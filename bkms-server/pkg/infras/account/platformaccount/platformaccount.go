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

// Package platformaccount 提供平台级虚拟账号的身份，用于在 BKMS 完成鉴权后调用外部系统
package platformaccount

import (
	"context"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// Get 返回虚拟账号的身份，未启用时 ok 为 false
func Get() (auth.User, bool) {
	// 配置加载前 config.G 为空
	if config.G == nil || !config.G.PlatformAccount.Enabled {
		return auth.User{}, false
	}
	cfg := config.G.PlatformAccount
	return auth.User{ID: cfg.Username, Cred: auth.UserCredential{AccessToken: cfg.AccessToken}}, true
}

// GetCaller 获取调用外部系统使用的身份：虚拟账号，未启用时为当前登录用户
func GetCaller(ctx context.Context) (auth.User, error) {
	if account, ok := Get(); ok {
		return account, nil
	}
	return auth.GetUser(ctx)
}

// MustGetCaller 获取调用外部系统使用的身份，如果失败则 panic
func MustGetCaller(ctx context.Context) auth.User {
	if account, ok := Get(); ok {
		return account
	}
	return auth.MustGetUser(ctx)
}
