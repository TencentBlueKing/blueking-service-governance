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
	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("App component serializers", func() {
	It("builds an instantiated app component model from create input", func() {
		version := "v2.0.0"
		name := "resource-limits"
		input := serializer.CreateAppComponentInput{
			CompName:   &name,
			Type:       "ResourceLimits",
			Version:    &version,
			Properties: map[string]any{"cpu": "500m"},
		}

		model := input.ToModel()

		Expect(model).To(Equal(&component.Component{
			Name: "resource-limits",
			ComponentInst: component.ComponentInst{
				Type:       "ResourceLimits",
				Version:    "v2.0.0",
				Properties: map[string]any{"cpu": "500m"},
			},
		}))
	})

	It("builds a referenced app component model from create input", func() {
		name := "shared-ref"
		refName := "workspace-comp"
		input := serializer.CreateAppComponentInput{
			CompName:             &name,
			RefWorkspaceCompName: &refName,
			Type:                 "Ignored",
			Properties:           map[string]any{"ignored": true},
		}

		model := input.ToModel()

		Expect(model).To(Equal(&component.Component{
			Name:         "shared-ref",
			ComponentRef: component.ComponentRef{RefWorkspaceCompName: "workspace-comp"},
		}))
	})

	It("defaults component version for instantiated app components", func() {
		input := serializer.CreateAppComponentInput{Type: "ResourceLimits"}

		model := input.ToModel()

		Expect(model.Version).To(Equal(component.DefaultComponentDefVersion))
	})

	It("builds app component update data from patch input", func() {
		name := "renamed-component"
		input := serializer.PatchAppComponentInput{
			Name:       &name,
			Properties: map[string]any{"memory": "512Mi"},
		}

		updateData := input.ToModel()

		Expect(updateData).To(Equal(&appmodel.ComponentUpdateData{
			Name:       &name,
			Properties: map[string]any{"memory": "512Mi"},
		}))
	})

	DescribeTable(
		"validates create input fields",
		func(input serializer.CreateAppComponentInput, expectedErrSubstrings []string) {
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
		Entry("valid instantiated component", serializer.CreateAppComponentInput{
			Type: "ResourceLimits",
		}, nil),
		Entry("valid referenced component", serializer.CreateAppComponentInput{
			RefWorkspaceCompName: ptr("workspace-comp"),
		}, nil),
		Entry("missing type when not referenced", serializer.CreateAppComponentInput{}, []string{
			"CreateAppComponentInput.Type",
			"failed on the 'required' tag",
		}),
		Entry("invalid component name", serializer.CreateAppComponentInput{
			CompName: ptr("Invalid"),
			Type:     "ResourceLimits",
		}, []string{
			"CreateAppComponentInput.CompName",
			"failed on the 'component_name' tag",
		}),
		Entry("too long component name", serializer.CreateAppComponentInput{
			CompName: ptr("resource-limits-name-x"),
			Type:     "ResourceLimits",
		}, []string{
			"CreateAppComponentInput.CompName",
			"failed on the 'component_name' tag",
		}),
	)
})

func ptr[T any](v T) *T {
	return &v
}
