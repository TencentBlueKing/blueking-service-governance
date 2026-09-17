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
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// credentialCall 记录对蓝盾凭证接口的调用
type credentialCall struct {
	op, credentialID, username, password string
}

// recordingBkCIClient 记录调用的 stub 客户端
type recordingBkCIClient struct {
	*cloudbkci.StubApiClient

	credentials    map[string]credentialCall
	credentialOps  []string
	createRepoOpts []cloudbkci.CreateRepositoryOptions
	refQueries     []string
}

func newRecordingBkCIClient() *recordingBkCIClient {
	return &recordingBkCIClient{
		StubApiClient: cloudbkci.NewStub(mockUser()),
		credentials:   map[string]credentialCall{},
	}
}

func (c *recordingBkCIClient) CreateCredential(
	_ context.Context, _, credentialID, _, username, password string,
) error {
	c.credentialOps = append(c.credentialOps, "create")
	c.credentials[credentialID] = credentialCall{"create", credentialID, username, password}
	return nil
}

// UpdateCredential 凭证不存在时返回 ObjectNotFound
func (c *recordingBkCIClient) UpdateCredential(
	_ context.Context, _, credentialID, _, username, password string,
) error {
	c.credentialOps = append(c.credentialOps, "update")
	if _, ok := c.credentials[credentialID]; !ok {
		return cloudbkci.ObjectNotFound
	}
	c.credentials[credentialID] = credentialCall{"update", credentialID, username, password}
	return nil
}

func (c *recordingBkCIClient) CreateRepository(
	ctx context.Context, projectCode string, opts cloudbkci.CreateRepositoryOptions,
) (string, error) {
	c.createRepoOpts = append(c.createRepoOpts, opts)
	return c.StubApiClient.CreateRepository(ctx, projectCode, opts)
}

func (c *recordingBkCIClient) ListRepositoryBranches(
	ctx context.Context, projectCode, repositoryID, repositoryType, search string, page, pageSize int64,
) ([]cloudbkci.RepositoryRef, error) {
	c.refQueries = append(c.refQueries, repositoryType+":"+repositoryID)
	return c.StubApiClient.ListRepositoryBranches(
		ctx,
		projectCode,
		repositoryID,
		repositoryType,
		search,
		page,
		pageSize,
	)
}

