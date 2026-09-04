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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("AppConfigFileDefStoreMongo", func() {
	var store *appcfg.AppConfigFileDefStoreMongo
	var ctx context.Context
	var appID string

	BeforeEach(func() {
		var err error
		store, err = appcfg.NewAppConfigFileDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("app_config_file_defs")
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
		appID = "def-test-app-" + stringx.Random(6)
	})

	newDef := func() appcfg.AppConfigFileDef {
		return appcfg.AppConfigFileDef{
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

	Context("Add and GetByID", func() {
		It("should persist and retrieve a def record", func() {
			def := newDef()
			id, err := store.Add(ctx, def)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(Equal(bson.NilObjectID))

			got, err := store.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("values.yaml"))
			Expect(got.AppID).To(Equal(appID))
			Expect(got.MountDir).To(Equal("/data/conf"))
			Expect(got.EnvConfigMode.IsUnifiedConfig).To(BeTrue())
		})
	})

	Context("Update", func() {
		It("should update name and envConfigMode", func() {
			def := newDef()
			id, err := store.Add(ctx, def)
			Expect(err).NotTo(HaveOccurred())

			got, err := store.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())

			got.Name = "new-values.yaml"
			got.EnvConfigMode.IsUnifiedConfig = false
			cnt, err := store.Update(ctx, *got)
			Expect(err).NotTo(HaveOccurred())
			Expect(cnt).To(Equal(int64(1)))

			updated, err := store.GetByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Name).To(Equal("new-values.yaml"))
			Expect(updated.EnvConfigMode.IsUnifiedConfig).To(BeFalse())
		})
	})

	Context("ListByApp", func() {
		It("should list all metas for an app and filter by ConfigKind", func() {
			m1 := newDef()
			m1.Name = "a.yaml"
			m1.MountDir = "/data/conf/a"
			_, err := store.Add(ctx, m1)
			Expect(err).NotTo(HaveOccurred())

			m2 := newDef()
			m2.Name = "b.yaml"
			m2.MountDir = "/data/conf/b"
			_, err = store.Add(ctx, m2)
			Expect(err).NotTo(HaveOccurred())

			all, err := store.ListByApp(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(all).To(HaveLen(2))

			filtered, err := store.ListByApp(ctx, appID,
				appcfg.DefFilterConfigKind(appcfg.ConfigKindFramework))
			Expect(err).NotTo(HaveOccurred())
			Expect(filtered).To(HaveLen(2))
		})
	})

	Context("DeleteByID", func() {
		It("should remove the def record", func() {
			id, err := store.Add(ctx, newDef())
			Expect(err).NotTo(HaveOccurred())

			cnt, err := store.DeleteByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(cnt).To(Equal(int64(1)))

			_, err = store.GetByID(ctx, id)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("DeleteByApp", func() {
		It("should remove all def records for the app", func() {
			m1 := newDef()
			m1.Name = "x.yaml"
			m1.MountDir = "/data/conf/x"
			_, err := store.Add(ctx, m1)
			Expect(err).NotTo(HaveOccurred())
			m2 := newDef()
			m2.Name = "y.yaml"
			m2.MountDir = "/data/conf/y"
			_, err = store.Add(ctx, m2)
			Expect(err).NotTo(HaveOccurred())

			cnt, err := store.DeleteByApp(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(cnt).To(Equal(int64(2)))

			all, err := store.ListByApp(ctx, appID)
			Expect(err).NotTo(HaveOccurred())
			Expect(all).To(BeEmpty())
		})
	})
})
