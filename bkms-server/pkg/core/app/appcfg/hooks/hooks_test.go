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

package hooks_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	appcfghooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/hooks"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("Env delete hooks from appcfg", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var envSvc *bkmsenv.EnvService
	var appStore bkmsapp.ApplicationStore
	var envStore envmodel.EnvironmentStore
	var fileStore appcfg.AppConfigFileStore
	var defStore appcfg.AppConfigFileDefStore
	var versionStore appcfg.AppConfigFileVersionStore

	BeforeEach(func() {
		ctx = context.Background()
		bkmsenv.ResetHooksForTest()

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appcfg.FxModule,
			bkmsenv.FxModule,
			fx.Populate(
				&envSvc,
				&appStore,
				&envStore,
				&fileStore,
				&defStore,
				&versionStore,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		Expect(testutil.CleanupCollection("app_config_file_versions")).To(Succeed())
		Expect(testutil.CleanupCollection("app_config_files")).To(Succeed())
		Expect(testutil.CleanupCollection("app_config_file_defs")).To(Succeed())
		Expect(testutil.CleanupCollection("environments")).To(Succeed())
		Expect(testutil.CleanupCollection("applications")).To(Succeed())
		bkmsenv.ResetHooksForTest()
		diApp.RequireStop()
	})

	createMountedPlainEnvInstance := func(app *bkmsapp.Application, envName string) appcfg.AppConfigFileDef {
		base := appcfg.NewBaseAppCfgFileService(defStore, fileStore, versionStore)
		cfgSvc := appcfg.NewAppCfgFileDefService(base)
		defaultContent := "key: default\n"
		result, err := cfgSvc.Create(ctx, appcfg.CreateCfgFileParams{
			AppID:             app.ID,
			EnvName:           appcfg.EnvNameDefault,
			Name:              "plain-" + stringx.Random(6) + ".yaml",
			MountDir:          "/data/" + stringx.Random(4),
			Type:              appcfg.AppConfigFileTypeNormal,
			ContentSourceType: appcfg.ContentSourceTypeLocal,
			Format:            appcfg.FileFormatYAML,
			Content:           &defaultContent,
			Creator:           "tester",
			Description:       "init",
			ConfigKind:        appcfg.ConfigKindPlain,
		})
		Expect(err).NotTo(HaveOccurred())

		def, err := defStore.GetByID(ctx, result.Def.ID)
		Expect(err).NotTo(HaveOccurred())

		isUnified := false
		mounted := []string{envName}
		err = cfgSvc.UpdateAppCfgFileDef(ctx, def, appcfg.FileDefUpdate{
			IsUnifiedConfig: &isUnified,
			MountedEnvNames: &mounted,
			Operator:        "tester",
		})
		Expect(err).NotTo(HaveOccurred())

		def, err = defStore.GetByID(ctx, result.Def.ID)
		Expect(err).NotTo(HaveOccurred())

		envContent := "key: env\n"
		targetFile, _, isNewFile, err := cfgSvc.PrepareEnvContentUpdate(ctx, def, envName, envContent, "tester")
		Expect(err).NotTo(HaveOccurred())
		Expect(isNewFile).To(BeTrue())

		_, err = cfgSvc.CreateFileWithVersion(ctx, *targetFile, def.Name, "create env file", "tester")
		Expect(err).NotTo(HaveOccurred())

		return *def
	}

	It("should clean plain env instances for all apps in the deleted workspace env", func() {
		workspaceID := "test-ws-" + stringx.Random(6)
		appA := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
		appB := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
		environment := dbfactory.Env(ctx, envSvc, workspaceID)

		defA := createMountedPlainEnvInstance(appA, environment.Name)
		defB := createMountedPlainEnvInstance(appB, environment.Name)

		appcfghooks.RegisterDeleteHooks(appStore, fileStore, defStore, versionStore)

		err := envSvc.Delete(ctx, environment.ID)
		Expect(err).NotTo(HaveOccurred())

		_, err = fileStore.GetByDefIDAndEnv(ctx, defA.ID, environment.Name)
		Expect(errors.Is(err, appcfg.ErrAppConfigFileNotFound)).To(BeTrue())
		_, err = fileStore.GetByDefIDAndEnv(ctx, defB.ID, environment.Name)
		Expect(errors.Is(err, appcfg.ErrAppConfigFileNotFound)).To(BeTrue())

		defAUpdated, err := defStore.GetByID(ctx, defA.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(defAUpdated.EnvConfigMode.MountedEnvNames).NotTo(ContainElement(environment.Name))

		defBUpdated, err := defStore.GetByID(ctx, defB.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(defBUpdated.EnvConfigMode.MountedEnvNames).NotTo(ContainElement(environment.Name))
	})

	It("should fail fast when feature env owner app id is missing", func() {
		hook := appcfghooks.NewCleanupPlainEnvInstancesHook(appStore, fileStore, defStore, versionStore)

		err := hook(ctx, envmodel.Environment{
			Kind:        envmodel.EnvironmentKindFeature,
			WorkspaceID: "test-ws-" + stringx.Random(6),
			Name:        "feature-" + stringx.Random(6),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("owner app id is empty"))
	})
})
