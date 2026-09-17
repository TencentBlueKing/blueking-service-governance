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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

// credentialCall 记录对蓝盾凭证接口的调用
type credentialCall struct {
	op, credentialID, username, password string
}

// recordingBkCIClient 在 stub 客户端基础上记录凭证与代码库相关调用
type recordingBkCIClient struct {
	*cloudbkci.StubApiClient

	credentialCalls []credentialCall
	createRepoOpts  []cloudbkci.CreateRepositoryOptions
	// updateNotFound 为 true 时 UpdateCredential 返回 ObjectNotFound，模拟蓝盾侧凭证被删除
	updateNotFound bool
	// createErr 非空时 CreateCredential 返回该错误，模拟蓝盾侧残留同 ID 凭证
	createErr error
}

func (c *recordingBkCIClient) CreateCredential(
	_ context.Context, _, credentialID, _, username, password string,
) error {
	c.credentialCalls = append(c.credentialCalls, credentialCall{"create", credentialID, username, password})
	return c.createErr
}

func (c *recordingBkCIClient) UpdateCredential(
	_ context.Context, _, credentialID, _, username, password string,
) error {
	c.credentialCalls = append(c.credentialCalls, credentialCall{"update", credentialID, username, password})
	if c.updateNotFound {
		return cloudbkci.ObjectNotFound
	}
	return nil
}

func (c *recordingBkCIClient) DeleteCredential(_ context.Context, _, credentialID string) error {
	c.credentialCalls = append(c.credentialCalls, credentialCall{op: "delete", credentialID: credentialID})
	return cloudbkci.ObjectNotFound
}

func (c *recordingBkCIClient) CreateRepository(
	ctx context.Context, projectCode string, opts cloudbkci.CreateRepositoryOptions,
) (string, error) {
	c.createRepoOpts = append(c.createRepoOpts, opts)
	return c.StubApiClient.CreateRepository(ctx, projectCode, opts)
}

// gitPkt 按 pkt-line 格式编码一行
func gitPkt(s string) string {
	return fmt.Sprintf("%04x%s", len(s)+4, s)
}

// newFakeGitServer 模拟 Git Smart HTTP 服务，仅接受指定账号密码
func newFakeGitServer(username, password string) *httptest.Server {
	sha := strings.Repeat("a", 40)
	body := gitPkt("# service=git-upload-pack\n") + "0000" +
		gitPkt(sha+" HEAD\x00symref=HEAD:refs/heads/master\n") +
		gitPkt(sha+" refs/heads/feature/login\n") +
		gitPkt(sha+" refs/heads/master\n") +
		gitPkt(sha+" refs/heads/release\n") +
		gitPkt(sha+" refs/tags/nightly\n") +
		gitPkt(sha+" refs/tags/v1.2.0\n") +
		gitPkt(sha+" refs/tags/v1.10.0\n") +
		"0000"
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != username || p != password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		_, _ = w.Write([]byte(body))
	}))
}

