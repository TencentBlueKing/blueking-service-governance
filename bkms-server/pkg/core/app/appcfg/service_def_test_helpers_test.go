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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
)

// defServiceFixture 封装 AppCfgFileDefService 集成测试所需的共享上下文。
type defServiceFixture struct {
	DiApp        *fxtest.App
	AppStore     bkmsapp.ApplicationStore
	FileStore    appcfg.AppConfigFileStore
	DefStore     appcfg.AppConfigFileDefStore
	VersionStore appcfg.AppConfigFileVersionStore
	Svc          *appcfg.AppCfgFileDefService
	Ctx          context.Context
	AppID        string
}

// setupDefServiceFixture 初始化测试 fixture，应在 BeforeEach 中调用。
func setupDefServiceFixture() *defServiceFixture {
	f := &defServiceFixture{Ctx: context.Background()}

	Expect(testutil.CleanupCollection("app_config_file_versions")).To(Succeed())
	Expect(testutil.CleanupCollection("app_config_files")).To(Succeed())
	Expect(testutil.CleanupCollection("app_config_file_defs")).To(Succeed())
	Expect(testutil.CleanupCollection("applications")).To(Succeed())

	f.DiApp = fxtest.New(
		GinkgoT(),
		bkmsapp.FxModule,
		appcfg.FxModule,
		fx.Populate(&f.AppStore, &f.FileStore, &f.DefStore, &f.VersionStore),
	)
	f.DiApp.RequireStart()

	app := dbfactory.Application(f.Ctx, f.AppStore)
	f.AppID = app.ID

	base := appcfg.NewBaseAppCfgFileService(f.DefStore, f.FileStore, f.VersionStore)
	f.Svc = appcfg.NewAppCfgFileDefService(base)
	return f
}

// createFrameworkFile 创建 framework 类型的配置文件。
func (f *defServiceFixture) createFrameworkFile(name string) *appcfg.AppConfigFileWithDef {
	content := "key: value"
	result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
		AppID:             f.AppID,
		EnvName:           appcfg.EnvNameDefault,
		Name:              name,
		Type:              appcfg.AppConfigFileTypeNormal,
		ContentSourceType: appcfg.ContentSourceTypeLocal,
		Format:            appcfg.FileFormatYAML,
		Content:           &content,
		Creator:           "tester",
		Description:       "init",
		ConfigKind:        appcfg.ConfigKindFramework,
	})
	Expect(err).NotTo(HaveOccurred())
	return result
}

// createPlainFile 创建 plain 类型的配置文件。
func (f *defServiceFixture) createPlainFile(name, mountDir, content string) *appcfg.AppConfigFileWithDef {
	result, err := f.Svc.Create(f.Ctx, appcfg.CreateCfgFileParams{
		AppID:             f.AppID,
		EnvName:           appcfg.EnvNameDefault,
		Name:              name,
		MountDir:          mountDir,
		Type:              appcfg.AppConfigFileTypeNormal,
		ContentSourceType: appcfg.ContentSourceTypeLocal,
		Format:            appcfg.FileFormatYAML,
		Content:           &content,
		Creator:           "tester",
		Description:       "init",
		ConfigKind:        appcfg.ConfigKindPlain,
	})
	Expect(err).NotTo(HaveOccurred())
	return result
}
