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

package component

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ScopeType 组件生效范围类型
type ScopeType string

const (
	// ScopeTypeGlobal 全局生效
	ScopeTypeGlobal ScopeType = "global"
	// ScopeTypeWorkspace 工作空间生效
	ScopeTypeWorkspace ScopeType = "workspace"
	// ScopeTypeEnvironment 环境生效
	ScopeTypeEnvironment ScopeType = "environment"
)

const (
	// DefaultComponentDefVersion 默认组件定义版本
	// 目前产品上不对外暴露版本定义，默认所有组件都使用 v1.0.0 版本
	DefaultComponentDefVersion = "v1.0.0"
)

// ComponentDef 代表一个组件定义，也可以被理解成一个未被实例化的组件模板。目前所有的组件都遵循同一种工作模式，
// 即通过输入 Properties 来渲染 Patchers 和 Specs。
// 未来，该功能可能会被扩展为支持更多种工作模式。
type ComponentDef struct {
	// Name 组件名称，比如 ResourceLimits
	Name string `bson:"name" validate:"required"`
	// Version 组件版本，比如 v1.0.0，**一个 Name 和 Version 共同唯一标识一个组件定义**
	Version string `bson:"version" validate:"required"`

	// DisplayName 是组件的展示用名称
	DisplayName string `bson:"displayName"`
	// Description 是组件的详细描述
	Description string `bson:"description"`

	// Properties 是组件的属性定义，也可被理解成是组件的输入参数 parameter 的 schema 定义，它明确了
	// 使用组件所需要提供的输入参数。
	Properties []Property `bson:"properties" validate:"dive"`
	// Patchers 是按顺序执行的根节点 YAML Patch 模板。
	Patchers []string `bson:"patchers" validate:"dive,comp_fragment"`
	// Specs 是额外 Kubernetes 资源 YAML 模板。
	Specs []string `bson:"specs" validate:"dive,comp_fragment"`

	// ScopeType 组件生效范围类型
	ScopeType ScopeType `bson:"scopeType"`
	// ScopeWorkspaceIDs 组件生效的工作空间列表，当 ScopeType 为 Workspace 时有效
	ScopeWorkspaceIDs []string `bson:"scopeWorkspaceIDs"`

	// IsBuiltin 是否为官方内置组件, 仅通过 component/assets 目录下的组件文件创建的组件定义会设置为 true
	IsBuiltin bool `bson:"isBuiltin"`

	// Invisible 标记组件定义是否对前端不可见。当设置为 true 时，该组件定义不会返回给前端，
	// 但仍然可以在后端使用。默认值为 false（可见）。
	Invisible bool `bson:"invisible"`

	// ManagedByWorkspaceIDs 标记在哪些工作空间下可以管理该组件定义，包括在组件管理页查看、编辑
	ManagedByWorkspaceIDs []string `bson:"managedByWorkspaceIDs"`

	// AppCompInstanceCount 由该组件定义生成的应用组件实例数量
	AppCompInstanceCount int32 `bson:"appCompInstanceCount"`
	// WorkspaceCompInstanceCount 由该组件定义生成的空间组件实例数量
	WorkspaceCompInstanceCount int32 `bson:"workspaceCompInstanceCount"`

	// Creator 创建人
	Creator string `bson:"creator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// Updater 更新人
	Updater string `bson:"updater"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// Key 返回组件定义的唯一标识
func (c *ComponentDef) Key() string {
	return fmt.Sprintf("%s:%s", c.Name, c.Version)
}

// Property 是组件属性定义中的单个属性
type Property struct {
	// Name 是属性名称，比如“replicas”
	Name string `bson:"name" validate:"required"`
	// Type 是属性类型，它决定了本属性接收何种类型的数据
	Type PropType `bson:"type" validate:"required,prop_type"`
	// Options 是 SELECT 类型的候选项配置（非 SELECT 类型可为空）
	Options []PropertyOption `bson:"options,omitempty"`
	// DefaultValue 是属性的默认值，当用户未提供该属性时会使用此默认值
	DefaultValue any `bson:"defaultValue"`

	// Description 是属性的描述，比如“副本数”
	Description string `bson:"description"`
}

// PropertyOption 表示 SELECT 类型属性的单个候选项。
type PropertyOption struct {
	// Label 是展示给用户看的文案
	Label string `bson:"label"`
	// Value 是实际写入并用于渲染的值
	Value string `bson:"value"`
}

// NormalizedDefaultValue 将属性默认值规范化为 Go 原生类型（仅针对 MAP 类型）。
//
// 默认情况下，any 类型的值被存入 MongoDB 后将使用 BSON 格式序列化，而 BSON 使用 ExtJSON 规范。
// 对于 map 类型，值被读取出来后可能会变成 `{"apple": {"$numberInt": "10"}}` 这种格式，因此，
// 需要使用 json 模块对其二次反序列化，转换为 Go 原生类型。
//
// NOTE: 因为 json 天生的局限性，无法区分 int 和 float64 类型，所以数值类型的值会全部被转换为 float64。
func (p Property) NormalizedDefaultValue() any {
	if p.Type != PropTypeMap {
		return p.DefaultValue
	}

	data, err := bson.MarshalExtJSON(p.DefaultValue, false, false)
	if err != nil {
		// Fallback to the original value if marshal fails
		return p.DefaultValue
	}
	var out any
	if err = json.Unmarshal(data, &out); err != nil {
		// Fallback to the original value if unmarshal fails
		return p.DefaultValue
	}
	return out
}

// ComponentInst 表示一个组件实例，包含实例化组件的详细信息
type ComponentInst struct {
	// Type 为组件类型，也是组件在市场中的名字，等同于 ComponentDef 的 name
	Type string `bson:"type"`
	// Version 是组件版本
	Version string `bson:"version"`

	// Properties 中包含组件具体内容, 当引用空间组件时，该字段没有实际值，需要根据引用关系使用空间组件的值。
	Properties map[string]any `bson:"properties,omitempty"`
}

// NormalizeProperties 将 Properties 中的 BSON 类型规范化为 Go 原生类型。
//
// 默认情况下，any 类型的值被存入 MongoDB 后将使用 BSON 格式序列化。当读取时，嵌套的
// map/array 会被解码为 bson.M/bson.A 等 BSON 特定类型，而不是 Go 原生的 map[string]any/[]any。
// 该方法使用 bson.MarshalExtJSON + json.Unmarshal 的方式，将所有 BSON 类型转换为 Go 原生类型，
// 使得用户可以直接使用类型断言来获取值。
//
// NOTE: 因为 json 天生的局限性，无法区分 int 和 float64 类型，所以数值类型的值会全部被转换为 float64。
func (c *ComponentInst) NormalizeProperties() {
	if c == nil || c.Properties == nil {
		return
	}
	for k, v := range c.Properties {
		c.Properties[k] = normalizeValue(v)
	}
}

// GenerateName 生成组件名称
func (c *ComponentInst) GenerateName() string {
	if c.Type == "" {
		return ""
	}
	return strings.ToLower(fmt.Sprintf("%s-%s", c.Type, stringx.Random(5)))
}

// normalizeValue 将单个值从 BSON 类型转换为 Go 原生类型
func normalizeValue(v any) any {
	data, err := bson.MarshalExtJSON(v, false, false)
	if err != nil {
		return v
	}
	var out any
	if err = json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

// ComponentRef 表示一个组件引用，指向引用的组件
type ComponentRef struct {
	// RefWorkspaceName indicates the referenced workspace component ID.
	// When it is not nil, the actual component definition should be loaded from
	// workspace by this ID.
	RefWorkspaceCompName string `bson:"refWorkspaceCompName,omitempty"`
}

// GenerateName 生成组件名称
func (c *ComponentRef) GenerateName() string {
	if c.RefWorkspaceCompName == "" {
		return ""
	}
	return strings.ToLower(fmt.Sprintf("%s-%s", c.RefWorkspaceCompName, stringx.Random(5)))
}

// Component 是一种用来自定义应用负载的关键类型，一个 Component 可能是组件实例或者引用了某个环境组件。
// 如果是组件实例，则通过 Type 和 Version 指向到一个唯一的组件定义（ComponentDef），比如“设置资源限制”。
// 然后使用 Properties，将该组件具体化为一批 payload，其中可能包含 patches 和 specs 等数据。
// 如果是引用了某个环境组件，则通过 RefWorkspaceCompName 引用某个环境组件，
// 此时依赖环境组件的 Properties 渲染出最终的 payload。
// 最终这些 payload 被应用到实际的资源定义上，完成自定义。
// 校验逻辑见： bkms-server/pkg/workload/appmodelcore/appmodel/validate.go
type Component struct {
	// Name 是组件被使用后的自定义名称，用于环境中定义的组件引用
	// 该字段可能影响到最后渲染出的实例名称，因此不能重复
	// 默认由后端负责生成，生成规则为 "[type]-stringx.Random(5)"
	// 校验规则：Name 必须包含小写字母、数字、中划线，必须以字母开头，必须以字母或数字结尾，长度限制20位以内
	Name string `bson:"name" validate:"required"`
	// 组件只可能是 ComponentInst 或 ComponentRef 之一
	// ComponentInst 表示组件实例
	ComponentInst `bson:",inline"`
	// ComponentRef 表示组件引用
	ComponentRef `bson:",inline"`
}

// EnsureName 确保组件有名称。如果 Name 为空，则根据 Type 生成一个小写的随机名称。
func (c *Component) EnsureName() {
	if c.Name != "" {
		return
	}
	if c.RefWorkspaceCompName != "" {
		c.Name = c.ComponentRef.GenerateName()
	} else if c.Type != "" {
		c.Name = c.ComponentInst.GenerateName()
	}
}
