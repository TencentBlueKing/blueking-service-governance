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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	devmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("PublishRecord store", func() {
	const envName = "staging"
	var (
		ctx     context.Context
		db      *mongo.Database
		records *devmode.PublishRecordStore
		appID   string
	)

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		records = store.Of(db, devmode.PublishRecordMeta)
		appID = "test-app-" + stringx.Random(6)
	})

	newRecord := func(instance, binary string, status devmode.PublishStatus) *devmode.PublishRecord {
		return &devmode.PublishRecord{
			AppID: appID, EnvName: envName, Instance: instance,
			BinaryName: binary, MD5: "m-" + binary, Status: status, Operator: "admin",
		}
	}

	mustCreate := func(objs ...*devmode.PublishRecord) {
		Expect(records.CreateMany(ctx, objs)).To(Succeed())
	}

	Describe("listing by app and env", func() {
		BeforeEach(func() {
			mustCreate(
				newRecord("pod-1", "alpha", devmode.PublishStatusSuccess),
				newRecord("pod-2", "beta", devmode.PublishStatusSuccess),
			)
			// 其它环境的记录不应被列出
			other := newRecord("pod-1", "alpha", devmode.PublishStatusSuccess)
			other.EnvName = "prod"
			mustCreate(other)
		})

		list := func(keyword string) *store.Page[devmode.PublishRecord] {
			page, err := records.ListPage(ctx, 1, 10,
				devmode.ByAppEnv(appID, envName), devmode.KeywordMatches(keyword), devmode.NewestFirst())
			Expect(err).NotTo(HaveOccurred())
			return page
		}

		It("should list the records of the env in a stable newest-first order", func() {
			page := list("")
			Expect(page.Total).To(Equal(int64(2)))
			// 同一次上报的记录创建时间相同，按 _id 倒序
			Expect(lo.Map(page.Items, func(r devmode.PublishRecord, _ int) string { return r.Instance })).
				To(Equal([]string{"pod-2", "pod-1"}))
		})

		It("should filter by binary name, operator or instance ignoring case", func() {
			Expect(list("BETA").Items).To(HaveLen(1))
			Expect(list("pod-1").Items[0].BinaryName).To(Equal("alpha"))
			Expect(list("admin").Total).To(Equal(int64(2)))
		})
	})

	Describe("ListLatestByInstance", func() {
		It("should return the latest record for each instance", func() {
			mustCreate(newRecord("pod-1", "v1", devmode.PublishStatusSuccess))
			// 间隔一段时间，确保第二次 push 的 createdAt 更晚
			time.Sleep(10 * time.Millisecond)
			failed := newRecord("pod-1", "v2", devmode.PublishStatusFailed)
			failed.Message = "boom"
			mustCreate(failed, newRecord("pod-2", "v1", devmode.PublishStatusSuccess))

			latest, err := devmode.ListLatestByInstance(ctx, db, appID, envName, []string{"pod-1", "pod-2"})
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(2))
			Expect(latest["pod-1"].Status).To(Equal(devmode.PublishStatusFailed))
			Expect(latest["pod-1"].BinaryName).To(Equal("v2"))
			Expect(latest["pod-2"].Status).To(Equal(devmode.PublishStatusSuccess))
		})

		It("should omit instances without records", func() {
			mustCreate(newRecord("pod-1", "v1", devmode.PublishStatusSuccess))

			latest, err := devmode.ListLatestByInstance(ctx, db, appID, envName, []string{"pod-1", "pod-2"})
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(HaveLen(1))
			Expect(latest).To(HaveKey("pod-1"))
		})

		It("should return an empty map for no instances", func() {
			latest, err := devmode.ListLatestByInstance(ctx, db, appID, envName, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(latest).To(BeEmpty())
		})
	})
})
