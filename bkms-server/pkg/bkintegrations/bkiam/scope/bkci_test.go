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

var _ = Describe("BKCIRoleScopesGenerator", func() {
	const (
		projectID   = "bkci-proj-01"
		projectName = "BKCI Project"
	)

	It("should render admin scopes scoped to the BKCI system id", func() {
		g := BKCIRoleScopesGenerator{
			ProjectID:   projectID,
			ProjectName: projectName,
			TplRoleCode: role.BuiltinRoleCode.Admin,
		}
		scopes := g.Generate()
		Expect(scopes).NotTo(BeEmpty())

		for _, s := range scopes {
			Expect(s.System).To(Equal(testBkCISystemID))
			Expect(s.Resources).NotTo(BeEmpty())
		}

		// First scope should contain key project actions.
		Expect(scopes[0].Actions).To(ContainElements(
			types.Action{ID: "project_visit"},
			types.Action{ID: "project_view"},
			types.Action{ID: "project_edit"},
		))
		Expect(scopes[0].Resources[0].Paths[0][0].ID).To(Equal(projectID))
		Expect(scopes[0].Resources[0].Paths[0][0].Name).To(Equal(projectName))
	})

	It("should render operator scopes with at least one action", func() {
		g := BKCIRoleScopesGenerator{
			ProjectID:   projectID,
			ProjectName: projectName,
			TplRoleCode: role.BuiltinRoleCode.Operator,
		}
		scopes := g.Generate()
		Expect(scopes).NotTo(BeEmpty())
		Expect(scopes[0].Actions).NotTo(BeEmpty())
	})
})
