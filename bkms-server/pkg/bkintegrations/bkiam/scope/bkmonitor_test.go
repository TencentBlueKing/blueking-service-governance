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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam/role"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

var _ = Describe("BKMonitorRoleScopesGenerator", func() {
	const (
		spaceID   = "monitor-space-01"
		spaceName = "Monitor Space"
	)

	It("should render admin scopes scoped to the BKMonitor system id", func() {
		g := BKMonitorRoleScopesGenerator{
			SpaceID:     spaceID,
			SpaceName:   spaceName,
			TplRoleCode: role.BuiltinRoleCode.Admin,
		}
		scopes := g.Generate()
		Expect(scopes).NotTo(BeEmpty())

		for _, s := range scopes {
			Expect(s.System).To(Equal(testBkMonitorSystemID))
		}

		// First scope should at least include view_business_v2.
		Expect(scopes[0].Actions).To(ContainElement(types.Action{ID: "view_business_v2"}))
		Expect(scopes[0].Resources[0].Paths[0][0].ID).To(Equal(spaceID))
	})

	It("should render sre scopes with at least one action", func() {
		g := BKMonitorRoleScopesGenerator{
			SpaceID:     spaceID,
			SpaceName:   spaceName,
			TplRoleCode: role.BuiltinRoleCode.SRE,
		}
		scopes := g.Generate()
		Expect(scopes).NotTo(BeEmpty())
		Expect(scopes[0].Actions).NotTo(BeEmpty())
	})

	Context("MCP actions", func() {
		// mcpViewActions 是所有 view 类型的 MCP actions
		mcpViewActions := []types.Action{
			{ID: "using_dashboard_mcp"},
			{ID: "using_metrics_mcp"},
			{ID: "using_log_mcp"},
			{ID: "using_alarm_mcp"},
			{ID: "using_metadata_mcp"},
			{ID: "using_apm_mcp"},
			{ID: "using_operation_mcp"},
		}
		// mcpManageAction 是 manage 类型的 MCP action
		mcpManageAction := types.Action{ID: "using_alarm_handling_mcp"}

		DescribeTable("admin/developer/sre should include all 8 MCP actions",
			func(roleCode string) {
				g := BKMonitorRoleScopesGenerator{
					SpaceID:     spaceID,
					SpaceName:   spaceName,
					TplRoleCode: roleCode,
				}
				scopes := g.Generate()
				Expect(scopes).NotTo(BeEmpty())

				// 第一个 scope block（type: space）应包含所有 MCP actions
				spaceActions := scopes[0].Actions
				for _, action := range mcpViewActions {
					Expect(spaceActions).To(ContainElement(action))
				}
				Expect(spaceActions).To(ContainElement(mcpManageAction))
			},
			Entry("admin", role.BuiltinRoleCode.Admin),
			Entry("developer", role.BuiltinRoleCode.Developer),
			Entry("sre", role.BuiltinRoleCode.SRE),
		)

		It("operator should include 7 view MCP actions but NOT using_alarm_handling_mcp", func() {
			g := BKMonitorRoleScopesGenerator{
				SpaceID:     spaceID,
				SpaceName:   spaceName,
				TplRoleCode: role.BuiltinRoleCode.Operator,
			}
			scopes := g.Generate()
			Expect(scopes).NotTo(BeEmpty())

			spaceActions := scopes[0].Actions
			for _, action := range mcpViewActions {
				Expect(spaceActions).To(ContainElement(action))
			}
			Expect(spaceActions).NotTo(ContainElement(mcpManageAction))
		})

		Context("when DisableBkMonitorMCPPerms is enabled", func() {
			BeforeEach(func() {
				config.G.FeatureIntegrations.DisableBkMonitorMCPPerms = true
				DeferCleanup(func() { config.G.FeatureIntegrations.DisableBkMonitorMCPPerms = false })
			})

			DescribeTable("should drop all MCP actions and keep the rest",
				func(roleCode string) {
					g := BKMonitorRoleScopesGenerator{
						SpaceID:     spaceID,
						SpaceName:   spaceName,
						TplRoleCode: roleCode,
					}
					scopes := g.Generate()
					Expect(scopes).NotTo(BeEmpty())
					Expect(scopes[0].Actions).To(ContainElement(types.Action{ID: "view_business_v2"}))

					for _, s := range scopes {
						Expect(s.Actions).NotTo(BeEmpty())
						for _, action := range append(mcpViewActions, mcpManageAction) {
							Expect(s.Actions).NotTo(ContainElement(action))
						}
					}
				},
				Entry("admin", role.BuiltinRoleCode.Admin),
				Entry("developer", role.BuiltinRoleCode.Developer),
				Entry("sre", role.BuiltinRoleCode.SRE),
				Entry("operator", role.BuiltinRoleCode.Operator),
			)
		})
	})
})
