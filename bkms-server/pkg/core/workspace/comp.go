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
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

// Component 表示工作空间级别的组件实例，仅 tRPC 应用使用。
//
// 工作空间组件是一种可被多个环境下的不同应用共享的预设配置（模板）。应用可以：
//   - 根据组件市场（apps/bkms-server/pkg/extension/component/entities.go/ComponentDef）中的组件定义，实例化出一个组件实例并使用
//   - 通过 Component.Name 字段引用工作空间组件
//
// 引用生效条件：ScopeGlobal=true 或当前环境在 ScopeEnvNames 列表中
type Component struct {
	// ComponentInst 组件实例
	component.ComponentInst `bson:",inline"`

	// ID 组件唯一标识
	ID bson.ObjectID `bson:"_id,omitempty"`
	// Name 组件名称，用于引用标识
	// 默认由后端负责生成，生成规则为 "[type]-stringx.Random(5)"
	Name string `bson:"name" validate:"required"`
	// WorkspaceID 组件所属工作空间 ID
	WorkspaceID string `bson:"workspaceID" validate:"required"`

	// ScopeGlobal 组件是否全局生效（对所有环境可用）
	ScopeType component.ScopeType `bson:"scopeType" validate:"required"`
	// ScopeEnvNames 组件生效的环境列表（环境英文标识, workspace 下唯一），当 ScopeGlobal 为 false 时有效
	ScopeEnvNames []string `bson:"scopeEnvNames"`

	// CreatedAt 记录添加组件到工作空间中的时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 记录更新组件配置的时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// IsAvailableInEnv 检查组件是否在指定环境中可用
// 返回 true 表示组件在该环境中生效，可以被引用使用
func (c *Component) IsAvailableInEnv(envName string) bool {
	switch c.ScopeType {
	case component.ScopeTypeGlobal:
		// 全局生效，对所有环境可用
		return true
	case component.ScopeTypeEnvironment:
		// 仅对指定环境列表生效
		return lo.Contains(c.ScopeEnvNames, envName)
	default:
		// 未知类型，默认不可用
		return false
	}
}
