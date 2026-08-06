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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// EnvService 环境基础配置服务
type EnvService struct {
	model.EnvironmentStore
}

// NewEnvService 创建环境基础配置服务
func NewEnvService(environmentStore model.EnvironmentStore) *EnvService {
	return &EnvService{
		EnvironmentStore: environmentStore,
	}
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

// Create 创建环境数据.
func (s *EnvService) Create(ctx context.Context, environment *model.Environment) (bson.ObjectID, error) {
	envID, err := s.EnvironmentStore.Create(ctx, environment)
	if err != nil {
		return bson.NilObjectID, errors.Wrap(err, "create environment")
	}
	return envID, nil
}

// Update 更新环境
func (s *EnvService) Update(
	ctx context.Context,
	envID bson.ObjectID,
	updateData *model.EnvironmentUpdateData,
) error {
	// 更新集群信息时, 需要检查环境是否有部署应用
	if updateData.ClusterID != nil || updateData.Namespace != nil {
		environment, err := s.Get(ctx, envID)
		if err != nil {
			return errors.Wrap(err, "get environment")
		}

		appCount := len(environment.AppIDs)
		if appCount != 0 {
			return errors.Errorf("environment has %d apps, cannot update cluster", appCount)
		}
	}

	return s.EnvironmentStore.Update(ctx, envID, updateData)
}

// Delete 删除环境
func (s *EnvService) Delete(ctx context.Context, envID bson.ObjectID) error {
	environment, err := s.Get(ctx, envID)
	if err != nil {
		return errors.Wrap(err, "get environment")
	}

	appCount := len(environment.AppIDs)
	if appCount != 0 {
		return errors.Errorf("environment has %d apps, cannot delete", appCount)
	}

	// 删除环境往往意味着下游模块中的环境级资源也必须一起清理。这里在物理删除环境记录前执行 Hook：
	// - Hook 失败时保留环境，调用方可以修复下游数据后重试；
	// - Hook 成功后再删环境，避免留下“环境已不存在但下游清理无法再定位环境”的孤儿数据。
	if err = runDeleteHooks(ctx, *environment); err != nil {
		return err
	}

	if err = s.EnvironmentStore.Delete(ctx, envID); err != nil {
		return err
	}

	return nil
}
