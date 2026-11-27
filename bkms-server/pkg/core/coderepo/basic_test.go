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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
)

var _ = Describe("basic controller", func() {
	var (
		ctx        context.Context
		controller coderepo.Controller
	)

	BeforeEach(func() {
		useSourceTypes(tgitSourceType, gitSourceType)
		ctx = testutil.NewDefaultTenantContext()
		repo := testutil.NewGitRepo(GinkgoT(), "v1.2.0", "v1.10.0", "latest")
		repo.NewGitBranch(GinkgoT(), "feature-a")
		repo.NewGitBranch(GinkgoT(), "aaa")

		var err error
		controller, err = coderepo.NewController(ctx, testutil.NewDB(GinkgoT()), &coderepo.CodeRepository{
			URL: repo.URL, SourceType: testutil.BasicRepoSourceTypeName,
			BasicAuth: &coderepo.BasicAuth{
				Username: testutil.GitServerUsername(),
				Password: testutil.GitServerPassword(),
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	names := func(refs []coderepo.Ref) []string {
		result := make([]string, 0, len(refs))
		for _, ref := range refs {
			result = append(result, ref.Name)
		}
		return result
	}

	It("lists branches with the default branch first", func() {
		refs, err := controller.ListRefs(ctx, coderepo.RefsQuery{Kind: coderepo.RefKindBranch, Page: 1, PageSize: 10})
		Expect(err).NotTo(HaveOccurred())
		Expect(names(refs)).To(Equal([]string{"main", "aaa", "feature-a"}))
		Expect(refs[0].Path).To(Equal("refs/heads/main"))
		Expect(refs[0].SHA).NotTo(BeEmpty())
	})

	It("lists tags with semver tags descending first", func() {
		refs, err := controller.ListRefs(ctx, coderepo.RefsQuery{Kind: coderepo.RefKindTag, Page: 1, PageSize: 10})
		Expect(err).NotTo(HaveOccurred())
		Expect(names(refs)).To(Equal([]string{"v1.10.0", "v1.2.0", "latest"}))
	})

	It("filters by keyword ignoring case and paginates", func() {
		refs, err := controller.ListRefs(ctx, coderepo.RefsQuery{
			Kind: coderepo.RefKindTag, Search: "V1", Page: 2, PageSize: 1,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(names(refs)).To(Equal([]string{"v1.2.0"}))

		refs, err = controller.ListRefs(ctx, coderepo.RefsQuery{Kind: coderepo.RefKindTag, Page: 5, PageSize: 10})
		Expect(err).NotTo(HaveOccurred())
		Expect(refs).To(BeEmpty())
	})

	It("requires a credential", func() {
		_, err := coderepo.NewController(ctx, testutil.NewDB(GinkgoT()), &coderepo.CodeRepository{
			URL: "https://git.example.com/group/app.git", SourceType: testutil.BasicRepoSourceTypeName,
		})
		Expect(err).To(MatchError(coderepo.ErrCredentialRequired))
	})
})
