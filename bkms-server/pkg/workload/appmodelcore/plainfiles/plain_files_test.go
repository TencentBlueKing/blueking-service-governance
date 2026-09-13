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

package plainfiles_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/plainfiles"
)

var _ = Describe("BuildPlainConfigFiles", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var appStore bkmsapp.ApplicationStore
	var store appcfg.AppConfigFileStore
	var defStore appcfg.AppConfigFileDefStore
	var versionStore appcfg.AppConfigFileVersionStore
	var app *bkmsapp.Application
	var provider appcfg.MountableFileProvider

	BeforeEach(func() {
		var err error

		ctx = context.Background()
		err = testutil.CleanupCollection("app_config_file_defs")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("app_config_files")
		Expect(err).NotTo(HaveOccurred())
		err = testutil.CleanupCollection("applications")
		Expect(err).NotTo(HaveOccurred())

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			fx.Populate(&appStore, &store, &defStore, &versionStore),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		provider = appcfg.NewMountableFileProvider(store, defStore, versionStore)
	})

	AfterEach(func() {
		diApp.RequireStop()
	})

	createPlainDef := func(name, mountDir string) bson.ObjectID {
		id, err := defStore.Add(ctx, appcfg.AppConfigFileDef{
			AppID:      app.ID,
			Name:       name,
			ConfigKind: appcfg.ConfigKindPlain,
			MountDir:   mountDir,
			Creator:    "tester",
		})
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	addDefaultFile := func(defID bson.ObjectID, content string) {
		_, err := store.Add(ctx, appcfg.AppConfigFile{
			DefID:   defID,
			AppID:   app.ID,
			EnvName: appcfg.EnvNameDefault,
			Type:    appcfg.AppConfigFileTypeNormal,
			VersionedContent: appcfg.VersionedContent{
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
			},
		})
		Expect(err).NotTo(HaveOccurred())
	}

	It("should return nil when provider is nil", func() {
		envVars := map[string]string{}
		collector := envvarrefs.NewCollector(envVars)

		result, err := plainfiles.BuildPlainConfigFiles(ctx, nil, app.ID, "prod", envVars, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeNil())
	})

	It("should return empty when no plain defs exist", func() {
		envVars := map[string]string{}
		collector := envvarrefs.NewCollector(envVars)

		result, err := plainfiles.BuildPlainConfigFiles(ctx, provider, app.ID, "prod", envVars, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("should render plain config files with env var substitution", func() {
		defID := createPlainDef("nginx.conf", "/etc/nginx")
		addDefaultFile(defID, "server ${{ env.HOST }};")

		envVars := map[string]string{"HOST": "localhost"}
		collector := envvarrefs.NewCollector(envVars)

		result, err := plainfiles.BuildPlainConfigFiles(ctx, provider, app.ID, "prod", envVars, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].FileName).To(Equal("nginx.conf"))
		Expect(result[0].FilePath).To(Equal("/etc/nginx"))
		Expect(result[0].FileContent).To(Equal("server localhost;"))
	})

	It("should render multiple plain config files", func() {
		defA := createPlainDef("a.conf", "/etc/a")
		addDefaultFile(defA, "key=${{ env.A }}")

		defB := createPlainDef("b.conf", "/etc/b")
		addDefaultFile(defB, "key=static")

		envVars := map[string]string{"A": "1"}
		collector := envvarrefs.NewCollector(envVars)

		result, err := plainfiles.BuildPlainConfigFiles(ctx, provider, app.ID, "prod", envVars, collector)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(2))
	})

	It("should reject plain config files with empty mount dir", func() {
		defID := createPlainDef("bad.conf", "")
		addDefaultFile(defID, "content")

		envVars := map[string]string{}
		collector := envvarrefs.NewCollector(envVars)

		_, err := plainfiles.BuildPlainConfigFiles(ctx, provider, app.ID, "prod", envVars, collector)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("empty mount dir"))
	})
})
