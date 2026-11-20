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

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	bkciproject "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("ListBkCIRepositoryBranches", func() {
	const (
		repoAlias = "group/helm-repo"
		repoURL   = "https://git.example.com/group/helm-repo.git"
	)

	var (
		ctx         context.Context
		h           *Handler
		workspaceID string
		repos       *bkciproject.RepositoryStore
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		db := testutil.NewDB(GinkgoT())
		h = &Handler{registry: &storereg.Registry{DB: db}}
		workspaceID = "ws" + stringx.Random(8)
		repos = testutil.NewStore(GinkgoT(), bkciproject.RepositoryMeta)

		Expect(store.Of(db, bkciproject.ProjectMeta).Create(ctx, &bkciproject.Project{
			ID:          "pid-" + workspaceID,
			Code:        "bkms-" + workspaceID,
			WorkspaceID: workspaceID,
			Creator:     "test-user",
		})).To(Succeed())
	})

	createHelmApp := func() {
		dbfactory.HelmApplication(ctx, &dbfactory.HelmApplicationStores{
			AppStore: testutil.NewStore(GinkgoT(), bkmsapp.AppMeta),
		}, &dbfactory.HelmApplicationOpts{
			WorkspaceID: workspaceID,
			HelmSource: &bkmsapp.HelmSource{
				RepoType: bkmsapp.HelmSourceRepoTypeGit,
				GitRepoConfig: &bkmsapp.GitRepoConfig{
					Type:      bkmsapp.GitRepoTypeTGit,
					RepoAlias: repoAlias,
					RepoURL:   repoURL,
					Revision:  "master",
				},
			},
		})
	}

	// listBranchesOnceNotFound 首次查询返回 ObjectNotFound，之后返回固定分支
	listBranchesOnceNotFound := func() *int {
		calls := 0
		mockey.Mock((*cloudbkci.StubApiClient).ListRepositoryBranches).To(
			func(
				_ *cloudbkci.StubApiClient, _ context.Context, _, _, _, _ string, _, _ int64,
			) ([]cloudbkci.RepositoryRef, error) {
				calls++
				if calls == 1 {
					return nil, cloudbkci.ObjectNotFound
				}
				return []cloudbkci.RepositoryRef{{Name: "main"}}, nil
			},
		).Build()
		return &calls
	}

	request := func() (int, map[string]any) {
		r := gin.New()
		r.Use(bkerrs.ErrorHandler())
		r.GET("/workspaces/:workspaceID/bkci-repositories/branches", h.ListBkCIRepositoryBranches)

		req := httptest.NewRequestWithContext(
			ctx, http.MethodGet,
			"/workspaces/"+workspaceID+"/bkci-repositories/branches?repositoryID=group%2Fhelm-repo"+
				"&repositoryType=NAME&page=1&pageSize=100",
			nil,
		)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		var body map[string]any
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		return w.Code, body
	}

	errorDetailCode := func(body map[string]any) string {
		details, _ := body["error"].(map[string]any)["details"].([]any)
		Expect(details).To(HaveLen(1))
		return details[0].(map[string]any)["code"].(string)
	}

	It("returns branches without rebinding when bkci finds the repository", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()

			code, body := request()

			Expect(code).To(Equal(http.StatusOK))
			Expect(body["data"]).To(HaveLen(1))
			_, err := repos.Get(ctx, bkciproject.ByWorkspaceAndAlias(workspaceID, repoAlias))
			Expect(err).To(MatchError(bkciproject.ErrRepositoryNotFound))
		})
	})

	It("binds the helm git repository and retries when bkci cannot find it", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
			calls := listBranchesOnceNotFound()
			createHelmApp()

			code, body := request()

			Expect(code).To(Equal(http.StatusOK))
			Expect(body["data"]).To(HaveLen(1))
			Expect(*calls).To(Equal(2))
			repo, err := repos.Get(ctx, bkciproject.ByWorkspaceAndAlias(workspaceID, repoAlias))
			Expect(err).NotTo(HaveOccurred())
			Expect(repo.URL).To(Equal(repoURL))
		})
	})

	It("binds the code repository of a build config when no helm app uses the alias", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
			listBranchesOnceNotFound()
			app := dbfactory.ApplicationWithOpts(
				ctx, testutil.NewStore(GinkgoT(), bkmsapp.AppMeta),
				&dbfactory.ApplicationOpts{WorkspaceID: workspaceID},
			)
			Expect(testutil.NewStore(GinkgoT(), build.ConfigMeta).Create(ctx, &build.Config{
				AppID:      app.ID,
				SourceType: build.SourceTypeCodeRepository,
				CodeRepo:   &build.RepositoryConfig{RepoAlias: repoAlias, RepoURL: repoURL},
			})).To(Succeed())

			code, _ := request()

			Expect(code).To(Equal(http.StatusOK))
			repo, err := repos.Get(ctx, bkciproject.ByWorkspaceAndAlias(workspaceID, repoAlias))
			Expect(err).NotTo(HaveOccurred())
			Expect(repo.URL).To(Equal(repoURL))
		})
	})

	It("replaces a stale local record with the repository registered again in bkci", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
			listBranchesOnceNotFound()
			Expect(repos.Create(ctx, &bkciproject.Repository{
				ID:          "stale-repo-id",
				Alias:       repoAlias,
				URL:         repoURL,
				WorkspaceID: workspaceID,
				ProjectCode: "bkms-" + workspaceID,
				Creator:     "test-user",
			})).To(Succeed())

			code, _ := request()

			Expect(code).To(Equal(http.StatusOK))
			stored, err := repos.List(ctx, bkciproject.ByWorkspaceAndAlias(workspaceID, repoAlias))
			Expect(err).NotTo(HaveOccurred())
			Expect(stored).To(HaveLen(1))
			Expect(stored[0].ID).NotTo(Equal("stale-repo-id"))
		})
	})

	It("returns not bound error when no repository url can be found", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
			listBranchesOnceNotFound()

			code, body := request()

			Expect(code).To(Equal(http.StatusNotFound))
			Expect(errorDetailCode(body)).To(Equal(string(bkerrs.ErrDetailCodeBkCIRepositoryNotBound)))
		})
	})

	It("returns permission denied error when bkci rejects binding the repository", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()
			listBranchesOnceNotFound()
			mockey.Mock((*cloudbkci.StubApiClient).CreateRepository).Return(
				"", errors.Wrap(cloudbkci.PermissionDenied, "403 Forbidden"),
			).Build()
			createHelmApp()

			code, body := request()

			Expect(code).To(Equal(http.StatusForbidden))
			Expect(errorDetailCode(body)).To(Equal(string(bkerrs.ErrDetailCodeBkCIRepositoryPermissionDenied)))
			_, err := repos.Get(ctx, bkciproject.ByWorkspaceAndAlias(workspaceID, repoAlias))
			Expect(err).To(MatchError(bkciproject.ErrRepositoryNotFound))
		})
	})
})
