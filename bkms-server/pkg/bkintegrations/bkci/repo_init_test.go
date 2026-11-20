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

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

var _ = Describe("RepositoryManager Rebind", func() {
	const (
		alias = "group/repo"
		url   = "https://git.example.com/group/repo.git"
	)

	var (
		ctx         context.Context
		workspaceID string
		manager     *RepositoryManager
		repos       *RepositoryStore
	)

	BeforeEach(func() {
		useStub := config.G.Development.UseStubBkCI
		config.G.Development.UseStubBkCI = true
		DeferCleanup(func() { config.G.Development.UseStubBkCI = useStub })

		ctx = testutil.NewDefaultTenantContext()
		workspaceID = "ws" + stringx.Random(8)
		manager = NewRepositoryManager(testutil.NewDB(GinkgoT()), workspaceID)
		repos = testutil.NewStore(GinkgoT(), RepositoryMeta)
		Expect(testutil.NewStore(GinkgoT(), ProjectMeta).Create(ctx, &Project{
			ID:          "pid-" + workspaceID,
			Code:        "bkms-" + workspaceID,
			WorkspaceID: workspaceID,
			Creator:     "test-user",
		})).To(Succeed())
		Expect(repos.Create(ctx, &Repository{
			ID:          "stale-repo-id",
			Alias:       alias,
			URL:         url,
			WorkspaceID: workspaceID,
			ProjectCode: "bkms-" + workspaceID,
			Creator:     "test-user",
		})).To(Succeed())
	})

	It("should replace the local record with the repository registered in bkci", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()

			repo, err := manager.Rebind(ctx, url, alias)

			Expect(err).NotTo(HaveOccurred())
			Expect(repo.ID).NotTo(Equal("stale-repo-id"))
			stored, err := repos.List(ctx, ByWorkspaceAndAlias(workspaceID, alias))
			Expect(err).NotTo(HaveOccurred())
			Expect(stored).To(HaveLen(1))
			Expect(stored[0].ID).To(Equal(repo.ID))
		})
	})

	It("should keep the local record when bkci rejects the registration", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(mockUser()).Build()
			mockey.Mock((*cloudbkci.StubApiClient).CreateRepository).Return(
				"", errors.Wrap(cloudbkci.PermissionDenied, "403 Forbidden"),
			).Build()

			_, err := manager.Rebind(ctx, url, alias)

			Expect(err).To(MatchError(cloudbkci.PermissionDenied))
			stored, err := repos.Get(ctx, ByWorkspaceAndAlias(workspaceID, alias))
			Expect(err).NotTo(HaveOccurred())
			Expect(stored.ID).To(Equal("stale-repo-id"))
		})
	})
})

var _ = Describe("WrapRepositoryErr", func() {
	It("should map bkci permission denied to no permission with detail code", func() {
		err := WrapRepositoryErr(errors.Wrap(cloudbkci.PermissionDenied, "403 Forbidden"), "ensure bkci repositories")

		Expect(err.Code()).To(Equal(bkerrs.ErrCodeNoPermission))
		Expect(err.Details()).To(HaveLen(1))
		Expect(err).To(MatchError(cloudbkci.PermissionDenied))
	})

	It("should map other errors to internal server error", func() {
		err := WrapRepositoryErr(errors.New("boom"), "ensure bkci repositories")

		Expect(err.Code()).To(Equal(bkerrs.ErrCodeInternalServerError))
		Expect(err.Details()).To(BeEmpty())
	})
})
