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

package workspace

import (
	"context"
	"errors"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

var _ = Describe("getExistingBCSProject", func() {
	var (
		ctx  context.Context
		stub *bcs.StubApiClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		stub = bcs.NewStub(auth.User{ID: "tester"})
	})

	It("returns existing project", func() {
		proj, err := getExistingBCSProject(ctx, stub, "bkms-ws")
		Expect(err).NotTo(HaveOccurred())
		Expect(proj).NotTo(BeNil())
		Expect(proj.ID).NotTo(BeEmpty())
		Expect(proj.Code).To(Equal("bkms-ws"))
	})

	It("returns get error including not found", func() {
		mockey.PatchConvey("project not found", GinkgoT(), func() {
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(nil, bcs.ErrProjectNotFound).Build()

			_, err := getExistingBCSProject(ctx, stub, "bkms-ws")
			Expect(err).To(MatchError(bcs.ErrProjectNotFound))
		})
	})
})

var _ = Describe("createBCSProject", func() {
	var (
		ctx  context.Context
		stub *bcs.StubApiClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		stub = bcs.NewStub(auth.User{ID: "tester"})
	})

	It("creates project from display name and biz id", func() {
		proj, err := createBCSProject(ctx, stub, "workspace-name", "bkms-ws", 100)
		Expect(err).NotTo(HaveOccurred())
		Expect(proj.Code).To(Equal("bkms-ws"))
		Expect(proj.Name).To(Equal("workspace-name"))
		Expect(proj.Kind).To(Equal(bcs.ProjectKindK8s))
		Expect(proj.BizID).To(Equal("100"))
	})

	It("falls back to project code as name", func() {
		proj, err := createBCSProject(ctx, stub, "", "bkms-ws", 0)
		Expect(err).NotTo(HaveOccurred())
		Expect(proj.Name).To(Equal("bkms-ws"))
		Expect(proj.BizID).To(BeEmpty())
		Expect(proj.Kind).To(Equal(bcs.ProjectKindK8s))
	})

	It("surfaces create errors such as duplicate project code", func() {
		mockey.PatchConvey("create failed", GinkgoT(), func() {
			mockey.Mock((*bcs.StubApiClient).CreateProject).
				Return(nil, errors.New("project code already exists")).
				Build()

			_, err := createBCSProject(ctx, stub, "workspace-name", "bkms-ws", 100)
			Expect(err).To(MatchError(ContainSubstring("already exists")))
		})
	})
})

var _ = Describe("getBCSProjectBizID", func() {
	var (
		ctx  context.Context
		user auth.User
	)

	BeforeEach(func() {
		ctx = context.Background()
		user = auth.User{ID: "tester"}
	})

	It("uses the request biz id when creating a new project", func() {
		bizID, err := getBCSProjectBizID(ctx, "", 398)
		Expect(err).NotTo(HaveOccurred())
		Expect(bizID).To(Equal("398"))
	})

	It("loads biz id from the BCS project when binding an existing project", func() {
		mockey.PatchConvey("bind path", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
				ID:    "bcs-uid",
				Code:  "existing-bcs",
				BizID: "398",
			}, nil).Build()

			bizID, err := getBCSProjectBizID(ctx, "existing-bcs", 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(bizID).To(Equal("398"))
		})
	})

	It("fails when the bound BCS project has no biz id", func() {
		mockey.PatchConvey("missing biz", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
				ID:   "bcs-uid",
				Code: "existing-bcs",
			}, nil).Build()

			_, err := getBCSProjectBizID(ctx, "existing-bcs", 0)
			Expect(err).To(MatchError(ContainSubstring("has no associated bizID")))
		})
	})
})
