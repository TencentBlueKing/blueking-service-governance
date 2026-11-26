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

// Package apierrs 把 env 领域错误映射为 API 错误的规则，供 env 以及会创建环境的 workspace 等 handler 共用。
//
// 环境不存在、环境名重复由 store 声明错误类别、错误出口统一转换为 404 / 409，这里无需重复映射
package apierrs

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var (
	// HasAppsCannotUpdateCluster 环境下仍有应用时不允许修改集群
	HasAppsCannotUpdateCluster = bkerrs.WhenAs(func(err error, e *bkmsenv.EnvHasAppsError) *bkerrs.Error {
		return bkerrs.Wrapf(
			err,
			bkerrs.ErrCodeInvalidRequest,
			i18n.Msg("environment has %d deployed apps, cannot update cluster"),
			e.AppCount,
		)
	})

	// SharedNamespaceCannotUpdateCluster 共用来源环境命名空间的特性环境不能修改集群或命名空间，只能重建
	SharedNamespaceCannotUpdateCluster = bkerrs.WhenIs(
		envmodel.ErrSharedNamespaceUnsupported,
		bkerrs.ErrCodeInvalidRequest,
		i18n.Msg(
			"environments sharing a namespace cannot change cluster or namespace, recreate the environment instead",
		),
	)

	// HasNamespaceSharersCannotUpdateCluster 仍有特性环境共用该环境的命名空间时不允许修改集群或命名空间
	HasNamespaceSharersCannotUpdateCluster = bkerrs.WhenIs(
		bkmsenv.ErrEnvHasNamespaceSharers,
		bkerrs.ErrCodeInvalidRequest,
		i18n.Msg("namespace is shared by feature environments, delete them before changing cluster or namespace"),
	)

	// HasAppsCannotDelete 环境下仍有应用时不允许删除
	HasAppsCannotDelete = bkerrs.WhenAs(func(err error, e *bkmsenv.EnvHasAppsError) *bkerrs.Error {
		return bkerrs.Wrapf(
			err,
			bkerrs.ErrCodeInvalidRequest,
			i18n.Msg("environment has %d deployed apps, cannot delete"),
			e.AppCount,
		)
	})

	// HasFeatureEnvsCannotDelete 仍存在派生自该环境的特性环境时不允许删除来源环境
	HasFeatureEnvsCannotDelete = bkerrs.WhenIs(
		bkmsenv.ErrEnvHasFeatureEnvs,
		bkerrs.ErrCodeInvalidRequest,
		i18n.Msg("environment has derived feature environments, delete them first"),
	)

	// InvalidFeatureEnvInput 创建特性环境的入参不符合业务约束
	InvalidFeatureEnvInput = bkerrs.WhenIs(
		bkmsenv.ErrInvalidFeatureEnvInput, bkerrs.ErrCodeInvalidRequest, i18n.Msg("invalid feature environment input"),
	)

	// FeatureEnvNameConflict 特性环境名由系统生成，与已有环境重名属于内部错误，不能按 store 类别返回 409
	FeatureEnvNameConflict = bkerrs.WhenIs(
		envmodel.ErrEnvAlreadyExists,
		bkerrs.ErrCodeInternalServerError,
		"generated feature environment name conflicts",
	)
)

// NamespaceOccupied 集群命名空间已被其他环境占用，错误详情附带占用方信息供前端展示
func NamespaceOccupied(err error) *bkerrs.Error {
	info, ok := bkmsenv.GetEnvClusterNamespaceConflictInfo(err)
	if !ok {
		return nil
	}
	return bkerrs.Wrapf(
		err, bkerrs.ErrCodeAlreadyExists,
		i18n.Msg("cluster %s namespace %s is already bound to environment %q in workspace %s"),
		info.ClusterID, info.Namespace, info.OccupiedByEnvName, info.OccupiedByWorkspaceID,
	).SetDetails(
		bkerrs.NewDetail(
			bkerrs.ErrDetailCodeEnvClusterNamespaceOccupied,
			"cluster namespace is already bound to another environment",
			bkerrs.WithSystem(bkerrs.SystemName),
			bkerrs.WithModule("env"),
			bkerrs.WithExtras(map[string]string{
				"clusterID":             info.ClusterID,
				"namespace":             info.Namespace,
				"occupiedByEnvName":     info.OccupiedByEnvName,
				"occupiedByWorkspaceID": info.OccupiedByWorkspaceID,
			}),
		),
	)
}
