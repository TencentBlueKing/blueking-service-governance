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

package auth

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
)

const (
	// MaintenanceUserID 命令行维护任务使用的系统用户 ID
	MaintenanceUserID = "bkms-server-maintenance"
)

// WithUser 将已认证用户写入 context
func WithUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, ctxkey.AuthUser, user)
}

// WithMaintenanceUser 将命令行维护身份写入 context
func WithMaintenanceUser(ctx context.Context) context.Context {
	return WithUser(ctx, User{ID: MaintenanceUserID})
}

// GetUser 获取当前登录用户。
func GetUser(ctx context.Context) (User, error) {
	anonymous := User{}

	val := ctx.Value(ctxkey.AuthUser)
	if val == nil {
		return anonymous, errors.New("authed user not found")
	}
	user, ok := val.(User)
	if !ok {
		return anonymous, errors.New("authed user type error")
	}
	return user, nil
}

// MustGetUser 获取当前登录用户，如果失败则 panic。
func MustGetUser(ctx context.Context) User {
	user, err := GetUser(ctx)
	if err != nil {
		panic(err)
	}
	return user
}
