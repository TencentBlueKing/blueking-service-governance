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
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	apphooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/hooks"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("Env delete hooks from app visible envs", func() {
	var diApp *fxtest.App
	var ctx context.Context
	var envSvc *bkmsenv.EnvService
	var appStore bkmsapp.ApplicationStore
	var envStore envmodel.EnvironmentStore

	BeforeEach(func() {
		ctx = context.Background()
		bkmsenv.ResetHooksForTest()

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			bkmsenv.FxModule,
			fx.Populate(
				&envSvc,
				&appStore,
				&envStore,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		Expect(testutil.CleanupCollection("environments")).To(Succeed())
		Expect(testutil.CleanupCollection("applications")).To(Succeed())
		bkmsenv.ResetHooksForTest()
		diApp.RequireStop()
	})

	It("should remove the deleted standard env name from workspace apps", func() {
		workspaceID := "test-ws-" + stringx.Random(6)
		appA := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
		appB := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
		environment := dbfactory.Env(ctx, envSvc, workspaceID)

		Expect(appStore.UpdateVisibleEnvNames(ctx, appA, []string{environment.Name, "stag"})).To(Succeed())
		Expect(appStore.UpdateVisibleEnvNames(ctx, appB, []string{environment.Name})).To(Succeed())

		apphooks.RegisterDeleteHooks(appStore)
		Expect(envSvc.Delete(ctx, environment.ID)).To(Succeed())

		appA, err := appStore.GetApp(ctx, appA.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(appA.VisibleEnvNames).To(Equal([]string{"stag"}))

		appB, err = appStore.GetApp(ctx, appB.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(appB.VisibleEnvNames).To(BeEmpty())
		// 摘掉最后一个 name 后名单为空，按 R-001 等同「未配置」，appB 不再受可见环境限制。
		// 这是有意接受的取舍，改动此行为前先看 NewCleanupVisibleEnvNamesHook 的说明。
	})

	It("should leave other workspaces' visible env names unchanged", func() {
		workspaceID := "test-ws-" + stringx.Random(6)
		otherWorkspaceID := "test-ws-" + stringx.Random(6)
		app := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
		otherApp := dbfactory.ApplicationWithOpts(
			ctx,
			appStore,
			&dbfactory.ApplicationOpts{WorkspaceID: otherWorkspaceID},
		)
		environment := dbfactory.Env(ctx, envSvc, workspaceID)

		Expect(appStore.UpdateVisibleEnvNames(ctx, app, []string{environment.Name})).To(Succeed())
		Expect(appStore.UpdateVisibleEnvNames(ctx, otherApp, []string{environment.Name})).To(Succeed())

		apphooks.RegisterDeleteHooks(appStore)
		Expect(envSvc.Delete(ctx, environment.ID)).To(Succeed())

		otherApp, err := appStore.GetApp(ctx, otherApp.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(otherApp.VisibleEnvNames).To(Equal([]string{environment.Name}))
	})

	It("should skip feature environments", func() {
		workspaceID := "test-ws-" + stringx.Random(6)
		app := dbfactory.ApplicationWithOpts(ctx, appStore, &dbfactory.ApplicationOpts{WorkspaceID: workspaceID})
		Expect(appStore.UpdateVisibleEnvNames(ctx, app, []string{"test", "stag"})).To(Succeed())

		hook := apphooks.NewCleanupVisibleEnvNamesHook(appStore)
		err := hook(ctx, envmodel.Environment{
			Kind:        envmodel.EnvironmentKindFeature,
			WorkspaceID: workspaceID,
			Name:        "feat-" + stringx.Random(6),
			OwnerAppID:  app.ID,
		})
		Expect(err).NotTo(HaveOccurred())

		app, err = appStore.GetApp(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(app.VisibleEnvNames).To(Equal([]string{"test", "stag"}))
	})
})
