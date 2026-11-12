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

package frameworktmpl

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// CollectionName 存储空间自定义框架配置模板的集合。
const CollectionName = "workspace_framework_templates"

// TemplateStore 空间自定义模板的存储。
type TemplateStore = store.GenericStore[Template]

// TemplateMeta 空间自定义模板的存储元数据。
//
// 索引（由 golang-migrate 维护，见 db/migrations/000026_workspace_framework_templates_idx）：
// - 唯一：workspaceID + framework + language + name，同一分组下模板名称不重复
// - 唯一（partial，isDefault 为 true）：workspaceID + framework + language，每个分组至多一个默认模板
var TemplateMeta = store.Meta[Template]{
	Collection:   CollectionName,
	TenantScoped: true,
	NotFound:     ErrTemplateNotFound,
	Duplicate:    ErrTemplateConflict,
}

// ByWorkspace 匹配工作空间下的全部模板。
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByGroup 匹配工作空间下某个分组的模板。
func ByGroup(workspaceID string, group Group) store.Opt {
	return store.Combine(
		ByWorkspace(workspaceID),
		store.Where("framework", group.Framework),
		store.Where("language", group.Language),
	)
}

// ByID 按工作空间和 ID 匹配模板，其它工作空间的同 ID 模板视为不存在。
func ByID(workspaceID string, id bson.ObjectID) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.ByID(id))
}

// DefaultOf 匹配分组下被设为默认的模板。
func DefaultOf(workspaceID string, group Group) store.Opt {
	return store.Combine(ByGroup(workspaceID, group), store.Where("isDefault", true))
}
