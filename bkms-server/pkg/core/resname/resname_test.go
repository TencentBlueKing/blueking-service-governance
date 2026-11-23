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

	Describe("Build and TrimEnvPrefix", func() {
		It("handles nil environment safely", func() {
			Expect(resname.Build(nil, "web")).To(Equal("web"))
			Expect(resname.Build(nil, "web", resname.WithTrafficLane("gray"))).To(Equal("gray-web"))
			Expect(resname.TrimEnvPrefix(nil, "fab12-web")).To(Equal("fab12-web"))
		})

		DescribeTable("builds scoped names across environments and traffic lanes",
			func(resourcePrefix, base, laneName, want string) {
				env := &envmodel.Environment{ResourcePrefix: resourcePrefix}
				Expect(resname.Build(env, base, resname.WithTrafficLane(laneName))).To(Equal(want))
				if laneName == "" {
					Expect(resname.Build(env, base)).To(Equal(want))
					Expect(resname.TrimEnvPrefix(env, want)).To(Equal(base))
				}
			},
			Entry("standard env without lane", "", "web", "", "web"),
			Entry("standard env with lane", "", "web", "gray", "gray-web"),
			Entry("shared-namespace env without lane", "fab12", "web", "", "fab12-web"),
			Entry("shared-namespace env with lane", "fab12", "web", "gray", "gray-fab12-web"),
			Entry("empty base name stays empty", "fab12", "", "gray", ""),
		)

		It("leaves names without the exact prefix+separator unchanged when trimming", func() {
			env := &envmodel.Environment{ResourcePrefix: "fab12"}
			Expect(resname.TrimEnvPrefix(env, "web")).To(Equal("web"))
			Expect(resname.TrimEnvPrefix(env, "fab12web")).To(Equal("fab12web"))
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
