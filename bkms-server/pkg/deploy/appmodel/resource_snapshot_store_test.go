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

package appmodel_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
)

var _ = Describe("ResourceSnapshotStoreMongo", func() {
	var diApp *fxtest.App

	var ctx context.Context
	var snapStore ResourceSnapshotStore

	BeforeEach(func() {
		ctx = context.Background()

		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(
				&snapStore,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	It("should create resources and list by deploy record", func() {
		oid := bson.NewObjectID()
		recordHex := oid.Hex()
		appID := "snap-app-" + stringx.Random(6)
		oidParsed, perr := bson.ObjectIDFromHex(recordHex)
		Expect(perr).NotTo(HaveOccurred())
		err := snapStore.CreateResources(ctx, []ResourceSnapshot{
			{
				AppID: appID, DeployRecordID: oidParsed,
				APIVersion: "tkex/v1", Kind: "GameDeployment", Name: "gd1",
				Manifest: "dep-yaml", IsTruncated: true,
			},
			{
				AppID: appID, DeployRecordID: oidParsed,
				APIVersion: "v1", Kind: "ConfigMap", Name: "cm-b",
				Manifest: "cm-b-yaml", IsTruncated: false,
			},
			{
				AppID: appID, DeployRecordID: oidParsed,
				APIVersion: "v1", Kind: "ConfigMap", Name: "cm-a",
				Manifest: "cm-a-yaml", IsTruncated: true,
			},
		})
		Expect(err).NotTo(HaveOccurred())

		got, err := snapStore.ListByDeployRecord(ctx, appID, recordHex)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(3))
		// Sort is kind asc, name asc: "ConfigMap" < "GameDeployment" lexicographically.
		Expect(got[0].Kind).To(Equal("ConfigMap"))
		Expect(got[0].Name).To(Equal("cm-a"))
		Expect(got[0].Manifest).To(Equal("cm-a-yaml"))
		Expect(got[0].IsTruncated).To(BeTrue())
		Expect(got[0].AppID).To(Equal(appID))
		Expect(got[1].Name).To(Equal("cm-b"))
		Expect(got[1].Manifest).To(Equal("cm-b-yaml"))
		Expect(got[2].Kind).To(Equal("GameDeployment"))
		Expect(got[2].Name).To(Equal("gd1"))
		Expect(got[2].Manifest).To(Equal("dep-yaml"))
		Expect(got[2].IsTruncated).To(BeTrue())
	})

	It("should delete then create to refresh rows for same deploy", func() {
		oid := bson.NewObjectID()
		recordHex := oid.Hex()
		appID := "snap-app2-" + stringx.Random(6)
		Expect(snapStore.CreateResources(ctx, []ResourceSnapshot{
			{AppID: appID, DeployRecordID: oid, Kind: "GameDeployment", Name: "g", Manifest: "v1"},
		})).To(Succeed())
		got, err := snapStore.ListByDeployRecord(ctx, appID, recordHex)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))

		Expect(snapStore.DeleteByDeployRecord(ctx, appID, oid.Hex())).To(Succeed())
		Expect(snapStore.CreateResources(ctx, []ResourceSnapshot{
			{AppID: appID, DeployRecordID: oid, Kind: "GameDeployment", Name: "g", Manifest: "v2"},
			{AppID: appID, DeployRecordID: oid, Kind: "Secret", Name: "s", Manifest: "s1"},
		})).To(Succeed())
		got, err = snapStore.ListByDeployRecord(ctx, appID, recordHex)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(2))
		Expect(got[0].Manifest).To(Equal("v2"))
	})

	It("should return ErrResourceSnapshotNotFound when missing", func() {
		_, err := snapStore.ListByDeployRecord(ctx, "no-such-app", bson.NewObjectID().Hex())
		Expect(err).To(MatchError(ErrResourceSnapshotNotFound))
	})

	It("should list meta without manifest and return empty when none", func() {
		meta, total, err := snapStore.ListMetaByDeployRecord(ctx, "no-meta-app", bson.NewObjectID().Hex(), 1, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(total).To(BeZero())
		Expect(meta).To(BeEmpty())

		oid := bson.NewObjectID()
		appID := "snap-meta-" + stringx.Random(6)
		Expect(snapStore.CreateResources(ctx, []ResourceSnapshot{
			{
				AppID: appID, DeployRecordID: oid,
				Kind: "GameDeployment", Name: "g", Manifest: "secret-yaml-body",
			},
		})).To(Succeed())

		meta, total, err = snapStore.ListMetaByDeployRecord(ctx, appID, oid.Hex(), 1, 10)
		Expect(err).NotTo(HaveOccurred())
		Expect(total).To(Equal(int64(1)))
		Expect(meta).To(HaveLen(1))
		Expect(meta[0].Manifest).To(BeEmpty())
		Expect(meta[0].Kind).To(Equal("GameDeployment"))

		full, err := snapStore.GetByID(ctx, appID, oid.Hex(), meta[0].ID.Hex())
		Expect(err).NotTo(HaveOccurred())
		Expect(full.Manifest).To(Equal("secret-yaml-body"))

		_, err = snapStore.GetByID(ctx, "other-app", oid.Hex(), meta[0].ID.Hex())
		Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))

		// 即便 appID + snapshotID 都正确，但 deployRecordID 不匹配，也应返回 NotFound（防越权）
		_, err = snapStore.GetByID(ctx, appID, bson.NewObjectID().Hex(), meta[0].ID.Hex())
		Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))

		// 非法 hex 等价于「找不到」
		_, err = snapStore.GetByID(ctx, appID, "not-a-hex", meta[0].ID.Hex())
		Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))
		_, err = snapStore.GetByID(ctx, appID, oid.Hex(), "not-a-hex")
		Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))
	})

	It("should paginate list meta by deploy record", func() {
		oid := bson.NewObjectID()
		appID := "snap-meta-page-" + stringx.Random(6)
		Expect(snapStore.CreateResources(ctx, []ResourceSnapshot{
			{AppID: appID, DeployRecordID: oid, Kind: "GameDeployment", Name: "gd", Manifest: "m1"},
			{AppID: appID, DeployRecordID: oid, Kind: "ConfigMap", Name: "cm-b", Manifest: "m2"},
			{AppID: appID, DeployRecordID: oid, Kind: "ConfigMap", Name: "cm-a", Manifest: "m3"},
		})).To(Succeed())

		page1, total, err := snapStore.ListMetaByDeployRecord(ctx, appID, oid.Hex(), 1, 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(total).To(Equal(int64(3)))
		Expect(page1).To(HaveLen(2))
		Expect(page1[0].Kind).To(Equal("ConfigMap"))
		Expect(page1[0].Name).To(Equal("cm-a"))
		Expect(page1[1].Kind).To(Equal("ConfigMap"))
		Expect(page1[1].Name).To(Equal("cm-b"))

		page2, total2, err := snapStore.ListMetaByDeployRecord(ctx, appID, oid.Hex(), 2, 2)
		Expect(err).NotTo(HaveOccurred())
		Expect(total2).To(Equal(int64(3)))
		Expect(page2).To(HaveLen(1))
		Expect(page2[0].Kind).To(Equal("GameDeployment"))
		Expect(page2[0].Name).To(Equal("gd"))
	})
})
