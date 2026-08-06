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

package service_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/service"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("Manager", func() {
	var configStore model.Store
	var mgr *service.Manager
	var ctx context.Context
	var testAppID string
	var diApp *fxtest.App

	BeforeEach(func() {
		ctx = context.Background()
		testAppID = "test-app-" + stringx.Random(5)

		diApp = fxtest.New(
			GinkgoT(),
			model.FxModule,
			fx.Supply(auth.User{ID: "test-user", Cred: auth.UserCredential{BkToken: "stub-token"}}),
			fx.Provide(service.NewManager),
			fx.Populate(&configStore, &mgr),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		// 清理测试数据
		_ = configStore.DeleteEnvBindingsByApp(ctx, testAppID)
		_ = configStore.DeleteMetadata(ctx, testAppID)
		diApp.RequireStop()
	})

	Describe("InitMetadata", func() {
		Context("when app config does not exist", func() {
			It("should create app config successfully", func() {
				appConfig, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
					AppID:     testAppID,
					BscpBizID: "100001",
					Operator:  "test-user",
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(appConfig.AppID).To(Equal(testAppID))
				Expect(appConfig.BscpBizID).To(Equal("100001"))
				Expect(appConfig.CredentialName).To(Equal("bkms-credential"))
				Expect(appConfig.Token).NotTo(BeEmpty())
				Expect(appConfig.FeedAddr).NotTo(BeEmpty())
			})
		})

		Context("when app config already exists", func() {
			It("should return existing config without error (idempotent)", func() {
				// 第一次创建
				first, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
					AppID:     testAppID,
					BscpBizID: "100001",
					Operator:  "test-user",
				})
				Expect(err).NotTo(HaveOccurred())

				// 第二次调用应返回相同结果
				second, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
					AppID:     testAppID,
					BscpBizID: "100001",
					Operator:  "test-user",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(second.AppID).To(Equal(first.AppID))
				Expect(second.Token).To(Equal(first.Token))
			})
		})
	})

	Describe("PatchMetadata", func() {
		BeforeEach(func() {
			_, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
				AppID:     testAppID,
				BscpBizID: "100001",
				Operator:  "test-user",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update mountPath and workload in one call", func() {
			mountPath := "/data/new-path"
			workload := "my-deployment"
			err := mgr.PatchMetadata(ctx, testAppID, &model.MetadataUpdate{
				MountPath:    &mountPath,
				WorkloadName: &workload,
			})
			Expect(err).NotTo(HaveOccurred())

			appConfig, err := configStore.GetMetadata(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfig.MountPath).To(Equal("/data/new-path"))
			Expect(appConfig.WorkloadName).To(Equal("my-deployment"))
		})

		It("should only update specified fields (nil fields unchanged)", func() {
			// 先设置 workload
			workload := "my-deployment"
			err := mgr.PatchMetadata(ctx, testAppID, &model.MetadataUpdate{
				WorkloadName: &workload,
			})
			Expect(err).NotTo(HaveOccurred())

			// 只更新 mountPath，workload 应保持不变
			mountPath := "/data/another-path"
			err = mgr.PatchMetadata(ctx, testAppID, &model.MetadataUpdate{
				MountPath: &mountPath,
			})
			Expect(err).NotTo(HaveOccurred())

			appConfig, err := configStore.GetMetadata(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfig.MountPath).To(Equal("/data/another-path"))
			Expect(appConfig.WorkloadName).To(Equal("my-deployment"))
		})

		It("should clear workload with empty string", func() {
			workload := "my-deployment"
			err := mgr.PatchMetadata(ctx, testAppID, &model.MetadataUpdate{
				WorkloadName: &workload,
			})
			Expect(err).NotTo(HaveOccurred())

			emptyWorkload := ""
			err = mgr.PatchMetadata(ctx, testAppID, &model.MetadataUpdate{
				WorkloadName: &emptyWorkload,
			})
			Expect(err).NotTo(HaveOccurred())

			appConfig, err := configStore.GetMetadata(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(appConfig.WorkloadName).To(Equal(""))
		})
	})

	Describe("GetSnapshot", func() {
		Context("when app config not enabled", func() {
			It("should return ErrMetadataNotFound", func() {
				_, err := mgr.GetSnapshot(ctx, testAppID, "prod")
				Expect(err).To(MatchError(model.ErrMetadataNotFound))
			})
		})

		Context("when env config not found", func() {
			BeforeEach(func() {
				_, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
					AppID:     testAppID,
					BscpBizID: "100001",
					Operator:  "test-user",
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return ErrEnvBindingNotFound", func() {
				_, err := mgr.GetSnapshot(ctx, testAppID, "non-existent-env")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("ListSnapshots", func() {
		Context("when app config not enabled", func() {
			It("should return ErrEnvBindingNotFound", func() {
				_, err := mgr.ListSnapshots(ctx, testAppID)
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})

		Context("when app config enabled but no env configs", func() {
			BeforeEach(func() {
				_, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
					AppID:     testAppID,
					BscpBizID: "100001",
					Operator:  "test-user",
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return empty list", func() {
				configs, err := mgr.ListSnapshots(ctx, testAppID)
				Expect(err).NotTo(HaveOccurred())
				Expect(configs).To(BeEmpty())
			})
		})
	})

	Describe("DeleteByApp", func() {
		BeforeEach(func() {
			// 创建 app 级配置
			_, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
				AppID:     testAppID,
				BscpBizID: "100001",
				Operator:  "test-user",
			})
			Expect(err).NotTo(HaveOccurred())

			// 手动创建 env 级配置
			err = configStore.CreateEnvBinding(ctx, &model.EnvBinding{
				AppID:   testAppID,
				EnvName: "prod",
				Services: []model.ServiceRef{
					{ID: "1001", Name: "stub-service-file"},
				},
				Operator: "test-user",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete both app and env configs", func() {
			err := mgr.DeleteByApp(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())

			// 验证 app 级配置已删除
			_, err = configStore.GetMetadata(ctx, testAppID)
			Expect(err).To(MatchError(model.ErrMetadataNotFound))

			// 验证 env 级配置已删除
			envConfigs, err := configStore.ListEnvBindingsByApp(ctx, testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(envConfigs).To(BeEmpty())
		})
	})

	Describe("DeleteEnvBinding", func() {
		BeforeEach(func() {
			err := configStore.CreateEnvBinding(ctx, &model.EnvBinding{
				AppID:   testAppID,
				EnvName: "staging",
				Services: []model.ServiceRef{
					{ID: "1001", Name: "stub-service-file"},
				},
				Operator: "test-user",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete the specified env config", func() {
			err := mgr.DeleteEnvBinding(ctx, testAppID, "staging")
			Expect(err).NotTo(HaveOccurred())

			_, err = configStore.GetEnvBinding(ctx, testAppID, "staging")
			Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
		})

		Context("when env config does not exist", func() {
			It("should return ErrEnvBindingNotFound", func() {
				err := mgr.DeleteEnvBinding(ctx, testAppID, "non-existent")
				Expect(err).To(MatchError(model.ErrEnvBindingNotFound))
			})
		})
	})

	Describe("BindServices", func() {
		BeforeEach(func() {
			// 创建 app 级配置
			_, err := mgr.InitMetadata(ctx, &service.InitMetadataParams{
				AppID:     testAppID,
				BscpBizID: "100001",
				Operator:  "test-user",
			})
			Expect(err).NotTo(HaveOccurred())

			// 创建 env 级配置（带 defaultServiceRefID）
			err = configStore.CreateEnvBinding(ctx, &model.EnvBinding{
				AppID:   testAppID,
				EnvName: "prod",
				Services: []model.ServiceRef{
					{ID: "default-svc-id", Name: "default-svc"},
				},
				DefaultServiceID: "default-svc-id",
				Operator:         "test-user",
			})
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when newApps contains the default app", func() {
			It("should update apps successfully", func() {
				newApps := []model.ServiceRef{
					{ID: "default-svc-id", Name: "default-svc"},
					{ID: "extra-svc-id", Name: "extra-svc"},
				}
				err := mgr.BindServices(ctx, testAppID, "prod", "100001", newApps)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when newApps does not contain the default app", func() {
			It("should return error", func() {
				newApps := []model.ServiceRef{
					{ID: "other-svc-id", Name: "other-svc"},
				}
				err := mgr.BindServices(ctx, testAppID, "prod", "100001", newApps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must contain the default file service"))
			})
		})

		Context("when env config does not exist", func() {
			It("should return error", func() {
				newApps := []model.ServiceRef{
					{ID: "any-id", Name: "any"},
				}
				err := mgr.BindServices(ctx, testAppID, "non-existent", "100001", newApps)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetOrCreateCredential", func() {
		It("should return credential from stub client", func() {
			cred, err := mgr.GetOrCreateCredential(ctx, "100001")
			Expect(err).NotTo(HaveOccurred())
			Expect(cred).NotTo(BeNil())
			Expect(cred.Name).To(Equal("bkms-credential"))
			Expect(cred.EncCredential).NotTo(BeEmpty())
		})
	})

	Describe("RefreshCredentialScopes", func() {
		It("should not return error with stub client", func() {
			err := mgr.RefreshCredentialScopes(ctx, "100001", 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetOrCreatePostHook", func() {
		It("should create the post hook when it does not exist", func() {
			id, err := mgr.GetOrCreatePostHook(ctx, "100001", testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(BeNumerically(">", int64(0)))
		})

		It("should be idempotent and reuse the existing hook on subsequent calls", func() {
			first, err := mgr.GetOrCreatePostHook(ctx, "100001", testAppID)
			Expect(err).NotTo(HaveOccurred())

			second, err := mgr.GetOrCreatePostHook(ctx, "100001", testAppID)
			Expect(err).NotTo(HaveOccurred())
			Expect(second).To(Equal(first))
		})
	})
})
