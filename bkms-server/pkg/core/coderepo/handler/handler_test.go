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

package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo/handler"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CodeRepo Handler Suite")
}

var _ = BeforeSuite(func() {
	if err := testutil.SetUpGlobalDatabase(); err != nil {
		panic("failed to set up global database: " + err.Error())
	}
})

var _ = AfterSuite(func() {
	if err := testutil.TeardownGlobalDatabase(); err != nil {
		panic("failed to teardown global database: " + err.Error())
	}
})

var _ = Describe("CodeRepo Handler", func() {
	var (
		router   *gin.Engine
		repo     *testutil.GitRepo
		permType ginperm.Type
	)

	post := func(path string, body any) *httptest.ResponseRecorder {
		payload, err := json.Marshal(body)
		Expect(err).NotTo(HaveOccurred())
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(auth.WithUser(testutil.NewDefaultTenantContext(), auth.User{ID: "tester"}))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	BeforeEach(func() {
		testutil.UseRepoSourceTypes(GinkgoT(), testutil.BkCIManagedRepoSourceType(), testutil.BasicRepoSourceType())
		Expect(testutil.CleanupMetaColl(coderepo.CodeRepositoryMeta)).To(Succeed())

		repo = testutil.NewGitRepo(GinkgoT(), "v1.0.0")
		gin.SetMode(gin.TestMode)
		router = gin.New()
		router.Use(bkerrs.ErrorHandler())
		coderepo.Register(router.Group(""), handler.New(&storereg.Registry{DB: testutil.NewDB(GinkgoT())}))

		mockey.Mock(ginperm.ValidateWorkspaceByID).To(
			func(_ context.Context, _ *storereg.Registry, workspaceID string, t ginperm.Type) (*workspace.Workspace, error) {
				permType = t
				return &workspace.Workspace{ID: workspaceID}, nil
			},
		).
			Build()
		DeferCleanup(mockey.UnPatchAll)
	})

	Describe("TestConnectivity", func() {
		It("succeeds with a correct credential and checks the view permission", func() {
			rec := post("/workspaces/ws-1/code-repositories/connectivity-test", map[string]any{
				"type": testutil.BasicRepoSourceTypeName, "repoURL": repo.URL,
				"username": testutil.GitServerUsername(), "password": testutil.GitServerPassword(),
			})
			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			Expect(permType).To(Equal(ginperm.TypeView))
		})

		It("returns a readable error for a wrong password", func() {
			rec := post("/workspaces/ws-1/code-repositories/connectivity-test", map[string]any{
				"type": testutil.BasicRepoSourceTypeName, "repoURL": repo.URL, "username": testutil.GitServerUsername(), "password": "wrong",
			})
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(rec.Body.String()).To(ContainSubstring("仓库账号或密码不正确"))
		})

		It("rejects an unknown source type", func() {
			rec := post("/workspaces/ws-1/code-repositories/connectivity-test", map[string]any{
				"type": "unknown", "repoURL": repo.URL, "username": "u", "password": "p",
			})
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(rec.Body.String()).To(ContainSubstring("不支持该代码库类型"))
		})
	})

	Describe("ListRefs", func() {
		It("lists tags of an unsaved basic repository", func() {
			rec := post("/workspaces/ws-1/code-repositories/refs", map[string]any{
				"type": testutil.BasicRepoSourceTypeName, "repoURL": repo.URL,
				"username": testutil.GitServerUsername(), "password": testutil.GitServerPassword(),
				"kind": "tag", "page": 1, "pageSize": 10,
			})
			Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
			var resp slz.ListRefsOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).To(HaveLen(1))
			Expect(resp.Data[0].Name).To(Equal("v1.0.0"))
			Expect(resp.Data[0].Path).To(Equal("refs/tags/v1.0.0"))
		})

		It("validates the query", func() {
			rec := post("/workspaces/ws-1/code-repositories/refs", map[string]any{
				"type": testutil.BasicRepoSourceTypeName, "repoURL": repo.URL, "kind": "commit", "page": 1, "pageSize": 10,
			})
			Expect(rec.Code).To(Equal(http.StatusBadRequest))
		})
	})
})
