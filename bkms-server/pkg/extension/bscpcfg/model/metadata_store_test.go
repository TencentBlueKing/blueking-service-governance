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
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
)

var _ = Describe("MetadataStore", func() {
	var store model.MetadataStore
	var ctx context.Context
	var testAppID string
	var diApp *fxtest.App

	BeforeEach(func() {
		diApp = fxtest.New(
			GinkgoT(),
			model.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()

		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)
	})

	AfterEach(func() {
		_ = store.Delete(ctx, testAppID)
		diApp.RequireStop()
	})

	Describe("Create", func() {
		Context("when creating a valid metadata", func() {
			It("should create successfully", func() {
				meta := &model.Metadata{
					AppID:          testAppID,
					BscpBizID:      "12345",
					MountPath:      "/data/bscp",
					CredentialID:   "cred-1",
					CredentialName: "bkms-credential",
					Token:          "test-token",
					FeedAddr:       "bscp-feed.example.com:9500",
					Operator:       "tester",
				}

				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				// 验证写入
				stored, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.BscpBizID).To(Equal("12345"))
				Expect(stored.MountPath).To(Equal("/data/bscp"))
				Expect(stored.CredentialID).To(Equal("cred-1"))
				Expect(stored.CredentialName).To(Equal("bkms-credential"))
				Expect(stored.Token).To(Equal("test-token"))
				Expect(stored.FeedAddr).To(Equal("bscp-feed.example.com:9500"))
				Expect(stored.CreatedAt).NotTo(BeZero())
				Expect(stored.UpdatedAt).NotTo(BeZero())
			})
		})

		Context("when creating duplicate metadata", func() {
			It("should return ErrMetadataAlreadyExists", func() {
				meta := &model.Metadata{
					AppID:     testAppID,
					BscpBizID: "12345",
					MountPath: "/data/bscp",
				}
				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				meta2 := &model.Metadata{
					AppID:     testAppID,
					BscpBizID: "12345",
					MountPath: "/data/bscp2",
				}
				err = store.Create(ctx, meta2)
				Expect(err).To(MatchError(model.ErrMetadataAlreadyExists))
			})
		})

		Context("when required fields are missing", func() {
			It("should return validation error for missing appID", func() {
				meta := &model.Metadata{
					BscpBizID: "12345",
					MountPath: "/data/bscp",
				}
				err := store.Create(ctx, meta)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("validation failed"))
			})

			It("should return validation error for missing bscpBizID", func() {
				meta := &model.Metadata{
					AppID:     testAppID,
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
				BscpBizID: "99",
				MountPath: "/etc/bscp",
				Operator:  "admin",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when meta exists", func() {
			It("should return the meta", func() {
				meta, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(meta.BscpBizID).To(Equal("99"))
				Expect(meta.MountPath).To(Equal("/etc/bscp"))
				Expect(meta.Operator).To(Equal("admin"))
			})
		})

		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				_, err := store.Get(ctx, "non-existent-app")
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})
	})

	Describe("Update", func() {
		BeforeEach(func() {
			meta := &model.Metadata{
				AppID:     testAppID,
				BscpBizID: "100",
				MountPath: "/old/path",
				Token:     "old-token",
				Operator:  "user1",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when updating token", func() {
			It("should update token successfully", func() {
				newToken := "new-token"
				err := store.Update(ctx, testAppID, &model.MetadataUpdate{
					Token: &newToken,
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Token).To(Equal("new-token"))
			})
		})

		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				newPath := "/any"
				err := store.Update(ctx, "non-existent-app", &model.MetadataUpdate{
					MountPath: &newPath,
				})
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})

		Context("when updateData is nil", func() {
			It("should return nil without error", func() {
				err := store.Update(ctx, testAppID, nil)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when updating workload", func() {
			It("should update workload successfully", func() {
				newWorkload := "my-deployment"
				err := store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadName: &newWorkload,
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadName).To(Equal("my-deployment"))
			})
		})

		Context("when workload is nil in updateData", func() {
			It("should not modify existing workload value", func() {
				// 先设置 workload
				workload := "existing-workload"
				err := store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadName: &workload,
				})
				Expect(err).NotTo(HaveOccurred())

				// 更新其他字段，workload 为 nil
				newToken := "another-token"
				err = store.Update(ctx, testAppID, &model.MetadataUpdate{
					Token: &newToken,
				})
				Expect(err).NotTo(HaveOccurred())

				// 验证 workload 未被修改
				updated, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadName).To(Equal("existing-workload"))
				Expect(updated.Token).To(Equal("another-token"))
			})
		})

		Context("when clearing workload with empty string", func() {
			It("should set workload to empty string", func() {
				// 先设置 workload
				workload := "my-deployment"
				err := store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadName: &workload,
				})
				Expect(err).NotTo(HaveOccurred())

				// 清除 workload
				emptyWorkload := ""
				err = store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadName: &emptyWorkload,
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadName).To(Equal(""))
			})
		})
	})

	Describe("Delete", func() {
		BeforeEach(func() {
			meta := &model.Metadata{
				AppID:     testAppID,
				BscpBizID: "200",
				MountPath: "/tmp",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when meta exists", func() {
			It("should delete successfully", func() {
				err := store.Delete(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get(ctx, testAppID)
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})

		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				err := store.Delete(ctx, "non-existent-app")
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})
	})

	Describe("Update WorkloadKind", func() {
		BeforeEach(func() {
			meta := &model.Metadata{
				AppID:     testAppID,
				BscpBizID: "100",
				MountPath: "/data/bscp",
				Operator:  "user1",
			}
			err := store.Create(ctx, meta)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when updating workloadKind", func() {
			It("should update workloadKind successfully", func() {
				kind := "StatefulSet"
				err := store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadKind: &kind,
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadKind).To(Equal("StatefulSet"))
			})
		})

		Context("when workloadKind is nil in updateData", func() {
			It("should not modify existing workloadKind value", func() {
				// 先设置 workloadKind
				kind := "Deployment"
				err := store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadKind: &kind,
				})
				Expect(err).NotTo(HaveOccurred())

				// 更新其他字段，workloadKind 为 nil
				newToken := "another-token"
				err = store.Update(ctx, testAppID, &model.MetadataUpdate{
					Token: &newToken,
				})
				Expect(err).NotTo(HaveOccurred())

				// 验证 workloadKind 未被修改
				updated, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadKind).To(Equal("Deployment"))
				Expect(updated.Token).To(Equal("another-token"))
			})
		})

		Context("when clearing workloadKind with empty string", func() {
			It("should set workloadKind to empty string", func() {
				// 先设置 workloadKind
				kind := "DaemonSet"
				err := store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadKind: &kind,
				})
				Expect(err).NotTo(HaveOccurred())

				// 清除 workloadKind
				empty := ""
				err = store.Update(ctx, testAppID, &model.MetadataUpdate{
					WorkloadKind: &empty,
				})
				Expect(err).NotTo(HaveOccurred())

				updated, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.WorkloadKind).To(Equal(""))
			})
		})

		Context("when creating metadata with workloadKind", func() {
			It("should persist workloadKind on create", func() {
				anotherAppID := "test-app-wk-" + stringx.Random(5)
				defer func() { _ = store.Delete(ctx, anotherAppID) }()

				meta := &model.Metadata{
					AppID:        anotherAppID,
					BscpBizID:    "100",
					MountPath:    "/data/bscp",
					WorkloadKind: "Deployment",
					WorkloadName: "my-deploy",
					Operator:     "user1",
				}
				err := store.Create(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				stored, err := store.Get(ctx, anotherAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.WorkloadKind).To(Equal("Deployment"))
				Expect(stored.WorkloadName).To(Equal("my-deploy"))
			})
		})

		Context("when reading old data without workloadKind field", func() {
			It("should default to empty string (backward compatible)", func() {
				// 已有记录不含 workloadKind 字段时，读取应默认为空
				stored, err := store.Get(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(stored.WorkloadKind).To(Equal(""))
			})
		})
	})
})

