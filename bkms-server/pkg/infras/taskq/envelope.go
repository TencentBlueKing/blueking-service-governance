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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// payloadEnvelope 把身份与业务 Args 分开放进 asynq payload
// 字段名前缀 `_` 避免与业务 Args 字段冲突；身份禁止放进业务 Args
type payloadEnvelope struct {
	AuthUser auth.User       `json:"_authUser"`
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

// wrapEnvelope 把用户身份与业务 Args JSON 打成 envelope，身份不进入业务 Args
func wrapEnvelope(user auth.User, argsPayload []byte) ([]byte, error) {
	return json.Marshal(payloadEnvelope{AuthUser: user, Args: argsPayload})
}

// restoreEnvelope 从 envelope 恢复用户到 ctx，并抽出业务 Args JSON
// 没有 `_args` 的旧 payload 原样返回，兼容单测与存量任务
func restoreEnvelope(ctx context.Context, payload []byte) (context.Context, []byte) {
	var env payloadEnvelope
	if err := json.Unmarshal(payload, &env); err != nil || len(env.Args) == 0 {
		return ctx, payload
	}
	if env.AuthUser.ID != "" {
		ctx = auth.WithUser(ctx, env.AuthUser)
	}
	return ctx, env.Args
}
