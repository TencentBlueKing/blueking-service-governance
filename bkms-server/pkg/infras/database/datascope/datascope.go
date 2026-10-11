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

// Package datascope 提供 Store 的数据访问范围。
// Store 持有原生 *mongo.Collection，在每个方法里显式调用 TenantFilter / TenantForWrite 完成租户过滤和写入。
package datascope

import (
	"context"
	"maps"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// ErrWriteAllTenants 表示在 AllTenants 范围下写入。写入必须落到一个确定的租户。
var ErrWriteAllTenants = errors.New("write requires a tenant, but scope is AllTenants")

// ErrScopeRequired 表示调用 Scopes 时没有传入范围。
var ErrScopeRequired = errors.New("scope is required")

// ErrScopeAlreadySet 表示对已经设置过范围的访问再次设置范围。
var ErrScopeAlreadySet = errors.New("scope can only be set once")

const (
	// modeContext 每次执行时从 ctx 读租户。零值，不调用 Scopes 时就是它。
	modeContext = ""
	// modeTenant 使用 ForTenant 指定的租户。
	modeTenant = "tenant"
	// modeAllTenants 不限定租户。
	modeAllTenants = "all_tenants"
)

// Scope 是一次访问的租户范围，由 Store 作为字段持有。零值从 ctx 读租户。
type Scope struct {
	mode     string
	tenantID string
	// scoped 表示已经设置过范围
	scoped bool
	// err 是设置范围时记下的错误，在 TenantFilter / TenantForWrite 返回。
	err error
}

// Option 修改租户范围。
type Option func(*Scope)

// ForTenant 限定为指定租户，不再读 ctx。
func ForTenant(id string) Option {
	return func(s *Scope) {
		s.mode = modeTenant
		s.tenantID = id
	}
}

// FromContext 每次执行时从 ctx 读租户，与不调用 Scopes 时的默认行为相同。
func FromContext() Option {
	return func(s *Scope) {
		s.mode = modeContext
		s.tenantID = ""
	}
}

// AllTenants 不限定租户。只用于平台管理和迁移命令。
func AllTenants() Option {
	return func(s *Scope) {
		s.mode = modeAllTenants
		s.tenantID = ""
	}
}

// ApplyScope 返回应用了 opt 的新 Scope，供 Store 实现 Scopes 时调用。
// opt 为 nil，或 scope 已经设置过范围时，不在这里报错，错误会在后续 TenantFilter / TenantForWrite 返回。
func ApplyScope(scope Scope, opt Option) Scope {
	if opt == nil {
		if scope.err == nil {
			scope.err = ErrScopeRequired
		}
		return scope
	}
	if scope.scoped {
		if scope.err == nil {
			scope.err = ErrScopeAlreadySet
		}
		return scope
	}
	opt(&scope)
	scope.scoped = true
	return scope
}

// TenantFilter 返回带租户条件的 filter 副本，调用方传入的 tenant_id 会被覆盖。
// AllTenants 范围下返回不加租户条件的副本。
func (s Scope) TenantFilter(ctx context.Context, filter bson.M) (bson.M, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := bson.M{}
	maps.Copy(result, filter)
	if s.mode == modeAllTenants {
		return result, nil
	}
	tenantID, err := s.resolve(ctx)
	if err != nil {
		return nil, err
	}
	result[tenant.FieldTenantID] = tenantID
	return result, nil
}

// TenantForWrite 返回写入时使用的租户。AllTenants 范围下返回 ErrWriteAllTenants。
func (s Scope) TenantForWrite(ctx context.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.mode == modeAllTenants {
		return "", ErrWriteAllTenants
	}
	return s.resolve(ctx)
}

func (s Scope) resolve(ctx context.Context) (string, error) {
	if s.mode == modeTenant {
		if s.tenantID == "" {
			return "", tenant.ErrTenantIDRequired
		}
		return s.tenantID, nil
	}
	tenantID, ok := tenant.GetTenantID(ctx)
	if !ok {
		return "", tenant.ErrTenantIDRequired
	}
	return tenantID, nil
}
