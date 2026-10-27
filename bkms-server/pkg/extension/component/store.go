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

// Package component storage module for the component package, it helps with component data persistence.
package component

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrComponentDefNotFound is returned when a component-def is not found in the store.
var ErrComponentDefNotFound = errors.New("component def not found")

// ErrComponentDefAlreadyExists is returned when creating a component-def whose name and version already exist.
var ErrComponentDefAlreadyExists = errors.New("component def already exists")

// ComponentDefStore is the storage of component-defs.
type ComponentDefStore = store.GenericStore[ComponentDef]

// ComponentDefMeta 组件定义的存储元数据，写入前校验完整模型。
//
// 集合按租户划分，平台租户中的内置定义（isBuiltin 为 true）对所有租户可读，写入只作用于本租户。
// 租户自建定义可能与内置定义同名，按名称取定义请使用 GetDef，不要直接 Get。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：tenantID + name + version
var ComponentDefMeta = store.Meta[ComponentDef]{
	Collection:         "component_defs",
	TenantScoped:       true,
	SharedFromPlatform: bson.D{{Key: "isBuiltin", Value: true}},
	NotFound:           ErrComponentDefNotFound,
	Duplicate:          ErrComponentDefAlreadyExists,
	BeforeSave:         []func(*ComponentDef) error{validateDef},
	Encode:             []func(*ComponentDef) error{defaultEmptyFragments},
}

func validateDef(compDef *ComponentDef) error {
	return errors.Wrap(ValidateComponentDef(compDef), "component validation failed")
}

// defaultEmptyFragments Patchers 和 Specs 允许任一侧为空，但数据库中始终以数组存储，避免 nil 被编码为 null。
func defaultEmptyFragments(compDef *ComponentDef) error {
	if compDef.Patchers == nil {
		compDef.Patchers = []string{}
	}
	if compDef.Specs == nil {
		compDef.Specs = []string{}
	}
	return nil
}

// ByNameVersion 按名称与版本匹配组件定义
func ByNameVersion(name, version string) store.Opt {
	return store.Filter(bson.M{"name": name, "version": version})
}

// ExcludeInvisible 过滤掉不可见的组件定义
func ExcludeInvisible() store.Opt {
	return store.Where("invisible", bson.M{"$ne": true})
}

// UsableInWorkspace 匹配工作空间可使用的组件定义（全局 + 该工作空间专属），workspaceID 为空时不过滤
func UsableInWorkspace(workspaceID string) store.Opt {
	if workspaceID == "" {
		return nil
	}
	return store.Where("$or", []bson.M{
		{"scopeType": ScopeTypeGlobal},
		{"scopeWorkspaceIDs": workspaceID},
	})
}

// ManagedByWorkspace 匹配可由工作空间管理的组件定义，workspaceID 为空时不过滤
func ManagedByWorkspace(workspaceID string) store.Opt {
	if workspaceID == "" {
		return nil
	}
	return store.Where("managedByWorkspaceIDs", workspaceID)
}

// KeywordMatches 按名称、展示名称模糊匹配组件定义，keyword 为空时不过滤
func KeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "name", "displayName")
}

// GetDef 按名称与版本获取当前租户可用的组件定义，不存在时返回 ErrComponentDefNotFound。
// 租户自建定义与平台内置定义同名时以自建的为准
func GetDef(ctx context.Context, defs *ComponentDefStore, name, version string) (*ComponentDef, error) {
	// 自建定义的 isBuiltin 总是 false，升序排在内置定义之前
	return defs.Get(ctx, ByNameVersion(name, version), store.Asc("isBuiltin"))
}

// BuiltinDefExists 判断是否存在同名的平台内置定义，租户新建定义时不能与之重名
func BuiltinDefExists(ctx context.Context, defs *ComponentDefStore, name, version string) (bool, error) {
	exists, err := defs.Exists(ctx, ByNameVersion(name, version), store.Where("isBuiltin", true))
	return exists, errors.Wrapf(err, "check builtin component def %s:%s", name, version)
}

// ExcludeShadowedBuiltins 返回过滤掉被本租户同名定义覆盖的内置定义的条件，没有被覆盖的内置定义时返回 nil。
// 新建定义时会拒绝与内置定义重名，但内置定义可能在租户自建之后才加入
func ExcludeShadowedBuiltins(ctx context.Context, defs *ComponentDefStore) (store.Opt, error) {
	builtins, err := defs.List(ctx, store.Where("isBuiltin", true), store.SelectFields("name", "version"))
	if err != nil {
		return nil, errors.Wrap(err, "list builtin component defs")
	}
	if len(builtins) == 0 {
		return nil, nil
	}
	names := lo.Uniq(lo.Map(builtins, func(d ComponentDef, _ int) string { return d.Name }))
	owned, err := defs.List(ctx,
		store.Where("isBuiltin", false), store.Where("name", bson.M{"$in": names}),
		store.SelectFields("name", "version"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "list component defs shadowing builtin ones")
	}
	if len(owned) == 0 {
		return nil, nil
	}
	shadowed := lo.Map(owned, func(d ComponentDef, _ int) bson.M {
		return bson.M{"isBuiltin": true, "name": d.Name, "version": d.Version}
	})
	return store.Where("$nor", shadowed), nil
}

// UpsertDef 按 name + version 创建或覆盖组件定义
func UpsertDef(ctx context.Context, defs *ComponentDefStore, compDef *ComponentDef, opts ...store.Opt) error {
	opts = append([]store.Opt{ByNameVersion(compDef.Name, compDef.Version)}, opts...)
	if _, err := defs.Upsert(ctx, compDef, opts...); err != nil {
		return errors.Wrapf(err, "upsert component def %s:%s", compDef.Name, compDef.Version)
	}
	return nil
}
