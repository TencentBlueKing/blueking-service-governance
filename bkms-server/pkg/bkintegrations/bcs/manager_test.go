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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbcs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
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

	spec := ProjectSpec{ID: "fixed-project-id", Code: "bkms-ws", Name: "工作空间(ws)", BizID: 398}

	It("looks up by the given project id and reuses the existing project", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			var lookedUp string
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).To(
				func(_ *cloudbcs.StubApiClient, _ context.Context, id string) (*cloudbcs.Project, error) {
					lookedUp = id
					// 多租户下 BCS 返回的 code 带租户前缀
					return &cloudbcs.Project{
						ID: id, Code: "tenant-a-bkms-ws", Kind: cloudbcs.ProjectKindK8s, BizID: "398",
					}, nil
				},
			).Build()
			mockey.Mock((*cloudbcs.StubApiClient).CreateProject).
				Return(nil, errors.New("should not create when project already exists")).
				Build()

			project, err := manager.EnsureProject(ctx, spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(lookedUp).To(Equal("fixed-project-id"))
			Expect(project.ID).To(Equal("fixed-project-id"))
			Expect(project.Code).To(Equal("tenant-a-bkms-ws"))
		})
	})

	It("rejects reusing an existing project of another biz", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(&cloudbcs.Project{
				ID: "fixed-project-id", Code: "bkms-ws", Kind: cloudbcs.ProjectKindK8s, BizID: "399",
			}, nil).Build()

			_, err := manager.EnsureProject(ctx, spec)
			Expect(err).To(MatchError(ContainSubstring("belongs to biz 399, not requested biz 398")))
		})
	})

	It("creates the project with the given id, name and code when it does not exist", func() {
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
					return &cloudbcs.Project{
						ID: in.ProjectID, Code: "tenant-a-" + in.ProjectCode, Kind: in.Kind, BizID: in.BusinessID,
					}, nil
				},
			).Build()

			project, err := manager.EnsureProject(ctx, spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(created).To(Equal(cloudbcs.CreateProjectInput{
				ProjectID:   "fixed-project-id",
				Name:        "工作空间(ws)",
				ProjectCode: "bkms-ws",
				Kind:        cloudbcs.ProjectKindK8s,
				BusinessID:  "398",
			}))
			Expect(project.Code).To(Equal("tenant-a-bkms-ws"))
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

var _ = Describe("WorkspaceProjectID", func() {
	It("is deterministic and has the 32-char BCS project id format", func() {
		id := WorkspaceProjectID("tenant-a", "ws")
		Expect(id).To(Equal(WorkspaceProjectID("tenant-a", "ws")))
		Expect(id).To(MatchRegexp(`^[0-9a-f]{32}$`))
	})

	It("differs across tenants and workspaces", func() {
		id := WorkspaceProjectID("tenant-a", "ws")
		Expect(WorkspaceProjectID("tenant-b", "ws")).NotTo(Equal(id))
		Expect(WorkspaceProjectID("tenant-a", "ws2")).NotTo(Equal(id))
	})
})

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
