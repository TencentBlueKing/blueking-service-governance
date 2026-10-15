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
	"encoding/json"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Service instance store", func() {
	var ctx context.Context
	var db *mongo.Database
	var insts *ServiceInstanceStore

	var initInst *ServiceInstance
	var initConfig map[string]any
	var initCreds map[string]any

	newInstance := func(scopeType ScopeType, scopeValue string) *ServiceInstance {
		return &ServiceInstance{
			Name:         "test-name" + stringx.Random(6),
			WorkspaceID:  "test-workspace-id" + stringx.Random(6),
			ServiceName:  "test-service-name" + stringx.Random(6),
			PlanName:     "test-plan-name" + stringx.Random(6),
			ProviderType: "test-provider-type",
			ScopeType:    scopeType,
			ScopeValue:   scopeValue,
			Operator:     "test-operator" + stringx.Random(6),
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		insts = store.Of(db, ServiceInstanceMeta)

		// 重置测试密钥
		originCryptoKey := config.G.Encrypt.Secret
		secret, err := crypto.GenerateKey(32)
		Expect(err).NotTo(HaveOccurred())
		config.G.Encrypt.Secret = secret
		DeferCleanup(func() {
			config.G.Encrypt.Secret = originCryptoKey
			Expect(testutil.CleanupMetaColl(ServiceInstanceMeta)).To(Succeed())
		})

		initConfig = map[string]any{"foo": "bar", "funny": true}
		initCreds = map[string]any{"username": stringx.Random(6), "password": stringx.Random(6)}
		initInst = newInstance(ScopeTypeEnv, "stage-"+stringx.Random(4))
		initInst.Config = initConfig
		initInst.Credentials = initCreds
		initInst.Status = ProvisioningStatus
		initInst.Message = stringx.Random(6)
		Expect(insts.Create(ctx, initInst)).To(Succeed())
		Expect(initInst.ID.IsZero()).To(BeFalse())
	})

	getInst := func(id bson.ObjectID) *ServiceInstance {
		inst, err := insts.Get(ctx, store.ByID(id))
		Expect(err).NotTo(HaveOccurred())
		return inst
	}

	Context("Create and Get", func() {
		It("round trips config and credentials", func() {
			inst := getInst(initInst.ID)
			Expect(inst.Name).To(Equal(initInst.Name))
			Expect(inst.ServiceName).To(Equal(initInst.ServiceName))
			Expect(inst.Config).To(Equal(initConfig))
			Expect(inst.Credentials).To(Equal(initCreds))
		})

		It("stores config as json and credentials encrypted", func() {
			raw, err := db.Collection(ServiceInstanceMeta.Collection).
				FindOne(ctx, bson.M{"_id": initInst.ID}).Raw()
			Expect(err).NotTo(HaveOccurred())

			rawConfig, ok := raw.Lookup("config", configValueKey).StringValueOK()
			Expect(ok).To(BeTrue())
			Expect(rawConfig).To(MatchJSON(`{"foo": "bar", "funny": true}`))

			ciphertext, ok := raw.Lookup("credentials", configValueKey).StringValueOK()
			Expect(ok).To(BeTrue())
			Expect(ciphertext).NotTo(ContainSubstring(initCreds["password"].(string)))
			plaintext, err := crypto.AESDecrypt(config.G.Encrypt.Secret, ciphertext)
			Expect(err).NotTo(HaveOccurred())
			Expect(plaintext).To(MatchJSON(lo.Must(json.Marshal(initCreds))))
		})

		It("rejects duplicate names in the same workspace and service", func() {
			dup := *initInst
			dup.ID = bson.NilObjectID
			Expect(insts.Create(ctx, &dup)).To(MatchError(ErrInstanceNameExists))
		})

		It("rejects workspace scope with non-empty scopeValue", func() {
			Expect(insts.Create(ctx, newInstance(ScopeTypeWorkspace, "should-be-empty"))).NotTo(Succeed())
		})

		It("rejects envType scope with invalid scopeValue", func() {
			Expect(insts.Create(ctx, newInstance(ScopeTypeEnvType, "invalid-env-type"))).NotTo(Succeed())
		})

		It("rejects unknown scopeType", func() {
			Expect(insts.Create(ctx, newInstance("unknown", "whatever"))).NotTo(Succeed())
		})

		It("reports a missing instance as not found", func() {
			_, err := insts.Get(ctx, store.ByID(bson.NewObjectID()))
			Expect(AsNotFoundError(err)).To(BeTrue())
		})
	})

	Context("InstancesByIDs", func() {
		It("omits missing ids", func() {
			list, err := insts.List(ctx, InstancesByIDs([]bson.ObjectID{initInst.ID, bson.NewObjectID()}))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(HaveLen(1))
			Expect(list[0].ID).To(Equal(initInst.ID))
		})

		It("returns empty for empty ids", func() {
			list, err := insts.List(ctx, InstancesByIDs(nil))
			Expect(err).NotTo(HaveOccurred())
			Expect(list).To(BeEmpty())
		})
	})

	Context("Update config and credentials", func() {
		It("replaces config without touching credentials", func() {
			newConfig := map[string]any{stringx.Random(6): stringx.Random(6)}
			Expect(insts.Update(ctx, map[string]any{"config": newConfig}, store.ByID(initInst.ID))).To(Succeed())

			inst := getInst(initInst.ID)
			Expect(inst.Config).To(Equal(newConfig))
			Expect(inst.Credentials).To(Equal(initCreds))
		})

		It("replaces credentials without touching config", func() {
			newCreds := map[string]any{stringx.Random(6): stringx.Random(6)}
			Expect(insts.Update(ctx, map[string]any{"credentials": newCreds}, store.ByID(initInst.ID))).To(Succeed())

			inst := getInst(initInst.ID)
			Expect(inst.Credentials).To(Equal(newCreds))
			Expect(inst.Config).To(Equal(initConfig))
		})

		It("patches config by merging into existing keys", func() {
			Expect(PatchInstanceConfig(ctx, db, initInst.ID, map[string]any{
				"foo":            "patched",
				"createTicketID": 123,
			})).To(Succeed())

			inst := getInst(initInst.ID)
			Expect(inst.Config["foo"]).To(Equal("patched"))
			Expect(inst.Config["funny"]).To(Equal(true))
			Expect(inst.Config["createTicketID"]).To(BeEquivalentTo(123))
			Expect(inst.Credentials).To(Equal(initCreds))
		})

		It("patches credentials by merging into existing keys", func() {
			Expect(PatchInstanceCredentials(ctx, db, initInst.ID, map[string]any{
				"password": "new-password",
				"extra":    "keep-me",
			})).To(Succeed())

			inst := getInst(initInst.ID)
			Expect(inst.Credentials["username"]).To(Equal(initCreds["username"]))
			Expect(inst.Credentials["password"]).To(Equal("new-password"))
			Expect(inst.Credentials["extra"]).To(Equal("keep-me"))
			Expect(inst.Config).To(Equal(initConfig))
		})

		It("ignores an empty patch", func() {
			Expect(PatchInstanceConfig(ctx, db, bson.NewObjectID(), nil)).To(Succeed())
		})

		It("reports patching a missing instance as not found", func() {
			err := PatchInstanceConfig(ctx, db, bson.NewObjectID(), map[string]any{"foo": "bar"})
			Expect(AsNotFoundError(err)).To(BeTrue())
		})
	})

	Context("UpdateInstanceStatus", func() {
		It("updates status and message", func() {
			Expect(UpdateInstanceStatus(ctx, db, initInst.ID, AvailableStatus, "")).To(Succeed())

			inst := getInst(initInst.ID)
			Expect(inst.Status).To(Equal(AvailableStatus))
			Expect(inst.Message).To(BeEmpty())
			Expect(inst.Config).To(Equal(initConfig))
			Expect(inst.Credentials).To(Equal(initCreds))
		})

		It("reports a missing instance as not found", func() {
			err := UpdateInstanceStatus(ctx, db, bson.NewObjectID(), AvailableStatus, "")
			Expect(AsNotFoundError(err)).To(BeTrue())
		})
	})

	Context("List with SvcInstQueryOptions", func() {
		var otherWorkspaceInst, envTypeInst *ServiceInstance

		BeforeEach(func() {
			otherWorkspaceInst = newInstance(ScopeTypeWorkspace, "")
			otherWorkspaceInst.ServiceName = initInst.ServiceName
			Expect(insts.Create(ctx, otherWorkspaceInst)).To(Succeed())

			envTypeInst = newInstance(ScopeTypeEnvType, "test")
			envTypeInst.WorkspaceID = initInst.WorkspaceID
			envTypeInst.ServiceName = initInst.ServiceName
			Expect(insts.Create(ctx, envTypeInst)).To(Succeed())
		})

		listIDs := func(opts *SvcInstQueryOptions) []bson.ObjectID {
			list, err := insts.List(ctx, opts.Cond(), RecentlyUpdatedFirst())
			Expect(err).NotTo(HaveOccurred())
			ids := make([]bson.ObjectID, 0, len(list))
			for _, inst := range list {
				ids = append(ids, inst.ID)
			}
			return ids
		}

		It("lists by workspace, most recently updated first", func() {
			// initInst (env scope) + envTypeInst (envType scope) 同 workspace
			Expect(listIDs(&SvcInstQueryOptions{WorkspaceID: initInst.WorkspaceID})).
				To(Equal([]bson.ObjectID{envTypeInst.ID, initInst.ID}))

			// 显式推后 updatedAt，避免与创建时间落在同一毫秒
			later := map[string]any{"updatedAt": time.Now().Add(time.Hour)}
			Expect(insts.Update(ctx, later, store.ByID(initInst.ID))).To(Succeed())
			Expect(listIDs(&SvcInstQueryOptions{WorkspaceID: initInst.WorkspaceID})).
				To(Equal([]bson.ObjectID{initInst.ID, envTypeInst.ID}))
		})

		It("lists by service name", func() {
			Expect(listIDs(&SvcInstQueryOptions{ServiceName: initInst.ServiceName})).To(HaveLen(3))
		})

		It("treats nil options as no filter", func() {
			Expect(listIDs(nil)).To(HaveLen(3))
		})

		It("matches env scope and envType scope by env", func() {
			ids := listIDs(&SvcInstQueryOptions{
				WorkspaceID: initInst.WorkspaceID,
				EnvName:     initInst.ScopeValue,
				EnvType:     "test",
			})
			Expect(ids).To(ConsistOf(initInst.ID, envTypeInst.ID))
		})

		It("matches only envType scope when env name does not match", func() {
			ids := listIDs(&SvcInstQueryOptions{
				WorkspaceID: initInst.WorkspaceID,
				EnvName:     "non-existent-env",
				EnvType:     "test",
			})
			Expect(ids).To(Equal([]bson.ObjectID{envTypeInst.ID}))
		})

		It("matches workspace scope for any env", func() {
			ids := listIDs(&SvcInstQueryOptions{
				WorkspaceID: otherWorkspaceInst.WorkspaceID,
				EnvName:     "any",
				EnvType:     "production",
			})
			Expect(ids).To(Equal([]bson.ObjectID{otherWorkspaceInst.ID}))
		})

		It("filters by status", func() {
			Expect(UpdateInstanceStatus(ctx, db, initInst.ID, AvailableStatus, "")).To(Succeed())
			Expect(UpdateInstanceStatus(ctx, db, envTypeInst.ID, ProvisioningStatus, "")).To(Succeed())

			Expect(listIDs(&SvcInstQueryOptions{WorkspaceID: initInst.WorkspaceID, Status: AvailableStatus})).
				To(Equal([]bson.ObjectID{initInst.ID}))
			Expect(listIDs(&SvcInstQueryOptions{WorkspaceID: initInst.WorkspaceID, Status: ProvisioningStatus})).
				To(Equal([]bson.ObjectID{envTypeInst.ID}))
		})

		It("filters by scope type", func() {
			Expect(listIDs(&SvcInstQueryOptions{ServiceName: initInst.ServiceName, ScopeType: ScopeTypeWorkspace})).
				To(Equal([]bson.ObjectID{otherWorkspaceInst.ID}))
		})
	})
})
