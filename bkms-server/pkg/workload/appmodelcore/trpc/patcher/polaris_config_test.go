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

package patcher_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc/patcher"
)

var _ = Describe("PolarisRegistryPatcher", func() {
	var (
		ctx                 context.Context
		diApp               *fxtest.App
		appStore            bkmsapp.ApplicationStore
		envService          *env.EnvService
		store               polaris.PolarisConfigStore
		app                 *bkmsapp.Application
		environment         *bkmsenv.Environment
		otherEnvironment    *bkmsenv.Environment
		unscopedEnvironment *bkmsenv.Environment
		p                   *patcher.PolarisRegistryPatcher
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			env.FxModule,
			polaris.FxModule,
			fx.Populate(&appStore, &envService, &store),
		)
		diApp.RequireStart()

		app = dbfactory.Application(ctx, appStore)
		environment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		otherEnvironment = dbfactory.Env(ctx, envService, app.WorkspaceID)
		unscopedEnvironment = dbfactory.Env(ctx, envService, app.WorkspaceID)

		p = patcher.NewPolarisRegistryPatcher(store)
	})

	AfterEach(func() {
		_ = store.DeleteByApp(ctx, app.ID)
		diApp.RequireStop()
	})

	Context("when there are no polaris configs", func() {
		It("should return content as-is", func() {
			result, err := p.Patch(ctx, app.ID, environment.Name, "server:\n  app: test")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("server:\n  app: test"))
		})
	})

	Context("when all configs have health check disabled", func() {
		BeforeEach(func() {
			config := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-no-hc",
					PolarisName:       "svc-no-hc",
					PolarisNamespace:  "ns-no-hc",
					PolarisToken:      "token-no-hc",
					ServicePort:       8080,
					EnableHealthCheck: false,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{environment.Name},
			}
			err := store.Create(ctx, config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return content as-is", func() {
			result, err := p.Patch(ctx, app.ID, environment.Name, "server:\n  app: test")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("server:\n  app: test"))
		})
	})

	Context("when there are configs with health check enabled", func() {
		BeforeEach(func() {
			config := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-hc",
					PolarisName:       "svc-a",
					PolarisNamespace:  "ns-a",
					PolarisToken:      "token-a",
					ServicePort:       8080,
					EnableHealthCheck: true,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{environment.Name},
			}
			err := store.Create(ctx, config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should build complete registry YAML when content is empty", func() {
			result, err := p.Patch(ctx, app.ID, environment.Name, "")
			Expect(err).NotTo(HaveOccurred())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(result), &configMap)).To(Succeed())

			// 验证结构: plugins.registry.polaris.service
			plugins := configMap["plugins"].(map[string]any)
			registry := plugins["registry"].(map[string]any)
			polarisMap := registry["polaris"].(map[string]any)
			serviceList := polarisMap["service"].([]any)
			Expect(serviceList).To(HaveLen(1))

			svc := serviceList[0].(map[string]any)
			Expect(svc["name"]).To(Equal("svc-a"))
			Expect(svc["namespace"]).To(Equal("ns-a"))
			Expect(svc["token"]).To(Equal("token-a"))
		})

		It("should inject when plugins.registry.polaris.service is absent", func() {
			content := "server:\n  app: test\nplugins:\n  log:\n    level: debug"
			result, err := p.Patch(ctx, app.ID, environment.Name, content)
			Expect(err).NotTo(HaveOccurred())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(result), &configMap)).To(Succeed())

			// 验证原有配置仍然存在
			server := configMap["server"].(map[string]any)
			Expect(server["app"]).To(Equal("test"))

			// 验证注入的配置
			plugins := configMap["plugins"].(map[string]any)
			registry := plugins["registry"].(map[string]any)
			polarisMap := registry["polaris"].(map[string]any)
			serviceList := polarisMap["service"].([]any)
			Expect(serviceList).To(HaveLen(1))

			// 验证原有的 log 配置仍然保留
			logMap := plugins["log"].(map[string]any)
			Expect(logMap["level"]).To(Equal("debug"))
		})

		It("should not overwrite when plugins.registry.polaris.service already exists", func() {
			content := `plugins:
  registry:
    polaris:
      service:
        - name: existing-svc
          namespace: existing-ns
          token: existing-token`
			result, err := p.Patch(ctx, app.ID, environment.Name, content)
			Expect(err).NotTo(HaveOccurred())
			// 应原样返回，不修改
			Expect(result).To(Equal(content))
		})

		It("should return error when YAML parsing fails", func() {
			invalidYAML := "invalid: yaml: content:\n  - [broken"
			_, err := p.Patch(ctx, app.ID, environment.Name, invalidYAML)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("parsing tRPC config YAML"))
		})

		It("should inject correctly when plugins exists but registry is absent", func() {
			content := "plugins:\n  log:\n    level: info"
			result, err := p.Patch(ctx, app.ID, environment.Name, content)
			Expect(err).NotTo(HaveOccurred())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(result), &configMap)).To(Succeed())

			plugins := configMap["plugins"].(map[string]any)
			// 原有 log 保留
			logMap := plugins["log"].(map[string]any)
			Expect(logMap["level"]).To(Equal("info"))
			// registry 被注入
			registry := plugins["registry"].(map[string]any)
			polarisMap := registry["polaris"].(map[string]any)
			serviceList := polarisMap["service"].([]any)
			Expect(serviceList).To(HaveLen(1))
		})

		It("should inject correctly when plugins.registry exists but polaris is absent", func() {
			content := "plugins:\n  registry:\n    consul:\n      address: localhost:8500"
			result, err := p.Patch(ctx, app.ID, environment.Name, content)
			Expect(err).NotTo(HaveOccurred())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(result), &configMap)).To(Succeed())

			plugins := configMap["plugins"].(map[string]any)
			registry := plugins["registry"].(map[string]any)
			// consul 保留
			consul := registry["consul"].(map[string]any)
			Expect(consul["address"]).To(Equal("localhost:8500"))
			// polaris 被注入
			polarisMap := registry["polaris"].(map[string]any)
			serviceList := polarisMap["service"].([]any)
			Expect(serviceList).To(HaveLen(1))
		})
	})

	Context("mixed multiple configs scenario", func() {
		BeforeEach(func() {
			// 启用健康检查的配置 A
			configA := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-a",
					PolarisName:       "svc-a",
					PolarisNamespace:  "ns-a",
					PolarisToken:      "token-a",
					ServicePort:       8080,
					EnableHealthCheck: true,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{environment.Name},
			}
			err := store.Create(ctx, configA)
			Expect(err).NotTo(HaveOccurred())

			// 未启用健康检查的配置 B
			configB := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-b",
					PolarisName:       "svc-b",
					PolarisNamespace:  "ns-b",
					PolarisToken:      "token-b",
					ServicePort:       8081,
					EnableHealthCheck: false,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{environment.Name},
			}
			err = store.Create(ctx, configB)
			Expect(err).NotTo(HaveOccurred())

			// 启用健康检查的配置 C
			configC := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-c",
					PolarisName:       "svc-c",
					PolarisNamespace:  "ns-c",
					PolarisToken:      "token-c",
					ServicePort:       8082,
					EnableHealthCheck: true,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{environment.Name},
			}
			err = store.Create(ctx, configC)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should only inject configs with health check enabled", func() {
			result, err := p.Patch(ctx, app.ID, environment.Name, "server:\n  app: test")
			Expect(err).NotTo(HaveOccurred())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(result), &configMap)).To(Succeed())

			plugins := configMap["plugins"].(map[string]any)
			registry := plugins["registry"].(map[string]any)
			polarisMap := registry["polaris"].(map[string]any)
			serviceList := polarisMap["service"].([]any)
			// svc-b 的 EnableHealthCheck 为 false，应被过滤
			Expect(serviceList).To(HaveLen(2))

			// 收集注入的 service name
			var names []string
			for _, svc := range serviceList {
				svcMap := svc.(map[string]any)
				names = append(names, svcMap["name"].(string))
			}
			Expect(names).To(ContainElements("svc-a", "svc-c"))
			Expect(names).NotTo(ContainElement("svc-b"))
		})
	})

	Context("environment scope filtering", func() {
		BeforeEach(func() {
			// 仅对当前环境生效的配置
			configEnv := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-env",
					PolarisName:       "svc-env",
					PolarisNamespace:  "ns-env",
					PolarisToken:      "token-env",
					ServicePort:       8080,
					EnableHealthCheck: true,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{environment.Name},
			}
			err := store.Create(ctx, configEnv)
			Expect(err).NotTo(HaveOccurred())

			// 仅对另一个环境生效的配置
			configProd := &polaris.PolarisConfig{
				AppID: app.ID,
				Properties: polaris.Properties{
					InstanceKey:       "inst-prod",
					PolarisName:       "svc-prod",
					PolarisNamespace:  "ns-prod",
					PolarisToken:      "token-prod",
					ServicePort:       8081,
					EnableHealthCheck: true,
				},
				ScopeType:     component.ScopeTypeEnvironment,
				ScopeEnvNames: []string{otherEnvironment.Name},
			}
			err = store.Create(ctx, configProd)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should only inject configs available for the current environment", func() {
			result, err := p.Patch(ctx, app.ID, environment.Name, "server:\n  app: test")
			Expect(err).NotTo(HaveOccurred())

			var configMap map[string]any
			Expect(yaml.Unmarshal([]byte(result), &configMap)).To(Succeed())

			plugins := configMap["plugins"].(map[string]any)
			registry := plugins["registry"].(map[string]any)
			polarisMap := registry["polaris"].(map[string]any)
			serviceList := polarisMap["service"].([]any)
			Expect(serviceList).To(HaveLen(1))

			svc := serviceList[0].(map[string]any)
			Expect(svc["name"]).To(Equal("svc-env"))
		})

		It("should return content as-is when no configs are available for the environment", func() {
			result, err := p.Patch(ctx, app.ID, unscopedEnvironment.Name, "server:\n  app: test")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("server:\n  app: test"))
		})
	})
})
