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

package git_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/repoutil/git"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("Client", func() {
	ctx := context.Background()
	cred := git.Credential{Username: testutil.GitServerUsername(), Password: testutil.GitServerPassword()}

	Describe("NewClient", func() {
		DescribeTable("rejects invalid repository urls",
			func(repoURL string) {
				_, err := git.NewClient(repoURL, cred)
				Expect(err).To(MatchError(git.ErrInvalidRepoURL))
			},
			Entry("ssh scheme", "git@example.com:group/repo.git"),
			Entry("ftp scheme", "ftp://example.com/group/repo.git"),
			Entry("empty host", "https:///group/repo.git"),
			Entry("embedded credentials", "https://user:pass@example.com/group/repo.git"),
		)
	})

	Describe("ListRefs", func() {
		It("lists branches, tags and the default branch", func() {
			repo := testutil.NewGitRepo(GinkgoT(), "v1.0.0", "v1.1.0")
			repo.NewGitBranch(GinkgoT(), "feature")

			client, err := git.NewClient(repo.URL, cred)
			Expect(err).NotTo(HaveOccurred())
			refs, err := client.ListRefs(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(refs.DefaultBranch).To(Equal("main"))
			Expect(refs.Branches).To(HaveLen(2))
			Expect(
				refs.Branches[0],
			).To(MatchFields(IgnoreExtras, Fields{"Name": Equal("feature"), "Path": Equal("refs/heads/feature")}))
			Expect(refs.Branches[1].Name).To(Equal("main"))
			Expect(refs.Tags).To(HaveLen(2))
			Expect(refs.Tags[0].Name).To(Equal("v1.0.0"))
			// Tag 指向默认分支的提交
			Expect(refs.Tags[0].SHA).To(Equal(refs.Branches[1].SHA))
		})

		It("tolerates a trailing slash in the repository url", func() {
			repo := testutil.NewGitRepo(GinkgoT())
			client, err := git.NewClient(repo.URL+"/", cred)
			Expect(err).NotTo(HaveOccurred())
			Expect(client.Touch(ctx)).To(Succeed())
		})

		It("returns ErrAuthFailed for a wrong password", func() {
			repo := testutil.NewGitRepo(GinkgoT())
			client, err := git.NewClient(
				repo.URL,
				git.Credential{Username: testutil.GitServerUsername(), Password: "wrong"},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client.Touch(ctx)).To(MatchError(git.ErrAuthFailed))
		})

		It("returns ErrAuthFailed for a private repository without credentials", func() {
			repo := testutil.NewGitRepo(GinkgoT())
			client, err := git.NewClient(repo.URL, git.Credential{})
			Expect(err).NotTo(HaveOccurred())
			Expect(client.Touch(ctx)).To(MatchError(git.ErrAuthFailed))
		})

		It("returns ErrRepoNotFound for a missing repository", func() {
			client, err := git.NewClient(
				testutil.GitServerURL()+"/"+testutil.GitServerUsername()+"/not-exist.git",
				cred,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(client.Touch(ctx)).To(MatchError(git.ErrRepoNotFound))
		})
	})
})
