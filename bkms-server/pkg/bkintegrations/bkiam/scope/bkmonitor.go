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

package scope

import (
	"strings"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/scope/template"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

// bkMonitorMCPActionSuffix 监控 MCP 相关 action 的 ID 后缀，如 using_dashboard_mcp
const bkMonitorMCPActionSuffix = "_mcp"

// BKMonitorRoleScopesGenerator 是 bk-monitor 角色权限范围生成器
type BKMonitorRoleScopesGenerator struct {
	SpaceID     string
	SpaceName   string
	TplRoleCode string
}

// Generate 生成权限范围
func (g BKMonitorRoleScopesGenerator) Generate() []types.AuthorizationScope {
	scopes := generateFromTemplate(
		template.GetRoleScopeTemplatePath("bkmonitor", g.TplRoleCode),
		map[string]any{
			"BKMonitorSystemID": config.G.BkIAMSystemIDs.BkMonitor,
			"SpaceID":           g.SpaceID,
			"SpaceName":         g.SpaceName,
		},
	)
	if config.G.FeatureIntegrations.DisableBkMonitorMCPPerms {
		scopes = dropMCPActions(scopes)
	}
	return scopes
}

// dropMCPActions 移除所有 MCP action，移除后没有 action 的 scope 一并丢弃
func dropMCPActions(scopes []types.AuthorizationScope) []types.AuthorizationScope {
	result := make([]types.AuthorizationScope, 0, len(scopes))
	for _, s := range scopes {
		s.Actions = lo.Reject(s.Actions, func(a types.Action, _ int) bool {
			return strings.HasSuffix(a.ID, bkMonitorMCPActionSuffix)
		})
		if len(s.Actions) > 0 {
			result = append(result, s)
		}
	}
	return result
}

var _ AuthScopesGenerator = (*BKMonitorRoleScopesGenerator)(nil)
