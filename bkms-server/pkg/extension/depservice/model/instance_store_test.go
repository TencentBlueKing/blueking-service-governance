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

package model

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Test ServiceInstanceStoreMongo", func() {
	var ctx context.Context

	var originCryptoKey string

	var mongoClient *mongo.Client
	var store ServiceInstanceStore
	var dbName string

	BeforeEach(func() {
		var err error

		mongoClient, dbName = database.Client(), database.Name()

		ctx = context.Background()
		store, err = NewServiceInstanceStoreMongo(mongoClient, dbName)
		Expect(err).NotTo(HaveOccurred())

		// 重置测试密钥
		originCryptoKey = config.G.Encrypt.Secret
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		config.G.Encrypt.Secret = secret
	})
	AfterEach(func() {
		config.G.Encrypt.Secret = originCryptoKey

		err := store.DeleteAll(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Test ServiceInstanceStoreMongo methods", func() {
		var initInst *ServiceInstance
		var initInstID bson.ObjectID

		var initConfig map[string]any
		var initCreds map[string]any
		var initMessage string
		var initStatus InstanceStatus

		BeforeEach(func() {
			var err error

			initStatus = ProvisioningStatus
			initMessage = stringx.Random(6)
			initConfig = map[string]any{"foo": "bar", "funny": true}
			initCreds = map[string]any{"username": stringx.Random(6), "password": stringx.Random(6)}
			initInst = &ServiceInstance{
				Name:         "test-name" + stringx.Random(6),
				WorkspaceID:  "test-workspace-id" + stringx.Random(6),
				ServiceName:  "test-service-name" + stringx.Random(6),
				PlanName:     "test-plan-name" + stringx.Random(6),
				ProviderType: "test-provider-type",
				ScopeType:    ScopeTypeEnv,
				ScopeValue:   "stage-" + stringx.Random(4),
				Config:       initConfig,
				Credentials:  initCreds,
				Status:       initStatus,
				Message:      initMessage,
				Operator:     "test-operator" + stringx.Random(6),
			}

			// test: create service instance
			initInstID, err = store.Create(ctx, initInst)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			// test: delete service instance
			err := store.Delete(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("test create rejects workspace scope with non-empty scopeValue", func() {
			inst := &ServiceInstance{
				Name:         "test-name" + stringx.Random(6),
				WorkspaceID:  "test-workspace-id" + stringx.Random(6),
				ServiceName:  "test-service-name" + stringx.Random(6),
				PlanName:     "test-plan-name" + stringx.Random(6),
				ProviderType: "test-provider-type",
				ScopeType:    ScopeTypeWorkspace,
				ScopeValue:   "should-be-empty",
				Operator:     "test-operator" + stringx.Random(6),
			}
			_, err := store.Create(ctx, inst)
			Expect(err).To(HaveOccurred())
		})

		It("test create rejects envType scope with invalid scopeValue", func() {
			inst := &ServiceInstance{
				Name:         "test-name" + stringx.Random(6),
				WorkspaceID:  "test-workspace-id" + stringx.Random(6),
				ServiceName:  "test-service-name" + stringx.Random(6),
				PlanName:     "test-plan-name" + stringx.Random(6),
				ProviderType: "test-provider-type",
				ScopeType:    ScopeTypeEnvType,
				ScopeValue:   "invalid-env-type",
				Operator:     "test-operator" + stringx.Random(6),
			}
			_, err := store.Create(ctx, inst)
			Expect(err).To(HaveOccurred())
		})

		It("test create rejects unknown scopeType", func() {
			inst := &ServiceInstance{
				Name:         "test-name" + stringx.Random(6),
				WorkspaceID:  "test-workspace-id" + stringx.Random(6),
				ServiceName:  "test-service-name" + stringx.Random(6),
				PlanName:     "test-plan-name" + stringx.Random(6),
				ProviderType: "test-provider-type",
				ScopeType:    "unknown",
				ScopeValue:   "whatever",
				Operator:     "test-operator" + stringx.Random(6),
			}
			_, err := store.Create(ctx, inst)
			Expect(err).To(HaveOccurred())
		})

		It("test get service instance", func() {
			inst, err := store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.ID).To(Equal(initInstID))
			Expect(inst.Name).To(Equal(initInst.Name))
			Expect(inst.ServiceName).To(Equal(initInst.ServiceName))
			Expect(inst.Config).To(Equal(initInst.Config))
			Expect(inst.Credentials).To(Equal(initInst.Credentials))
		})

		It("test update config", func() {
			newConfig := map[string]any{stringx.Random(6): stringx.Random(6)}

			err := store.UpdateConfig(ctx, initInstID, newConfig)
			Expect(err).NotTo(HaveOccurred())

			inst, err := store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			// test: 此处验证了 serviceInstancePrepDBValue 和 serviceInstanceFromDBValue
			Expect(inst.Config).To(Equal(newConfig))

			// 还原 config
			Expect(store.UpdateConfig(ctx, initInstID, initConfig)).NotTo(HaveOccurred())
		})

		It("test update credentials", func() {
			newCreds := map[string]any{stringx.Random(6): stringx.Random(6)}

			err := store.UpdateCredentials(ctx, initInstID, newCreds)
			Expect(err).NotTo(HaveOccurred())

			inst, err := store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			// test: 此处验证了 serviceInstancePrepDBValue 和 serviceInstanceFromDBValue
			Expect(inst.Credentials).To(Equal(newCreds))

			// 还原 credentials
			Expect(store.UpdateCredentials(ctx, initInstID, initCreds)).NotTo(HaveOccurred())
		})

		It("test update status", func() {
			err := store.UpdateStatus(ctx, initInstID, AvailableStatus, "")
			Expect(err).NotTo(HaveOccurred())

			inst, err := store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.Status).To(Equal(AvailableStatus))
			Expect(inst.Message).To(BeEmpty())

			// 还原 status
			Expect(store.UpdateStatus(ctx, initInstID, initStatus, initMessage)).NotTo(HaveOccurred())
		})

		It("test update", func() {
			newConfig := map[string]any{stringx.Random(6): stringx.Random(6), "updated": true}
			newCreds := map[string]any{"new_username": stringx.Random(6), "new_password": stringx.Random(6)}
			newCustomEnvVars := map[string]string{
				"FOO_DSN": "foo://${{FOO_USER}}@${{FOO_HOST}}",
			}
			newOperator := "test-operator-" + stringx.Random(6)

			updateData := &SvcInstUpdateData{
				ScopeType:     ScopeTypeWorkspace,
				ScopeValue:    "",
				Config:        newConfig,
				Credentials:   newCreds,
				CustomEnvVars: newCustomEnvVars,
				Operator:      newOperator,
			}

			err := store.Update(ctx, initInstID, updateData)
			Expect(err).NotTo(HaveOccurred())

			inst, err := store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			// test: 验证所有字段都已更新
			Expect(inst.ScopeType).To(Equal(ScopeTypeWorkspace))
			Expect(inst.ScopeValue).To(Equal(""))
			Expect(inst.Config).To(Equal(newConfig))
			Expect(inst.Credentials).To(Equal(newCreds))
			Expect(inst.CustomEnvVars).To(Equal(newCustomEnvVars))
			Expect(inst.Operator).To(Equal(newOperator))

			// 还原数据
			originalUpdateData := &SvcInstUpdateData{
				ScopeType:   initInst.ScopeType,
				ScopeValue:  initInst.ScopeValue,
				Config:      initConfig,
				Credentials: initCreds,
				Operator:    initInst.Operator,
			}
			Expect(store.Update(ctx, initInstID, originalUpdateData)).NotTo(HaveOccurred())
		})

		It("test update rejects invalid scope", func() {
			err := store.Update(ctx, initInstID, &SvcInstUpdateData{
				ScopeType:  ScopeTypeWorkspace,
				ScopeValue: "should-be-empty",
				Operator:   "x",
			})
			Expect(err).To(HaveOccurred())
		})

		It("test attach/detach app", func() {
			appID1 := "app-id-" + stringx.Random(6)
			appID2 := "app-id-" + stringx.Random(6)
			appID3 := "app-id-" + stringx.Random(6)

			// 关联第一个应用
			err := store.AttachApp(ctx, initInstID, appID1)
			Expect(err).NotTo(HaveOccurred())

			inst, err := store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(ContainElement(appID1))
			Expect(inst.AttachedApps).To(HaveLen(1))

			// 关联第二个应用
			err = store.AttachApp(ctx, initInstID, appID2)
			Expect(err).NotTo(HaveOccurred())

			inst, err = store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(ContainElement(appID1))
			Expect(inst.AttachedApps).To(ContainElement(appID2))
			Expect(inst.AttachedApps).To(HaveLen(2))

			// 重复关联同一个应用，应该不会重复
			err = store.AttachApp(ctx, initInstID, appID1)
			Expect(err).NotTo(HaveOccurred())

			inst, err = store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(HaveLen(2))

			// 关联第三个应用
			err = store.AttachApp(ctx, initInstID, appID3)
			Expect(err).NotTo(HaveOccurred())

			inst, err = store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(HaveLen(3))

			// 解关联第一个应用
			err = store.DetachApp(ctx, initInstID, appID1)
			Expect(err).NotTo(HaveOccurred())

			inst, err = store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(ContainElement(appID2))
			Expect(inst.AttachedApps).To(ContainElement(appID3))
			Expect(inst.AttachedApps).To(HaveLen(2))

			// 解关联第二个应用
			err = store.DetachApp(ctx, initInstID, appID2)
			Expect(err).NotTo(HaveOccurred())

			inst, err = store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(ContainElement(appID3))
			Expect(inst.AttachedApps).To(HaveLen(1))

			// 重复解关联同一个应用，应该不会报错
			err = store.DetachApp(ctx, initInstID, appID1)
			Expect(err).NotTo(HaveOccurred())

			inst, err = store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(HaveLen(1))

			// 解关联最后一个应用
			err = store.DetachApp(ctx, initInstID, appID3)
			Expect(err).NotTo(HaveOccurred())

			inst, err = store.Get(ctx, initInstID)
			Expect(err).NotTo(HaveOccurred())
			Expect(inst.AttachedApps).To(BeEmpty())
		})

		Context("test list service instances", func() {
			var err error
			var tempInstID bson.ObjectID
			var envTypeInstID bson.ObjectID

			BeforeEach(func() {
				tempInstID, err = store.Create(ctx, &ServiceInstance{
					Name:         "test-name" + stringx.Random(6),
					WorkspaceID:  "test-workspace-id" + stringx.Random(6),
					ServiceName:  initInst.ServiceName,
					PlanName:     "test-plan-name" + stringx.Random(6),
					ProviderType: "test-provider-type",
					ScopeType:    ScopeTypeWorkspace,
					ScopeValue:   "",
					Operator:     stringx.Random(6),
				})
				Expect(err).NotTo(HaveOccurred())

				envTypeInstID, err = store.Create(ctx, &ServiceInstance{
					Name:         "test-name" + stringx.Random(6),
					WorkspaceID:  initInst.WorkspaceID,
					ServiceName:  initInst.ServiceName,
					PlanName:     "test-plan-name" + stringx.Random(6),
					ProviderType: "test-provider-type",
					ScopeType:    ScopeTypeEnvType,
					ScopeValue:   "test",
					Operator:     stringx.Random(6),
				})
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := store.Delete(ctx, tempInstID)
				Expect(err).NotTo(HaveOccurred())
				err = store.Delete(ctx, envTypeInstID)
				Expect(err).NotTo(HaveOccurred())
			})

			It("test list by workspace", func() {
				instList, err := store.List(ctx, &SvcInstQueryOptions{WorkspaceID: initInst.WorkspaceID})
				Expect(err).NotTo(HaveOccurred())
				// initInst (env scope) + envTypeInst (envType scope) 同 workspace
				Expect(instList).To(HaveLen(2))
			})

			It("test list by service name", func() {
				instList, err := store.List(ctx, &SvcInstQueryOptions{ServiceName: initInst.ServiceName})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(3))
			})

			It("test list by env (matches env scope and envType scope)", func() {
				instList, err := store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID: initInst.WorkspaceID,
					EnvName:     initInst.ScopeValue,
					EnvType:     "test",
				})
				Expect(err).NotTo(HaveOccurred())
				// 命中: initInst(env scope) + envTypeInst(envType=test)
				Expect(instList).To(HaveLen(2))
			})

			It("test list by env: only envType matches", func() {
				instList, err := store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID: initInst.WorkspaceID,
					EnvName:     "non-existent-env",
					EnvType:     "test",
				})
				Expect(err).NotTo(HaveOccurred())
				// 命中: envTypeInst(envType=test)
				Expect(instList).To(HaveLen(1))
				Expect(instList[0].ID).To(Equal(envTypeInstID))
			})

			It("test list by env: workspace-scoped instance always visible", func() {
				// tempInst 用于另一个 workspace, 但 ScopeType=workspace 时也该在自己 workspace 中可见
				instList, err := store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID: "non-existent-workspace",
					EnvName:     "any",
					EnvType:     "production",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(BeEmpty())
			})

			It("test list by attachedAppID: only returns instances attached to the given app", func() {
				appID1 := "app-id-" + stringx.Random(6)
				appID2 := "app-id-" + stringx.Random(6)

				// initInst attach appID1, envTypeInst attach appID2
				Expect(store.AttachApp(ctx, initInstID, appID1)).NotTo(HaveOccurred())
				Expect(store.AttachApp(ctx, envTypeInstID, appID2)).NotTo(HaveOccurred())

				// 按 appID1 过滤, 只应返回 initInst
				instList, err := store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID:   initInst.WorkspaceID,
					AttachedAppID: appID1,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(1))
				Expect(instList[0].ID).To(Equal(initInstID))

				// 按 appID2 过滤, 只应返回 envTypeInst
				instList, err = store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID:   initInst.WorkspaceID,
					AttachedAppID: appID2,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(1))
				Expect(instList[0].ID).To(Equal(envTypeInstID))

				// 按不存在的 appID 过滤, 应返回空
				instList, err = store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID:   initInst.WorkspaceID,
					AttachedAppID: "non-existent-app-id",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(BeEmpty())
			})

			It("test list by attachedAppID: combined with env filters", func() {
				appID := "app-id-" + stringx.Random(6)

				// initInst 和 envTypeInst 都 attach 同一个 app
				Expect(store.AttachApp(ctx, initInstID, appID)).NotTo(HaveOccurred())
				Expect(store.AttachApp(ctx, envTypeInstID, appID)).NotTo(HaveOccurred())

				// 同时按 appID + env 过滤, 应命中两个实例
				instList, err := store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID:   initInst.WorkspaceID,
					AttachedAppID: appID,
					EnvName:       initInst.ScopeValue,
					EnvType:       "test",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(2))

				// envType 不匹配时, envTypeInst 不应命中
				instList, err = store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID:   initInst.WorkspaceID,
					AttachedAppID: appID,
					EnvName:       initInst.ScopeValue,
					EnvType:       "production",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(1))
				Expect(instList[0].ID).To(Equal(initInstID))
			})

			It("test list by status: only returns instances with the given status", func() {
				// 确保 initInst 为 AvailableStatus, envTypeInst 为 ProvisioningStatus
				Expect(store.UpdateStatus(ctx, initInstID, AvailableStatus, "")).NotTo(HaveOccurred())
				Expect(store.UpdateStatus(ctx, envTypeInstID, ProvisioningStatus, "")).NotTo(HaveOccurred())

				// 只查 AvailableStatus, 应只返回 initInst
				instList, err := store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID: initInst.WorkspaceID,
					Status:      AvailableStatus,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(1))
				Expect(instList[0].ID).To(Equal(initInstID))

				// 只查 ProvisioningStatus, 应只返回 envTypeInst
				instList, err = store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID: initInst.WorkspaceID,
					Status:      ProvisioningStatus,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(1))
				Expect(instList[0].ID).To(Equal(envTypeInstID))

				// 不设 Status 过滤, 应返回全部
				instList, err = store.List(ctx, &SvcInstQueryOptions{
					WorkspaceID: initInst.WorkspaceID,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(instList).To(HaveLen(2))

				// 还原 status
				Expect(store.UpdateStatus(ctx, initInstID, ProvisioningStatus, "")).NotTo(HaveOccurred())
			})
		})
	})
})
