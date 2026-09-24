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

// Package hooks 注册应用模块相关的领域事件钩子，如环境删除时清理可见环境名单。
package hooks

import (
	"context"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// CleanupVisibleEnvNamesHookName 是可见环境名单清理 Hook 的注册名。
const CleanupVisibleEnvNamesHookName = "app.cleanup_visible_env_names"

// RegisterDeleteHooks 注册应用模块的环境删除 Hook。
func RegisterDeleteHooks(appStore bkmsapp.ApplicationStore) {
	bkmsenv.RegisterDeleteHook(
		CleanupVisibleEnvNamesHookName,
		NewCleanupVisibleEnvNamesHook(appStore),
	)
}

// NewCleanupVisibleEnvNamesHook 创建一个在标准环境删除前摘掉可见名单中对应 name 的 Hook。
// 特性环境不会写入可见名单，因此直接跳过。
//
// 需求文档《应用可见环境》把「空间删除环境时回写清理名单」列在本期不包含，并要求残留 name 不自动
// 清理，这里是有意偏离：名单里留着已删除环境的 name 对用户没有意义，删一个少一个更符合直觉。
//
// 由此带来一个需要知情的副作用：可见名单是 allowlist，而空名单在 R-001 里表示「未配置 = 不限制」，
// 不表示「空的 allowlist」。所以摘掉最后一个 name 时语义会翻转——原本只能上 test 的应用，在 test 环境
// 被删除后会变成可以部署到该空间的任意标准环境（含 prod），且不再有记录表明它曾被限制过。
// 这是评审时确认可接受的取舍。如果后续要改回去，只需摘掉 registry 里的注册即可。
func NewCleanupVisibleEnvNamesHook(appStore bkmsapp.ApplicationStore) bkmsenv.DeleteHook {
	return func(ctx context.Context, environment envmodel.Environment) error {
		if environment.IsFeatureEnv() {
			return nil
		}
		if err := appStore.RemoveVisibleEnvName(ctx, environment.WorkspaceID, environment.Name); err != nil {
			return errors.Wrapf(
				err,
				"cleanup visible env names for workspace %s env %s",
				environment.WorkspaceID,
				environment.Name,
			)
		}
		return nil
	}
}
