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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("ResourceSnapshotStoreMongo", func() {
	var (
		store *ResourceSnapshotStoreMongo
		ctx   context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		store, err = NewResourceSnapshotStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(store).NotTo(BeNil())

		// 每次测试前清空 Collection
		err = store.DeleteAll(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		_ = store.DeleteAll(ctx)
	})

	Describe("UpsertWithVersion", func() {
		It("should create a new resource snapshot when not exists", func() {
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				ReleaseName:     "test-release",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusSuccess,
				RefreshedAt:     time.Now(),
				Resources: []ResourceEntry{
					{
						Kind:       "Deployment",
						APIVersion: "apps/v1",
						Namespace:  "default",
						Name:       "nginx",
						IsManaged:  true,
						SourceType: SourceTypeHelmManifest,
					},
				},
				Relations: []ResourceRelation{},
			}

			err := store.UpsertWithVersion(ctx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())

			// 验证数据已写入
			result, err := store.Get(ctx, "test-app", "dev", "")
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
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusSuccess,
				Resources:       []ResourceEntry{},
				Relations:       []ResourceRelation{},
			}
			err := store.UpsertWithVersion(ctx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())

			// 第二次更新（版本匹配：expectedVersion=1 == 当前 dataVersion=1）
			snapshot.DataVersion = 2
			snapshot.Resources = []ResourceEntry{
				{
					Kind:       "Service",
					Namespace:  "default",
					Name:       "nginx-svc",
					IsManaged:  true,
					SourceType: SourceTypeHelmManifest,
				},
			}
			err = store.UpsertWithVersion(ctx, snapshot, 1)
			Expect(err).NotTo(HaveOccurred())

			// 验证更新
			result, err := store.Get(ctx, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.DataVersion).To(Equal(int64(2)))
			Expect(result.Resources).To(HaveLen(1))
			Expect(result.Resources[0].Kind).To(Equal("Service"))
		})

		It("should return ErrVersionConflict when version has been advanced", func() {
			// 写入初始数据（dataVersion=2）
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     2,
				RefreshStatus:   RefreshStatusSuccess,
				Resources:       []ResourceEntry{},
				Relations:       []ResourceRelation{},
			}
			err := store.UpsertWithVersion(ctx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())

			// 尝试用过期版本号写入（expectedVersion=1 < 当前 dataVersion=2）
			snapshot.DataVersion = 3
			err = store.UpsertWithVersion(ctx, snapshot, 1)
			Expect(err).To(MatchError(ErrVersionConflict))

			// 验证数据未被修改
			result, err := store.Get(ctx, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.DataVersion).To(Equal(int64(2)))
		})
	})

	Describe("Get", func() {
		It("should return nil when resource snapshot not found", func() {
			result, err := store.Get(ctx, "non-existent", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return the resource snapshot when found", func() {
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "staging",
				TrafficLaneName: "canary",
				ClusterID:       "BCS-K8S-00002",
				Namespace:       "staging-ns",
				DataVersion:     5,
				RefreshStatus:   RefreshStatusSuccess,
				Resources:       []ResourceEntry{},
				Relations:       []ResourceRelation{},
			}
			err := store.UpsertWithVersion(ctx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())

			result, err := store.Get(ctx, "test-app", "staging", "canary")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.TrafficLaneName).To(Equal("canary"))
			Expect(result.ClusterID).To(Equal("BCS-K8S-00002"))
		})
	})

	Describe("Delete", func() {
		It("should delete existing resource snapshot", func() {
			snapshot := &ResourceSnapshot{
				AppID:           "test-app",
				EnvName:         "dev",
				TrafficLaneName: "",
				ClusterID:       "BCS-K8S-00001",
				Namespace:       "default",
				DataVersion:     1,
				RefreshStatus:   RefreshStatusProgressing,
				Resources:       []ResourceEntry{},
				Relations:       []ResourceRelation{},
			}
			err := store.UpsertWithVersion(ctx, snapshot, 0)
			Expect(err).NotTo(HaveOccurred())

			// 删除
			err = store.Delete(ctx, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())

			// 验证已删除
			result, err := store.Get(ctx, "test-app", "dev", "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should not error when deleting non-existent snapshot", func() {
			err := store.Delete(ctx, "non-existent", "dev", "")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
