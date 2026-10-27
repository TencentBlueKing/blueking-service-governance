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

package taskq

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// payloadEnvelope 把身份、租户与业务 Args 分开放进 asynq payload
// 字段名前缀 `_` 避免与业务 Args 字段冲突；身份与租户禁止放进业务 Args
type payloadEnvelope struct {
	AuthUser auth.User       `json:"_authUser"`
	TenantID string          `json:"_tenantID,omitempty"`
	Args     json.RawMessage `json:"_args"`
}

// userFromContext 从 ctx 取已认证用户；ID 为空视为未认证，禁止投递匿名任务
func userFromContext(ctx context.Context) (auth.User, error) {
	user, err := auth.GetUser(ctx)
	if err != nil || user.ID == "" {
		return auth.User{}, errors.New("taskq: auth user not found in context")
	}
	return user, nil
}

// tenantFromContext 从 ctx 取租户；缺租户直接失败，否则任务消费时会读不到按租户隔离的数据
func tenantFromContext(ctx context.Context) (string, error) {
	tenantID, ok := tenant.GetTenantID(ctx)
	if !ok {
		return "", errors.New("taskq: tenant id not found in context")
	}
	return tenantID, nil
}

// wrapEnvelope 把用户身份、租户与业务 Args JSON 打成 envelope，两者都不进入业务 Args；
// 全局任务的 tenantID 为空，envelope 中不出现 `_tenantID`
func wrapEnvelope(user auth.User, tenantID string, argsPayload []byte) ([]byte, error) {
	return json.Marshal(payloadEnvelope{AuthUser: user, TenantID: tenantID, Args: argsPayload})
}

// restoreEnvelope 从 envelope 恢复用户与租户到 ctx，并抽出业务 Args JSON
// 没有 `_args` 的旧 payload 原样返回，兼容单测与存量任务；
// global 为 true 时不恢复租户，即使 payload 中带有 `_tenantID`（升级前入队的任务）也忽略；
// todo 其余任务没有 `_tenantID` 时（多租户上线前已入队）回落到默认租户并打 warn，
// 投递端已强制写入租户，多租户上线后队列排空即可去掉这个回落
func restoreEnvelope(ctx context.Context, name string, payload []byte, global bool) (context.Context, []byte) {
	var env payloadEnvelope
	if err := json.Unmarshal(payload, &env); err != nil || len(env.Args) == 0 {
		return ctx, payload
	}
	if env.AuthUser.ID != "" {
		ctx = auth.WithUser(ctx, env.AuthUser)
	}
	if !global {
		if env.TenantID == "" {
			taskID, _, _ := handlerMeta(ctx)
			logging.Warnf(ctx, "taskq: task type=%s id=%s has no _tenantID, falling back to tenant %s",
				name, taskID, tenant.DefaultTenantID)
		}
		ctx = tenant.WithTenantID(ctx, lo.CoalesceOrEmpty(env.TenantID, tenant.DefaultTenantID))
	}
	return ctx, env.Args
}
