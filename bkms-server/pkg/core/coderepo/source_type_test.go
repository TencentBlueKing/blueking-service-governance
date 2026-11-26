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

package coderepo_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
)

// 单测使用的接入方式：蓝盾托管与账号密码
var (
	tgitSourceType = testutil.BkCIManagedRepoSourceType()
	gitSourceType  = testutil.BasicRepoSourceType()
)

// unknownSourceTypeName 未配置的接入方式名
const unknownSourceTypeName = "SVN"

// useSourceTypes 在当前用例内替换生效的接入方式
func useSourceTypes(types ...config.RepoSourceTypeConfig) {
	GinkgoHelper()
	testutil.UseRepoSourceTypes(GinkgoT(), types...)
}

var _ = Describe("SourceType", func() {
	It("lists enabled source types only", func() {
		disabled := gitSourceType
		disabled.Name, disabled.Enabled = "Disabled", false
		useSourceTypes(gitSourceType, disabled, tgitSourceType)
		Expect(coderepo.SourceTypes()).To(Equal([]coderepo.SourceType{gitSourceType, tgitSourceType}))
	})

	It("prefers a bkci managed source type as the default", func() {
		useSourceTypes(gitSourceType, tgitSourceType)
		Expect(coderepo.DefaultSourceType()).To(Equal(tgitSourceType))
	})

	It("falls back to the first enabled source type when no bkci managed one is enabled", func() {
		useSourceTypes(gitSourceType)
		Expect(coderepo.DefaultSourceType()).To(Equal(gitSourceType))
	})

	DescribeTable(
		"resolves the source type of a repository type",
		func(types []config.RepoSourceTypeConfig, name, expected string) {
			useSourceTypes(types...)
			sourceType, err := coderepo.ResolveSourceType(name)
			Expect(err).NotTo(HaveOccurred())
			Expect(sourceType.Name).To(Equal(expected))
		},
		Entry(
			"configured name",
			[]config.RepoSourceTypeConfig{tgitSourceType, gitSourceType},
			testutil.BasicRepoSourceTypeName,
			testutil.BasicRepoSourceTypeName,
		),
		Entry(
			"empty name",
			[]config.RepoSourceTypeConfig{tgitSourceType, gitSourceType},
			"",
			config.DefaultRepoSourceTypeName,
		),
		Entry(
			"legacy GitHub",
			[]config.RepoSourceTypeConfig{tgitSourceType, gitSourceType},
			coderepo.LegacyTypeGitHub,
			config.DefaultRepoSourceTypeName,
		),
		Entry(
			"legacy TGit not configured",
			[]config.RepoSourceTypeConfig{gitSourceType},
			config.DefaultRepoSourceTypeName,
			testutil.BasicRepoSourceTypeName,
		),
	)

	It("rejects unknown or disabled source types", func() {
		disabled := gitSourceType
		disabled.Enabled = false
		useSourceTypes(tgitSourceType, disabled)
		_, err := coderepo.ResolveSourceType(testutil.BasicRepoSourceTypeName)
		Expect(err).To(MatchError(coderepo.ErrSourceTypeNotFound))
		_, err = coderepo.ResolveSourceType(unknownSourceTypeName)
		Expect(err).To(MatchError(coderepo.ErrSourceTypeNotFound))
	})
})

var _ = Describe("ResolveAlias", func() {
	It("uses the given alias for bkci managed repositories", func() {
		alias, err := coderepo.ResolveAlias(tgitSourceType, "https://git.example.com/group/app.git", "group/app")
		Expect(err).NotTo(HaveOccurred())
		Expect(alias).To(Equal("group/app"))
	})

	It("requires an alias for bkci managed repositories", func() {
		_, err := coderepo.ResolveAlias(tgitSourceType, "https://git.example.com/group/app.git", "")
		Expect(err).To(MatchError(coderepo.ErrAliasRequired))
	})

	DescribeTable("derives the alias from the url for other repositories, ignoring the given alias",
		func(repoURL, expected string) {
			alias, err := coderepo.ResolveAlias(gitSourceType, repoURL, "ignored")
			Expect(err).NotTo(HaveOccurred())
			Expect(alias).To(Equal(expected))
		},
		Entry("with .git suffix", "https://git.example.com/Group/App.git", "git:group/app"),
		Entry("without suffix", "https://git.example.com/group/sub/app", "git:group/sub/app"),
		Entry("trailing slash", "http://git.example.com/group/app.git/", "git:group/app"),
	)

	DescribeTable("rejects urls that cannot derive an alias",
		func(repoURL string) {
			_, err := coderepo.ResolveAlias(gitSourceType, repoURL, "")
			Expect(err).To(MatchError(coderepo.ErrAliasInvalid))
		},
		Entry("ssh url", "git@git.example.com:group/app.git"),
		Entry("empty path", "https://git.example.com/"),
		Entry("embedded credentials", "https://u:p@git.example.com/group/app.git"),
	)
})
