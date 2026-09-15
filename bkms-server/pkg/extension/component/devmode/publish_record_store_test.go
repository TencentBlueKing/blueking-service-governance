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

package devmode_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	devmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
)

var _ = Describe("PublishRecordStoreMongo", func() {
	var store devmode.PublishRecordStore
	var ctx context.Context
	var diApp *fxtest.App

	BeforeEach(func() {
		diApp = fxtest.New(
			GinkgoT(),
			devmode.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()
		ctx = context.Background()
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	Describe("Create", func() {
		It("creates one record per instance", func() {
			appID := "test-app-" + stringx.Random(6)
			envName := "staging"

			ids, err := store.Create(ctx, []devmode.PublishRecord{
				{
					AppID: appID, EnvName: envName, Instance: "pod-1",
					BinaryName: "server", MD5: "m1",
					Status: devmode.PublishStatusSuccess, Operator: "admin",
				},
				{
					AppID: appID, EnvName: envName, Instance: "pod-2",
					BinaryName: "server", MD5: "m1",
					Status: devmode.PublishStatusFailed, Message: "boom", Operator: "admin",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(ids).To(HaveLen(2))
			Expect(ids[0]).NotTo(BeEmpty())
			Expect(ids[1]).NotTo(BeEmpty())
		})
	})

	Describe("List", func() {
		It("lists records with pagination", func() {
			appID := "test-app-" + stringx.Random(6)
			envName := "staging"

			_, err := store.Create(ctx, []devmode.PublishRecord{
				{
					AppID: appID, EnvName: envName, Instance: "pod-1",
					BinaryName: "alpha", MD5: "m1",
					Status: devmode.PublishStatusSuccess, Operator: "u1",
				},
				{
					AppID: appID, EnvName: envName, Instance: "pod-2",
					BinaryName: "beta", MD5: "m2",
					Status: devmode.PublishStatusSuccess, Operator: "u2",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			records, total, err := store.List(ctx, appID, envName, "", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(records).To(HaveLen(2))
		})

		It("filters records by keyword", func() {
			appID := "test-app-" + stringx.Random(6)
			envName := "staging"

			_, err := store.Create(ctx, []devmode.PublishRecord{
				{
					AppID: appID, EnvName: envName, Instance: "pod-1",
					BinaryName: "alpha", MD5: "m1",
					Status: devmode.PublishStatusSuccess, Operator: "u1",
				},
				{
					AppID: appID, EnvName: envName, Instance: "pod-2",
					BinaryName: "beta", MD5: "m2",
					Status: devmode.PublishStatusSuccess, Operator: "u2",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			records, total, err := store.List(ctx, appID, envName, "beta", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(records).To(HaveLen(1))
			Expect(records[0].BinaryName).To(Equal("beta"))
		})
	})

	Describe("ListLatestByInstance", func() {
		It("returns the latest record for each instance", func() {
			appID := "test-app-" + stringx.Random(6)
			envName := "staging"

			// pod-1 第一次 push 成功
			_, err := store.Create(ctx, []devmode.PublishRecord{
				{
					AppID: appID, EnvName: envName, Instance: "pod-1",
					BinaryName: "v1", MD5: "m1",
					Status: devmode.PublishStatusSuccess, Operator: "admin",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// 间隔一段时间，确保第二次 push 的 createdAt 更晚
			time.Sleep(10 * time.Millisecond)

			// pod-1 第二次 push 失败；pod-2 首次 push 成功
			_, err = store.Create(ctx, []devmode.PublishRecord{
				{
					AppID: appID, EnvName: envName, Instance: "pod-1",
					BinaryName: "v2", MD5: "m2",
					Status: devmode.PublishStatusFailed, Message: "boom", Operator: "admin",
				},
				{
					AppID: appID, EnvName: envName, Instance: "pod-2",
					BinaryName: "v1", MD5: "m1",
					Status: devmode.PublishStatusSuccess, Operator: "admin",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			latest, err := store.ListLatestByInstance(ctx, appID, envName, []string{"pod-1", "pod-2"})
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(2))
			Expect(latest["pod-1"].Status).To(Equal(devmode.PublishStatusFailed))
			Expect(latest["pod-1"].BinaryName).To(Equal("v2"))
			Expect(latest["pod-2"].Status).To(Equal(devmode.PublishStatusSuccess))
		})

		It("omits instances without records", func() {
			appID := "test-app-" + stringx.Random(6)
			envName := "staging"

			_, err := store.Create(ctx, []devmode.PublishRecord{
				{
					AppID: appID, EnvName: envName, Instance: "pod-1",
					BinaryName: "v1", MD5: "m1",
					Status: devmode.PublishStatusSuccess, Operator: "admin",
				},
			})
			Expect(err).NotTo(HaveOccurred())

			latest, err := store.ListLatestByInstance(ctx, appID, envName, []string{"pod-1", "pod-2"})
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest).To(HaveKey("pod-1"))
		})
	})
})
