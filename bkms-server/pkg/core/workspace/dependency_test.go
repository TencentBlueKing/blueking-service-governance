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
	getCMDBInfo     func(ctx context.Context, bkCCBizID int64) (*bkcmdb.BusinessDetail, error)
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
	if f.getCMDBInfo == nil {
		return nil, nil
	}
	return f.getCMDBInfo(ctx, bkCCBizID)
}

func (f *fakeCMDBService) ListBusinessesWithLevel2Details(ctx context.Context) ([]bkcmdb.BusinessDetail, error) {
	return nil, nil
}

var _ = Describe("fetchCMDBInfo", func() {
	var (
		ctx       context.Context
		user      auth.User
		queriedID int64
	)

	BeforeEach(func() {
		ctx = context.Background()
		user = auth.User{ID: "tester"}
		queriedID = 0
	})

	mockCMDB := func() {
		mockey.Mock(bkcmdb.NewService).Return(&fakeCMDBService{
			getCMDBInfo: func(_ context.Context, bkCCBizID int64) (*bkcmdb.BusinessDetail, error) {
				queriedID = bkCCBizID
				return &bkcmdb.BusinessDetail{}, nil
			},
		}, nil).Build()
	}

	It("uses the request biz id without querying BCS", func() {
		mockey.PatchConvey("request biz", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(nil, errors.New("should not query bcs")).Build()
			mockCMDB()

			_, err := FetchCMDBInfo(ctx, "existing-bcs", 398)
			Expect(err).NotTo(HaveOccurred())
			Expect(queriedID).To(Equal(int64(398)))
		})
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
			mockCMDB()

			_, err := FetchCMDBInfo(ctx, "existing-bcs", 0)
			Expect(err).NotTo(HaveOccurred())
			Expect(queriedID).To(Equal(int64(398)))
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

			_, err := FetchCMDBInfo(ctx, "existing-bcs", 0)
			Expect(err).To(MatchError(ContainSubstring("has no associated bizID")))
		})
	})
})

var _ = Describe("ensureIndependentBCSProject", func() {
	var (
		ctx  context.Context
		user auth.User
	)

	BeforeEach(func() {
		ctx = testutil.NewDefaultTenantContext()
		user = auth.User{ID: "tester"}
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

			project, err := EnsureIndependentBCSProject(ctx, "current-workspace", "workspace-name", 398)
			Expect(err).NotTo(HaveOccurred())
			Expect(project.ID).To(Equal("bcs-uid"))
			Expect(project.Code).To(Equal("bkms-ws"))
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

			_, err := EnsureIndependentBCSProject(ctx, "current-workspace", "workspace-name", 398)
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

			project, err := EnsureIndependentBCSProject(ctx, "current-workspace", "workspace-name", 398)
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
		_, err := EnsureIndependentBCSProject(context.Background(), "current-workspace", "workspace-name", 398)
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
		// 不向当前租户暴露其他租户的工作空间
		Expect(err.Error()).To(ContainSubstring("already bound by another tenant"))
		Expect(err.Error()).NotTo(ContainSubstring(ws.ID))
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
					return nil, bkcc.ErrBusinessNotFound
				},
			}, nil).Build()

			_, _, err := PrepareIndependentBCSBinding(ctx, nil, "ws-1", "", 398)
			Expect(err).To(MatchError(ErrIndependentWorkspaceBizInvalid))
			Expect(err.Error()).To(ContainSubstring("validate bkcc biz 398"))
		})
	})

	It("keeps biz query failures other than not found as internal errors", func() {
		mockey.PatchConvey("biz query failed", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bkcmdb.NewService).Return(&fakeCMDBService{
				getBusinessByID: func(ctx context.Context, bizID int64) (*bkcc.Business, error) {
					return nil, errors.New("bkcc: request timeout")
				},
			}, nil).Build()

			_, _, err := PrepareIndependentBCSBinding(ctx, nil, "ws-1", "", 398)
			Expect(err).To(MatchError(ContainSubstring("bkcc: request timeout")))
			Expect(errors.Is(err, ErrIndependentWorkspaceBizInvalid)).To(BeFalse())
		})
	})

	DescribeTable("rejects binding an invalid existing BCS project as a request error",
		func(project *bcs.Project, getErr error, wantMsg string) {
			mockey.PatchConvey("invalid bound project", GinkgoT(), func() {
				mockey.Mock(auth.MustGetUser).Return(user).Build()
				mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
				mockey.Mock((*bcs.StubApiClient).GetProject).Return(project, getErr).Build()

				_, _, err := PrepareIndependentBCSBinding(ctx, nil, "ws-1", "existing-bcs", 0)
				Expect(err).To(MatchError(ErrBoundBCSProjectInvalid))
				Expect(err.Error()).To(ContainSubstring(wantMsg))
			})
		},
		Entry("project not found", nil, bcs.ErrProjectNotFound, "project not found"),
		Entry("project is not k8s",
			&bcs.Project{ID: "bcs-uid", Code: "existing-bcs", Kind: "mesos", BizID: "398"}, nil,
			"not a k8s project",
		),
		Entry("project has no biz",
			&bcs.Project{ID: "bcs-uid", Code: "existing-bcs", Kind: bcs.ProjectKindK8s}, nil,
			"has no associated bizID",
		),
	)

	It("keeps BCS query failures other than not found as internal errors", func() {
		mockey.PatchConvey("bcs query failed", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(bcs.New).Return(bcs.NewStub(user), nil).Build()
			mockey.Mock((*bcs.StubApiClient).GetProject).Return(nil, errors.New("bcs: gateway timeout")).Build()

			_, _, err := PrepareIndependentBCSBinding(ctx, nil, "ws-1", "existing-bcs", 0)
			Expect(err).To(MatchError(ContainSubstring("bcs: gateway timeout")))
			Expect(errors.Is(err, ErrBoundBCSProjectInvalid)).To(BeFalse())
		})
	})
})