var _ = Describe("RepositoryManager Ensure", func() {
	const repoAlias = "group/app"

	var (
		ctx        context.Context
		manager    *RepositoryManager
		repoStore  *RepositoryStore
		gitServer  *httptest.Server
		repoURL    string
		client     *recordingBkCIClient
		prevConfig *config.Config
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db := testutil.NewDB(GinkgoT())
		manager = NewRepositoryManager(db, managerTestWorkspaceID)
		projectStore, _, _ := setupManagerTestStores()
		Expect(testutil.CleanupMetaColl(RepositoryMeta)).To(Succeed())
		Expect(projectStore.Create(ctx, mockProject())).To(Succeed())
		repoStore = testutil.NewStore(GinkgoT(), RepositoryMeta)

		gitServer = newFakeGitServer("git", "secret")
		DeferCleanup(gitServer.Close)
		repoURL = gitServer.URL + "/group/app.git"

		prevConfig = config.G
		cfg := *config.G
		config.G = &cfg
		DeferCleanup(func() { config.G = prevConfig })

		client = &recordingBkCIClient{StubApiClient: cloudbkci.NewStub(mockUser())}
	})

	// withClient 在注入 recordingBkCIClient 的环境下执行 fn
	withClient := func(fn func()) {
		mockey.PatchConvey("with recording bkci client", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
			mockey.Mock(cloudbkci.New).Return(client, nil).Build()
			fn()
		})
	}

	Context("when git oauth is disabled", func() {
		BeforeEach(func() {
			config.G.FeatureIntegrations.DisableBkCIGitOAuth = true
		})

		It("creates a TGit repository bound with a username/password credential", func() {
			withClient(func() {
				repo, err := manager.Ensure(ctx, RepositoryInitSpec{
					URL: repoURL, Alias: repoAlias, Username: "git", Password: "secret",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(repo.UsesCredential()).To(BeTrue())
				Expect(repo.CredentialID).To(Equal("bkms_repo_group_app"))

				Expect(client.createRepoOpts).To(HaveLen(1))
				Expect(client.createRepoOpts[0].RepoType).To(Equal(cloudbkci.RepoTypeCodeTGit))
				// 测试 Git 服务为 http 地址
				Expect(client.createRepoOpts[0].AuthType).To(Equal(cloudbkci.AuthTypeHTTP))
				Expect(client.createRepoOpts[0].CredentialID).To(Equal("bkms_repo_group_app"))
				Expect(client.credentialCalls).To(ContainElement(
					credentialCall{"create", "bkms_repo_group_app", "git", "secret"},
				))

				// 密码加密落库，读取时自动解密
				raw := bson.M{}
				Expect(testutil.NewDB(GinkgoT()).Collection(RepositoryMeta.Collection).
					FindOne(ctx, bson.M{"alias": repoAlias}).Decode(&raw)).To(Succeed())
				Expect(raw["password"]).NotTo(BeEmpty())
				Expect(raw["password"]).NotTo(Equal("secret"))
				saved, err := repoStore.Get(ctx, ByWorkspaceAndAlias(managerTestWorkspaceID, repoAlias))
				Expect(err).NotTo(HaveOccurred())
				Expect(saved.Username).To(Equal("git"))
				Expect(saved.Password).To(Equal("secret"))
			})
		})

		It("does not delete credentials before creating one", func() {
			withClient(func() {
				_, err := manager.Ensure(ctx, RepositoryInitSpec{
					URL: repoURL, Alias: repoAlias, Username: "git", Password: "secret",
				})
				Expect(err).NotTo(HaveOccurred())
				// 社区版蓝盾删除不存在的凭证返回 403，首次关联不能依赖预删除
				Expect(client.credentialCalls).To(Equal([]credentialCall{
					{"create", "bkms_repo_group_app", "git", "secret"},
				}))
			})
		})

		It("updates a stale credential left in bkci when creating fails", func() {
			client.createErr = errors.New("credential already exists")
			withClient(func() {
				repo, err := manager.Ensure(ctx, RepositoryInitSpec{
					URL: repoURL, Alias: repoAlias, Username: "git", Password: "secret",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(repo.CredentialID).To(Equal("bkms_repo_group_app"))
				Expect(client.credentialCalls).To(Equal([]credentialCall{
					{"create", "bkms_repo_group_app", "git", "secret"},
					{"update", "bkms_repo_group_app", "git", "secret"},
				}))
			})
		})

		It("rejects wrong credentials without any side effect", func() {
			withClient(func() {
				_, err := manager.Ensure(ctx, RepositoryInitSpec{
					URL: repoURL, Alias: repoAlias, Username: "git", Password: "wrong",
				})
				Expect(errors.Is(err, ErrRepoCredentialInvalid)).To(BeTrue(), "unexpected error: %v", err)
				Expect(client.credentialCalls).To(BeEmpty())
				Expect(client.createRepoOpts).To(BeEmpty())
				_, err = repoStore.Get(ctx, ByWorkspaceAndAlias(managerTestWorkspaceID, repoAlias))
				Expect(errors.Is(err, ErrRepositoryNotFound)).To(BeTrue())
			})
		})

		It("verifies credentials without creating anything", func() {
			Expect(VerifyRepositoryCredential(ctx, repoURL, "git", "secret")).To(Succeed())
			Expect(errors.Is(VerifyRepositoryCredential(ctx, repoURL, "git", "wrong"), ErrRepoCredentialInvalid)).
				To(BeTrue())
			Expect(errors.Is(VerifyRepositoryCredential(ctx, repoURL, "git", ""), ErrRepoCredentialRequired)).
				To(BeTrue())
			Expect(errors.Is(
				VerifyRepositoryCredential(ctx, "https://127.0.0.1:1/app.git", "git", "secret"), ErrRepoUnreachable,
			)).To(BeTrue())

			Expect(client.credentialCalls).To(BeEmpty())
			Expect(client.createRepoOpts).To(BeEmpty())
		})

		It("requires username and password for the first binding", func() {
			withClient(func() {
				_, err := manager.Ensure(ctx, RepositoryInitSpec{URL: repoURL, Alias: repoAlias, Username: "git"})
				Expect(errors.Is(err, ErrRepoCredentialRequired)).To(BeTrue(), "unexpected error: %v", err)
			})
		})

		Context("when the repository is already bound", func() {
			BeforeEach(func() {
				withClient(func() {
					_, err := manager.Ensure(ctx, RepositoryInitSpec{
						URL: repoURL, Alias: repoAlias, Username: "git", Password: "secret",
					})
					Expect(err).NotTo(HaveOccurred())
				})
				client.credentialCalls = nil
				client.createRepoOpts = nil
			})

			It("keeps the saved credential when password is empty", func() {
				withClient(func() {
					repo, err := manager.Ensure(
						ctx,
						RepositoryInitSpec{URL: repoURL, Alias: repoAlias, Username: "git"},
					)
					Expect(err).NotTo(HaveOccurred())
					Expect(repo.Password).To(Equal("secret"))
					// 构建时不携带账号密码，同样沿用
					_, err = manager.Initialize(ctx, repoURL, repoAlias)
					Expect(err).NotTo(HaveOccurred())
					Expect(client.credentialCalls).To(BeEmpty())
					Expect(client.createRepoOpts).To(BeEmpty())
				})
			})

			It("requires password when username changes", func() {
				withClient(func() {
					_, err := manager.Ensure(
						ctx,
						RepositoryInitSpec{URL: repoURL, Alias: repoAlias, Username: "other"},
					)
					Expect(errors.Is(err, ErrRepoPasswordRequired)).To(BeTrue(), "unexpected error: %v", err)
				})
			})

			It("keeps the saved credential when the new password is wrong", func() {
				withClient(func() {
					_, err := manager.Ensure(ctx, RepositoryInitSpec{
						URL: repoURL, Alias: repoAlias, Username: "git", Password: "wrong",
					})
					Expect(errors.Is(err, ErrRepoCredentialInvalid)).To(BeTrue(), "unexpected error: %v", err)
					Expect(client.credentialCalls).To(BeEmpty())
					saved, err := repoStore.Get(ctx, ByWorkspaceAndAlias(managerTestWorkspaceID, repoAlias))
					Expect(err).NotTo(HaveOccurred())
					Expect(saved.Password).To(Equal("secret"))
				})
			})

			It("updates the bkci credential and local record with a new password", func() {
				// 仓库侧改了密码
				gitServer.Config.Handler = newFakeGitServer("git", "new-secret").Config.Handler
				client.updateNotFound = true
				withClient(func() {
					_, err := manager.Ensure(ctx, RepositoryInitSpec{
						URL: repoURL, Alias: repoAlias, Username: "git", Password: "new-secret",
					})
					Expect(err).NotTo(HaveOccurred())
					// 蓝盾侧凭证不存在时重新创建
					Expect(client.credentialCalls).To(Equal([]credentialCall{
						{"update", "bkms_repo_group_app", "git", "new-secret"},
						{"create", "bkms_repo_group_app", "git", "new-secret"},
					}))
					saved, err := repoStore.Get(ctx, ByWorkspaceAndAlias(managerTestWorkspaceID, repoAlias))
					Expect(err).NotTo(HaveOccurred())
					Expect(saved.Password).To(Equal("new-secret"))
				})
			})

			It("lists branches and tags from the git repository with the saved credential", func() {
				withClient(func() {
					query := RepositoryRefsQuery{
						Kind: RefKindBranch, RepositoryID: repoAlias, RepositoryType: RepositoryIdentityTypeName,
						Page: 1, PageSize: 2,
					}
					db := testutil.NewDB(GinkgoT())
					branches, err := ListRepositoryRefs(ctx, db, managerTestWorkspaceID, query)
					Expect(err).NotTo(HaveOccurred())
					// 默认分支在前
					Expect(names(branches)).To(Equal([]string{"master", "feature/login"}))

					query.Page = 2
					branches, err = ListRepositoryRefs(ctx, db, managerTestWorkspaceID, query)
					Expect(err).NotTo(HaveOccurred())
					Expect(names(branches)).To(Equal([]string{"release"}))

					query = RepositoryRefsQuery{
						Kind: RefKindTag, RepositoryID: repoAlias, RepositoryType: RepositoryIdentityTypeName,
						Page: 1, PageSize: 10,
					}
					tags, err := ListRepositoryRefs(ctx, db, managerTestWorkspaceID, query)
					Expect(err).NotTo(HaveOccurred())
					Expect(names(tags)).To(Equal([]string{"v1.10.0", "v1.2.0", "nightly"}))

					query.Search = "V1.1"
					tags, err = ListRepositoryRefs(ctx, db, managerTestWorkspaceID, query)
					Expect(err).NotTo(HaveOccurred())
					Expect(names(tags)).To(Equal([]string{"v1.10.0"}))
				})
			})
		})
	})

	Context("when git oauth is enabled", func() {
		It("creates a git repository with oauth and no credential", func() {
			withClient(func() {
				repo, err := manager.Ensure(ctx, RepositoryInitSpec{URL: repoURL, Alias: repoAlias})
				Expect(err).NotTo(HaveOccurred())
				Expect(repo.UsesCredential()).To(BeFalse())
				Expect(client.createRepoOpts).To(HaveLen(1))
				Expect(client.createRepoOpts[0].RepoType).To(Equal(cloudbkci.RepoTypeCodeGit))
				Expect(client.createRepoOpts[0].AuthType).To(BeEmpty())
				Expect(client.credentialCalls).To(BeEmpty())
			})
		})
	})
})

var _ = Describe("repositoryCredentialID", func() {
	It("keeps a readable id for short aliases", func() {
		Expect(repositoryCredentialID("group/sub-group/app")).To(Equal("bkms_repo_group_sub_group_app"))
	})

	It("uses a digest for long aliases", func() {
		id := repositoryCredentialID(strings.Repeat("very-long-group/", 4) + "app")
		Expect(id).To(HavePrefix("bkms_repo_"))
		Expect(len(id)).To(BeNumerically("<=", repoCredentialIDMaxLen))
		Expect(id).To(Equal(repositoryCredentialID(strings.Repeat("very-long-group/", 4) + "app")))
	})
})

func names(refs []cloudbkci.RepositoryRef) []string {
	result := make([]string, 0, len(refs))
	for _, ref := range refs {
		result = append(result, ref.Name)
	}
	return result
}
