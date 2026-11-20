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

package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

var _ = Describe("keepK8sProjects", func() {
	var projects []bcs.Project

	BeforeEach(func() {
		projects = []bcs.Project{
			{Code: "k8s-managed", Kind: bcs.ProjectKindK8s},
			{Code: "k8s-unmanaged", Kind: bcs.ProjectKindK8s},
			{Code: "mesos-managed", Kind: "mesos"},
			{Code: "empty-kind", Kind: ""},
		}
	})

	It("keeps all k8s projects when BCS is independent of BKCI", func() {
		filtered := keepK8sProjects(projects, nil)
		Expect(projectCodes(filtered)).To(Equal([]string{"k8s-managed", "k8s-unmanaged"}))
	})

	It("intersects with BKCI-managed codes when identities are shared", func() {
		filtered := keepK8sProjects(projects, map[string]bool{
			"k8s-managed":   true,
			"mesos-managed": true,
		})
		Expect(projectCodes(filtered)).To(Equal([]string{"k8s-managed"}))
	})

	It("returns empty when identities are shared but the user manages no BKCI project", func() {
		Expect(keepK8sProjects(projects, map[string]bool{})).To(BeEmpty())
	})

	It("returns empty for an empty project list", func() {
		Expect(keepK8sProjects(nil, nil)).To(BeEmpty())
	})
})

func projectCodes(projects []bcs.Project) []string {
	return lo.Map(projects, func(item bcs.Project, _ int) string {
		return item.Code
	})
}
