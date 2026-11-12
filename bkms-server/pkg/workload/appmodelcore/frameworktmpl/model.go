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

// Package frameworktmpl 管理工作空间的框架配置模板。
//
// 模板分平台内置与空间自定义两种：内置模板随代码发布、只读，不入库；
// 自定义模板存储在 workspace_framework_templates 集合。模板只用于创建 tRPC、TAF
// 应用时填充框架配置文件，不影响已创建的应用。
package frameworktmpl

import (
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	// ErrTemplateNotFound 模板不存在。
	ErrTemplateNotFound = errors.New("framework config template not found")
	// ErrTemplateConflict 同一分组下模板名称重复，或并发设置默认模板冲突。
	ErrTemplateConflict = errors.New("framework config template conflicts with an existing one")
	// ErrInvalidTemplate 模板参数非法。
	ErrInvalidTemplate = errors.New("invalid framework config template")
	// ErrBuiltinReadOnly 平台内置模板只读。
	ErrBuiltinReadOnly = errors.New("builtin framework config template is read-only")
)

// Group 模板分组。同一工作空间的每个分组至多有一个默认模板，没有时平台内置模板为默认。
type Group struct {
	// Framework 框架，如 trpc、taf。
	Framework string `bson:"framework"`
	// Language 语言，如 go、cpp。
	Language string `bson:"language"`
}

// Definition 模板中可编辑的内容。
type Definition struct {
	// Name 模板名称。
	Name string
	// FileName 框架配置文件名。
	FileName string
	// Content 框架配置文件内容。
	Content string
}

// Template 空间自定义模板。
type Template struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID    string `json:"-" bson:"tenantID"`
	WorkspaceID string `bson:"workspaceID"`
	Group       `bson:",inline"`
	Name        string `bson:"name"`
	FileName    string `bson:"fileName"`
	Content     string `bson:"content"`
	// IsDefault 是否为所在分组的默认模板。
	IsDefault bool      `bson:"isDefault"`
	Creator   string    `bson:"creator"`
	Updater   string    `bson:"updater"`
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// View 模板的对外视图，统一表示平台内置模板与空间自定义模板。
type View struct {
	// ID 自定义模板为 ObjectID 的十六进制形式，内置模板为 builtin-{framework}-{language}。
	ID string
	Group
	Name     string
	FileName string
	// Content 列表场景下为空。
	Content   string
	Builtin   bool
	IsDefault bool
	// 以下字段内置模板为零值。
	Creator   string
	Updater   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (t *Template) view() *View {
	return &View{
		ID:        t.ID.Hex(),
		Group:     t.Group,
		Name:      t.Name,
		FileName:  t.FileName,
		Content:   t.Content,
		IsDefault: t.IsDefault,
		Creator:   t.Creator,
		Updater:   t.Updater,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
