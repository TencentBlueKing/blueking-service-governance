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

package bkmonitor_test

import (
	"context"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var _ = Describe("ApmService", func() {
	var (
		ctx               context.Context
		diApp             *fxtest.App
		store             ApmInstConfigStore
		scopedEnvVarStore envvars.ScopedEnvVarStore
		svc               *ApmService
	)

	BeforeEach(func() {
		ctx = context.Background()

		// 清理测试数据
		err := testutil.CleanupCollection("bkmonitor_apm_inst_config")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("env_variables")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("scoped_env_vars")
		Expect(err).NotTo(HaveOccurred())

		// 使用 FxModule 统一注入所有依赖
		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(&store, &svc),
		)
		diApp.RequireStart()

		scopedEnvVarStore, err = envvars.NewScopedEnvVarStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	// ==================== Get ====================

	Describe("Get", func() {
		Context("when local APM record already exists", func() {
			var existingApm *ApmInstConfig

			BeforeEach(func() {
				existingApm = &ApmInstConfig{
					WorkspaceID: "test-workspace",
					ApmID:       2001,
					Name:        "existing-apm",
					Token:       "existing-token",
					Creator:     "test-user",
				}
				id, err := store.Create(ctx, existingApm)
				Expect(err).NotTo(HaveOccurred())
				existingApm.ID = id
			})

			// 本地已有记录时，应直接返回本地数据，不调用远程 API
			It("should return from local without calling remote API", func() {
				result, err := svc.Get(ctx, 2001, CreateApmInstParams{
					WorkspaceID:  "test-workspace",
					Username:     "test-user",
					BkmProjectID: 100,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.ApmID).To(Equal(int64(2001)))
				Expect(result.Name).To(Equal("existing-apm"))
				Expect(result.Token).To(Equal("existing-token"))
			})
		})

		Context("when not found locally and needs to fetch from remote", func() {
			// 本地不存在时，应从远程 API 获取并持久化到本地
			It("should fetch from remote API and persist locally", func() {
				mockey.PatchConvey("fetch APM from remote successfully", GinkgoT(), func() {
					mockClient := new(bkmonitor.ApiClient)
					mockey.Mock(bkmonitor.New).Return(mockClient, nil).Build()
					mockey.Mock((*bkmonitor.ApiClient).ListApmApp).Return([]*bkmonitor.ApmApp{
						{ID: 3001, AppName: "remote-apm", Token: "remote-token", BkBizID: 100},
						{ID: 3002, AppName: "other-apm", Token: "other-token", BkBizID: 100},
					}, nil).Build()

					result, err := svc.Get(ctx, 3001, CreateApmInstParams{
						WorkspaceID:  "test-workspace",
						Username:     "test-user",
						BkmProjectID: 100,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.ApmID).To(Equal(int64(3001)))
					Expect(result.Name).To(Equal("remote-apm"))
					Expect(result.Token).To(Equal("remote-token"))
					Expect(result.ID).NotTo(Equal(bson.NilObjectID))

					// 验证已持久化到本地
					var localApm *ApmInstConfig
					localApm, err = store.GetByApmID(ctx, 3001)
					Expect(err).NotTo(HaveOccurred())
					Expect(localApm.Name).To(Equal("remote-apm"))
				})
			})

			// 远程 API 中找不到目标 APM 时，应返回错误
			It("should return error when target APM not found in remote API", func() {
				mockey.PatchConvey("target APM not found in remote", GinkgoT(), func() {
					mockClient := new(bkmonitor.ApiClient)
					mockey.Mock(bkmonitor.New).Return(mockClient, nil).Build()
					mockey.Mock((*bkmonitor.ApiClient).ListApmApp).Return([]*bkmonitor.ApmApp{
						{ID: 9999, AppName: "other-apm", Token: "other-token", BkBizID: 100},
					}, nil).Build()

					result, err := svc.Get(ctx, 3001, CreateApmInstParams{
						WorkspaceID:  "test-workspace",
						Username:     "test-user",
						BkmProjectID: 100,
					})
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("apm 3001 not found"))
					Expect(result).To(BeNil())
				})
			})
		})
	})

	// ==================== BindToEnv ====================

	Describe("BindToEnv", func() {
		var (
			apm   *ApmInstConfig
			envID bson.ObjectID
		)

		BeforeEach(func() {
			envID = bson.NewObjectID()
			apm = &ApmInstConfig{
				WorkspaceID: "test-workspace",
				ApmID:       4001,
				Name:        "bind-test-apm",
				Token:       "bind-test-token",
				Creator:     "test-user",
			}
			id, err := store.Create(ctx, apm)
			Expect(err).NotTo(HaveOccurred())
			apm.ID = id
		})

		// 应成功将环境绑定到 APM
		It("should successfully bind env to APM", func() {
			err := svc.BindToEnv(ctx, apm, envID, "test-env")
			Expect(err).NotTo(HaveOccurred())

			// 验证绑定关系已建立
			result, err := store.GetByEnvID(ctx, envID)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.ApmID).To(Equal(int64(4001)))
			Expect(result.AssociatedEnvs).To(HaveLen(1))
			Expect(result.AssociatedEnvs[0].EnvName).To(Equal("test-env"))
			assertApmEnvVars(ctx, scopedEnvVarStore, apm.WorkspaceID, "test-env", apm.Token)
		})

		// 重复绑定相同环境时，不应报错
		It("should not error on duplicate binding of the same env", func() {
			err := svc.BindToEnv(ctx, apm, envID, "test-env")
			Expect(err).NotTo(HaveOccurred())

			// 再次绑定
			err = svc.BindToEnv(ctx, apm, envID, "test-env")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when env is already bound to another APM", func() {
			var apm2 *ApmInstConfig

			BeforeEach(func() {
				// 先将 env 绑定到 apm
				err := store.BindEnv(ctx, apm.ID, envID, "test-env")
				Expect(err).NotTo(HaveOccurred())

				// 创建另一个 APM
				apm2 = &ApmInstConfig{
					WorkspaceID: "test-workspace",
					ApmID:       4002,
					Name:        "bind-test-apm-2",
					Token:       "bind-test-token-2",
					Creator:     "test-user",
				}
				id, err := store.Create(ctx, apm2)
				Expect(err).NotTo(HaveOccurred())
				apm2.ID = id
			})

			// 应先解绑旧的关联，再绑定到新的 APM
			It("should unbind old association first then bind to new APM", func() {
				err := svc.BindToEnv(ctx, apm2, envID, "test-env")
				Expect(err).NotTo(HaveOccurred())

				// 验证旧 APM 已解绑
				oldApm, err := store.GetByApmID(ctx, 4001)
				Expect(err).NotTo(HaveOccurred())
				Expect(oldApm.AssociatedEnvs).To(HaveLen(0))

				// 验证新 APM 已绑定
				newApm, err := store.GetByApmID(ctx, 4002)
				Expect(err).NotTo(HaveOccurred())
				Expect(newApm.AssociatedEnvs).To(HaveLen(1))
				Expect(newApm.AssociatedEnvs[0].EnvID).To(Equal(envID))
			})
		})
	})

	// ==================== CreateAndBindToEnv ====================

	Describe("CreateAndBindToEnv", func() {
		Context("when local APM with the same name already exists", func() {
			var existingApm *ApmInstConfig

			BeforeEach(func() {
				existingApm = &ApmInstConfig{
					WorkspaceID: "test-workspace",
					ApmID:       5001,
					Name:        "test-env",
					Token:       "existing-token",
					Creator:     "test-user",
				}
				id, err := store.Create(ctx, existingApm)
				Expect(err).NotTo(HaveOccurred())
				existingApm.ID = id
			})

			// 应复用已存在的同名 APM 并绑定到环境
			It("should reuse existing APM and bind to env", func() {
				envID := bson.NewObjectID()
				result, err := svc.CreateAndBindToEnv(ctx, envID, "test-env", "bcs-project", CreateApmInstParams{
					WorkspaceID:  "test-workspace",
					Username:     "test-user",
					BkmProjectID: 100,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.ApmID).To(Equal(int64(5001)))

				// 验证绑定关系
				bound, err := store.GetByEnvID(ctx, envID)
				Expect(err).NotTo(HaveOccurred())
				Expect(bound.ApmID).To(Equal(int64(5001)))
				assertApmEnvVars(ctx, scopedEnvVarStore, existingApm.WorkspaceID, "test-env", existingApm.Token)
			})
		})

		Context("when local APM with the same name does not exist and needs remote creation", func() {
			// 应调用远程 API 创建 APM，持久化并绑定到环境
			It("should call remote API to create, persist and bind", func() {
				mockey.PatchConvey("remote APM creation succeeded", GinkgoT(), func() {
					mockClient := new(bkmonitor.ApiClient)
					mockey.Mock(bkmonitor.New).Return(mockClient, nil).Build()
					mockey.Mock((*bkmonitor.ApiClient).GetOrCreate).Return(&bkmonitor.ApmApp{
						ID:      6001,
						AppName: "new-env",
						Token:   "new-token",
						BkBizID: 100,
					}, nil).Build()
					mockey.Mock((*bkmonitor.ApiClient).ListApmApp).Return([]*bkmonitor.ApmApp{
						{ID: 6001, AppName: "new-env", Token: "new-token", BkBizID: 100},
					}, nil).Build()

					envID := bson.NewObjectID()
					result, err := svc.CreateAndBindToEnv(ctx, envID, "new-env", "bcs-project", CreateApmInstParams{
						WorkspaceID:  "test-workspace",
						Username:     "test-user",
						BkmProjectID: 100,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.ApmID).To(Equal(int64(6001)))
					Expect(result.Name).To(Equal("new-env"))
					Expect(result.Token).To(Equal("new-token"))

					// 验证已持久化
					localApm, err := store.GetByApmID(ctx, 6001)
					Expect(err).NotTo(HaveOccurred())
					Expect(localApm.Name).To(Equal("new-env"))

					// 验证绑定关系
					bound, err := store.GetByEnvID(ctx, envID)
					Expect(err).NotTo(HaveOccurred())
					Expect(bound.ApmID).To(Equal(int64(6001)))
					assertApmEnvVars(ctx, scopedEnvVarStore, "test-workspace", "new-env", "new-token")
				})
			})
		})
	})
})

func assertApmEnvVars(
	ctx context.Context,
	store envvars.ScopedEnvVarStore,
	workspaceID string,
	envName string,
	expectedToken string,
) {
	envVars, err := store.List(
		ctx,
		workspaceID,
		envvars.WithScopes(envvartypes.ScopeEnv(envName)),
		envvars.WithOnlyBuiltin(),
	)
	Expect(err).NotTo(HaveOccurred())
	Expect(envVars).To(HaveLen(3))

	varsByKey := make(map[string]envvars.ScopedEnvVar, len(envVars))
	for _, item := range envVars {
		varsByKey[item.Key] = item
	}
	Expect(varsByKey[bkmsenv.EnvVarNameApmGRPCAPI].IsBuiltin).To(BeTrue())
	Expect(varsByKey[bkmsenv.EnvVarNameApmHTTPAPI].IsBuiltin).To(BeTrue())
	Expect(varsByKey[bkmsenv.EnvVarNameApmToken].Value).To(Equal(expectedToken))
	Expect(varsByKey[bkmsenv.EnvVarNameApmToken].IsBuiltin).To(BeTrue())
	Expect(varsByKey[bkmsenv.EnvVarNameApmToken].IsSensitive).To(BeTrue())
}
