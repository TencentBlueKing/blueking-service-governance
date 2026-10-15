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

package model_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	genericstore "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("MetadataStore", func() {
	var store *model.MetadataStore
	var ctx context.Context
	var testAppID string

	BeforeEach(func() {
		store = testutil.NewStore(GinkgoT(), model.MetadataMeta)
		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		_ = store.DeleteIfExists(ctx, model.ByApp(testAppID))
	})

	// update 按字段局部更新指定 app 的 Metadata
	update := func(appID string, fields map[string]any) error {
		return store.Update(ctx, fields, model.ByApp(appID))
	}

	Describe("Create", func() {
		Context("when creating a valid metadata", func() {
			It("should create successfully", func() {
				meta := &model.Metadata{
					AppID:        testAppID,
					Enable:       true,
					MountPath:    "/data/bscp",
					WorkloadName: "test-workload",
					Operator:     "tester",
				}

				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				// 验证写入
				stored, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.Enable).To(BeTrue())
				Expect(stored.MountPath).To(Equal("/data/bscp"))
				Expect(stored.WorkloadName).To(Equal("test-workload"))
				Expect(stored.Operator).To(Equal("tester"))
				Expect(stored.CreatedAt).NotTo(BeZero())
				Expect(stored.UpdatedAt).NotTo(BeZero())
			})
		})

		Context("when creating duplicate metadata", func() {
			It("should return ErrMetadataAlreadyExists", func() {
				meta := &model.Metadata{
					AppID:     testAppID,
					MountPath: "/data/bscp",
				}
				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				meta2 := &model.Metadata{
					AppID:     testAppID,
					MountPath: "/data/bscp2",
				}
				err = store.Create(ctx, meta2)
				Expect(err).To(MatchError(model.ErrMetadataAlreadyExists))
			})
		})

		Context("when required fields are missing", func() {
			It("should return validation error for missing appID", func() {
				meta := &model.Metadata{
					MountPath: "/data/bscp",
				}
				err := store.Create(ctx, meta)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})
		})
	})

	Describe("Get", func() {
		BeforeEach(func() {
			meta := &model.Metadata{
				AppID:     testAppID,
				Enable:    true,
				MountPath: "/etc/bscp",
				Operator:  "admin",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when meta exists", func() {
			It("should return the meta", func() {
				meta, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(meta.Enable).To(BeTrue())
				Expect(meta.MountPath).To(Equal("/etc/bscp"))
				Expect(meta.Operator).To(Equal("admin"))
			})
		})

		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				_, err := store.Get(ctx, model.ByApp("non-existent-app"))
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			meta := &model.Metadata{
				AppID:     testAppID,
				Enable:    true,
				MountPath: "/old/path",
				Operator:  "user1",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when updating enable", func() {
			It("should update enable successfully", func() {
				newEnable := false
				err := update(testAppID, map[string]any{"enable": newEnable})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Enable).To(BeFalse())
			})
		})

		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				newPath := "/any"
				err := update("non-existent-app", map[string]any{"mountPath": newPath})
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})

		Context("when no field is given", func() {
			It("should return ErrNoFieldsToUpdate", func() {
				err := update(testAppID, nil)
				Expect(err).To(MatchError(genericstore.ErrNoFieldsToUpdate))
			})
		})

		Context("when updating workload", func() {
			It("should update workload successfully", func() {
				newWorkload := "my-deployment"
				err := update(testAppID, map[string]any{"workloadName": newWorkload})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadName).To(Equal("my-deployment"))
			})
		})

		Context("when workload is nil in updateData", func() {
			It("should not modify existing workload value", func() {
				// 先设置 workload
				workload := "existing-workload"
				err := update(testAppID, map[string]any{"workloadName": workload})
				Expect(err).NotTo(HaveOccurred())

				// 更新其他字段，workload 为 nil
				newEnable := false
				err = update(testAppID, map[string]any{"enable": newEnable})
				Expect(err).NotTo(HaveOccurred())

				// 验证 workload 未被修改
				updated, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadName).To(Equal("existing-workload"))
				Expect(updated.Enable).To(BeFalse())
			})
		})

		Context("when clearing workload with empty string", func() {
			It("should set workload to empty string", func() {
				// 先设置 workload
				workload := "my-deployment"
				err := update(testAppID, map[string]any{"workloadName": workload})
				Expect(err).NotTo(HaveOccurred())

				// 清除 workload
				emptyWorkload := ""
				err = update(testAppID, map[string]any{"workloadName": emptyWorkload})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadName).To(Equal(""))
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			meta := &model.Metadata{
				AppID:     testAppID,
				MountPath: "/tmp",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when meta exists", func() {
			It("should delete successfully", func() {
				err := store.Delete(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get(ctx, model.ByApp(testAppID))
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})

		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				err := store.Delete(ctx, model.ByApp("non-existent-app"))
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})
	})

	Describe("Update WorkloadKind", func() {
		BeforeEach(func() {
			meta := &model.Metadata{
				AppID:     testAppID,
				MountPath: "/data/bscp",
				Operator:  "user1",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when updating workloadKind", func() {
			It("should update workloadKind successfully", func() {
				kind := "StatefulSet"
				err := update(testAppID, map[string]any{"workloadKind": kind})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadKind).To(Equal("StatefulSet"))
			})
		})

		Context("when workloadKind is nil in updateData", func() {
			It("should not modify existing workloadKind value", func() {
				// 先设置 workloadKind
				kind := "Deployment"
				err := update(testAppID, map[string]any{"workloadKind": kind})
				Expect(err).NotTo(HaveOccurred())

				// 更新其他字段，workloadKind 为 nil
				newEnable := false
				err = update(testAppID, map[string]any{"enable": newEnable})
				Expect(err).NotTo(HaveOccurred())

				// 验证 workloadKind 未被修改
				updated, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadKind).To(Equal("Deployment"))
				Expect(updated.Enable).To(BeFalse())
			})
		})

		Context("when clearing workloadKind with empty string", func() {
			It("should set workloadKind to empty string", func() {
				// 先设置 workloadKind
				kind := "DaemonSet"
				err := update(testAppID, map[string]any{"workloadKind": kind})
				Expect(err).NotTo(HaveOccurred())

				// 清除 workloadKind
				empty := ""
				err = update(testAppID, map[string]any{"workloadKind": empty})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadKind).To(Equal(""))
			})
		})

		Context("when creating metadata with workloadKind", func() {
			It("should persist workloadKind on create", func() {
				anotherAppID := "test-app-wk-" + stringx.Random(5)
				defer func() { _ = store.DeleteIfExists(ctx, model.ByApp(anotherAppID)) }()

				meta := &model.Metadata{
					AppID:        anotherAppID,
					MountPath:    "/data/bscp",
					WorkloadKind: "Deployment",
					WorkloadName: "my-deploy",
					Operator:     "user1",
				}
				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				stored, err := store.Get(ctx, model.ByApp(anotherAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.WorkloadKind).To(Equal("Deployment"))
				Expect(stored.WorkloadName).To(Equal("my-deploy"))
			})
		})

		Context("when reading old data without workloadKind field", func() {
			It("should default to empty string (backward compatible)", func() {
				// 已有记录不含 workloadKind 字段时，读取应默认为空
				stored, err := store.Get(ctx, model.ByApp(testAppID))
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.WorkloadKind).To(Equal(""))
			})
		})
	})
})
