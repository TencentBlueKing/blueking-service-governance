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

// Package helmcomponent 提供 Helm 应用组件引用的数据模型和存储能力
package helmcomponent

import (
	"time"

	"github.com/TencentBlueKing/gopkg/mapx"
	"go.mongodb.org/mongo-driver/v2/bson"

	appcomponent "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

// helmAppComponentCollectionName MongoDB collection 名称
const helmAppComponentCollectionName = "helm_app_components"

// HelmAppComponent 表示 Helm 应用的组件引用
// 每条记录 = 一个组件实例 + 一个目标资源，存储在独立的 MongoDB collection 中
type HelmAppComponent struct {
	// ID 唯一标识
	ID bson.ObjectID `bson:"_id,omitempty"`
	// AppID 关联的应用 ID
	AppID string `bson:"appID" validate:"required"`
	// EnvName 生效的环境名称
	EnvName string `bson:"envName" validate:"required"`

	// Component 内嵌组件引用（复用现有模型，包含 Name、ComponentInst、ComponentRef）
	appcomponent.Component `bson:",inline"`

	// Target 目标资源选择器，指定 patch 的目标资源
	Target TargetResourceSelector `bson:"target" validate:"required"`

	// Priority 执行优先级，数值越小越先执行，默认 0
	Priority int `bson:"priority"`

	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// TargetResourceSelector 目标资源选择器
// 用于在 Helm 渲染后的 manifest 中精确匹配目标资源（类似 HPA scaleTargetRef）
type TargetResourceSelector struct {
	// APIVersion 资源 API 版本（如 "apps/v1"），为空时不参与匹配
	APIVersion string `bson:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	// Kind 资源类型（如 "Deployment"），必填
	Kind string `bson:"kind" json:"kind" validate:"required"`
	// Name 资源名称（metadata.name），必填
	Name string `bson:"name" json:"name" validate:"required"`
}

// Matches 判断选择器是否匹配给定的资源文档
// manifest 应为解析后的 YAML 文档（map[string]any 格式）
func (s *TargetResourceSelector) Matches(manifest map[string]any) bool {
	// 匹配 kind
	if mapx.GetStr(manifest, "kind") != s.Kind {
		return false
	}

	// 匹配 metadata.name
	if mapx.GetStr(manifest, "metadata.name") != s.Name {
		return false
	}

	// 如果指定了 apiVersion，则也需要匹配
	if s.APIVersion != "" && mapx.GetStr(manifest, "apiVersion") != s.APIVersion {
		return false
	}
	return true
}
