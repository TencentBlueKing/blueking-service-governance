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
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// linkedProjectView 关联项目的导出字段，用于断言
type linkedProjectView struct {
	System        LinkedSystem
	ProjectID     string
	ProjectName   string
	Origin        LinkedProjectOrigin
	State         LinkedProjectState
	BlockedReason LinkedProjectBlockedReason
}

func toLinkedProjectView(p LinkedProject, _ int) linkedProjectView {
	return linkedProjectView{
		System:        p.System,
		ProjectID:     p.ProjectID,
		ProjectName:   p.ProjectName,
		Origin:        p.Origin,
		State:         p.State,
		BlockedReason: p.BlockedReason,
	}
}

var _ = Describe("Linked projects", func() {
	var (
		originalCfg *config.Config
		ctx         context.Context
		code        string
		ws          *Workspace
	)

	BeforeEach(func() {
		originalCfg = config.G
		config.G = &config.Config{}
		config.G.Development.UseStubBkCI = true
		config.G.Development.UseStubBCS = true
		config.G.Development.UseStubBkMonitor = true
		ctx = auth.WithUser(context.Background(), auth.User{ID: "admin"})

		// stub 按项目 Code 全局记录启停状态，每个用例使用不同的 Code 避免相互影响
		code = fmt.Sprintf("bkms-lp-%d", time.Now().UnixNano())
		ws = &Workspace{
			ID: "demo",
			BkSystems: BkSystems{
				BkCIProjectID:      code,
				BkBCSProjectCode:   code,
				BkMonitorProjectID: "-100001",
			},
		}
	})

	AfterEach(func() {
		config.G = originalCfg
	})

	Describe("ListLinkedProjects", func() {
		It("lists bkci, bkmonitor and bcs projects in order", func() {
			projects := ListLinkedProjects(ctx, ws)

			Expect(lo.Map(projects, toLinkedProjectView)).To(Equal([]linkedProjectView{
				{
					System: LinkedSystemBKCI, ProjectID: code, ProjectName: "Stub Project",
					Origin: LinkedProjectCreated, State: LinkedProjectReady,
				},
				{
					System: LinkedSystemBKMonitor, ProjectID: "-100001", ProjectName: "Stub 项目",
					Origin: LinkedProjectCreated, State: LinkedProjectReady,
				},
				{
					System: LinkedSystemBCS, ProjectID: code, ProjectName: fmt.Sprintf("Stub 项目 (%s)", code),
					Origin: LinkedProjectCreated, State: LinkedProjectReady, BlockedReason: LinkedProjectUnsupported,
				},
			}))
		})

		It("marks bound projects and bkmonitor without bkms prefix as unsupported", func() {
			ws.BkSystems.IsBoundExistedBKCIProject = true
			ws.BkSystems.BkBCSProjectCode = "existing"

			projects := ListLinkedProjects(ctx, ws)

			Expect(lo.Map(projects, func(p LinkedProject, _ int) LinkedProjectOrigin { return p.Origin })).
				To(HaveEach(LinkedProjectBound))
			Expect(projects[1].System).To(Equal(LinkedSystemBKMonitor))
			Expect(projects[1].BlockedReason).To(Equal(LinkedProjectUnsupported))
		})

		It("skips bkmonitor before the workspace is activated", func() {
			ws.BkSystems.BkMonitorProjectID = ""

			projects := ListLinkedProjects(ctx, ws)

			Expect(lo.Map(projects, func(p LinkedProject, _ int) LinkedSystem { return p.System })).
				To(Equal([]LinkedSystem{LinkedSystemBKCI, LinkedSystemBCS}))
		})

		It("reports missing manage permission", func() {
			mockey.PatchConvey("no manage permission", GinkgoT(), func() {
				mockey.Mock((*bkci.StubApiClient).GetProject).Return(&bkci.Project{
					Code: code, Name: "蓝盾项目", Enabled: lo.ToPtr(true),
				}, nil).Build()

				projects := ListLinkedProjects(ctx, ws)

				Expect(projects[0].BlockedReason).To(Equal(LinkedProjectNoPermission))
				Expect(projects[0].Operable()).To(BeFalse())
			})
		})

		It("keeps bkci state unknown when bkci does not return the enabled state", func() {
			mockey.PatchConvey("no enabled field", GinkgoT(), func() {
				mockey.Mock((*bkci.StubApiClient).GetProject).Return(&bkci.Project{
					Code: code, Name: "蓝盾项目", HasManagePerm: true,
				}, nil).Build()

				projects := ListLinkedProjects(ctx, ws)

				Expect(projects[0].State).To(Equal(LinkedProjectUnknown))
				Expect(projects[0].Operable()).To(BeTrue())
				targets := SelectLinkedProjectsToEnable(projects)
				Expect(targets).To(ContainElement(HaveField("System", LinkedSystemBKCI)))
			})
		})

		It("falls back to unknown state and project ID as name when platform queries fail", func() {
			mockey.PatchConvey("platform down", GinkgoT(), func() {
				mockey.Mock((*bkci.StubApiClient).GetProject).Return(nil, errors.New("bkci down")).Build()
				mockey.Mock((*bkmonitor.StubClient).GetMetadataSpaceDetail).
					Return(nil, errors.New("bkmonitor down")).Build()
				mockey.Mock((*bcs.StubApiClient).GetProject).Return(nil, errors.New("bcs down")).Build()

				projects := ListLinkedProjects(ctx, ws)

				Expect(lo.Map(projects, toLinkedProjectView)).To(Equal([]linkedProjectView{
					{
						System: LinkedSystemBKCI, ProjectID: code, ProjectName: code,
						Origin: LinkedProjectCreated, State: LinkedProjectUnknown,
					},
					{
						System: LinkedSystemBKMonitor, ProjectID: "-100001", ProjectName: "-100001",
						Origin: LinkedProjectCreated, State: LinkedProjectUnknown,
					},
					{
						System: LinkedSystemBCS, ProjectID: code, ProjectName: code,
						Origin: LinkedProjectCreated, State: LinkedProjectReady, BlockedReason: LinkedProjectUnsupported,
					},
				}))
			})
		})
	})

	Describe("SetLinkedProjectsEnabled", func() {
		It("disables and re-enables bkci and bkmonitor projects", func() {
			targets, err := SelectLinkedProjectsToDisable(
				ListLinkedProjects(ctx, ws), []LinkedSystem{LinkedSystemBKMonitor, LinkedSystemBKCI},
			)
			Expect(err).NotTo(HaveOccurred())

			results := SetLinkedProjectsEnabled(ctx, targets, false)

			Expect(results).To(HaveEach(HaveField("Success", BeTrue())))
			Expect(lo.Map(results, func(r LinkedProjectResult, _ int) LinkedSystem { return r.System })).
				To(Equal([]LinkedSystem{LinkedSystemBKCI, LinkedSystemBKMonitor}))

			projects := ListLinkedProjects(ctx, ws)
			Expect(projects[0].State).To(Equal(LinkedProjectDisabled))
			Expect(projects[1].State).To(Equal(LinkedProjectDisabled))

			targets = SelectLinkedProjectsToEnable(projects)
			Expect(targets).To(HaveLen(2))
			Expect(SetLinkedProjectsEnabled(ctx, targets, true)).To(HaveEach(HaveField("Success", BeTrue())))

			projects = ListLinkedProjects(ctx, ws)
			Expect(projects[0].State).To(Equal(LinkedProjectReady))
			Expect(projects[1].State).To(Equal(LinkedProjectReady))
			Expect(SelectLinkedProjectsToEnable(projects)).To(BeEmpty())
		})

		It("calls bkmonitor with the bcs project code instead of the space ID", func() {
			mockey.PatchConvey("monitor code", GinkgoT(), func() {
				var gotCode, gotStatus string
				mockey.Mock((*bkmonitor.StubClient).SetBkmsSpaceStatus).To(
					func(_ *bkmonitor.StubClient, _ context.Context, bcsProjectCode, status string) error {
						gotCode, gotStatus = bcsProjectCode, status
						return nil
					},
				).Build()

				targets, err := SelectLinkedProjectsToDisable(
					ListLinkedProjects(ctx, ws), []LinkedSystem{LinkedSystemBKMonitor},
				)
				Expect(err).NotTo(HaveOccurred())
				SetLinkedProjectsEnabled(ctx, targets, false)

				Expect(gotCode).To(Equal(code))
				Expect(gotStatus).To(Equal(bkmonitor.SpaceStatusDisabled))
			})
		})

		It("keeps going and classifies the error when one project fails", func() {
			mockey.PatchConvey("bkci forbidden", GinkgoT(), func() {
				mockey.Mock((*bkci.StubApiClient).SetProjectEnabled).Return(&bkci.APIError{
					OpName: "v4_user_project_enable", HTTPStatus: http.StatusForbidden,
					Message: "no perm", RequestID: "req-1",
				}).Build()

				targets, err := SelectLinkedProjectsToDisable(
					ListLinkedProjects(ctx, ws), []LinkedSystem{LinkedSystemBKCI, LinkedSystemBKMonitor},
				)
				Expect(err).NotTo(HaveOccurred())
				results := SetLinkedProjectsEnabled(ctx, targets, false)

				Expect(results).To(ContainElement(HaveField("Success", BeFalse())))
				Expect(results[0]).To(Equal(LinkedProjectResult{
					System: LinkedSystemBKCI, ProjectID: code, ProjectName: "Stub Project",
					Reason: LinkedProjectFailurePermission, Detail: "no perm", RequestID: "req-1",
				}))
				Expect(results[1].Success).To(BeTrue())
			})
		})
	})
})

