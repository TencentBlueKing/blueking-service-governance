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
	DescribeTable("derives alias prefixed with the repository type",
		func(repoURL, alias string) {
			got, err := DeriveRepoAlias(config.CodeRepoTypeTGit, repoURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(alias))
		},
		Entry("plain", "https://git.tencent.com/huiwenzhang/bkms-test-app.git", "tgit:huiwenzhang/bkms-test-app"),
		Entry("nested group", "https://git.tencent.com/group/sub/app", "tgit:group/sub/app"),
		Entry("trailing slash and upper .GIT", "https://git.tencent.com/Group/App.GIT/", "tgit:group/app"),
		Entry("http with port", "http://git.example.com:8080/group/app.git", "tgit:group/app"),
	)

	DescribeTable("rejects urls that cannot produce an alias",
		func(repoURL string) {
			_, err := DeriveRepoAlias(config.CodeRepoTypeTGit, repoURL)
			Expect(errors.Is(err, ErrRepoAliasInvalid)).To(BeTrue(), "unexpected error: %v", err)
		},
		Entry("empty path", "https://git.tencent.com/"),
		Entry("only .git", "https://git.tencent.com/.git"),
		Entry("ssh url", "git@git.tencent.com:group/app.git"),
		Entry("embedded credentials", "https://user:pass@git.tencent.com/group/app.git"),
		Entry("too long", "https://git.tencent.com/"+strings.Repeat("a", 260)),
	)
})

var _ = Describe("ResolveCodeRepo", func() {
	const repoURL = "https://git.tencent.com/Group/App.git"

	var prevConfig *config.Config

	BeforeEach(func() {
		prevConfig = config.G
		cfg := *config.G
		cfg.FeatureIntegrations.CodeRepoTypes = nil
		config.G = &cfg
		DeferCleanup(func() { config.G = prevConfig })
	})

	It("keeps the given alias for TGit oauth by default", func() {
		authType, alias, err := ResolveCodeRepo(config.CodeRepoTypeTGit, "", repoURL, "Group/App")
		Expect(err).NotTo(HaveOccurred())
		Expect(authType).To(Equal(config.CodeRepoAuthOAuth))
		Expect(alias).To(Equal("Group/App"))

		_, _, err = ResolveCodeRepo(config.CodeRepoTypeTGit, "", repoURL, "")
		Expect(errors.Is(err, ErrRepoAliasRequired)).To(BeTrue())
	})

	It("rejects unsupported types and auth types", func() {
		_, _, err := ResolveCodeRepo("GitHub", "", repoURL, "group/app")
		Expect(errors.Is(err, ErrCodeRepoTypeUnsupported)).To(BeTrue())
		_, _, err = ResolveCodeRepo(config.CodeRepoTypeTGit, config.CodeRepoAuthPassword, repoURL, "")
		Expect(errors.Is(err, ErrCodeRepoAuthUnsupported)).To(BeTrue())
	})

	It("derives alias and ignores the given one for password auth", func() {
		config.G.FeatureIntegrations.CodeRepoTypes = []config.CodeRepoTypeConfig{{
			Type: config.CodeRepoTypeTGit, BkCIRepoType: "codeTGit", AuthTypes: []string{config.CodeRepoAuthPassword},
		}}
		authType, alias, err := ResolveCodeRepo(config.CodeRepoTypeTGit, "", repoURL, "manual/alias")
		Expect(err).NotTo(HaveOccurred())
		Expect(authType).To(Equal(config.CodeRepoAuthPassword))
		Expect(alias).To(Equal("tgit:group/app"))
	})
})
