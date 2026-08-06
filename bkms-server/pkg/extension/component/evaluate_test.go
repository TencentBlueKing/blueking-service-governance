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

package component_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

var _ = Describe("evaluate tests", func() {
	var ctx context.Context
	var diApp *fxtest.App
	var appenv *envmodel.Environment

	// app related fixtures
	var appStore bkmsapp.ApplicationStore
	var app *bkmsapp.Application

	// env & envvars related fixtures
	var envSvc *env.EnvService
	var scopedEnvVarStore envvars.ScopedEnvVarStore
	var appDepsVarReader *depenvvars.Reader
	var polarisVarReader *polarisenvvars.Reader

	// component definition related fixtures
	var compDefStore component.ComponentDefStore
	var compDef *component.ComponentDef
	var propsBuilder *component.AppPropertiesBuilder

	BeforeEach(func() {
		ctx = context.Background()

		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			component.FxModule,
			env.FxModule,
			envvars.FxModule,
			depmodel.FxModule,
			depenvvars.FxModule,
			polaris.FxModule,
			polarisenvvars.FxModule,
			fx.Populate(
				&appStore,
				&compDefStore,
				&envSvc,
				&scopedEnvVarStore,
				&appDepsVarReader,
				&polarisVarReader,
			),
		)
		diApp.RequireStart()

		// Create fixtures
		app = dbfactory.Application(ctx, appStore)
		compDef = dbfactory.CompDef(ctx, compDefStore, &dbfactory.ComponentDefOpts{
			Properties: []component.Property{
				{Name: "replicas", Type: "INT", DefaultValue: int64(1)},
				{Name: "homepageTitle", Type: "STRING", DefaultValue: "defaultApp"},
			},
			Patchers: []string{
				"replicas: {{ .replicas }}\nhomepageTitle: {{ .homepageTitle }}\nenvName: {{ .bkmsEnvName }}\n",
			},
		})
		appenv = dbfactory.Env(ctx, envSvc, app.WorkspaceID)

		propsBuilder = component.NewAppPropertiesBuilder(compDefStore, envSvc)
	})

	AfterEach(func() {
		_, _ = compDefStore.Delete(ctx, compDef.Name, compDef.Version)
		diApp.RequireStop()
	})

	Describe("AppComponentApplier", func() {
		Context("Evaluate", func() {
			It("renders template with overrides, env vars, and builtin props", func() {
				applier := component.NewAppComponentApplier(compDefStore, envSvc)
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"replicas":      int64(3),
							"homepageTitle": "Welcome ${{ env.FOO_TITLE }}",
							"mapProp": map[string]any{
								"key": "${{ env.FOO_TITLE }}",
							},
						},
					},
				}
				_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *appenv, "FOO_TITLE", "foo_value", "")
				Expect(err).NotTo(HaveOccurred())

				// 构建渲染变量：环境作用域变量 + 内置变量
				vars := buildTestVars(ctx, scopedEnvVarStore, app, appenv, appDepsVarReader, polarisVarReader)
				evaluated, err := applier.Evaluate(ctx, app, comp, appenv.ID, vars, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(evaluated.Patchers).To(Equal([]map[string]any{{
					"replicas":      3,
					"homepageTitle": "Welcome foo_value",
					"envName":       appenv.Name,
				}}))
				Expect(evaluated.Specs).To(BeEmpty())
			})
		})
	})

	Describe("AppPropertiesBuilder", func() {
		Context("Build", func() {
			It("No app properties, no env vars", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:       compDef.Name,
						Version:    compDef.Version,
						Properties: nil,
					},
				}
				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, nil, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(appProps["replicas"]).To(Equal(int64(1)))
			})
			It("with app properties, no env vars", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"replicas": int64(5),
						},
					},
				}
				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, nil, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(appProps["replicas"]).To(Equal(int64(5)))
			})
			It("should contain basic builtin props", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:       compDef.Name,
						Version:    compDef.Version,
						Properties: nil,
					},
				}

				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, nil, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(testutil.IsSuperMap(appProps, map[string]any{
					"bkmsAppName": app.Name,
					"bkmsEnvName": appenv.Name,
				})).To(BeTrue())
			})
			It("with env var placeholder in value", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"homepageTitle": "New title ${{ env.FOO_VAR }}",
						},
					},
				}
				_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *appenv, "FOO_VAR", "foo_value", "")
				Expect(err).NotTo(HaveOccurred())

				// 构建渲染变量：环境作用域变量 + 内置变量
				vars := buildTestVars(ctx, scopedEnvVarStore, app, appenv, appDepsVarReader, polarisVarReader)
				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, vars, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(appProps["homepageTitle"]).To(Equal("New title foo_value"))
			})
			It("raw function should preserve template syntax as-is", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"homepageTitle": `{{.bcs.pod_name}}`,
						},
					},
				}

				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, nil, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(appProps["homepageTitle"]).To(Equal("{{.bcs.pod_name}}"))
			})
			It("should render both legacy and new template formats", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"homepageTitle": "${{env.FOO_VAR}} and ${{env.BKMS_APP_NAME}}",
						},
					},
				}
				_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *appenv, "FOO_VAR", "foo_value", "")
				Expect(err).NotTo(HaveOccurred())

				// 构建渲染变量：包含环境作用域变量和内置变量
				vars := buildTestVars(ctx, scopedEnvVarStore, app, appenv, appDepsVarReader, polarisVarReader)
				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, vars, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(appProps["homepageTitle"]).To(Equal("foo_value and " + app.Name))
			})
		})
		Context("Build with map type property", func() {
			BeforeEach(func() {
				compDef = dbfactory.CompDef(ctx, compDefStore, &dbfactory.ComponentDefOpts{
					Properties: []component.Property{
						{Name: "fruits", Type: "MAP", DefaultValue: map[string]any{"apple": "red"}},
					},
					Patchers: []string{"kinds: {{ .kinds }}\n"},
				})
			})

			It("should remain as map if already is", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:       compDef.Name,
						Version:    compDef.Version,
						Properties: nil,
					},
				}
				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, nil, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(appProps["fruits"]).To(Equal(map[string]any{"apple": "red"}))
			})
			It("string value should be unmarshaled to map", func() {
				comp := component.Component{
					Name: "foobar",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"fruits": `{"banana": "yellow"}`,
						},
					},
				}
				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, nil, nil)

				Expect(err).NotTo(HaveOccurred())
				// The string value should be unmarshaled to map
				Expect(appProps["fruits"]).To(Equal(map[string]any{"banana": "yellow"}))
			})
		})

		Context("Build with map type property containing env var placeholders", func() {
			BeforeEach(func() {
				compDef = dbfactory.CompDef(ctx, compDefStore, &dbfactory.ComponentDefOpts{
					Properties: []component.Property{
						{Name: "serviceLabels", Type: "MAP", DefaultValue: map[string]any{}},
					},
					Patchers: []string{
						"labels:\n{{- range $key, $value := .serviceLabels }}\n  {{ $key }}: {{ $value }}\n{{- end }}\n",
					},
				})
			})

			It("should render env var placeholders in map values", func() {
				// 添加环境变量
				_, err := scopedEnvVarStore.CreateSimpleEnvScopeVar(ctx, *appenv, "set_env", "my-env-value", "")
				Expect(err).NotTo(HaveOccurred())

				serviceLabels := map[string]any{
					"static_key":  "static_value",
					"dynamic_key": "${{env.set_env}}",
				}
				serviceLabelsJSON, _ := json.Marshal(serviceLabels)
				comp := component.Component{
					Name: "test-labels",
					ComponentInst: component.ComponentInst{
						Type:    compDef.Name,
						Version: compDef.Version,
						Properties: map[string]any{
							"serviceLabels": string(serviceLabelsJSON),
						},
					},
				}

				// 构建渲染变量：环境作用域变量 + 内置变量
				vars := buildTestVars(ctx, scopedEnvVarStore, app, appenv, appDepsVarReader, polarisVarReader)
				appProps, err := propsBuilder.Build(ctx, app, comp, appenv.ID, vars, nil)
				Expect(err).NotTo(HaveOccurred())

				labelsMap, ok := appProps["serviceLabels"].(map[string]any)
				Expect(ok).To(BeTrue(), "serviceLabels should be map[string]any")
				Expect(labelsMap["static_key"]).To(Equal("static_value"))
				Expect(labelsMap["dynamic_key"]).To(Equal("my-env-value"), "Environment variable should be rendered")
			})
		})
	})
})

// buildTestVars 构建用于测试的渲染变量 map，包含环境作用域变量和内置变量。
func buildTestVars(
	ctx context.Context,
	scopedEnvVarStore envvars.ScopedEnvVarStore,
	app *bkmsapp.Application,
	appenv *envmodel.Environment,
	appDepsVarReader *depenvvars.Reader,
	polarisVarReader *polarisenvvars.Reader,
) map[string]string {
	reader := envvars.NewUnifiedEnvVarsReader(scopedEnvVarStore, appDepsVarReader, polarisVarReader)
	vars, err := reader.ListVars(ctx, *appenv, app, nil)
	Expect(err).NotTo(HaveOccurred())
	return vars.ToMap()
}
