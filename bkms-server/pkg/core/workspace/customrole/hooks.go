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

package customrole

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
)

// PruneEnvHookName 环境删除时收敛自定义角色范围的 Hook 注册名
const PruneEnvHookName = "customrole.prune_env"

// RegisterHooks 注册环境删除 Hook。
//
// 环境类型变更的重授权在 UpdateEnvBasicInfo 中同步执行，避免异步 UpdateHook 返回前的短暂 403 竞态，
// 也避免 IAM 异常阻断后续告警策略协调 Hook。
func RegisterHooks(db *mongo.Database) {
	bkmsenv.RegisterDeleteHook(PruneEnvHookName, func(ctx context.Context, env envmodel.Environment) error {
		if err := PruneEnv(ctx, db, perm.NewManager(), &env); err != nil {
			return errors.Wrapf(err, "prune env %s from custom roles", env.Name)
		}
		return nil
	})
}
