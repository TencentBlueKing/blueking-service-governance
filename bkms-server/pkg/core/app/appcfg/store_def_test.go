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

package appcfg_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

var _ = Describe("AppConfigFileDef store", func() {
	var defs *appcfg.AppConfigFileDefStore
	var ctx context.Context
	var appID string

	BeforeEach(func() {
		defs = testutil.NewStore(GinkgoT(), appcfg.AppConfigFileDefMeta)
		Expect(testutil.CleanupMetaColl(appcfg.AppConfigFileDefMeta)).To(Succeed())
		ctx = context.Background()
		appID = "def-test-app-" + stringx.Random(6)
	})

	newDef := func() *appcfg.AppConfigFileDef {
		return &appcfg.AppConfigFileDef{
			AppID:      appID,
			Name:       "values.yaml",
			ConfigKind: appcfg.ConfigKindFramework,
			MountDir:   "/data/conf",
			EnvConfigMode: appcfg.EnvConfigMode{
				IsUnifiedConfig: true,
			},
			Creator: "tester",
		}
	}

	mustCreate := func(def *appcfg.AppConfigFileDef) bson.ObjectID {
		Expect(defs.Create(ctx, def)).To(Succeed())
		return def.ID
	}

	Context("Create and GetByID", func() {
		It("should persist and retrieve a def record", func() {
			def := newDef()
			id := mustCreate(def)

			got, err := defs.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("values.yaml"))
			Expect(got.AppID).To(Equal(appID))
			Expect(got.MountDir).To(Equal("/data/conf"))
			Expect(got.EnvConfigMode.IsUnifiedConfig).To(BeTrue())

			var raw bson.M
			err = defs.Collection().FindOne(ctx, bson.M{"_id": id}).Decode(&raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(raw).To(HaveKeyWithValue("isUnifiedConfig", true))
			Expect(raw).NotTo(HaveKey("envConfigMode"))
		})

		It("should preserve explicit empty mounted env names", func() {
			def := newDef()
			def.EnvConfigMode.MountedEnvNames = []string{}
			id := mustCreate(def)

			got, err := defs.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.EnvConfigMode.MountedEnvNames).NotTo(BeNil())
			Expect(got.EnvConfigMode.MountedEnvNames).To(BeEmpty())
		})

		It("should allow the same name in different directories", func() {
			mustCreate(newDef())
			dup := newDef()
			dup.MountDir = "/data/other"
			Expect(defs.Create(ctx, dup)).To(Succeed())
		})

		It("should reject the same name when mountDir is empty", func() {
			first := newDef()
			first.MountDir = ""
			mustCreate(first)

			second := newDef()
			second.MountDir = ""
			Expect(defs.Create(ctx, second)).To(MatchError(appcfg.ErrAppConfigFileDefAlreadyExists))
		})

		It("should return ErrAppConfigFileDefNotFound for missing defs", func() {
			_, err := defs.GetByID(ctx, bson.NewObjectID())
			Expect(err).To(MatchError(appcfg.ErrAppConfigFileDefNotFound))
		})
	})

	Context("uniqueness", func() {
		It("should allow the same mountDir with different names", func() {
			first := newDef()
			first.Name = "a.conf"
			first.MountDir = "/etc/app"
			mustCreate(first)

			second := newDef()
			second.Name = "b.conf"
			second.MountDir = "/etc/app"
			mustCreate(second)
		})

		It("should reject the same mountDir and name", func() {
			first := newDef()
			first.Name = "app.conf"
			first.MountDir = "/etc/app"
			mustCreate(first)

			second := newDef()
			second.Name = "app.conf"
			second.MountDir = "/etc/app"
			Expect(defs.Create(ctx, second)).To(MatchError(appcfg.ErrAppConfigFileDefAlreadyExists))
		})
	})

	Context("UpdateDef", func() {
		It("should update name, flat env config fields and enableEnvVarRender", func() {
			id := mustCreate(newDef())

			got, err := defs.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())

			got.Name = "new-values.yaml"
			got.EnvConfigMode.IsUnifiedConfig = false
			got.EnvConfigMode.EnvContentStrategy = appcfg.EnvContentStrategyOverwrite
			got.EnableEnvVarRender = true
			Expect(appcfg.UpdateDef(ctx, defs, got)).To(Succeed())

			updated, err := defs.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("new-values.yaml"))
			Expect(updated.EnvConfigMode.IsUnifiedConfig).To(BeFalse())
			Expect(updated.EnvConfigMode.EnvContentStrategy).To(Equal(appcfg.EnvContentStrategyOverwrite))
			Expect(updated.EnableEnvVarRender).To(BeTrue())

			var raw bson.M
			err = defs.Collection().FindOne(ctx, bson.M{"_id": id}).Decode(&raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(raw).To(HaveKeyWithValue("isUnifiedConfig", false))
			Expect(raw).NotTo(HaveKey("envConfigMode"))
			Expect(updated.CreatedAt).To(Equal(got.CreatedAt))
		})

		It("should clear mountDir and keep other fields immutable", func() {
			id := mustCreate(newDef())

			Expect(appcfg.UpdateDef(ctx, defs, &appcfg.AppConfigFileDef{
				ID:         id,
				AppID:      "another-app",
				Name:       "values.yaml",
				ConfigKind: appcfg.ConfigKindPlain,
			})).To(Succeed())

			updated, err := defs.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.MountDir).To(BeEmpty())
			Expect(updated.AppID).To(Equal(appID))
			Expect(updated.ConfigKind).To(Equal(appcfg.ConfigKindFramework))

			var raw bson.M
			err = defs.Collection().FindOne(ctx, bson.M{"_id": id}).Decode(&raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(raw).NotTo(HaveKey("mountDir"))
		})

		It("should reject defs without ID and report missing defs", func() {
			Expect(appcfg.UpdateDef(ctx, defs, newDef())).To(MatchError(ContainSubstring("def ID is required")))

			def := newDef()
			def.ID = bson.NewObjectID()
			Expect(appcfg.UpdateDef(ctx, defs, def)).To(MatchError(appcfg.ErrAppConfigFileDefNotFound))
		})
	})

	Context("ListDefsByApp", func() {
		It("should list all defs for an app sorted by name and filter by ConfigKind", func() {
			m1 := newDef()
			m1.Name = "b.yaml"
			m1.MountDir = "/data/conf/b"
			mustCreate(m1)

			m2 := newDef()
			m2.Name = "a.yaml"
			m2.MountDir = "/data/conf/a"
			mustCreate(m2)

			all, err := appcfg.ListDefsByApp(ctx, defs, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(all).To(HaveLen(2))
			Expect(all[0].Name).To(Equal("a.yaml"))

			filtered, err := appcfg.ListDefsByApp(ctx, defs, appID,
				appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework))
			Expect(err).NotTo(HaveOccurred())
			Expect(filtered).To(HaveLen(2))
		})

		It("should not treat empty configKind as framework", func() {
			frameworkDef := newDef()
			frameworkDef.Name = "framework.yaml"
			mustCreate(frameworkDef)

			legacyDef := newDef()
			legacyDef.Name = "legacy.yaml"
			legacyDef.MountDir = "/data/conf/legacy"
			legacyDef.ConfigKind = ""
			mustCreate(legacyDef)

			all, err := appcfg.ListDefsByApp(ctx, defs, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(all).To(HaveLen(2))

			filtered, err := appcfg.ListDefsByApp(ctx, defs, appID,
				appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework))
			Expect(err).NotTo(HaveOccurred())
			Expect(filtered).To(HaveLen(1))
			Expect(filtered[0].Name).To(Equal("framework.yaml"))
			Expect(filtered[0].ConfigKind).To(Equal(appcfg.ConfigKindFramework))
		})
	})
})
