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
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/serializer"
)

var _ = Describe("ListCustomBuildImages", func() {
	var (
		ctx    context.Context
		store  customruntime.Store
		diApp  *fxtest.App
		router *gin.Engine
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		diApp = fxtest.New(
			GinkgoT(),
			customruntime.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
		ctx = context.Background()

		router = gin.New()
		router.Use(bkerrs.ErrorHandler())
		image.Register(router.Group(""), New(&storereg.Registry{CustomRuntimeImageStore: store}))
		mockey.Mock(ginperm.ValidateWorkspaceByID).To(
			func(
				_ context.Context,
				_ *storereg.Registry,
				workspaceID string,
				permType ginperm.Type,
			) (*workspace.Workspace, error) {
				Expect(permType).To(Equal(ginperm.TypeView))
				return &workspace.Workspace{ID: workspaceID}, nil
			},
		).Build()
	})

	AfterEach(func() {
		Expect(store.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
		mockey.UnPatchAll()
	})

	It("filters by type and does not leak records from other workspaces", func() {
		workspaceID := "ws-demo-" + stringx.Random(6)
		builder := &customruntime.Image{
			WorkspaceID: workspaceID,
			Type:        customruntime.ImageTypeBuilder,
			Name:        "docker.bkrepo.example.com/demo/repo/my-golang",
		}
		runner := &customruntime.Image{
			WorkspaceID: workspaceID,
			Type:        customruntime.ImageTypeRunner,
			Name:        "docker.bkrepo.example.com/demo/repo/my-golang",
		}
		otherWorkspace := &customruntime.Image{
			WorkspaceID: "ws-other-" + stringx.Random(6),
			Type:        customruntime.ImageTypeBuilder,
			Name:        "docker.bkrepo.example.com/demo/repo/my-golang",
		}
		Expect(store.Upsert(ctx, builder)).To(Succeed())
		Expect(store.Upsert(ctx, runner)).To(Succeed())
		Expect(store.Upsert(ctx, otherWorkspace)).To(Succeed())

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodGet,
			"/workspaces/"+workspaceID+"/custom-build-images?type=builder&keyword=golang",
			nil,
		)
		router.ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusOK))
		var resp serializer.ListCustomRuntimeImagesOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Data).NotTo(BeNil())
		Expect(resp.Data.Results).To(HaveLen(1))
		Expect(resp.Data.Results[0].ID).To(Equal(builder.ID))
		Expect(resp.Data.Results[0].Type).To(Equal(string(customruntime.ImageTypeBuilder)))
		Expect(resp.Data.Results[0].Name).To(Equal(builder.Name))
	})

	It("returns an empty results list when the workspace has no records", func() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(
			http.MethodGet,
			"/workspaces/ws-empty-"+stringx.Random(6)+"/custom-build-images?type=runner",
			nil,
		)
		router.ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusOK))
		var resp serializer.ListCustomRuntimeImagesOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.Data).NotTo(BeNil())
		Expect(resp.Data.Results).To(BeEmpty())
	})
})
