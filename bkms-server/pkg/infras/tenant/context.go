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
