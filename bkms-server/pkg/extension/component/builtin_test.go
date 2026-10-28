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
	"fmt"
	"os"
	"path/filepath"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Builtin ComponentDefs Tests", func() {
	// The path containing test component definition files
	const testCompsPath = "./assets/testcomps"
	var compDefStore *component.ComponentDefStore
	var ctx context.Context

	BeforeEach(func() {
		compDefStore = testutil.NewStore(GinkgoT(), component.ComponentDefMeta)
		ctx = testutil.WithTenantOrDefault(context.Background())
	})

	Context("LoadBuiltinFromFolder valid input", func() {
		It("should load valid components successfully", func() {
			err := component.LoadBuiltinFromFolder(ctx, compDefStore, filepath.Join(testCompsPath, "valid"))
			Expect(err).To(Not(HaveOccurred()))

			// Get the loaded component and verify its content
			compDef, err := compDefStore.Get(ctx, component.ByNameVersion("MemLimits", "v1.0.0"))
			Expect(err).To(Not(HaveOccurred()))
			Expect(compDef.Name).To(Equal("MemLimits"))
			Expect(len(compDef.Properties)).To(Equal(2))
		})

		It("should mark loaded components as builtin and share them with other tenants", func() {
			// 测试文件中没有写 isBuiltin
			Expect(component.LoadBuiltinFromFolder(ctx, compDefStore, filepath.Join(testCompsPath, "valid"))).
				To(Succeed())

			ctxOther := tenant.WithTenantID(context.Background(), "tenant-builtin-test")
			compDef, err := compDefStore.Get(ctxOther, component.ByNameVersion("MemLimits", "v1.0.0"))
			Expect(err).NotTo(HaveOccurred())
			Expect(compDef.IsBuiltin).To(BeTrue())
		})

		It("should load VolumeConfigMap and preserve newlines in data", func() {
			err := component.LoadBuiltinFromFolder(
				ctx,
				compDefStore,
				"./assets/comps/VolumeConfigMap_v1.0.0.yaml",
			)
			Expect(err).NotTo(HaveOccurred())

			compDef, err := compDefStore.Get(ctx, component.ByNameVersion("VolumeConfigMap", "v1.0.0"))
			Expect(err).NotTo(HaveOccurred())
			Expect(compDef.Specs).To(HaveLen(1))
			Expect(component.ValidateFragmentTemplate(compDef.Specs[0])).To(Succeed())

			rendered, renderErr := render.RenderGoTemplate(compDef.Specs[0], map[string]any{
				"name": "app-mycm",
				"data": "line1\nline2",
			})
			Expect(renderErr).NotTo(HaveOccurred())

			var payload map[string]any
			Expect(yaml.Unmarshal([]byte(rendered), &payload)).To(Succeed())
			data, ok := payload["data"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(data["app-mycm"]).To(Equal("line1\nline2"))
		})

		It("should load and render ImportPolaris fragments", func() {
			err := component.LoadBuiltinFromFolder(
				ctx,
				compDefStore,
				"./assets/comps/ImportPolaris_v1.0.0.yaml",
			)
			Expect(err).NotTo(HaveOccurred())

			compDef, err := compDefStore.Get(ctx, component.ByNameVersion("ImportPolaris", "v1.0.0"))
			Expect(err).NotTo(HaveOccurred())
			Expect(compDef.Patchers).To(HaveLen(1))
			Expect(compDef.Specs).To(HaveLen(2))

			props := map[string]any{
				"bkmsContainerName": "main",
				"instanceKey":       "main",
				"polarisToken":      "token",
				"servicePort":       8080,
				"name":              "demo",
				"polarisName":       "demo",
				"polarisNamespace":  "Test",
				"bkmsEnvNamespace":  "default",
				"direct":            false,
				"keepNotReadyPod":   true,
				"enableHealthCheck": false,
				"weight":            10,
				"serviceLabels":     map[string]any{"env": "test"},
				"bkmsAppName":       "demo",
			}
			for _, fragment := range append(compDef.Patchers, compDef.Specs...) {
				rendered, renderErr := render.RenderGoTemplate(fragment, props)
				Expect(renderErr).NotTo(HaveOccurred())
				var payload map[string]any
				Expect(yaml.Unmarshal([]byte(rendered), &payload)).To(Succeed())
				Expect(payload).NotTo(BeEmpty())
			}
		})
	})
	Context("LoadBuiltinFromFolder in the platform tenant", func() {
		var platform, name, path string

		writeDef := func(description string) {
			content := fmt.Sprintf("name: %s\nversion: %s\ndescription: %s\npatchers:\n- 'foo: bar'\n",
				name, component.DefaultComponentDefVersion, description)
			Expect(os.WriteFile(path, []byte(content), 0o600)).To(Succeed())
		}
		getDef := func() *component.ComponentDef {
			compDef, err := compDefStore.Get(ctx, component.ByNameVersion(name, component.DefaultComponentDefVersion),
				store.TenantOf(platform))
			Expect(err).NotTo(HaveOccurred())
			return compDef
		}

		BeforeEach(func() {
			var err error
			platform, err = tenant.PlatformTenantID()
			Expect(err).NotTo(HaveOccurred())
			name = "Builtin-" + stringx.Random(6)
			path = filepath.Join(GinkgoT().TempDir(), name+".yaml")
			DeferCleanup(func() {
				Expect(
					compDefStore.DeleteIfExists(
						ctx,
						component.ByNameVersion(name, component.DefaultComponentDefVersion),
						store.TenantAll(),
					),
				).To(Succeed())
			})
		})

		It("should update an existing builtin definition on reload", func() {
			writeDef("v1")
			Expect(component.LoadBuiltinFromFolder(ctx, compDefStore, path)).To(Succeed())
			writeDef("v2")
			Expect(component.LoadBuiltinFromFolder(ctx, compDefStore, path)).To(Succeed())

			compDef := getDef()
			Expect(compDef.Description).To(Equal("v2"))
			Expect(compDef.IsBuiltin).To(BeTrue())
		})

		DescribeTable("should reject a custom definition of the same name in any tenant",
			func(tenantOf func() string) {
				custom := &component.ComponentDef{
					Name:        name,
					Version:     component.DefaultComponentDefVersion,
					Description: "custom",
					Patchers:    []string{"foo: custom\n"},
					ScopeType:   component.ScopeTypeGlobal,
				}
				customTenant := tenantOf()
				Expect(compDefStore.Create(ctx, custom, store.TenantOf(customTenant))).To(Succeed())

				writeDef("builtin")
				err := component.LoadBuiltinFromFolder(ctx, compDefStore, path)
				Expect(err).To(MatchError(ContainSubstring(customTenant + "/" + custom.Key())))

				defs, err := compDefStore.List(
					ctx,
					component.ByNameVersion(name, component.DefaultComponentDefVersion),
					store.TenantAll(),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(defs).To(HaveLen(1))
				Expect(defs[0].Description).To(Equal("custom"))
				Expect(defs[0].IsBuiltin).To(BeFalse())
			},
			Entry("the platform tenant", func() string { return platform }),
			Entry("another tenant", func() string { return "tenant-" + stringx.Random(6) }),
		)
	})

	Context("LoadBuiltinFromFolder invalid input", func() {
		It("should fail with bad properties", func() {
			err := component.LoadBuiltinFromFolder(
				ctx,
				compDefStore,
				filepath.Join(testCompsPath, "broken/BadProp_v1.0.0.yaml"),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid property type"))
		})
		It("should fail with an invalid fragment", func() {
			err := component.LoadBuiltinFromFolder(
				ctx,
				compDefStore,
				filepath.Join(testCompsPath, "broken/BadPatcher_v1.0.0.yaml"),
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is not a valid YAML mapping template"))
		})
	})
})
