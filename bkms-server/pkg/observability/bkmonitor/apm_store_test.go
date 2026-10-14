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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("ApmInstConfigStore", func() {
	var (
		ctx  context.Context
		db   *mongo.Database
		apms *ApmInstConfigStore
	)

	newApm := func(workspaceID string, apmID int64, name string, envs ...EnvInfo) *ApmInstConfig {
		return &ApmInstConfig{
			WorkspaceID:    workspaceID,
			ApmID:          apmID,
			Name:           name,
			Token:          "token",
			Creator:        "test-user",
			AssociatedEnvs: envs,
		}
	}

	mustCreate := func(apm *ApmInstConfig) *ApmInstConfig {
		Expect(apms.Create(ctx, apm)).To(Succeed())
		return apm
	}

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupMetaColl(ApmInstConfigMeta)).To(Succeed())
		db = testutil.NewDB(GinkgoT())
		apms = store.Of(db, ApmInstConfigMeta)
	})

	Describe("Create", func() {
		It("should store associatedEnvs as an empty array", func() {
			apm := mustCreate(newApm("ws", 1001, "apm"))

			raw := bson.M{}
			Expect(db.Collection(ApmInstConfigMeta.Collection).
				FindOne(ctx, bson.M{"_id": apm.ID}).Decode(&raw)).To(Succeed())
			Expect(raw["associatedEnvs"]).To(BeEquivalentTo(bson.A{}))
		})

		It("should reject a config missing required fields", func() {
			err := apms.Create(ctx, &ApmInstConfig{WorkspaceID: "ws", ApmID: 1001})
			Expect(err).To(MatchError(ContainSubstring("validation failed")))
		})

		It("should return ErrApmInstConfigExists on duplicate apmID", func() {
			mustCreate(newApm("ws", 1001, "apm"))
			err := apms.Create(ctx, newApm("ws", 1001, "apm-2"))
			Expect(err).To(MatchError(ErrApmInstConfigExists))
		})
	})

	Describe("Get", func() {
		It("should find configs by apmID, name and bound env", func() {
			envID := bson.NewObjectID()
			mustCreate(newApm("ws", 1001, "apm", EnvInfo{EnvID: envID, EnvName: "env"}))

			byApmID, err := apms.Get(ctx, ApmByApmID(1001))
			Expect(err).NotTo(HaveOccurred())
			Expect(byApmID.Name).To(Equal("apm"))

			byName, err := apms.Get(ctx, ApmByName("ws", "apm"))
			Expect(err).NotTo(HaveOccurred())
			Expect(byName.ApmID).To(Equal(int64(1001)))

			byEnv, err := apms.Get(ctx, ApmBoundToEnv(envID))
			Expect(err).NotTo(HaveOccurred())
			Expect(byEnv.AssociatedEnvs).To(Equal([]EnvInfo{{EnvID: envID, EnvName: "env"}}))
		})

		It("should return ErrApmInstConfigNotFound when nothing matches", func() {
			mustCreate(newApm("ws", 1001, "apm"))
			_, err := apms.Get(ctx, ApmByName("other-ws", "apm"))
			Expect(err).To(MatchError(ErrApmInstConfigNotFound))
		})
	})

	Describe("GetApmIDMap", func() {
		It("should index configs of the workspace by apmID", func() {
			mustCreate(newApm("ws-a", 1001, "apm-1"))
			mustCreate(newApm("ws-a", 1002, "apm-2"))
			mustCreate(newApm("ws-b", 1003, "apm-3"))

			m, err := GetApmIDMap(ctx, db, "ws-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(m).To(HaveLen(2))
			Expect(m[1001].Name).To(Equal("apm-1"))
			Expect(m[1002].Name).To(Equal("apm-2"))
		})
	})

	Describe("BindEnv", func() {
		var apm *ApmInstConfig

		BeforeEach(func() {
			apm = mustCreate(newApm("ws", 1001, "apm"))
		})

		It("should add the env only once", func() {
			envID := bson.NewObjectID()
			Expect(BindEnv(ctx, db, apm.ID, envID, "env")).To(Succeed())
			Expect(BindEnv(ctx, db, apm.ID, envID, "env")).To(Succeed())

			got, err := apms.Get(ctx, store.ByID(apm.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got.AssociatedEnvs).To(Equal([]EnvInfo{{EnvID: envID, EnvName: "env"}}))
		})

		It("should return ErrApmInstConfigNotFound for a missing config", func() {
			err := BindEnv(ctx, db, bson.NewObjectID(), bson.NewObjectID(), "env")
			Expect(err).To(MatchError(ErrApmInstConfigNotFound))
		})
	})

	Describe("UnbindEnvFromAll", func() {
		It("should remove the env from every config that references it", func() {
			envID, envID2 := bson.NewObjectID(), bson.NewObjectID()
			// 模拟脏数据：同一个 env 出现在多个 APM 的 associatedEnvs 列表中
			apm1 := mustCreate(newApm("ws", 1001, "apm-1", EnvInfo{EnvID: envID, EnvName: "env"}))
			apm2 := mustCreate(newApm("ws", 1002, "apm-2",
				EnvInfo{EnvID: envID, EnvName: "env"},
				EnvInfo{EnvID: envID2, EnvName: "env-2"},
			))

			Expect(UnbindEnvFromAll(ctx, db, envID)).To(Succeed())

			got1, err := apms.Get(ctx, store.ByID(apm1.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got1.AssociatedEnvs).To(BeEmpty())

			got2, err := apms.Get(ctx, store.ByID(apm2.ID))
			Expect(err).NotTo(HaveOccurred())
			Expect(got2.AssociatedEnvs).To(Equal([]EnvInfo{{EnvID: envID2, EnvName: "env-2"}}))
		})

		It("should succeed when no config references the env", func() {
			mustCreate(newApm("ws", 1001, "apm"))
			Expect(UnbindEnvFromAll(ctx, db, bson.NewObjectID())).To(Succeed())
		})
	})
})
