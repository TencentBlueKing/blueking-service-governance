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

var _ = Describe("Store", func() {
	var store model.Store
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
		_ = store.DeleteMetadata(ctx, testAppID)
		_ = store.DeleteEnvBindingsByApp(ctx, testAppID)
		diApp.RequireStop()
	})

	// === Metadata Delete 边界测试 ===
	Describe("DeleteMetadata", func() {
		Context("when meta exists", func() {
			BeforeEach(func() {
				meta := &model.Metadata{
					AppID:     testAppID,
					BscpBizID: "12345",
					MountPath: "/data/bscp",
					Operator:  "tester",
				}
				err := store.CreateMetadata(ctx, meta)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should delete successfully", func() {
				err := store.DeleteMetadata(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())

				// 验证已删除
				_, err = store.GetMetadata(ctx, testAppID)
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})

		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				err := store.DeleteMetadata(ctx, "non-existent-app")
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})
	})

	// === Metadata Get 边界测试 ===
	Describe("GetMetadata", func() {
		Context("when meta does not exist", func() {
			It("should return ErrMetadataNotFound", func() {
				_, err := store.GetMetadata(ctx, "non-existent-app")
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})
	})

	// === Env 级 Delete 边界测试 ===
	Describe("DeleteEnvBinding", func() {
		Context("when binding exists", func() {
			BeforeEach(func() {
				binding := &model.EnvBinding{
					AppID:    testAppID,
					EnvName:  "dev",
					Services: []model.ServiceRef{{ID: "svc-1", Name: "file-svc"}},
				}
				err := store.CreateEnvBinding(ctx, binding)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should delete successfully", func() {
				err := store.DeleteEnvBinding(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())

				// 验证已删除
				_, err = store.GetEnvBinding(ctx, testAppID, "dev")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})

		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				err := store.DeleteEnvBinding(ctx, testAppID, "non-existent-env")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	// === Env 级 Get 边界测试 ===
	Describe("GetEnvBinding", func() {
		Context("when binding does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				_, err := store.GetEnvBinding(ctx, testAppID, "non-existent-env")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	// === GetSnapshot 边界测试 ===
	Describe("GetSnapshot", func() {
		Context("when metadata does not exist", func() {
			It("should return nil, nil", func() {
				detail, err := store.GetSnapshot(ctx, "non-existent-app", "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(detail).To(BeNil())
			})
		})

		Context("when metadata exists but env binding does not", func() {
			BeforeEach(func() {
				meta := &model.Metadata{
					AppID:     testAppID,
					BscpBizID: "12345",
					MountPath: "/data/bscp",
				}
				err := store.CreateMetadata(ctx, meta)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return nil, nil", func() {
				detail, err := store.GetSnapshot(ctx, testAppID, "non-existent-env")
				Expect(err).NotTo(HaveOccurred())
				Expect(detail).To(BeNil())
			})
		})

		Context("when both metadata and env binding exist", func() {
			BeforeEach(func() {
				meta := &model.Metadata{
					AppID:     testAppID,
					BscpBizID: "12345",
					MountPath: "/data/bscp",
					Token:     "test-token",
					FeedAddr:  "bscp-feed.example.com:9500",
				}
				err := store.CreateMetadata(ctx, meta)
				Expect(err).NotTo(HaveOccurred())

				binding := &model.EnvBinding{
					AppID:    testAppID,
					EnvName:  "dev",
					Services: []model.ServiceRef{{ID: "svc-1", Name: "file-svc"}},
				}
				err = store.CreateEnvBinding(ctx, binding)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return aggregated detail", func() {
				detail, err := store.GetSnapshot(ctx, testAppID, "dev")
				Expect(err).NotTo(HaveOccurred())
				Expect(detail).NotTo(BeNil())
				Expect(detail.Metadata).NotTo(BeNil())
				Expect(detail.Metadata.BscpBizID).To(Equal("12345"))
				Expect(detail.EnvBinding).NotTo(BeNil())
				Expect(detail.EnvBinding.Services).To(HaveLen(1))
			})
		})
	})
})
