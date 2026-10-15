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
// 索引（由 golang-migrate 维护）：
// - 唯一：name + version
var ComponentDefMeta = store.Meta[ComponentDef]{
	Collection: "component_defs",
	NotFound:   ErrComponentDefNotFound,
	Duplicate:  ErrComponentDefAlreadyExists,
	BeforeSave: []func(*ComponentDef) error{validateDef},
	Encode:     []func(*ComponentDef) error{defaultEmptyFragments},
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

// UpsertDef 按 name + version 创建或覆盖组件定义；实例计数由 UpdateInstanceCount 维护，不会被覆盖
func UpsertDef(ctx context.Context, defs *ComponentDefStore, compDef *ComponentDef) error {
	if _, err := defs.Upsert(
		ctx, compDef,
		store.UpdateOmit("appCompInstanceCount", "workspaceCompInstanceCount"),
		ByNameVersion(compDef.Name, compDef.Version),
	); err != nil {
		return errors.Wrapf(err, "upsert component def %s:%s", compDef.Name, compDef.Version)
	}
	return nil
}

// UpdateInstanceCount 原子更新组件定义的实例计数。
// 使用 MongoDB $inc 操作符实现原子递增/递减，适用于并发场景。
//
// - ctx: The context object for cancellation and timeout
// - name: 组件定义名称（使用默认版本 DefaultComponentDefVersion）
// - instanceType: 计数字段，FieldAppCompInstance 或 FieldWorkspaceCompInstance
// - delta: 增量，正数表示增加，负数表示减少
func UpdateInstanceCount(
	ctx context.Context, defs *ComponentDefStore, name string, instanceType InstanceType, delta int,
) error {
	var fieldName string
	switch instanceType {
	case FieldAppCompInstance:
		fieldName = "appCompInstanceCount"
	case FieldWorkspaceCompInstance:
		fieldName = "workspaceCompInstanceCount"
	default:
		return errors.Errorf("invalid instance type: %s", instanceType)
	}
	// 计数不是对组件定义的编辑，不刷新 updatedAt；组件定义不存在时静默忽略
	_, err := defs.Modify(ctx, bson.M{"$inc": bson.M{fieldName: delta}},
		store.Where("name", name), store.Where("version", DefaultComponentDefVersion), store.KeepUpdatedAt())
	if err = store.IgnoreNotFound(err); err != nil {
		return errors.Wrapf(err, "incr ref count for component def(%s) field(%s) delta(%d)", name, instanceType, delta)
	}
	return nil
}
