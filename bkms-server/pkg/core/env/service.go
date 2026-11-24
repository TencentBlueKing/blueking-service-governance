/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package env

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrEnvHasNamespaceSharers 仍有特性环境共用该环境的命名空间，不能修改集群或命名空间
	ErrEnvHasNamespaceSharers = errors.New("environment namespace is shared by feature environments")
	// ErrEnvHasFeatureEnvs 仍存在派生自该环境的特性环境，不能删除来源环境
	ErrEnvHasFeatureEnvs = errors.New("environment has derived feature environments")
)

// EnvService 环境基础配置服务
type EnvService struct {
	db *mongo.Database
}

// NewEnvService 创建环境基础配置服务
func NewEnvService(db *mongo.Database) *EnvService {
	return &EnvService{db: db}
}

// ImageRegistryInfoInput 镜像仓库信息
type ImageRegistryInfoInput struct {
	// Registry 镜像仓库地址
	Registry string
	// Username 镜像仓库用户名
	Username string
	// Password 镜像仓库密码
	Password string
}

// Create 创建环境数据，成功后 environment.ID 为新环境的 ID.
func (s *EnvService) Create(ctx context.Context, environment *model.Environment) error {
	if err := findClusterNamespaceOccupant(
		ctx, s.db, environment.Cluster, environment.NsSharing.ResPrefix, bson.NilObjectID,
	); err != nil {
		return errors.Wrap(err, "check environment cluster namespace conflict")
	}

	if err := store.Of(s.db, model.EnvMeta).Create(ctx, environment); err != nil {
		err = rewriteWriteErrIfClusterOccupied(
			ctx, s.db, environment.Cluster, environment.NsSharing.ResPrefix, bson.NilObjectID, err,
		)
		return errors.Wrapf(err, "create environment %s", environment.Name)
	}
	return nil
}

// Update 更新指定环境。
func (s *EnvService) Update(
	ctx context.Context,
	envID bson.ObjectID,
	updateData *model.EnvironmentUpdateData,
) error {
	environment, err := store.Of(s.db, model.EnvMeta).GetByID(ctx, envID)
	if err != nil {
		return errors.Wrap(err, "get environment")
	}

	finalCluster := applyClusterUpdate(environment.Cluster, updateData)

	// 更新集群信息时, 需要检查环境是否有部署应用
	if updateData.ClusterID != nil || updateData.Namespace != nil {
		// 共用命名空间的关系在创建时确定，迁移会绕过同一来源环境、同一工作空间的前提，只能重建
		if environment.SharesNamespace() && namespaceChanged(environment.Cluster, finalCluster) {
			return errors.Wrapf(
				model.ErrSharedNamespaceUnsupported,
				"cannot change cluster or namespace of env %s, recreate it instead", environment.Name,
			)
		}
		appCount := len(environment.AppIDs)
		if appCount != 0 {
			return errors.Errorf("environment has %d apps, cannot update cluster", appCount)
		}

		if err = s.checkNoNamespaceSharers(ctx, environment, finalCluster); err != nil {
			return err
		}
		if err = findClusterNamespaceOccupant(
			ctx, s.db, finalCluster, environment.NsSharing.ResPrefix, envID,
		); err != nil {
			return errors.Wrap(err, "check environment cluster namespace conflict")
		}
	}

	if err = model.UpdateEnv(ctx, s.db, envID, updateData); err != nil {
		if updateData.ClusterID != nil || updateData.Namespace != nil {
			err = rewriteWriteErrIfClusterOccupied(
				ctx, s.db, finalCluster, environment.NsSharing.ResPrefix, envID, err,
			)
		}
		return errors.Wrap(err, "update environment")
	}

	updatedEnv, err := store.Of(s.db, model.EnvMeta).GetByID(ctx, envID)
	if err != nil {
		return errors.Wrap(err, "get updated environment")
	}

	// Update hooks 采用异步后置通知语义：环境更新成功后再触发，失败仅记录日志，不回滚主流程。
	go func(before, after model.Environment) {
		if hookErr := runUpdateHooks(context.WithoutCancel(ctx), before, after); hookErr != nil {
			log.Errorf(
				context.WithoutCancel(ctx),
				"run env update hooks failed, envID=%s envName=%s err=%v",
				after.ID.Hex(),
				after.Name,
				hookErr,
			)
		}
	}(*environment, *updatedEnv)

	return nil
}

