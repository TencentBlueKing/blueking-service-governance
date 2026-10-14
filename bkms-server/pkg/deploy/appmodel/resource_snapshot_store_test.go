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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
)

var _ = Describe("ResourceSnapshotStore", func() {
	var ctx context.Context
	var snapshots *ResourceSnapshotStore

	createSnapshots := func(items ...ResourceSnapshot) {
		Expect(snapshots.CreateMany(ctx, lo.ToSlicePtr(items))).To(Succeed())
	}

	BeforeEach(func() {
		ctx = context.Background()
		snapshots = testutil.NewStore(GinkgoT(), ResourceSnapshotMeta)
	})

	Describe("ListSnapshotMeta", func() {
		It("should list meta without manifest and return empty when none", func() {
			page, err := ListSnapshotMeta(ctx, snapshots, "no-meta-app", bson.NewObjectID().Hex(), 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(BeZero())
			Expect(page.Items).To(BeEmpty())

			oid := bson.NewObjectID()
			appID := "snap-meta-" + stringx.Random(6)
			createSnapshots(ResourceSnapshot{
				AppID: appID, DeployRecordID: oid,
				Kind: "GameDeployment", Name: "g", Manifest: "secret-yaml-body", IsTruncated: true,
			})

			page, err = ListSnapshotMeta(ctx, snapshots, appID, oid.Hex(), 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(Equal(int64(1)))
			Expect(page.Items).To(HaveLen(1))
			Expect(page.Items[0].Manifest).To(BeEmpty())
			Expect(page.Items[0].Kind).To(Equal("GameDeployment"))
			Expect(page.Items[0].IsTruncated).To(BeTrue())
		})

		It("should treat an invalid deployID as no records", func() {
			page, err := ListSnapshotMeta(ctx, snapshots, "snap-app", "not-a-hex", 1, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(page.Total).To(BeZero())
			Expect(page.Items).To(BeEmpty())
		})

		It("should paginate ordered by kind and name", func() {
			oid := bson.NewObjectID()
			appID := "snap-meta-page-" + stringx.Random(6)
			snap := func(deployOID bson.ObjectID, kind, name string) ResourceSnapshot {
				return ResourceSnapshot{AppID: appID, DeployRecordID: deployOID, Kind: kind, Name: name, Manifest: "m"}
			}
			createSnapshots(
				snap(oid, "GameDeployment", "gd"),
				snap(oid, "ConfigMap", "cm-b"),
				snap(oid, "ConfigMap", "cm-a"),
				// 其它部署记录的快照不应出现在结果中
				snap(bson.NewObjectID(), "ConfigMap", "other"),
			)

			page1, err := ListSnapshotMeta(ctx, snapshots, appID, oid.Hex(), 1, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(page1.Total).To(Equal(int64(3)))
			Expect(page1.HasNext).To(BeTrue())
			Expect(page1.Items).To(HaveLen(2))
			// Sort is kind asc, name asc: "ConfigMap" < "GameDeployment" lexicographically.
			Expect(page1.Items[0].Kind).To(Equal("ConfigMap"))
			Expect(page1.Items[0].Name).To(Equal("cm-a"))
			Expect(page1.Items[1].Kind).To(Equal("ConfigMap"))
			Expect(page1.Items[1].Name).To(Equal("cm-b"))

			page2, err := ListSnapshotMeta(ctx, snapshots, appID, oid.Hex(), 2, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(page2.Total).To(Equal(int64(3)))
			Expect(page2.HasNext).To(BeFalse())
			Expect(page2.Items).To(HaveLen(1))
			Expect(page2.Items[0].Kind).To(Equal("GameDeployment"))
			Expect(page2.Items[0].Name).To(Equal("gd"))
		})
	})

	Describe("GetSnapshot", func() {
		It("should get the full snapshot only when app, deploy record and id all match", func() {
			oid := bson.NewObjectID()
			appID := "snap-get-" + stringx.Random(6)
			snap := ResourceSnapshot{
				AppID: appID, DeployRecordID: oid, APIVersion: "tkex/v1",
				Kind: "GameDeployment", Name: "g", Manifest: "secret-yaml-body",
			}
			Expect(snapshots.Create(ctx, &snap)).To(Succeed())

			full, err := GetSnapshot(ctx, snapshots, appID, oid.Hex(), snap.ID.Hex())
			Expect(err).NotTo(HaveOccurred())
			Expect(full.Manifest).To(Equal("secret-yaml-body"))
			Expect(full.APIVersion).To(Equal("tkex/v1"))

			_, err = GetSnapshot(ctx, snapshots, "other-app", oid.Hex(), snap.ID.Hex())
			Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))

			// 即便 appID + snapshotID 都正确，但 deployRecordID 不匹配，也应返回 NotFound（防越权）
			_, err = GetSnapshot(ctx, snapshots, appID, bson.NewObjectID().Hex(), snap.ID.Hex())
			Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))

			// 非法 hex 等价于「找不到」
			_, err = GetSnapshot(ctx, snapshots, appID, "not-a-hex", snap.ID.Hex())
			Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))
			_, err = GetSnapshot(ctx, snapshots, appID, oid.Hex(), "not-a-hex")
			Expect(err).To(MatchError(ErrResourceSnapshotRowNotFound))
		})
	})
})
