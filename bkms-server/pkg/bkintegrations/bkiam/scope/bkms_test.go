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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

var _ = Describe("BKMSRoleScopesGenerator", func() {
	const (
		wsID   = "my-workspace"
		wsName = "我的空间"
	)

	It("should render admin scopes covering workspace / app / env actions", func() {
		g := BKMSRoleScopesGenerator{WorkspaceID: wsID, WorkspaceName: wsName, TplRoleCode: role.BuiltinRoleCode.Admin}
		scopes := g.Generate()
		Expect(scopes).To(HaveLen(5))

		// All scopes must be tagged with the configured BKMS system id.
		for _, s := range scopes {
			Expect(s.System).To(Equal(testBkmsSystemID))
			Expect(s.Resources).NotTo(BeEmpty())
		}

		// First scope: workspace view/edit/delete on the workspace itself.
		Expect(scopes[0].Actions).To(ContainElements(
			types.Action{ID: "view_workspace"},
			types.Action{ID: "edit_workspace"},
			types.Action{ID: "delete_workspace"},
		))
		Expect(scopes[0].Resources[0].Type).To(Equal(types.WorkspaceResourceType))
		Expect(scopes[0].Resources[0].Paths[0][0].ID).To(Equal(wsID))
		Expect(scopes[0].Resources[0].Paths[0][0].Name).To(Equal(wsName))
	})

	It("should render developer scopes (no delete on workspace)", func() {
		g := BKMSRoleScopesGenerator{
			WorkspaceID:   wsID,
			WorkspaceName: wsName,
			TplRoleCode:   role.BuiltinRoleCode.Developer,
		}
		scopes := g.Generate()
		Expect(scopes).NotTo(BeEmpty())

		// Developer should only have view on workspace (no edit / delete).
		Expect(scopes[0].Actions).To(ConsistOf(types.Action{ID: "view_workspace"}))
	})

	It("should render empty scopes for unknown role code (anonymous fallback)", func() {
		g := BKMSRoleScopesGenerator{WorkspaceID: wsID, WorkspaceName: wsName, TplRoleCode: "unknown-role"}
		Expect(g.Generate()).To(BeEmpty())
	})
})
