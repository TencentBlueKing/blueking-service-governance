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

package testutil

import (
	"context"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// WithTenantOrDefault 返回带租户的 ctx：ctx 已有租户时原样返回，没有时补上默认租户。
//
// TenantScoped 集合（Meta.TenantScoped 为 true）要求 ctx 里有租户，
// 这类集合的用例用 WithTenantOrDefault(context.Background()) 替代 context.Background()。
// 验证租户过滤本身的用例应直接用 tenant.WithTenantID 构造两个不同租户的 ctx
func WithTenantOrDefault(ctx context.Context) context.Context {
	if _, ok := tenant.GetTenantID(ctx); ok {
		return ctx
	}
	return tenant.WithTenantID(ctx, tenant.DefaultTenantID)
}
