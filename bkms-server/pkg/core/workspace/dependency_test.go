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

package workspace_test

import (
	"context"
	"errors"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bcsmanager "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bcs"
	bkcmdb "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/cmdb"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkcc"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/txcmdb"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

type fakeCMDBService struct {
	getBusinessByID func(ctx context.Context, bizID int64) (*bkcc.Business, error)
}

func (f *fakeCMDBService) GetBusinessByID(ctx context.Context, bizID int64) (*bkcc.Business, error) {
	return f.getBusinessByID(ctx, bizID)
}

func (f *fakeCMDBService) ListBusinesses(ctx context.Context) ([]bkcc.Business, error) {
	return nil, nil
}

func (f *fakeCMDBService) GetLevel2BusinessDetail(
	ctx context.Context, level2BizID int64,
) (*txcmdb.Level2BusinessDetail, error) {
	return nil, nil
}

func (f *fakeCMDBService) ListLevel2BusinessDetails(
	ctx context.Context, level2BizIDs []int64,
) ([]txcmdb.Level2BusinessDetail, error) {
	return nil, nil
}

func (f *fakeCMDBService) GetCMDBInfo(ctx context.Context, bkCCBizID int64) (*bkcmdb.BusinessDetail, error) {
	return nil, nil
}

func (f *fakeCMDBService) ListBusinessesWithLevel2Details(ctx context.Context) ([]bkcmdb.BusinessDetail, error) {
	return nil, nil
}

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
		bizID, err := GetBCSProjectBizID(ctx, "", 398)
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

			bizID, err := GetBCSProjectBizID(ctx, "existing-bcs", 0)
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

			_, err := GetBCSProjectBizID(ctx, "existing-bcs", 0)
			Expect(err).To(MatchError(ContainSubstring("has no associated bizID")))
		})
	})
})

var _ = Describe("ensureIndependentBCSProject", func() {
	var (
		ctx   context.Context
		user  auth.User
		store *WorkspaceStore
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		user = auth.User{ID: "tester"}
		store = testutil.NewStore(GinkgoT(), Meta)
	})

	It("reuses the existing deterministic BCS project before creating a new one", func() {
		mockey.PatchConvey("reuse existing project", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
				ID:    "bcs-uid",
				Code:  "bkms-ws",
				Kind:  bcs.ProjectKindK8s,
				BizID: "398",
			}, nil).Build()
			mockey.Mock((*bcs.StubApiClient).CreateProject).
				Return(nil, errors.New("should not create when project already exists")).
				Build()

			project, err := EnsureIndependentBCSProject(ctx, store, "current-workspace", "workspace-name", 398)
			Expect(err).NotTo(HaveOccurred())
			Expect(project.ID).To(Equal("bcs-uid"))
			Expect(project.Code).To(Equal("bkms-ws"))
		})
	})

	It("rejects reusing an existing deterministic BCS project that is already bound by another workspace", func() {
		mockey.PatchConvey("occupied deterministic project", GinkgoT(), func() {
			ws := dbfactory.Workspace(ctx, store)
			ws.BkSystems = BkSystems{
				BkBCSProjectID:   "bcs-uid",
				BkBCSProjectCode: "bkms-ws",
			}
			Expect(Update(ctx, store, ws)).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = store.DeleteIfExists(ctx, ByWorkspaceID(ws.ID))
			})

			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
				ID:    "bcs-uid",
				Code:  "bkms-ws",
				Kind:  bcs.ProjectKindK8s,
				BizID: "398",
			}, nil).Build()

			_, err := EnsureIndependentBCSProject(ctx, store, "current-workspace", "workspace-name", 398)
			Expect(err).To(MatchError(ContainSubstring("already bound by workspace")))
		})
	})

	It("rejects reusing an existing deterministic BCS project from another biz", func() {
		mockey.PatchConvey("biz id mismatch", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
				ID:    "bcs-uid",
				Code:  "bkms-ws",
				Kind:  bcs.ProjectKindK8s,
				BizID: "399",
			}, nil).Build()
			mockey.Mock((*bcs.StubApiClient).CreateProject).
				Return(nil, errors.New("should not create when conflicting project already exists")).
				Build()

			_, err := EnsureIndependentBCSProject(ctx, store, "current-workspace", "workspace-name", 398)
			Expect(err).To(MatchError(ContainSubstring("belongs to biz 399, not requested biz 398")))
		})
	})

	It("creates the BCS project with the deterministic id and stores the code returned by BCS", func() {
		mockey.PatchConvey("create after not found", GinkgoT(), func() {
			tenantID, _ := tenant.GetTenantID(ctx)
			wantID := bcsmanager.WorkspaceProjectID(tenantID, "current-workspace")

			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			var lookedUp string
			mockey.Mock((*bcs.StubApiClient).GetProject).To(
				func(_ *bcs.StubApiClient, _ context.Context, id string) (*bcs.Project, error) {
					lookedUp = id
					return nil, bcs.ErrProjectNotFound
				},
			).Build()
			var created bcs.CreateProjectInput
			mockey.Mock((*bcs.StubApiClient).CreateProject).To(
				func(_ *bcs.StubApiClient, _ context.Context, in bcs.CreateProjectInput) (*bcs.Project, error) {
					created = in
					// 模拟多租户 BCS 改写 projectCode
					return &bcs.Project{
						ID: in.ProjectID, Code: tenantID + "-" + in.ProjectCode, Kind: in.Kind, BizID: in.BusinessID,
					}, nil
				},
			).Build()

			project, err := EnsureIndependentBCSProject(ctx, store, "current-workspace", "workspace-name", 398)
			Expect(err).NotTo(HaveOccurred())
			Expect(lookedUp).To(Equal(wantID))
			Expect(created.ProjectID).To(Equal(wantID))
			Expect(created.ProjectCode).To(Equal("bkms-current-workspace"))
			Expect(created.Name).To(Equal("workspace-name(current-workspace)"))
			Expect(project.ID).To(Equal(wantID))
			Expect(project.Code).To(Equal(tenantID + "-bkms-current-workspace"))
			Expect(project.BizID).To(Equal("398"))
		})
	})

	It("fails without tenant in context instead of guessing the project id", func() {
		_, err := EnsureIndependentBCSProject(context.Background(), store, "current-workspace", "workspace-name", 398)
		Expect(err).To(MatchError(tenant.ErrTenantIDRequired))
	})
})

