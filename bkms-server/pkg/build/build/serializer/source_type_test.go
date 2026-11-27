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

package serializer_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/serializer"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
)

var _ = Describe("RepositoryBuildConfigInput source type", func() {
	BeforeEach(func() {
		testutil.UseRepoSourceTypes(GinkgoT(), testutil.BkCIManagedRepoSourceType(), testutil.BasicRepoSourceType())
	})

	toModel := func(typ, repoURL, alias string) (*imagebuild.RepositoryConfig, error) {
		return (&serializer.RepositoryBuildConfigInput{
			Type: typ, RepoURL: repoURL, RepoAlias: alias, DefaultBranch: "main",
			Username: "user", Password: "pass",
		}).ToModel()
	}

	It("keeps the alias of a bkci managed repository", func() {
		cfg, err := toModel(config.DefaultRepoSourceTypeName, "https://git.example.com/group/app.git", "group/app")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Type).To(Equal(imagebuild.RepositoryTypeTGit))
		Expect(cfg.RepoAlias).To(Equal("group/app"))
	})

	It("normalizes the legacy GitHub type to the default source type", func() {
		cfg, err := toModel(coderepo.LegacyTypeGitHub, "https://github.com/group/app.git", "group/app")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Type).To(Equal(imagebuild.RepositoryTypeTGit))
	})

	It("derives the alias of a basic repository and carries the credential", func() {
		cfg, err := toModel(testutil.BasicRepoSourceTypeName, "https://git.example.com/group/app.git", "ignored")
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Type).To(Equal(imagebuild.RepositoryType(testutil.BasicRepoSourceTypeName)))
		Expect(cfg.RepoAlias).To(Equal(strings.ToLower(testutil.BasicRepoSourceTypeName) + ":group/app"))
		Expect(cfg.Username).To(Equal("user"))
		Expect(cfg.Password).To(Equal("pass"))
	})

	It("rejects unknown source types", func() {
		_, err := toModel("unknown", "https://git.example.com/group/app.git", "group/app")
		Expect(err).To(MatchError(coderepo.ErrSourceTypeNotFound))
	})

	It("requires the alias of a bkci managed repository", func() {
		_, err := toModel(config.DefaultRepoSourceTypeName, "https://git.example.com/group/app.git", "")
		Expect(err).To(MatchError(coderepo.ErrAliasRequired))
	})
})