// === 纯逻辑测试（不依赖数据库） ===

var _ = Describe("Metadata Model Logic", func() {
	Describe("MetadataUpdate.ApplyTo", func() {
		It("should apply WorkloadKind when non-nil", func() {
			meta := &model.Metadata{
				AppID:        "app-1",
				WorkloadKind: "",
			}
			kind := "StatefulSet"
			update := &model.MetadataUpdate{
				WorkloadKind: &kind,
			}

			update.ApplyTo(meta)

			Expect(meta.WorkloadKind).To(Equal("StatefulSet"))
		})

		It("should not change WorkloadKind when nil", func() {
			meta := &model.Metadata{
				AppID:        "app-1",
				WorkloadKind: "Deployment",
			}
			update := &model.MetadataUpdate{
				WorkloadKind: nil,
			}

			update.ApplyTo(meta)

			Expect(meta.WorkloadKind).To(Equal("Deployment"))
		})

		It("should allow clearing WorkloadKind to empty string", func() {
			meta := &model.Metadata{
				AppID:        "app-1",
				WorkloadKind: "Deployment",
			}
			empty := ""
			update := &model.MetadataUpdate{
				WorkloadKind: &empty,
			}

			update.ApplyTo(meta)

			Expect(meta.WorkloadKind).To(Equal(""))
		})

		It("should handle nil update gracefully", func() {
			meta := &model.Metadata{
				AppID:        "app-1",
				WorkloadKind: "Deployment",
			}
			var update *model.MetadataUpdate

			update.ApplyTo(meta)

			Expect(meta.WorkloadKind).To(Equal("Deployment"))
		})

		It("should handle nil metadata gracefully", func() {
			kind := "StatefulSet"
			update := &model.MetadataUpdate{
				WorkloadKind: &kind,
			}

			// 不应 panic
			update.ApplyTo(nil)
		})
	})
})
