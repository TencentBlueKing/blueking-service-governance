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

package store

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

var (
	// ErrTenantIDRequired 需要租户但没有拿到：ctx 中没有租户，或 ForTenant 传入了空字符串。
	// 数据层不回落到默认租户，避免漏挂中间件时静默写出跨租户的数据
	ErrTenantIDRequired = errors.New("tenant id is required")
	// ErrTenantScopeDuplicated 一次操作里传入了多个租户选项
	ErrTenantScopeDuplicated = errors.New("tenant scope is specified more than once")
	// ErrTenantScopeNotSupported 在非租户集合上传入了租户选项，通常意味着调用方误解了该集合的属性
	ErrTenantScopeNotSupported = errors.New(
		"tenant scope is not supported by a collection that is not tenant isolated",
	)
	// ErrTenantScopeCombined 租户选项被 Combine 包了起来。
	// 跨租户访问必须留在调用点上，不能藏进领域包的具名 Opt 里
	ErrTenantScopeCombined = errors.New("tenant scope cannot be wrapped in Combine, pass it at the call site")
	// ErrTenantScopeRequired ctx 被 RequireExplicitTenantScope 标记过，但操作没有传 AllTenants / ForTenant
	ErrTenantScopeRequired = errors.New("tenant scope must be specified explicitly, use AllTenants() or ForTenant()")
)

type explicitTenantScopeKey struct{}

// RequireExplicitTenantScope 标记 ctx：其上对租户隔离集合的操作都必须显式传 AllTenants / ForTenant，
// 不再使用 ctx 中的租户，漏传时返回 ErrTenantScopeRequired。
//
// 用于平台管理这类 ctx 里带着操作者租户、业务上却要跨租户的请求：漏传 AllTenants 时，
// 查询会静默只返回操作者所在租户的数据，标记把这种情况变成报错。
// 标记只要求显式选择范围，不授予跨租户能力，跨租户仍须在调用点写 AllTenants
func RequireExplicitTenantScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, explicitTenantScopeKey{}, true)
}

func explicitTenantScopeRequired(ctx context.Context) bool {
	required, _ := ctx.Value(explicitTenantScopeKey{}).(bool)
	return required
}

// TenantOpt 租户范围，由 AllTenants / ForTenant 创建，决定本次操作按哪个租户过滤。
// 不传时租户取自 ctx，取不到则返回 ErrTenantIDRequired；
// ctx 被 RequireExplicitTenantScope 标记过时必须传，否则返回 ErrTenantScopeRequired
type TenantOpt struct {
	all bool
	id  string
}

func (o TenantOpt) kind() optKind {
	if o.all {
		return kindTenantAll
	}
	return kindTenantFixed
}

func (o TenantOpt) apply(q *query) {
	q.tenantOpts++
	q.tenantAll = o.all
	q.tenantID = o.id
}

// AllTenants 跨租户访问，不按租户过滤，用于平台管理与运维命令。
//
// 只有读、局部更新（Update / Modify）与删除接受它；写入整个文档的操作（Create / Replace / Upsert）
// 不接受，因为新文档的 tenantID 必须是一个确定的值。例如：
//
//	page, err := workspaces.ListPage(ctx, page, size, store.AllTenants())
func AllTenants() TenantOpt {
	return TenantOpt{all: true}
}

// ForTenant 指定租户，忽略 ctx 中的租户，
// 用于 ctx 里没有租户的场景，如运维命令按租户遍历。tenantID 为空时返回 ErrTenantIDRequired
func ForTenant(tenantID string) TenantOpt {
	return TenantOpt{id: tenantID}
}

// resolveTenant 返回本次操作要用的租户。
// ok 为 false 表示不按租户过滤：集合本身不隔离，或调用方显式要求了跨租户
func (s *GenericStore[T]) resolveTenant(ctx context.Context, op string, q *query) (string, bool, error) {
	fail := func(err error) (string, bool, error) {
		return "", false, errors.Wrapf(err, "%s %s", op, s.coll.Name())
	}
	if q.tenantOpts > 1 {
		return fail(ErrTenantScopeDuplicated)
	}
	if !s.meta.TenantIsolated {
		// 非租户集合上指定租户不静默忽略：调用方写出来就是理解错了集合的属性
		if q.tenantOpts > 0 {
			return fail(ErrTenantScopeNotSupported)
		}
		return "", false, nil
	}
	if q.tenantAll {
		return "", false, nil
	}
	if q.tenantOpts > 0 {
		if q.tenantID == "" {
			return fail(ErrTenantIDRequired)
		}
		return q.tenantID, true, nil
	}
	if explicitTenantScopeRequired(ctx) {
		return fail(ErrTenantScopeRequired)
	}
	id, found := tenant.GetTenantID(ctx)
	if !found {
		return fail(ErrTenantIDRequired)
	}
	return id, true, nil
}

// injectTenant 把租户条件加入已解析的过滤条件。
//
// 调用方自己写 tenantID 条件会被拒绝，而不是被覆盖：条件不能靠追加来覆盖（query.filter 遇到重复 key
// 会退化成 $and，调用方的条件仍会参与匹配），而静默丢弃又会让「以为自己在按租户过滤」的代码继续跑。
// AllTenants 下不做这个检查，此时按 tenantID 过滤是平台管理筛选租户的正常用法
func (s *GenericStore[T]) injectTenant(ctx context.Context, op string, q *query) error {
	id, ok, err := s.resolveTenant(ctx, op, q)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if q.hasKey(keyTenantID) {
		return errors.Errorf(
			"%s %s: %s is maintained by the store and cannot be used as a condition, "+
				"use AllTenants() or ForTenant() to choose the tenant",
			op, s.coll.Name(), keyTenantID,
		)
	}
	q.conds = append(q.conds, bson.E{Key: keyTenantID, Value: id})
	return nil
}

// tenantForWrite 返回写入整个文档时要填入的租户，非租户集合返回空字符串。
// 这类操作不接受 AllTenants，由各操作的 optKind 挡住，这里只做兜底
func (s *GenericStore[T]) tenantForWrite(ctx context.Context, op string, q *query) (string, error) {
	id, ok, err := s.resolveTenant(ctx, op, q)
	if err != nil {
		return "", err
	}
	if !ok && s.meta.TenantIsolated {
		return "", errors.Errorf("%s %s: cannot write a whole document across tenants", op, s.coll.Name())
	}
	return id, nil
}
