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

var _ = Describe("BKRepoRoleScopesGenerator", func() {
	const (
		projectID   = "bkrepo-proj-01"
		projectName = "BKRepo Project"
	)

	It("should render admin scopes scoped to the BKRepo system id", func() {
		g := BKRepoRoleScopesGenerator{
			ProjectID:   projectID,
			ProjectName: projectName,
			TplRoleCode: role.BuiltinRoleCode.Admin,
		}
		scopes := g.Generate()
		Expect(scopes).NotTo(BeEmpty())

		for _, s := range scopes {
			Expect(s.System).To(Equal(testBkRepoSystemID))
		}

		// First scope should contain project_manage / project_view / project_edit / repo_create.
		Expect(scopes[0].Actions).To(ContainElements(
			types.Action{ID: "project_manage"},
			types.Action{ID: "project_view"},
			types.Action{ID: "project_edit"},
			types.Action{ID: "repo_create"},
		))
		Expect(scopes[0].Resources[0].Paths[0][0].ID).To(Equal(projectID))
	})

	It("should render developer scopes with at least one action", func() {
		g := BKRepoRoleScopesGenerator{
			ProjectID:   projectID,
			ProjectName: projectName,
			TplRoleCode: role.BuiltinRoleCode.Developer,
		}
		scopes := g.Generate()
		Expect(scopes).NotTo(BeEmpty())
		Expect(scopes[0].Actions).NotTo(BeEmpty())
	})
})
