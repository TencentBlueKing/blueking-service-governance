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

package resname_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/resname"
)

var _ = Describe("resname", func() {
	Describe("GenEnvPrefix", func() {
		It("generates a 5-char lowercase alphanumeric prefix starting with f", func() {
			prefix, err := resname.GenEnvPrefix()
			Expect(err).NotTo(HaveOccurred())
			Expect(prefix).To(MatchRegexp(`^f[a-z0-9]{4}$`))
		})
	})

	Describe("Scope", func() {
		It("handles zero value and nil environment safely without mutating base scope", func() {
			var zero resname.Scope
			Expect(zero.Build("web")).To(Equal("web"))
			Expect(zero.Restore("fab12-web")).To(Equal("fab12-web"))

			base := resname.ForEnv(nil)
			withLane := base.WithTrafficLane("gray")
			Expect(base.Build("web")).To(Equal("web"))
			Expect(withLane.Build("web")).To(Equal("gray-web"))
			Expect(withLane.Restore("gray-web")).To(Equal("web"))
		})

		DescribeTable("builds and restores scoped names symmetrically across environments and traffic lanes",
			func(resourcePrefix, base, laneName, want string) {
				env := &envmodel.Environment{NsSharing: envmodel.NsSharing{ResPrefix: resourcePrefix}}
				scope := resname.ForEnv(env).WithTrafficLane(laneName)
				Expect(scope.Build(base)).To(Equal(want))
				if base != "" {
					Expect(scope.Restore(want)).To(Equal(base))
				}
				if laneName == "" {
					envScope := resname.ForEnv(env)
					Expect(envScope.Build(base)).To(Equal(want))
					if base != "" {
						Expect(envScope.Restore(want)).To(Equal(base))
					}
				}
			},
			Entry("standard env without lane", "", "web", "", "web"),
			Entry("standard env with lane", "", "web", "gray", "gray-web"),
			Entry("shared-namespace env without lane", "fab12", "web", "", "fab12-web"),
			Entry("shared-namespace env with lane", "fab12", "web", "gray", "gray-fab12-web"),
			Entry("hyphenated app and lane names", "fab12", "my-order-svc", "lane-01", "lane-01-fab12-my-order-svc"),
			Entry("empty base name stays empty", "fab12", "", "gray", ""),
		)

		It("leaves names unchanged when restoring if the full scope prefix does not match", func() {
			env := &envmodel.Environment{NsSharing: envmodel.NsSharing{ResPrefix: "fab12"}}
			envScope := resname.ForEnv(env)
			Expect(envScope.Restore("web")).To(Equal("web"))
			Expect(envScope.Restore("fab12web")).To(Equal("fab12web"))
			Expect(envScope.Restore("fab12-")).To(Equal("fab12-"))

			// 带泳道的作用域必须完整匹配 <lane>-<prefix>-，不会误剥其他环境同泳道资源
			laneScope := envScope.WithTrafficLane("gray")
			Expect(laneScope.Restore("gray-web")).To(Equal("gray-web"))
			Expect(laneScope.Restore("fab12-web")).To(Equal("fab12-web"))
		})
	})

	Describe("ValidateAppNameForSharedNS", func() {
		It("accepts app names up to 41 characters and rejects 42+", func() {
			Expect(resname.ValidateAppNameForSharedNS(strings.Repeat("a", 41))).To(Succeed())
			err := resname.ValidateAppNameForSharedNS(strings.Repeat("a", 42))
			Expect(err).To(MatchError(ContainSubstring("too long to share namespace")))
		})
	})

	Describe("Validate", func() {
		It("validates cluster resource name format and length", func() {
			Expect(resname.Validate("fab12-web-polaris-service")).To(Succeed())
			Expect(resname.Validate(strings.Repeat("a", 64))).NotTo(Succeed())
			Expect(resname.Validate("1-starts-with-digit")).NotTo(Succeed())
		})
	})
})