var _ = Describe("ensureBCSProjectNotBound", func() {
	var (
		ctx   context.Context
		store *WorkspaceStore
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		store = testutil.NewStore(GinkgoT(), Meta)
	})

	It("rejects binding a BCS project that is already occupied by another workspace", func() {
		ws := dbfactory.Workspace(ctx, store)
		ws.BkSystems = BkSystems{
			BkBCSProjectID:   "bcs-uid",
			BkBCSProjectCode: "existing-bcs",
		}
		Expect(Update(ctx, store, ws)).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_ = store.DeleteIfExists(ctx, ByWorkspaceID(ws.ID))
		})

		err := EnsureBCSProjectNotBound(ctx, store, "current-workspace", &bcs.Project{
			ID:   "bcs-uid",
			Code: "existing-bcs",
		})
		Expect(err).To(MatchError(ErrBCSProjectAlreadyBound))
		Expect(err.Error()).To(ContainSubstring("already bound by workspace"))
	})

	It("rejects binding a BCS project that is already occupied by a workspace of another tenant", func() {
		ws := dbfactory.Workspace(ctx, store)
		ws.BkSystems = BkSystems{
			BkBCSProjectID:   "bcs-uid",
			BkBCSProjectCode: "existing-bcs",
		}
		Expect(Update(ctx, store, ws)).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_ = store.DeleteIfExists(ctx, ByWorkspaceID(ws.ID))
		})

		otherTenantCtx := tenant.WithTenantID(context.Background(), "other-tenant")
		err := EnsureBCSProjectNotBound(otherTenantCtx, store, "current-workspace", &bcs.Project{
			ID:   "bcs-uid",
			Code: "existing-bcs",
		})
		Expect(err).To(MatchError(ErrBCSProjectAlreadyBound))
	})

	It("allows the workspace that already owns the BCS project", func() {
		ws := dbfactory.Workspace(ctx, store)
		ws.BkSystems = BkSystems{
			BkBCSProjectID:   "bcs-uid",
			BkBCSProjectCode: "existing-bcs",
		}
		Expect(Update(ctx, store, ws)).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_ = store.DeleteIfExists(ctx, ByWorkspaceID(ws.ID))
		})

		err := EnsureBCSProjectNotBound(ctx, store, ws.ID, &bcs.Project{ID: "bcs-uid", Code: "existing-bcs"})
		Expect(err).NotTo(HaveOccurred())
	})

	It("allows binding when the BCS project is not occupied by any workspace", func() {
		err := EnsureBCSProjectNotBound(ctx, store, "current-workspace", &bcs.Project{
			ID:   "bcs-uid",
			Code: "existing-bcs",
		})
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("prepareIndependentBCSBinding", func() {
	var (
		ctx  context.Context
		user auth.User
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		user = auth.User{ID: "tester"}
	})

	It("rejects auto create when the deterministic BCS project belongs to another biz", func() {
		mockey.PatchConvey("bcs project belongs to another biz", GinkgoT(), func() {
			tenantID, _ := tenant.GetTenantID(ctx)
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bkcmdb.NewService).Return(&fakeCMDBService{
				getBusinessByID: func(ctx context.Context, bizID int64) (*bkcc.Business, error) {
					return &bkcc.Business{BizID: "398"}, nil
				},
			}, nil).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			var lookedUp string
			mockey.Mock((*bcs.StubApiClient).GetProject).To(
				func(_ *bcs.StubApiClient, _ context.Context, id string) (*bcs.Project, error) {
					lookedUp = id
					return &bcs.Project{
						ID: id, Code: tenantID + "-bkms-ws-1", Kind: bcs.ProjectKindK8s, BizID: "399",
					}, nil
				},
			).Build()

			_, _, err := PrepareIndependentBCSBinding(ctx, nil, "ws-1", "", 398)
			Expect(lookedUp).To(Equal(bcsmanager.WorkspaceProjectID(tenantID, "ws-1")))
			Expect(err).To(MatchError(ErrBCSProjectAlreadyExists))
			Expect(err.Error()).To(ContainSubstring("belongs to biz 399, not requested biz 398"))
		})
	})

	It("allows auto create when the deterministic BCS project does not exist", func() {
		mockey.PatchConvey("bcs project not found", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bkcmdb.NewService).Return(&fakeCMDBService{
				getBusinessByID: func(ctx context.Context, bizID int64) (*bkcc.Business, error) {
					return &bkcc.Business{BizID: "398"}, nil
				},
			}, nil).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(nil, bcs.ErrProjectNotFound).Build()

			project, cmdbInfo, err := PrepareIndependentBCSBinding(ctx, nil, "ws-1", "", 398)
			Expect(err).NotTo(HaveOccurred())
			Expect(project).To(BeNil())
			Expect(cmdbInfo.BizID).To(Equal("398"))
		})
	})

	Context("when an existing deterministic project can be reused", func() {
		var store *WorkspaceStore

		BeforeEach(func() {
			store = testutil.NewStore(GinkgoT(), Meta)
		})

		It("allows retry when the project has the same biz and is not bound", func() {
			mockey.PatchConvey("reusable leftover project", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(user).Build()
				mockey.Mock(bkcmdb.NewService).Return(&fakeCMDBService{
					getBusinessByID: func(ctx context.Context, bizID int64) (*bkcc.Business, error) {
						return &bkcc.Business{BizID: "398"}, nil
					},
				}, nil).Build()
				mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
				mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
					ID:    "bcs-uid",
					Code:  "bkms-ws-1",
					Kind:  bcs.ProjectKindK8s,
					BizID: "398",
				}, nil).Build()

				project, cmdbInfo, err := PrepareIndependentBCSBinding(ctx, store, "ws-1", "", 398)
				Expect(err).NotTo(HaveOccurred())
				Expect(project).To(BeNil())
				Expect(cmdbInfo.BizID).To(Equal("398"))
			})
		})

		It("rejects retry when another workspace already bound the project", func() {
			mockey.PatchConvey("occupied leftover project", GinkgoT(), func() {
				ws := dbfactory.Workspace(ctx, store)
				ws.BkSystems = BkSystems{
					BkBCSProjectID:   "bcs-uid",
					BkBCSProjectCode: "bkms-ws-1",
				}
				Expect(Update(ctx, store, ws)).NotTo(HaveOccurred())
				DeferCleanup(func() {
					_ = store.DeleteIfExists(ctx, ByWorkspaceID(ws.ID))
				})

				mockey.Mock(auth.MustGetUser).Return(user).Build()
				mockey.Mock(bkcmdb.NewService).Return(&fakeCMDBService{
					getBusinessByID: func(ctx context.Context, bizID int64) (*bkcc.Business, error) {
						return &bkcc.Business{BizID: "398"}, nil
					},
				}, nil).Build()
				mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
				mockey.Mock((*bcs.StubApiClient).GetProject).Return(&bcs.Project{
					ID:    "bcs-uid",
					Code:  "bkms-ws-1",
					Kind:  bcs.ProjectKindK8s,
					BizID: "398",
				}, nil).Build()

				_, _, err := PrepareIndependentBCSBinding(ctx, store, "ws-1", "", 398)
				Expect(err).To(MatchError(ErrBCSProjectAlreadyBound))
				Expect(err.Error()).To(ContainSubstring("already bound by workspace"))
			})
		})
	})

	It("rejects auto create when the requested biz is not accessible", func() {
		mockey.PatchConvey("biz inaccessible", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bkcmdb.NewService).Return(&fakeCMDBService{
				getBusinessByID: func(ctx context.Context, bizID int64) (*bkcc.Business, error) {
					return nil, errors.New("bkcc: business not found")
				},
			}, nil).Build()

			_, _, err := PrepareIndependentBCSBinding(ctx, nil, "ws-1", "", 398)
			Expect(err).To(MatchError(ErrIndependentWorkspaceBizInvalid))
			Expect(err.Error()).To(ContainSubstring("validate bkcc biz 398"))
		})
	})
})
