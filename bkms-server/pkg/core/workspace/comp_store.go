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

package workspace

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// The name of the MongoDB collection for storing workspace component data.
const workspaceComponentCollectionName = "workspace_components"

var (
	// ErrComponentNotFound is returned when a component is not found in the store.
	ErrComponentNotFound = errors.New("component not found")
	// ErrComponentAlreadyExists 同一工作空间内组件重名
	ErrComponentAlreadyExists = errors.New("component already exists")
)

// ComponentUpdateData 定义了更新 Component 时允许修改的数据
type ComponentUpdateData struct {
	Version       *string
	Properties    map[string]any
	Name          *string
	ScopeType     *component.ScopeType
	ScopeEnvNames []string
}

// WorkspaceCompsStore 工作空间组件的 GenericStore
type WorkspaceCompsStore = store.GenericStore[Component]

// ComponentMeta 工作空间组件的存储元信息：创建时补齐名称与 Properties，读出时规范化 Properties 中的 BSON 类型。
//
// 组件增删会影响 ComponentDef 的引用计数，须通过 AddComponents / RemoveComponents / DeleteComponentsOf 完成。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + name
var ComponentMeta = store.Meta[Component]{
	Collection: workspaceComponentCollectionName,
	NotFound:   ErrComponentNotFound,
	Duplicate:  ErrComponentAlreadyExists,
	BeforeSave: []func(*Component) error{setComponentDefaults},
	Decode:     []func(*Component) error{normalizeComponentProperties},
}

// ComponentsIn 匹配工作空间内的组件
func ComponentsIn(workspaceID string) store.Opt { return store.Where("workspaceID", workspaceID) }

// ComponentByName 按工作空间与组件名称匹配
func ComponentByName(workspaceID, name string) store.Opt {
	return store.Combine(ComponentsIn(workspaceID), store.Where("name", name))
}

// AddComponents 向工作空间添加组件，并增加对应 ComponentDef 的引用计数
func AddComponents(ctx context.Context, db *mongo.Database, comps ...*Component) error {
	if len(comps) == 0 {
		return nil
	}
	if err := store.Of(db, ComponentMeta).CreateMany(ctx, comps); err != nil {
		return errors.Wrap(err, "add workspace components")
	}
	return adjustInstanceCounts(ctx, db, comps, 1)
}

// RemoveComponents 删除工作空间组件，并减少对应 ComponentDef 的引用计数；一个都没有删除时返回 ErrComponentNotFound
func RemoveComponents(ctx context.Context, db *mongo.Database, compIDs ...bson.ObjectID) error {
	if len(compIDs) == 0 {
		return nil
	}
	return removeComponents(ctx, db, store.Where("_id", bson.M{"$in": compIDs}), true)
}

// DeleteComponentsOf 删除工作空间下的全部组件，并减少对应 ComponentDef 的引用计数
func DeleteComponentsOf(ctx context.Context, db *mongo.Database, workspaceID string) error {
	return errors.Wrapf(
		removeComponents(ctx, db, ComponentsIn(workspaceID), false),
		"delete components of workspace %s", workspaceID,
	)
}

// removeComponents 先查出将被删除的组件以便按类型扣减引用计数，再删除
func removeComponents(ctx context.Context, db *mongo.Database, cond store.Opt, mustExist bool) error {
	comps := store.Of(db, ComponentMeta)
	removed, err := comps.List(ctx, cond)
	if err != nil {
		return errors.Wrap(err, "list components to remove")
	}
	deleted, err := comps.DeleteMany(ctx, cond)
	if err != nil {
		return errors.Wrap(err, "remove workspace components")
	}
	if deleted == 0 && mustExist {
		return ErrComponentNotFound
	}
	return adjustInstanceCounts(ctx, db, lo.ToSlicePtr(removed), -1)
}

// UpdateComponent 更新工作空间组件；切换为全局生效时清空生效环境列表
func UpdateComponent(ctx context.Context, db *mongo.Database, compID bson.ObjectID, data *ComponentUpdateData) error {
	if data == nil {
		return nil
	}

	fields := map[string]any{}
	if data.Version != nil {
		fields["version"] = *data.Version
	}
	if data.Properties != nil {
		fields["properties"] = data.Properties
	}
	if data.ScopeEnvNames != nil {
		fields["scopeEnvNames"] = data.ScopeEnvNames
	}
	if data.ScopeType != nil {
		if !lo.Contains(
			[]component.ScopeType{component.ScopeTypeGlobal, component.ScopeTypeEnvironment},
			*data.ScopeType,
		) {
			return errors.New("invalid scope type")
		}
		fields["scopeType"] = *data.ScopeType
		if *data.ScopeType == component.ScopeTypeGlobal {
			// 全局生效时，清空生效环境列表
			fields["scopeEnvNames"] = []string{}
		}
	}
	if data.Name != nil {
		fields["name"] = *data.Name
	}
	if len(fields) == 0 {
		return nil
	}

	err := store.Of(db, ComponentMeta).Update(ctx, fields, store.ByID(compID))
	return errors.Wrapf(err, "update workspace component %s", compID.Hex())
}

// adjustInstanceCounts 按组件类型汇总后调整 ComponentDef.WorkspaceCompInstanceCount，sign 为 1 增加、-1 减少
func adjustInstanceCounts(ctx context.Context, db *mongo.Database, comps []*Component, sign int) error {
	typeCounts := map[string]int{}
	for _, comp := range comps {
		if comp.Type != "" {
			typeCounts[comp.Type]++
		}
	}
	defs := store.Of(db, component.ComponentDefMeta)
	for compType, count := range typeCounts {
		if err := component.UpdateInstanceCount(
			ctx, defs, compType, component.FieldWorkspaceCompInstance, sign*count,
		); err != nil {
			return errors.Wrapf(err, "adjust workspaceCompInstanceCount of %s by %d", compType, sign*count)
		}
	}
	return nil
}

func setComponentDefaults(comp *Component) error {
	if comp.Name == "" {
		comp.Name = comp.GenerateName()
	}
	if comp.Properties == nil {
		comp.Properties = make(map[string]any)
	}
	return nil
}

func normalizeComponentProperties(comp *Component) error {
	comp.NormalizeProperties()
	return nil
}
