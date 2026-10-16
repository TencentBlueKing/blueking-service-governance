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

package dashboard

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("AppDashboard store", func() {
	var (
		ctx        context.Context
		dashboards *Store
		service    *Service
	)

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupMetaColl(Meta)).To(Succeed())

		diApp := fxtest.New(GinkgoT(), FxModule, fx.Populate(&service))
		diApp.RequireStart()
		DeferCleanup(diApp.RequireStop)
		dashboards = testutil.NewStore(GinkgoT(), Meta)
	})

	mustCreate := func(d *AppDashboard) {
		Expect(dashboards.Create(ctx, d)).To(Succeed())
	}

	Describe("Create", func() {
		It("should reject bindings missing required fields", func() {
			err := dashboards.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1"})
			Expect(err).To(MatchError(ContainSubstring("AppDashboard validation failed")))
		})

		It("should reject duplicate (appID, uid)", func() {
			mustCreate(&AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})

			err := dashboards.Create(ctx, &AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t2"})
			Expect(errors.Is(err, ErrDuplicate)).To(BeTrue())
		})

		It("should allow same uid across different apps", func() {
			mustCreate(&AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
			mustCreate(&AppDashboard{AppID: "app-b", UID: "uid-1", Title: "t2"})
		})
	})

	Describe("ByApp", func() {
		It("should list bindings by app ID", func() {
			mustCreate(&AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
			mustCreate(&AppDashboard{AppID: "app-a", UID: "uid-2", Title: "t2"})
			mustCreate(&AppDashboard{AppID: "app-b", UID: "uid-3", Title: "t3"})

			list, err := dashboards.List(ctx, ByApp("app-a"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(2))

			list, err = dashboards.List(ctx, ByApp("app-b"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].UID).To(Equal("uid-3"))
		})
	})

	Describe("Service.Update", func() {
		BeforeEach(func() {
			mustCreate(&AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
		})

		It("should update title and updater", func() {
			newTitle := "New Title"
			err := service.Update(ctx, nil, "app-a", "uid-1", "tester", &AppDashboardUpdateData{Title: &newTitle})
			Expect(err).NotTo(HaveOccurred())

			got, err := dashboards.Get(ctx, ByAppAndUID("app-a", "uid-1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Title).To(Equal("New Title"))
			Expect(got.Updater).To(Equal("tester"))
		})

		It("should do nothing when there is nothing to update", func() {
			Expect(service.Update(ctx, nil, "app-a", "uid-1", "tester", &AppDashboardUpdateData{})).To(Succeed())

			got, err := dashboards.Get(ctx, ByAppAndUID("app-a", "uid-1"))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Updater).To(BeEmpty())
		})

		It("should return not found when binding missing", func() {
			newTitle := "x"
			err := service.Update(
				ctx, nil, "app-a", "uid-missing", "tester", &AppDashboardUpdateData{Title: &newTitle},
			)
			Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
		})

		It("should reject duplicate uid", func() {
			mustCreate(&AppDashboard{AppID: "app-a", UID: "uid-2", Title: "t2"})

			err := dashboards.Update(ctx, bson.M{"uid": "uid-2"}, ByAppAndUID("app-a", "uid-1"))
			Expect(errors.Is(err, ErrDuplicate)).To(BeTrue())
		})
	})

	Describe("Service.Delete", func() {
		BeforeEach(func() {
			mustCreate(&AppDashboard{AppID: "app-a", UID: "uid-1", Title: "t1"})
		})

		It("should delete binding", func() {
			Expect(service.Delete(ctx, "app-a", "uid-1")).To(Succeed())

			list, err := dashboards.List(ctx, ByApp("app-a"))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(BeEmpty())
		})

		It("should return not found when binding missing", func() {
			err := service.Delete(ctx, "app-a", "uid-missing")
			Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
		})
	})
})
