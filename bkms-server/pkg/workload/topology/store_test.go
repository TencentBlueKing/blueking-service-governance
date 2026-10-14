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

package topology

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
)

var _ = Describe("ResourceSnapshotStore", func() {
	var (
		snapshots *ResourceSnapshotStore
		ctx       context.Context
	)

	newSnapshot := func(envName, trafficLaneName string, dataVersion int64) *ResourceSnapshot {
		return &ResourceSnapshot{
			AppID:           "test-app",
			EnvName:         envName,
			TrafficLaneName: trafficLaneName,
			ClusterID:       "BCS-K8S-00001",
			Namespace:       "default",
			DataVersion:     dataVersion,
			RefreshStatus:   RefreshStatusSuccess,
			Resources:       []ResourceEntry{},
			Relations:       []ResourceRelation{},
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		snapshots = testutil.NewStore(GinkgoT(), ResourceSnapshotMeta)
	})

	AfterEach(func() {
		Expect(testutil.CleanupMetaColl(ResourceSnapshotMeta)).To(Succeed())
	})

	Describe("SaveWithVersion", func() {
		It("should create a new resource snapshot when not exists", func() {
			snapshot := newSnapshot("dev", "", 1)
			snapshot.ReleaseName = "test-release"
			snapshot.RefreshedAt = time.Now()
			snapshot.Resources = []ResourceEntry{
				{
					Kind:       "Deployment",
					APIVersion: "apps/v1",
					Namespace:  "default",
					Name:       "nginx",
					IsManaged:  true,
					SourceType: SourceTypeHelmManifest,
				},
			}

			Expect(SaveWithVersion(ctx, snapshots, snapshot, 0)).To(Succeed())

			// 验证数据已写入
			result, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.AppID).To(Equal("test-app"))
			Expect(result.EnvName).To(Equal("dev"))
			Expect(result.ClusterID).To(Equal("BCS-K8S-00001"))
			Expect(result.DataVersion).To(Equal(int64(1)))
			Expect(result.Resources).To(HaveLen(1))
			Expect(result.Resources[0].Kind).To(Equal("Deployment"))
			Expect(result.CreatedAt).NotTo(BeZero())
			Expect(result.UpdatedAt).NotTo(BeZero())
		})

		It("should update existing resource snapshot when version matches", func() {
			// 第一次写入
			snapshot := newSnapshot("dev", "", 1)
			Expect(SaveWithVersion(ctx, snapshots, snapshot, 0)).To(Succeed())
			created, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())

			// 第二次更新（版本匹配：expectedVersion=1 == 当前 dataVersion=1），使用新对象模拟一次新的刷新
			next := newSnapshot("dev", "", 2)
			next.Resources = []ResourceEntry{
				{
					Kind:       "Service",
					Namespace:  "default",
					Name:       "nginx-svc",
					IsManaged:  true,
					SourceType: SourceTypeHelmManifest,
				},
			}
			Expect(SaveWithVersion(ctx, snapshots, next, 1)).To(Succeed())

			// 验证更新，且 _id 与 createdAt 保持不变
			result, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.ID).To(Equal(created.ID))
			Expect(result.CreatedAt).To(Equal(created.CreatedAt))
			Expect(result.DataVersion).To(Equal(int64(2)))
			Expect(result.Resources).To(HaveLen(1))
			Expect(result.Resources[0].Kind).To(Equal("Service"))
		})

		It("should return ErrVersionConflict when version has been advanced", func() {
			// 写入初始数据（dataVersion=2）
			Expect(SaveWithVersion(ctx, snapshots, newSnapshot("dev", "", 2), 0)).To(Succeed())

			// 尝试用过期版本号写入（expectedVersion=1 < 当前 dataVersion=2）
			err := SaveWithVersion(ctx, snapshots, newSnapshot("dev", "", 3), 1)
			Expect(err).To(MatchError(ErrVersionConflict))

			// 验证数据未被修改
			result, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.DataVersion).To(Equal(int64(2)))
		})

		It("should return ErrVersionConflict when writing back a stale snapshot read earlier", func() {
			Expect(SaveWithVersion(ctx, snapshots, newSnapshot("dev", "", 1), 0)).To(Succeed())
			stale, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())

			// 其他刷新任务先推进了版本
			Expect(SaveWithVersion(ctx, snapshots, newSnapshot("dev", "", 2), 1)).To(Succeed())

			// 回写读出的旧快照（带 _id）同样视为版本冲突
			Expect(SaveWithVersion(ctx, snapshots, stale, stale.DataVersion)).To(MatchError(ErrVersionConflict))
			result, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.DataVersion).To(Equal(int64(2)))
		})
	})

	Describe("FindSnapshot", func() {
		It("should return nil when resource snapshot not found", func() {
			result, err := FindSnapshot(ctx, snapshots, "non-existent", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return the resource snapshot when found", func() {
			snapshot := newSnapshot("staging", "canary", 5)
			snapshot.ClusterID = "BCS-K8S-00002"
			Expect(SaveWithVersion(ctx, snapshots, snapshot, 0)).To(Succeed())

			result, err := FindSnapshot(ctx, snapshots, "test-app", "staging", "canary")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.TrafficLaneName).To(Equal("canary"))
			Expect(result.ClusterID).To(Equal("BCS-K8S-00002"))
		})
	})

	Describe("UpdateRefreshStatus", func() {
		It("should update status and warning summary but keep resources", func() {
			snapshot := newSnapshot("dev", "", 1)
			snapshot.Resources = []ResourceEntry{{Kind: "Deployment", Name: "nginx", IsManaged: true}}
			Expect(SaveWithVersion(ctx, snapshots, snapshot, 0)).To(Succeed())

			Expect(UpdateRefreshStatus(
				ctx, snapshots, "test-app", "dev", "", RefreshStatusFailed, "boom",
			)).To(Succeed())

			result, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RefreshStatus).To(Equal(RefreshStatusFailed))
			Expect(result.WarningSummary).To(Equal("boom"))
			Expect(result.Resources).To(HaveLen(1))
		})

		It("should do nothing when the snapshot does not exist", func() {
			Expect(UpdateRefreshStatus(
				ctx, snapshots, "non-existent", "dev", "", RefreshStatusProgressing, "",
			)).To(Succeed())

			result, err := FindSnapshot(ctx, snapshots, "non-existent", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("DeleteIfExists by scope", func() {
		It("should delete existing resource snapshot", func() {
			snapshot := newSnapshot("dev", "", 1)
			snapshot.RefreshStatus = RefreshStatusProgressing
			Expect(SaveWithVersion(ctx, snapshots, snapshot, 0)).To(Succeed())

			// 删除
			Expect(snapshots.DeleteIfExists(ctx, ByScope("test-app", "dev", ""))).To(Succeed())

			// 验证已删除
			result, err := FindSnapshot(ctx, snapshots, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
