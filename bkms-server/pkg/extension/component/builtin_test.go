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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("Builtin ComponentDefs Tests", func() {
	// The path containing test component definition files
	const testCompsPath = "./assets/testcomps"
	var compDefStore component.ComponentDefStore
	var ctx context.Context
	var err error

	BeforeEach(func() {
		compDefStore, err = component.NewComponentDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
	})

	Context("LoadBuiltinFromFolder valid input", func() {
		It("should load valid components successfully", func() {
			err := component.LoadBuiltinFromFolder(ctx, compDefStore, filepath.Join(testCompsPath, "valid"))
			Expect(err).To(Not(HaveOccurred()))

			// Get the loaded component and verify its content
			compDef, err := compDefStore.Get(ctx, "MemLimits", "v1.0.0")
			Expect(err).To(Not(HaveOccurred()))
			Expect(compDef.Name).To(Equal("MemLimits"))
			Expect(len(compDef.Properties)).To(Equal(2))
		})

		It("should load and render ImportPolaris fragments", func() {
			err := component.LoadBuiltinFromFolder(
				ctx,
				compDefStore,
				"./assets/comps/ImportPolaris_v1.0.0.yaml",
			)
			Expect(err).NotTo(HaveOccurred())

			compDef, err := compDefStore.Get(ctx, "ImportPolaris", "v1.0.0")
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
