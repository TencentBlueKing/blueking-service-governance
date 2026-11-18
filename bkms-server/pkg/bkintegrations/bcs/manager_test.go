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

package bcs

import (
	"context"
	"errors"
	"strings"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbcs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

var _ = Describe("ProjectManager", func() {
	var (
		ctx     context.Context
		user    auth.User
		manager *ProjectManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		user = auth.User{ID: "tester"}
		manager = NewProjectManager()
	})

	It("loads a bindable existing project", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(&cloudbcs.Project{
				ID:    "bcs-uid",
				Code:  "existing-bcs",
				Kind:  cloudbcs.ProjectKindK8s,
				BizID: "398",
			}, nil).Build()

			project, err := manager.LoadBoundProject(ctx, "existing-bcs")
			Expect(err).NotTo(HaveOccurred())
			Expect(project.Code).To(Equal("existing-bcs"))
			Expect(project.BizID).To(Equal("398"))
		})
	})

	It("rejects binding a non-k8s project", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(&cloudbcs.Project{
				ID:   "bcs-uid",
				Code: "existing-bcs",
				Kind: "mesos",
			}, nil).Build()

			_, err := manager.LoadBoundProject(ctx, "existing-bcs")
			Expect(err).To(MatchError(ContainSubstring("is not a k8s project")))
		})
	})

	spec := ProjectSpec{Code: "bkms-ws", Name: "工作空间(ws)", BizID: 398}

	Context("in multi-tenant mode", func() {
		BeforeEach(func() {
			enableMultiTenantMode()
			ctx = tenant.WithTenantID(ctx, "tenant-a")
		})

		It("looks up by the tenant prefixed code and reuses the existing project", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(user).Build()
				mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
				var lookedUp string
				mockey.Mock((*cloudbcs.StubApiClient).GetProject).To(
					func(_ *cloudbcs.StubApiClient, _ context.Context, idOrCode string) (*cloudbcs.Project, error) {
						lookedUp = idOrCode
						return &cloudbcs.Project{
							ID: "bcs-uid", Code: idOrCode, Kind: cloudbcs.ProjectKindK8s, BizID: "398",
						}, nil
					},
				).Build()
				mockey.Mock((*cloudbcs.StubApiClient).CreateProject).
					Return(nil, errors.New("should not create when project already exists")).
					Build()

				project, err := manager.EnsureProject(ctx, spec)
				Expect(err).NotTo(HaveOccurred())
				Expect(lookedUp).To(Equal("tenant-a-bkms-ws"))
				Expect(project.ID).To(Equal("bcs-uid"))
				Expect(project.Code).To(Equal("tenant-a-bkms-ws"))
			})
		})

		It("creates the project with the unprefixed code and lets BCS add the tenant prefix", func() {
			mockey.PatchConvey("test", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(user).Build()
				mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
				mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(nil, cloudbcs.ErrProjectNotFound).Build()
				var created cloudbcs.CreateProjectInput
				mockey.Mock((*cloudbcs.StubApiClient).CreateProject).To(
					func(_ *cloudbcs.StubApiClient, _ context.Context, in cloudbcs.CreateProjectInput) (
						*cloudbcs.Project, error,
					) {
						created = in
						// 模拟多租户 BCS 改写 projectCode
						return &cloudbcs.Project{
							ID: "bcs-uid", Code: "tenant-a-" + in.ProjectCode, Kind: in.Kind, BizID: in.BusinessID,
						}, nil
					},
				).Build()

				project, err := manager.EnsureProject(ctx, spec)
				Expect(err).NotTo(HaveOccurred())
				Expect(created).To(Equal(cloudbcs.CreateProjectInput{
					Name:        "工作空间(ws)",
					ProjectCode: "bkms-ws",
					Kind:        cloudbcs.ProjectKindK8s,
					BusinessID:  "398",
				}))
				Expect(project.Code).To(Equal("tenant-a-bkms-ws"))
			})
		})
	})

	It("rejects reusing an existing project of another biz", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(&cloudbcs.Project{
				ID: "bcs-uid", Code: "bkms-ws", Kind: cloudbcs.ProjectKindK8s, BizID: "399",
			}, nil).Build()

			_, err := manager.EnsureProject(ctx, spec)
			Expect(err).To(MatchError(ErrProjectBizMismatch))
			Expect(err).To(MatchError(ContainSubstring("belongs to biz 399, not requested biz 398")))
		})
	})

	It("looks up by the unprefixed code when multi-tenant mode is disabled", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			var lookedUp string
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).To(
				func(_ *cloudbcs.StubApiClient, _ context.Context, idOrCode string) (*cloudbcs.Project, error) {
					lookedUp = idOrCode
					return &cloudbcs.Project{
						ID: "bcs-uid", Code: idOrCode, Kind: cloudbcs.ProjectKindK8s, BizID: "398",
					}, nil
				},
			).Build()

			_, err := manager.EnsureProject(tenant.WithTenantID(ctx, "tenant-a"), spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(lookedUp).To(Equal("bkms-ws"))
		})
	})

	It("does not create when the lookup fails for reasons other than not found", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(nil, errors.New("gateway error")).Build()
			mockey.Mock((*cloudbcs.StubApiClient).CreateProject).
				Return(nil, errors.New("should not create when lookup fails")).
				Build()

			_, err := manager.EnsureProject(ctx, spec)
			Expect(err).To(MatchError(ContainSubstring("gateway error")))
			Expect(err).NotTo(MatchError(ContainSubstring("should not create")))
		})
	})
})

var _ = Describe("AddBCSProjectTenantPrefix", func() {
	It("adds the tenant prefix in multi-tenant mode", func() {
		enableMultiTenantMode()
		ctx := tenant.WithTenantID(context.Background(), "tenant-a")
		Expect(AddBCSProjectTenantPrefix(ctx, "bkms-ws")).To(Equal("tenant-a-bkms-ws"))
	})

	It("keeps a code that already has the tenant prefix", func() {
		enableMultiTenantMode()
		ctx := tenant.WithTenantID(context.Background(), "tenant-a")
		Expect(AddBCSProjectTenantPrefix(ctx, "tenant-a-bkms-ws")).To(Equal("tenant-a-bkms-ws"))
	})

	It("keeps the code when multi-tenant mode is disabled", func() {
		ctx := tenant.WithTenantID(context.Background(), "tenant-a")
		Expect(AddBCSProjectTenantPrefix(ctx, "bkms-ws")).To(Equal("bkms-ws"))
	})
})

// enableMultiTenantMode 在当前用例内开启多租户模式，用例结束后恢复全局配置
func enableMultiTenantMode() {
	oldConfig := config.G
	DeferCleanup(func() { config.G = oldConfig })
	config.G = &config.Config{}
	config.G.Tenant.EnableMultiTenantMode = true
}

var _ = Describe("WorkspaceProjectName", func() {
	It("appends the workspace id so same display names do not collide", func() {
		Expect(WorkspaceProjectName("支付", "pay-1")).To(Equal("支付(pay-1)"))
		Expect(WorkspaceProjectName("支付", "pay-2")).NotTo(Equal(WorkspaceProjectName("支付", "pay-1")))
	})

	It("stays within the BCS name length 2-64 for workspace input limits", func() {
		// displayName 1-32 字符，workspace ID 1-27 字符
		Expect(len([]rune(WorkspaceProjectName("a", "w")))).To(BeNumerically(">=", 2))
		longName := WorkspaceProjectName(strings.Repeat("名", 32), strings.Repeat("w", 27))
		Expect(len([]rune(longName))).To(BeNumerically("<=", 64))
	})
})
