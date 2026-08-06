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

package serializer_test

import (
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
)

var _ = Describe("Workspace component serializers", func() {
	It("builds a workspace component model from create input", func() {
		version := "v2.0.0"
		name := "import-polaris"
		input := serializer.CreateWorkspaceComponentInput{
			CompName:      &name,
			Type:          "VolumeSecret",
			Version:       &version,
			Properties:    map[string]any{"servicePort": float64(8080)},
			ScopeType:     "environment",
			ScopeEnvNames: []string{"dev"},
		}

		model := input.ToModel("workspace-1")

		Expect(model).To(Equal(&workspace.Component{
			ComponentInst: component.ComponentInst{
				Type:       "VolumeSecret",
				Version:    "v2.0.0",
				Properties: map[string]any{"servicePort": float64(8080)},
			},
			Name:          "import-polaris",
			WorkspaceID:   "workspace-1",
			ScopeType:     component.ScopeTypeEnvironment,
			ScopeEnvNames: []string{"dev"},
		}))
	})

	It("defaults component version when create input omits it", func() {
		input := serializer.CreateWorkspaceComponentInput{
			Type:      "VolumeSecret",
			ScopeType: "global",
		}

		model := input.ToModel("workspace-1")

		Expect(model.Version).To(Equal(component.DefaultComponentDefVersion))
		Expect(model.ScopeType).To(Equal(component.ScopeTypeGlobal))
	})

	It("builds update data from patch input", func() {
		name := "renamed-comp"
		scopeType := "global"
		input := serializer.PatchWorkspaceComponentInput{
			Name:          &name,
			Properties:    map[string]any{"weight": float64(100)},
			ScopeType:     &scopeType,
			ScopeEnvNames: []string{"dev"},
		}

		updateData := input.ToModel()

		Expect(updateData.Name).To(Equal(&name))
		Expect(updateData.Properties).To(Equal(map[string]any{"weight": float64(100)}))
		Expect(updateData.ScopeType).NotTo(BeNil())
		Expect(*updateData.ScopeType).To(Equal(component.ScopeTypeGlobal))
		Expect(updateData.ScopeEnvNames).To(Equal([]string{"dev"}))
	})

	It("maps workspace component output from model", func() {
		createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		model := &workspace.Component{
			ComponentInst: component.ComponentInst{
				Type:       "VolumeSecret",
				Version:    "v1.0.0",
				Properties: map[string]any{"key": "value"},
			},
			Name:          "import-polaris",
			WorkspaceID:   "workspace-1",
			ScopeType:     component.ScopeTypeEnvironment,
			ScopeEnvNames: []string{"dev"},
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		}

		output := new(serializer.WorkspaceComponentOutputObj).FromModel(model, []string{"app-1"})

		Expect(output).To(Equal(&serializer.WorkspaceComponentOutputObj{
			Name:          "import-polaris",
			WorkspaceID:   "workspace-1",
			Type:          "VolumeSecret",
			Version:       "v1.0.0",
			Properties:    map[string]any{"key": "value"},
			ScopeType:     "environment",
			ScopeEnvNames: []string{"dev"},
			RefAppIDs:     []string{"app-1"},
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		}))
	})

	DescribeTable(
		"validates create input fields",
		func(input serializer.CreateWorkspaceComponentInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}

			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid global scope", serializer.CreateWorkspaceComponentInput{
			Type:      "VolumeSecret",
			ScopeType: "global",
		}, nil),
		Entry("missing type", serializer.CreateWorkspaceComponentInput{
			ScopeType: "global",
		}, []string{
			"CreateWorkspaceComponentInput.Type",
			"failed on the 'required' tag",
		}),
		Entry("invalid scope type", serializer.CreateWorkspaceComponentInput{
			Type:      "VolumeSecret",
			ScopeType: "workspace",
		}, []string{
			"CreateWorkspaceComponentInput.ScopeType",
			"failed on the 'oneof' tag",
		}),
		Entry("too long component name", serializer.CreateWorkspaceComponentInput{
			CompName:  ptr("import-polaris-name-x"),
			Type:      "VolumeSecret",
			ScopeType: "global",
		}, []string{
			"CreateWorkspaceComponentInput.CompName",
			"failed on the 'component_name' tag",
		}),
	)
})

func ptr[T any](v T) *T {
	return &v
}
