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

package deploy

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/envvarrefs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

func newTestPolarisConfig(
	appID, name string,
	scopeEnvNames []string,
	serviceLabels map[string]string,
) *polaris.PolarisConfig {
	return &polaris.PolarisConfig{
		Name:  name,
		AppID: appID,
		Properties: polaris.Properties{
			InstanceKey:      "test-instance",
			PolarisName:      "test-service",
			PolarisNamespace: "Test",
			PolarisToken:     "token",
			ServicePort:      8080,
			ServiceLabels:    serviceLabels,
		},
		ScopeType:     component.ScopeTypeEnvironment,
		ScopeEnvNames: scopeEnvNames,
	}
}

var _ = Describe("EnvVarPreChecker Check", func() {
	var (
		ctx                       context.Context
		fxApp                     *fxtest.App
		appStore                  bkmsapp.ApplicationStore
		appModelStore             appmodel.AppModelStore
		appConfigFileStore        appcfg.AppConfigFileStore
		appConfigFileVersionStore appcfg.AppConfigFileVersionStore
		componentDefStore         component.ComponentDefStore
		polarisConfigStore        polaris.PolarisConfigStore
		buildConfigStore          build.ConfigStore
		workspaceCompsStore       workspace.WorkspaceCompsStore
		envService                *env.EnvService
		checker                   *EnvVarPreChecker
		newApp                    func(*dbfactory.TrpcApplicationOpts) (*bkmsapp.Application, *envmodel.Environment)
	)

	BeforeEach(func() {
		ctx = context.Background()
		fxApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appmodel.FxModule,
			appcfg.FxModule,
			appspec.FxModule,
			component.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			env.FxModule,
			envvars.FxModule,
			workspace.FxModule,
			build.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			workload.FxModule,
			fx.Provide(NewEnvVarPreChecker),
			fx.Populate(
				&appStore,
				&appModelStore,
				&appConfigFileStore,
				&appConfigFileVersionStore,
				&componentDefStore,
				&polarisConfigStore,
				&buildConfigStore,
				&workspaceCompsStore,
				&envService,
				&checker,
			),
		)
		fxApp.RequireStart()
		workload.InitPlugin(appConfigFileStore, polarisConfigStore)

		newApp = func(opts *dbfactory.TrpcApplicationOpts) (*bkmsapp.Application, *envmodel.Environment) {
			app, _ := dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, opts)
			return app, dbfactory.Env(ctx, envService, app.WorkspaceID)
		}
	})

	AfterEach(func() {
		fxApp.RequireStop()
	})

	DescribeTable("rejects nil inputs",
		func(app *bkmsapp.Application, appEnv *envmodel.Environment) {
			_, err := checker.Check(ctx, app, appEnv)

			Expect(err).To(MatchError("app and environment are required"))
		},
		Entry("when the app is nil", nil, &envmodel.Environment{}),
		Entry("when the environment is nil", &bkmsapp.Application{}, nil),
	)

	It("returns an error when the app model does not exist", func() {
		app := dbfactory.Application(ctx, appStore)
		appEnv := dbfactory.Env(ctx, envService, app.WorkspaceID)

		_, err := checker.Check(ctx, app, appEnv)

		Expect(err).To(MatchError(ContainSubstring("get app " + app.ID + " model")))
	})

	It("aggregates, deduplicates, and sorts undefined vars", func() {
		compDef := dbfactory.CompDef(ctx, componentDefStore, &dbfactory.ComponentDefOpts{
			Properties: []component.Property{
				{Name: "value", Type: component.PropTypeString, DefaultValue: "${{ env.SHARED }}"},
			},
		})
		comp := &component.Component{
			Name: "shared-component",
			ComponentInst: component.ComponentInst{
				Type:    compDef.Name,
				Version: compDef.Version,
			},
		}
		app, appEnv := newApp(&dbfactory.TrpcApplicationOpts{
			TrpcConfig: &appmodel.TrpcConfig{FileContent: `z: ${{ env.ZED }}
shared: ${{ env.SHARED }} ${{ env.SHARED }}
ignored: ${LEGACY}
`},
			Components: []*component.Component{comp},
		})
		Expect(polarisConfigStore.Create(ctx, newTestPolarisConfig(
			app.ID,
			"polaris-main",
			[]string{appEnv.Name},
			map[string]string{"shared": "${{ env.SHARED }}"},
		))).To(Succeed())

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&EnvVarPreCheckResult{UndefinedVars: []envvarrefs.UndefinedEnvVar{
			{
				Key: "SHARED",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourceAppConfigFile, Name: appcfg.DefaultAppConfigFileName},
					{Type: envvarrefs.SourceComponent, Name: comp.Name},
					{Type: envvarrefs.SourcePolaris, Name: "polaris-main"},
				},
			},
			{
				Key: "ZED",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourceAppConfigFile, Name: appcfg.DefaultAppConfigFileName},
				},
			},
		}}))
	})

	It("treats empty and sensitive env vars as defined", func() {
		app, appEnv := newApp(&dbfactory.TrpcApplicationOpts{
			TrpcConfig: &appmodel.TrpcConfig{
				FileContent: "empty: ${{ env.EMPTY }}\nsecret: ${{ env.SECRET }}\n",
			},
			EnvVars: []appmodel.Variable{
				{Key: "EMPTY", Value: ""},
				{Key: "SECRET", Value: "secret-value", IsSensitive: true},
			},
		})

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedVars).To(BeEmpty())
		Expect(result.UndefinedVars).NotTo(BeNil())
	})

	It("builds the pre-check report when the persisted workload image is empty", func() {
		app, appEnv := newApp(nil)
		model, err := appModelStore.GetAppModel(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		model.Workload.Image = ""
		Expect(appModelStore.UpdateAppModel(ctx, model)).To(Succeed())

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedVars).To(BeEmpty())
		Expect(result.UndefinedVars).NotTo(BeNil())
		persistedModel, err := appModelStore.GetAppModel(ctx, app.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(persistedModel.Workload.Image).To(BeEmpty())
	})

	It("uses the environment config file as the reference source", func() {
		app, appEnv := newApp(&dbfactory.TrpcApplicationOpts{
			TrpcConfig: &appmodel.TrpcConfig{FileContent: "default: ${{ env.DEFAULT_ONLY }}\n"},
		})
		content := "environment: ${{ env.ENV_ONLY }}\n"
		_, err := appcfg.NewAppConfigFileService(appConfigFileStore, appConfigFileVersionStore).Create(
			ctx,
			appcfg.CreateCfgFileParams{
				AppID:             app.ID,
				EnvName:           appEnv.Name,
				Name:              appEnv.Name,
				Type:              appcfg.AppConfigFileTypeNormal,
				ContentSourceType: appcfg.ContentSourceTypeLocal,
				Format:            appcfg.FileFormatYAML,
				Content:           &content,
				Creator:           appcfg.CfgSystemUser,
				Description:       appcfg.CfgSystemVersionDescription,
			},
		)
		Expect(err).NotTo(HaveOccurred())

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedVars).To(Equal([]envvarrefs.UndefinedEnvVar{
			{
				Key: "ENV_ONLY",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourceAppConfigFile, Name: appEnv.Name},
				},
			},
		}))
	})

	It("scans effective component property values", func() {
		compDef := dbfactory.CompDef(ctx, componentDefStore, &dbfactory.ComponentDefOpts{
			Properties: []component.Property{
				{Name: "value", Type: component.PropTypeString, DefaultValue: "${{ env.DEFAULT_VALUE }}"},
				{Name: "replicas", Type: component.PropTypeInt, DefaultValue: int64(1)},
			},
		})
		app, appEnv := newApp(&dbfactory.TrpcApplicationOpts{
			Components: []*component.Component{
				{
					Name:          "default-component",
					ComponentInst: component.ComponentInst{Type: compDef.Name, Version: compDef.Version},
				},
				{
					Name: "override-component",
					ComponentInst: component.ComponentInst{
						Type:       compDef.Name,
						Version:    compDef.Version,
						Properties: map[string]any{"value": "${{ env.OVERRIDE_VALUE }}"},
					},
				},
			},
		})

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedVars).To(Equal([]envvarrefs.UndefinedEnvVar{
			{
				Key: "DEFAULT_VALUE",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourceComponent, Name: "default-component"},
				},
			},
			{
				Key: "OVERRIDE_VALUE",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourceComponent, Name: "override-component"},
				},
			},
		}))
	})

	It("filters workspace components by environment", func() {
		compDef := dbfactory.CompDef(ctx, componentDefStore, &dbfactory.ComponentDefOpts{
			Properties: []component.Property{{Name: "value", Type: component.PropTypeString}},
		})
		app, appEnv := newApp(&dbfactory.TrpcApplicationOpts{
			Components: []*component.Component{
				{ComponentRef: component.ComponentRef{RefWorkspaceCompName: "available-component"}},
				{ComponentRef: component.ComponentRef{RefWorkspaceCompName: "hidden-component"}},
			},
		})
		for _, comp := range []*workspace.Component{
			{
				Name:          "available-component",
				WorkspaceID:   app.WorkspaceID,
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{appEnv.Name},
				ComponentInst: component.ComponentInst{
					Type:       compDef.Name,
					Version:    compDef.Version,
					Properties: map[string]any{"value": "${{ env.AVAILABLE }}"},
				},
			},
			{
				Name:          "hidden-component",
				WorkspaceID:   app.WorkspaceID,
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{"another-environment"},
				ComponentInst: component.ComponentInst{
					Type:       compDef.Name,
					Version:    compDef.Version,
					Properties: map[string]any{"value": "${{ env.HIDDEN }}"},
				},
			},
		} {
			Expect(workspaceCompsStore.Add(ctx, comp)).To(Succeed())
		}

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedVars).To(Equal([]envvarrefs.UndefinedEnvVar{
			{
				Key: "AVAILABLE",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourceComponent, Name: "available-component"},
				},
			},
		}))
	})

	It("scans Polaris service labels only", func() {
		app, appEnv := newApp(nil)
		config := newTestPolarisConfig(
			app.ID,
			"polaris-main",
			[]string{appEnv.Name},
			map[string]string{"region": "${{ env.SERVICE_REGION }}"},
		)
		config.InstanceKey = "${{ env.IGNORED_INSTANCE_KEY }}"
		config.PolarisName = "${{ env.IGNORED_POLARIS_NAME }}"
		config.PolarisNamespace = "${{ env.IGNORED_POLARIS_NAMESPACE }}"
		config.PolarisToken = "${{ env.IGNORED_POLARIS_TOKEN }}"
		Expect(polarisConfigStore.Create(ctx, config)).To(Succeed())

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedVars).To(Equal([]envvarrefs.UndefinedEnvVar{
			{
				Key: "SERVICE_REGION",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourcePolaris, Name: "polaris-main"},
				},
			},
		}))
	})

	It("filters Polaris configs by environment", func() {
		app, appEnv := newApp(nil)
		for _, config := range []*polaris.PolarisConfig{
			newTestPolarisConfig(
				app.ID,
				"visible-polaris",
				[]string{appEnv.Name},
				map[string]string{"env": "${{ env.VISIBLE_POLARIS }}"},
			),
			newTestPolarisConfig(
				app.ID,
				"hidden-polaris",
				[]string{"another-environment"},
				map[string]string{"env": "${{ env.HIDDEN_POLARIS }}"},
			),
		} {
			Expect(polarisConfigStore.Create(ctx, config)).To(Succeed())
		}

		result, err := checker.Check(ctx, app, appEnv)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.UndefinedVars).To(Equal([]envvarrefs.UndefinedEnvVar{
			{
				Key: "VISIBLE_POLARIS",
				Sources: []envvarrefs.Source{
					{Type: envvarrefs.SourcePolaris, Name: "visible-polaris"},
				},
			},
		}))
	})

	It("returns an error for invalid templates", func() {
		app, appEnv := newApp(&dbfactory.TrpcApplicationOpts{
			TrpcConfig: &appmodel.TrpcConfig{FileContent: "${{ env.BROKEN "},
		})

		_, err := checker.Check(ctx, app, appEnv)

		Expect(err).To(MatchError(ContainSubstring("collecting env vars from tRPC config")))
	})

	It("returns an error when a component definition is missing", func() {
		app, appEnv := newApp(&dbfactory.TrpcApplicationOpts{
			Components: []*component.Component{
				{
					Name: "missing-component",
					ComponentInst: component.ComponentInst{
						Type:    "MissingComponent",
						Version: component.DefaultComponentDefVersion,
					},
				},
			},
		})

		_, err := checker.Check(ctx, app, appEnv)

		Expect(err).To(MatchError(ContainSubstring("getting component definition")))
	})

	Context("when required data cannot be loaded", func() {
		var (
			app    *bkmsapp.Application
			appEnv *envmodel.Environment
		)

		BeforeEach(func() {
			app, appEnv = newApp(nil)
		})

		It("returns an error when the effective app config is missing", func() {
			_, err := appConfigFileStore.DeleteByApp(ctx, app.ID)
			Expect(err).NotTo(HaveOccurred())

			_, err = checker.Check(ctx, app, appEnv)

			Expect(err).To(MatchError(ContainSubstring("no config file found")))
		})

		It("returns workload build errors", func() {
			Expect(buildConfigStore.Delete(ctx, app.ID)).To(Succeed())

			_, err := checker.Check(ctx, app, appEnv)

			Expect(err).To(MatchError(ContainSubstring("get build config")))
		})
	})
})