// checkNoNamespaceSharers 来源环境的集群或命名空间实际发生变化时，若仍有特性环境共用其命名空间则拒绝：
// 这些特性环境的资源留在原命名空间，来源环境迁走后二者就不再共用。
func (s *EnvService) checkNoNamespaceSharers(
	ctx context.Context, environment *model.Environment, finalCluster model.BizCluster,
) error {
	current := environment.Cluster
	if !namespaceChanged(current, finalCluster) {
		return nil
	}
	sharers, err := model.ListNamespaceSharers(ctx, s.db, current.ClusterID, current.Namespace)
	if err != nil {
		return errors.Wrap(err, "list feature environments sharing namespace")
	}
	if len(sharers) == 0 {
		return nil
	}
	names := lo.Map(sharers, func(e model.Environment, _ int) string { return e.Name })
	return errors.Wrapf(
		ErrEnvHasNamespaceSharers,
		"feature environments [%s] share namespace %s", strings.Join(names, ", "), current.Namespace,
	)
}

// namespaceChanged 判断集群或命名空间是否实际发生变化
func namespaceChanged(current, final model.BizCluster) bool {
	return current.ClusterID != final.ClusterID || current.Namespace != final.Namespace
}

// applyClusterUpdate 将 partial update 中的集群字段合并到现有 BizCluster，返回合并后的结果。
func applyClusterUpdate(cluster model.BizCluster, updateData *model.EnvironmentUpdateData) model.BizCluster {
	finalCluster := cluster
	if updateData.ClusterID != nil {
		finalCluster.ClusterID = *updateData.ClusterID
	}
	if updateData.ClusterType != nil {
		finalCluster.ClusterType = *updateData.ClusterType
	}
	if updateData.Namespace != nil {
		finalCluster.Namespace = *updateData.Namespace
	}
	if updateData.IsFederation != nil {
		finalCluster.IsFederation = *updateData.IsFederation
	}
	return finalCluster
}

// Delete 删除环境
func (s *EnvService) Delete(ctx context.Context, envID bson.ObjectID) error {
	environment, err := store.Of(s.db, model.EnvMeta).GetByID(ctx, envID)
	if err != nil {
		return errors.Wrap(err, "get environment")
	}

	appCount := len(environment.AppIDs)
	if appCount != 0 {
		return errors.Errorf("environment has %d apps, cannot delete", appCount)
	}

	if !environment.IsFeatureEnv() {
		featEnvs, listErr := store.Of(s.db, model.EnvMeta).List(ctx, model.FeatEnvsFromSource(envID))
		if listErr != nil {
			return errors.Wrap(listErr, "list feature environments derived from environment")
		}
		if len(featEnvs) > 0 {
			names := lo.Map(featEnvs, func(e model.Environment, _ int) string { return e.Name })
			return errors.Wrapf(
				ErrEnvHasFeatureEnvs,
				"environment %s has %d feature environments [%s], cannot delete",
				environment.Name, len(featEnvs), strings.Join(names, ", "),
			)
		}
	}

	// 删除环境往往意味着下游模块中的环境级资源也必须一起清理。这里在物理删除环境记录前执行 Hook：
	// - Hook 失败时保留环境，调用方可以修复下游数据后重试；
	// - Hook 成功后再删环境，避免留下“环境已不存在但下游清理无法再定位环境”的孤儿数据。
	if err = runDeleteHooks(ctx, *environment); err != nil {
		return err
	}

	if err = store.Of(s.db, model.EnvMeta).DeleteIfExists(ctx, store.ByID(envID)); err != nil {
		return errors.Wrapf(err, "delete environment %s", environment.Name)
	}

	return nil
}
