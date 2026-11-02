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

// Package tenant provides tenant-aware request context helpers and middleware.
package tenant

import (
	"context"
	"net/http"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/ctxkey"
)

// WithTenantID stores the resolved tenant ID into context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, ctxkey.TenantID, tenantID)
}

// GetTenantID returns the tenant ID from context.
func GetTenantID(ctx context.Context) (string, bool) {
	val := ctx.Value(ctxkey.TenantID)
	if val == nil {
		return "", false
	}
	tenantID, ok := val.(string)
	if !ok || tenantID == "" {
		return "", false
	}
	return tenantID, true
}

// ScopedKey 为租户内资源的 Redis key（锁、缓存）加上 ctx 租户前缀，格式为 tenant:{tenantID}:{key}。
// ctx 中没有租户时返回 ErrTenantIDRequired，不回落到 default。
//
// 只用于归属某个租户的资源（工作空间、应用等）。对应物理资源的 key（如集群、命名空间）不要使用：
// 同一个集群可能被多个租户共用，锁和缓存需要跨租户共享，加前缀反而会让不同租户并发操作同一资源。
func ScopedKey(ctx context.Context, key string) (string, error) {
	tenantID, ok := GetTenantID(ctx)
	if !ok {
		return "", errors.Wrapf(ErrTenantIDRequired, "scope key %s", key)
	}
	return "tenant:" + tenantID + ":" + key, nil
}

// TenantHeader 返回调用蓝鲸平台接口时附带的租户请求头（X-Bk-Tenant-Id），值取自 ctx 租户。
// ctx 中没有租户（如全局任务、运维命令）时返回 nil，不回落到 default：
// 多租户模式下下游会因缺少租户头而拒绝，问题能及时暴露，而不是把请求算到 default 租户上。
func TenantHeader(ctx context.Context) map[string]string {
	tenantID, ok := GetTenantID(ctx)
	if !ok {
		return nil
	}
	return map[string]string{HeaderTenantID: tenantID}
}

// OptBkApiTenantHeader 返回蓝鲸网关 client 的构造选项：每次请求发出前，按请求 ctx 设置 TenantHeader。
// 请求 ctx 来自 Operation.SetContext，未调用 SetContext 时取不到租户，不带租户头。
// 请求已显式设置租户头时保持原值，例如 bk-user 需要在待校验的租户内查询用户。
func OptBkApiTenantHeader() define.BkApiOption {
	return bkapi.OptRequestCallback(func(req *http.Request) *http.Request {
		for key, value := range TenantHeader(req.Context()) {
			if req.Header.Get(key) == "" {
				req.Header.Set(key, value)
			}
		}
		return req
	})
}