var _ = Describe("Repository sync", func() {
	var (
		ctx    context.Context
		db     *mongo.Database
		client *recordingBkCIClient
		gitURL string
	)

	BeforeEach(func() {
		testutil.UseRepoSourceTypes(GinkgoT(), testutil.BkCIManagedRepoSourceType(), testutil.BasicRepoSourceType())

		ctx = auth.WithUser(testutil.NewDefaultTenantContext(), mockUser())
		db = testutil.NewDB(GinkgoT())
		projectStore, _, _ := setupManagerTestStores()
		Expect(projectStore.Create(ctx, mockProject())).To(Succeed())
		Expect(testutil.CleanupMetaColl(RepositoryMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(coderepo.CodeRepositoryMeta)).To(Succeed())

		gitURL = testutil.NewGitRepo(GinkgoT(), "v1.0.0").URL
		client = newRecordingBkCIClient()
	})

	// withClient 注入 recordingBkCIClient 后执行 fn
	withClient := func(fn func()) {
		mockey.PatchConvey("with recording bkci client", GinkgoT(), func() {
			mockey.Mock(cloudbkci.New).Return(client, nil).Build()
			fn()
		})
	}

	saveGitRepo := func(password string) *coderepo.CodeRepository {
		GinkgoHelper()
		repo, err := coderepo.Save(ctx, db, coderepo.SaveSpec{
			WorkspaceID: managerTestWorkspaceID, Type: testutil.BasicRepoSourceTypeName, URL: gitURL,
			Username: testutil.GitServerUsername(), Password: password,
		})
		Expect(err).NotTo(HaveOccurred())
		return repo
	}

	Describe("SyncRepository", func() {
		It("syncs a basic repository bound with a credential and overwrites the credential on every sync", func() {
			withClient(func() {
				repo := saveGitRepo(testutil.GitServerPassword())
				credentialID := repositoryCredentialID(repo.Alias)

				record, err := SyncRepository(ctx, db, repo)
				Expect(err).NotTo(HaveOccurred())
				Expect(record.CredentialID).To(Equal(credentialID))
				Expect(record.Alias).To(Equal(repo.Alias))
				Expect(client.createRepoOpts).To(Equal([]cloudbkci.CreateRepositoryOptions{{
					RepoType: cloudbkci.RepoTypeCodeTGit, URL: gitURL, Alias: repo.Alias, CredentialID: credentialID,
				}}))
				Expect(client.credentialOps).To(Equal([]string{"update", "create"}))
				Expect(client.credentials[credentialID].password).To(Equal(testutil.GitServerPassword()))

				// 再次同步：覆盖凭证，不重复创建代码库
				repo.BasicAuth.Password = "rotated"
				_, err = SyncRepository(ctx, db, repo)
				Expect(err).NotTo(HaveOccurred())
				Expect(client.createRepoOpts).To(HaveLen(1))
				Expect(client.credentials[credentialID]).To(Equal(
					credentialCall{"update", credentialID, testutil.GitServerUsername(), "rotated"},
				))
			})
		})

		It("syncs a bkci managed repository with oauth only once", func() {
			withClient(func() {
				repo, err := coderepo.Save(ctx, db, coderepo.SaveSpec{
					WorkspaceID: managerTestWorkspaceID, Type: config.DefaultRepoSourceTypeName,
					URL: "https://git.example.com/group/app.git", Alias: "group/app",
				})
				Expect(err).NotTo(HaveOccurred())

				record, err := SyncRepository(ctx, db, repo)
				Expect(err).NotTo(HaveOccurred())
				Expect(record.CredentialID).To(BeEmpty())
				_, err = SyncRepository(ctx, db, repo)
				Expect(err).NotTo(HaveOccurred())

				Expect(client.createRepoOpts).To(Equal([]cloudbkci.CreateRepositoryOptions{{
					RepoType: cloudbkci.RepoTypeCodeGit, URL: repo.URL, Alias: "group/app",
				}}))
				Expect(client.credentialOps).To(BeEmpty())
			})
		})
	})

	Describe("SyncBuildRepository", func() {
		It("associates a bkci managed repository missing from bkms before syncing", func() {
			withClient(func() {
				_, err := SyncBuildRepository(
					ctx,
					db,
					managerTestWorkspaceID,
					config.DefaultRepoSourceTypeName,
					"https://git.example.com/group/app.git",
					"group/app",
				)
				Expect(err).NotTo(HaveOccurred())
				repo, err := coderepo.Get(ctx, db, managerTestWorkspaceID, "group/app")
				Expect(err).NotTo(HaveOccurred())
				Expect(repo.SourceType).To(Equal(config.DefaultRepoSourceTypeName))
				Expect(client.createRepoOpts).To(HaveLen(1))
			})
		})

		It("requires the credential of a basic repository missing from bkms", func() {
			withClient(func() {
				_, err := SyncBuildRepository(
					ctx,
					db,
					managerTestWorkspaceID,
					testutil.BasicRepoSourceTypeName,
					gitURL,
					"",
				)
				Expect(err).To(MatchError(coderepo.ErrCredentialRequired))
				Expect(client.createRepoOpts).To(BeEmpty())
			})
		})
	})

	Describe("SaveWorkspaceRepositories", func() {
		It("syncs bkci managed repositories immediately and defers basic ones to build time", func() {
			withClient(func() {
				err := SaveWorkspaceRepositories(ctx, db, []coderepo.SaveSpec{
					{
						WorkspaceID: managerTestWorkspaceID, Type: config.DefaultRepoSourceTypeName,
						URL: "https://git.example.com/group/app.git", Alias: "group/app",
					},
					{
						WorkspaceID: managerTestWorkspaceID, Type: testutil.BasicRepoSourceTypeName, URL: gitURL,
						Username: testutil.GitServerUsername(), Password: testutil.GitServerPassword(),
					},
				})
				Expect(err).NotTo(HaveOccurred())

				n, err := store.Of(db, coderepo.CodeRepositoryMeta).
					Count(ctx, coderepo.ByWorkspace(managerTestWorkspaceID))
				Expect(err).NotTo(HaveOccurred())
				Expect(n).To(BeEquivalentTo(2))
				Expect(client.createRepoOpts).To(HaveLen(1))
				Expect(client.createRepoOpts[0].Alias).To(Equal("group/app"))
				Expect(client.credentialOps).To(BeEmpty())
			})
		})
	})

	Describe("ListWorkspaceRepositoryRefs", func() {
		query := coderepo.RefsQuery{Kind: coderepo.RefKindTag, Page: 1, PageSize: 10}

		It("lists refs of a basic repository via git by alias or bkci hash id", func() {
			withClient(func() {
				repo := saveGitRepo(testutil.GitServerPassword())
				refs, err := ListWorkspaceRepositoryRefs(
					ctx, db, managerTestWorkspaceID, repo.Alias, RepositoryIdentityTypeName, query,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(refs).To(HaveLen(1))
				Expect(refs[0].Name).To(Equal("v1.0.0"))

				record, err := SyncRepository(ctx, db, repo)
				Expect(err).NotTo(HaveOccurred())
				refs, err = ListWorkspaceRepositoryRefs(
					ctx,
					db,
					managerTestWorkspaceID,
					record.ID,
					RepositoryIdentityTypeID,
					query,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(refs).To(HaveLen(1))
				Expect(client.refQueries).To(BeEmpty())
			})
		})

		It("lists refs of a bkci managed repository via bkci", func() {
			withClient(func() {
				_, err := coderepo.Save(ctx, db, coderepo.SaveSpec{
					WorkspaceID: managerTestWorkspaceID, Type: config.DefaultRepoSourceTypeName,
					URL: "https://git.example.com/group/app.git", Alias: "group/app",
				})
				Expect(err).NotTo(HaveOccurred())

				refs, err := ListWorkspaceRepositoryRefs(
					ctx,
					db,
					managerTestWorkspaceID,
					"group/app",
					RepositoryIdentityTypeName,
					coderepo.RefsQuery{Kind: coderepo.RefKindBranch, Page: 1, PageSize: 10},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(refs).To(HaveLen(1))
				Expect(refs[0].Name).To(Equal("main"))
				Expect(client.refQueries).To(Equal([]string{"NAME:group/app"}))
			})
		})

		It("falls back to bkci for repositories not associated yet", func() {
			withClient(func() {
				_, err := ListWorkspaceRepositoryRefs(
					ctx,
					db,
					managerTestWorkspaceID,
					"unknown-hash",
					RepositoryIdentityTypeID,
					coderepo.RefsQuery{Kind: coderepo.RefKindBranch, Page: 1, PageSize: 10},
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(client.refQueries).To(Equal([]string{"ID:unknown-hash"}))
			})
		})
	})
})