var _ = Describe("SelectLinkedProjectsToDisable", func() {
	projects := []LinkedProject{
		{System: LinkedSystemBKCI, ProjectID: "bkms-demo"},
		{System: LinkedSystemBKMonitor, ProjectID: "-100", BlockedReason: LinkedProjectUnsupported},
		{System: LinkedSystemBCS, ProjectID: "bkms-demo", BlockedReason: LinkedProjectUnsupported},
	}

	It("returns requested operable projects", func() {
		targets, err := SelectLinkedProjectsToDisable(projects, []LinkedSystem{LinkedSystemBKCI})
		Expect(err).NotTo(HaveOccurred())
		Expect(targets).To(Equal(projects[:1]))
	})

	It("returns no targets when no system is requested", func() {
		targets, err := SelectLinkedProjectsToDisable(projects, nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(targets).To(BeEmpty())
	})

	DescribeTable("rejects projects that can not be disabled",
		func(projects []LinkedProject, sys LinkedSystem) {
			_, err := SelectLinkedProjectsToDisable(projects, []LinkedSystem{sys})
			Expect(err).To(MatchError(ErrLinkedProjectNotOperable))
		},
		Entry("unsupported project", projects, LinkedSystemBKMonitor),
		Entry("project without manage permission", []LinkedProject{
			{System: LinkedSystemBKCI, BlockedReason: LinkedProjectNoPermission},
		}, LinkedSystemBKCI),
		Entry("system not linked", projects[:1], LinkedSystemBKMonitor),
	)
})

var _ = Describe("SelectLinkedProjectsToEnable", func() {
	It("selects operable or permission-blocked projects that are not confirmed ready", func() {
		projects := []LinkedProject{
			{System: LinkedSystemBKCI, State: LinkedProjectDisabled, BlockedReason: LinkedProjectNoPermission},
			{System: LinkedSystemBKMonitor, State: LinkedProjectUnknown},
			{System: LinkedSystemBCS, State: LinkedProjectReady, BlockedReason: LinkedProjectUnsupported},
		}
		Expect(SelectLinkedProjectsToEnable(projects)).To(Equal(projects[:2]))

		projects[1].State = LinkedProjectReady
		Expect(SelectLinkedProjectsToEnable(projects)).To(Equal(projects[:1]))
	})
})

var _ = Describe("classifyLinkedProjectError", func() {
	DescribeTable("classifies platform errors",
		func(err error, expected LinkedProjectFailureReason) {
			reason, _, _ := ClassifyLinkedProjectError(errors.Wrap(err, "set enabled"))
			Expect(reason).To(Equal(expected))
		},
		Entry("bkci forbidden", &bkci.APIError{HTTPStatus: http.StatusForbidden}, LinkedProjectFailurePermission),
		Entry("bkci project not exist status", &bkci.APIError{
			HTTPStatus: http.StatusOK, Status: bkci.ProjectNotExistStatus,
		}, LinkedProjectFailureNotFound),
		Entry("bkci service unavailable", &bkci.APIError{HTTPStatus: http.StatusServiceUnavailable},
			LinkedProjectFailureUnavailable),
		Entry("bkci unrecognized business error", &bkci.APIError{HTTPStatus: http.StatusOK, Status: 2100001},
			LinkedProjectFailureUnknown),
		Entry("bkci object not found", bkci.ObjectNotFound, LinkedProjectFailureNotFound),
		Entry("bkmonitor not found code", &bkmonitor.APIError{HTTPStatus: http.StatusOK, Code: 404},
			LinkedProjectFailureNotFound),
		Entry("bkmonitor app not authorized", &bkmonitor.APIError{HTTPStatus: http.StatusForbidden},
			LinkedProjectFailureUnknown),
		Entry("bkmonitor server error", &bkmonitor.APIError{HTTPStatus: http.StatusBadGateway},
			LinkedProjectFailureUnavailable),
		Entry("bkmonitor space not found", errors.Wrap(bkmonitor.ErrSpaceNotFound, "get space"),
			LinkedProjectFailureNotFound),
		Entry("deadline exceeded", context.DeadlineExceeded, LinkedProjectFailureUnavailable),
		Entry("network error", &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			LinkedProjectFailureUnavailable),
		Entry("other error", errors.New("boom"), LinkedProjectFailureUnknown),
	)

	It("returns the raw platform message and request ID", func() {
		_, detail, requestID := ClassifyLinkedProjectError(errors.Wrap(&bkmonitor.APIError{
			HTTPStatus: http.StatusOK, Code: 3300001, Message: "仅支持 bkci 类型且空间 ID 以 bkms- 开头的空间",
			RequestID: "req-1",
		}, "set space status"))

		Expect(detail).To(Equal("仅支持 bkci 类型且空间 ID 以 bkms- 开头的空间"))
		Expect(requestID).To(Equal("req-1"))
	})

	It("falls back to the full error message for non-API errors", func() {
		_, detail, requestID := ClassifyLinkedProjectError(errors.Wrap(errors.New("boom"), "set enabled"))

		Expect(detail).To(Equal("set enabled: boom"))
		Expect(requestID).To(BeEmpty())
	})
})
