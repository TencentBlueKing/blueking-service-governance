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

package bkci

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

var _ = Describe("DeriveRepoAlias", func() {
	DescribeTable("derives alias with a type prefix",
		func(repoURL, alias string) {
			got, err := DeriveRepoAlias(repoURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(alias))
		},
		Entry("tgit", "https://git.tencent.com/huiwenzhang/bkms-test-app.git", "tgit:huiwenzhang/bkms-test-app"),
		Entry("github", "https://github.com/TencentBlueKing/bk-ci.git", "github:tencentblueking/bk-ci"),
		Entry("gitee", "https://gitee.com/group/sub/app", "gitee:group/sub/app"),
		Entry("trailing slash and upper .GIT", "https://GitHub.com/group/App.GIT/", "github:group/app"),
		Entry("unknown host uses host as prefix", "http://git.example.com:8080/group/app.git",
			"git.example.com:group/app"),
	)

	DescribeTable("rejects urls that cannot produce an alias",
		func(repoURL string) {
			_, err := DeriveRepoAlias(repoURL)
			Expect(errors.Is(err, ErrRepoAliasInvalid)).To(BeTrue(), "unexpected error: %v", err)
		},
		Entry("empty path", "https://github.com/"),
		Entry("only .git", "https://github.com/.git"),
		Entry("ssh url", "git@github.com:group/app.git"),
		Entry("embedded credentials", "https://user:pass@github.com/group/app.git"),
		Entry("too long", "https://github.com/"+strings.Repeat("a", 260)),
	)
})

var _ = Describe("ResolveRepoAlias", func() {
	var prevConfig *config.Config

	BeforeEach(func() {
		prevConfig = config.G
		cfg := *config.G
		config.G = &cfg
		DeferCleanup(func() { config.G = prevConfig })
	})

	It("derives alias and ignores the given one when git oauth is disabled", func() {
		config.G.FeatureIntegrations.DisableBkCIGitOAuth = true
		alias, err := ResolveRepoAlias("https://github.com/group/app.git", "manual/alias")
		Expect(err).NotTo(HaveOccurred())
		Expect(alias).To(Equal("github:group/app"))
	})

	It("uses the given alias when git oauth is enabled", func() {
		config.G.FeatureIntegrations.DisableBkCIGitOAuth = false
		alias, err := ResolveRepoAlias("https://git.example.com/Group/App.git", "Group/App")
		Expect(err).NotTo(HaveOccurred())
		Expect(alias).To(Equal("Group/App"))

		_, err = ResolveRepoAlias("https://git.example.com/group/app.git", "")
		Expect(errors.Is(err, ErrRepoAliasRequired)).To(BeTrue())
	})
})
