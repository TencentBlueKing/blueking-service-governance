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

package app

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// CheckVisibleEnv 按应用可见环境名单拦截部署目标环境。
// 名单为空表示未配置，不做限制；应用自己的特性环境无需写入名单，始终允许。
// 仅用于创建部署（含构建后自动部署）和 precheck，卸载、回滚、查记录、实例更新不做此校验。
func (a *Application) CheckVisibleEnv(env *envmodel.Environment) error {
	if len(a.VisibleEnvNames) == 0 {
		return nil
	}
	if env.IsFeatureEnv() && env.OwnerAppID == a.ID {
		return nil
	}
	if lo.Contains(a.VisibleEnvNames, env.Name) {
		return nil
	}
	return bkerrs.Errorf(
		bkerrs.ErrCodeInvalidRequest,
		"environment %q is not in the application's visible environments",
		env.Name,
	)
}
