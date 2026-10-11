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

package dal

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

const (
	// scopeTenant 按本次访问解析租户。全局表不会进入这个状态。
	scopeTenant = "tenant"
	// scopeCrossTenant 本次访问跳过注入，用于平台管理和迁移命令。
	scopeCrossTenant = "cross_tenant"
)

// ErrScopeRequired 表示调用 Scopes 时没有传入范围。
var ErrScopeRequired = errors.New("scope is required")

// ErrScopeAlreadySet 表示对已经设置过范围的访问再次设置范围。
var ErrScopeAlreadySet = errors.New("scope can only be set once")

// Table 是一张表上的查询会话。租户在 One/All/Count/Insert/Update/Delete/Aggregate 执行时注入。
// ForTenant / AllTenants / FromContext 只能调用其中一个且只能一次。再次调用时，后续操作返回 ErrScopeAlreadySet。
type Table interface {
	ForTenant(id string) Table
	AllTenants() Table
	FromContext() Table
	TenantID(ctx context.Context) (string, bool)
	Find(filter any) Query
	Insert(ctx context.Context, document any) error
	Update(ctx context.Context, filter, update any) error
	Delete(ctx context.Context, filter any) error
	Aggregate(ctx context.Context, pipeline, result any) error
}

type mongoTable struct {
	inner             *mongo.Collection
	name              string
	scope             string
	specifiedTenantID *string
	// scoped 表示已经设置过范围
	scoped bool
	// err 是设置范围时记下的错误，在后续操作返回。
	err error
}

func newTable(name string, collection *mongo.Collection) *mongoTable {
	scope := scopeTenant
	if isGlobal(name) {
		scope = scopeCrossTenant
	}
	return &mongoTable{inner: collection, name: name, scope: scope}
}

func (t *mongoTable) ForTenant(id string) Table {
	cloned := t.rescope()
	if isGlobal(cloned.name) {
		return cloned
	}
	cloned.scope = scopeTenant
	cloned.specifiedTenantID = &id
	return cloned
}

func (t *mongoTable) AllTenants() Table {
	cloned := t.rescope()
	if isGlobal(cloned.name) {
		return cloned
	}
	cloned.scope = scopeCrossTenant
	cloned.specifiedTenantID = nil
	return cloned
}

func (t *mongoTable) FromContext() Table {
	cloned := t.rescope()
	if isGlobal(cloned.name) {
		return cloned
	}
	cloned.scope = scopeTenant
	cloned.specifiedTenantID = nil
	return cloned
}

func (t *mongoTable) Find(filter any) Query {
	return &mongoQuery{table: t, filter: filter}
}

func (t *mongoTable) Insert(ctx context.Context, document any) error {
	if err := t.scopeErr(); err != nil {
		return err
	}
	doc := document
	if t.shouldInject() {
		tenantID, err := t.requireTenant(ctx)
		if err != nil {
			return err
		}
		doc, err = setTenantField(document, tenantID)
		if err != nil {
			return t.wrapErr(err)
		}
	}
	_, err := t.inner.InsertOne(ctx, doc)
	if err != nil {
		return err
	}
	return nil
}

func (t *mongoTable) Update(ctx context.Context, filter, update any) error {
	if err := t.scopeErr(); err != nil {
		return err
	}
	if t.shouldInject() {
		tenantID, err := t.requireTenant(ctx)
		if err != nil {
			return err
		}
		if err = ensureTenantImmutable(update); err != nil {
			return t.wrapErr(err)
		}
		filter, err = setTenantField(filter, tenantID)
		if err != nil {
			return t.wrapErr(err)
		}
	}
	_, err := t.inner.UpdateOne(ctx, filter, update)
	return err
}

func (t *mongoTable) Delete(ctx context.Context, filter any) error {
	if err := t.scopeErr(); err != nil {
		return err
	}
	if t.shouldInject() {
		tenantID, err := t.requireTenant(ctx)
		if err != nil {
			return err
		}
		filter, err = setTenantField(filter, tenantID)
		if err != nil {
			return t.wrapErr(err)
		}
	}
	_, err := t.inner.DeleteOne(ctx, filter)
	return err
}

func (t *mongoTable) Aggregate(ctx context.Context, pipeline, result any) error {
	if err := t.scopeErr(); err != nil {
		return err
	}
	stages := pipeline
	if t.shouldInject() {
		tenantID, err := t.requireTenant(ctx)
		if err != nil {
			return err
		}
		stages, err = applyPipeline(pipeline, tenantID)
		if err != nil {
			return t.wrapErr(err)
		}
	}
	cursor, err := t.inner.Aggregate(ctx, stages)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	if err = cursor.All(ctx, result); err != nil {
		return t.wrapErr(errors.Wrap(err, "decode documents"))
	}
	return nil
}

// TenantID 返回本次写入会使用的租户。不注入时返回 false。
func (t *mongoTable) TenantID(ctx context.Context) (string, bool) {
	if !t.shouldInject() {
		return "", false
	}
	if t.specifiedTenantID != nil {
		if *t.specifiedTenantID == "" {
			return "", false
		}
		return *t.specifiedTenantID, true
	}
	return tenant.GetTenantID(ctx)
}

func (t *mongoTable) shouldInject() bool {
	return t.scope == scopeTenant
}

func (t *mongoTable) requireTenant(ctx context.Context) (string, error) {
	if t.specifiedTenantID != nil {
		if *t.specifiedTenantID == "" {
			return "", tenant.ErrTenantIDRequired
		}
		return *t.specifiedTenantID, nil
	}
	tenantID, ok := tenant.GetTenantID(ctx)
	if !ok {
		return "", tenant.ErrTenantIDRequired
	}
	return tenantID, nil
}

// rescope 返回标记为已设置范围的副本。已经设置过范围时记下 ErrScopeAlreadySet。
func (t *mongoTable) rescope() *mongoTable {
	cloned := *t
	if t.scoped {
		if cloned.err == nil {
			cloned.err = ErrScopeAlreadySet
		}
		return &cloned
	}
	cloned.scoped = true
	return &cloned
}

func (t *mongoTable) scopeErr() error {
	if t == nil || t.err == nil {
		return nil
	}
	return t.wrapErr(t.err)
}

func (t *mongoTable) wrapErr(err error) error {
	if err == nil {
		return nil
	}
	return errors.Wrapf(err, "table %q", t.name)
}
