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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/serializer"
)

var _ = Describe("Component definition serializers", func() {
	It("builds a component definition model from create input", func() {
		input := serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			DisplayName: "Custom Component",
			Description: "description",
			Properties: []serializer.PropertyDefInput{{
				Name:         "replicas",
				Type:         "INT",
				DefaultValue: float64(2),
			}},
			Patchers:              []string{"spec:\n  replicas: 2\n"},
			ScopeType:             "workspace",
			ScopeWorkspaceIDs:     []string{"workspace-1"},
			ManagedByWorkspaceIDs: []string{"workspace-1"},
		}

		model := input.ToModel("creator")

		Expect(model).To(Equal(&component.ComponentDef{
			Name:        "custom-comp",
			Version:     component.DefaultComponentDefVersion,
			DisplayName: "Custom Component",
			Description: "description",
			Properties: []component.Property{{
				Name:         "replicas",
				Type:         component.PropTypeInt,
				Options:      []component.PropertyOption{},
				DefaultValue: float64(2),
			}},
			Patchers:              []string{"spec:\n  replicas: 2\n"},
			Specs:                 nil,
			ScopeType:             component.ScopeTypeWorkspace,
			ScopeWorkspaceIDs:     []string{"workspace-1"},
			IsBuiltin:             false,
			ManagedByWorkspaceIDs: []string{"workspace-1"},
			Creator:               "creator",
			Updater:               "creator",
		}))
	})

	It("builds patch data by applying input to an existing model", func() {
		existing := &component.ComponentDef{
			Name:        "custom-comp",
			Version:     component.DefaultComponentDefVersion,
			DisplayName: "Old Name",
			Patchers:    []string{"old: patcher\n"},
			Specs:       []string{"kind: ConfigMap\n"},
			ScopeType:   component.ScopeTypeGlobal,
		}
		input := serializer.PatchComponentDefInput{
			DisplayName:     ptr("New Name"),
			PropertiesInput: &serializer.ComponentDefPropertiesInput{Properties: []serializer.PropertyDefInput{}},
			Patchers:        ptr([]string{"new: patcher\n"}),
			Specs:           ptr([]string{}),
			ScopeType:       ptr("workspace"),
		}

		updated := input.ToModel(existing, "updater")

		Expect(updated.DisplayName).To(Equal("New Name"))
		Expect(updated.Properties).To(BeEmpty())
		Expect(updated.Patchers).To(Equal([]string{"new: patcher\n"}))
		Expect(updated.Specs).To(Equal([]string{}))
		Expect(updated.ScopeType).To(Equal(component.ScopeTypeWorkspace))
		Expect(updated.Updater).To(Equal("updater"))
	})

	It("allows clearing patchers when specs remain", func() {
		existing := &component.ComponentDef{
			Name: "custom-comp", Version: component.DefaultComponentDefVersion,
			Patchers: []string{"spec: {}\n"}, Specs: []string{"apiVersion: v1\nkind: ConfigMap\n"},
		}
		emptyPatchers := []string{}
		updated := (serializer.PatchComponentDefInput{Patchers: &emptyPatchers}).ToModel(existing, "updater")

		Expect(updated.Patchers).To(BeEmpty())
		Expect(updated.Specs).To(HaveLen(1))
	})

	It("maps component definition output from model", func() {
		createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		model := &component.ComponentDef{
			Name:                       "custom-comp",
			Version:                    "v1.0.0",
			DisplayName:                "Custom Component",
			Description:                "description",
			Properties:                 []component.Property{{Name: "replicas", Type: component.PropTypeInt}},
			Patchers:                   []string{"spec: {}\n"},
			Specs:                      []string{"apiVersion: v1\nkind: ConfigMap\n"},
			ScopeType:                  component.ScopeTypeGlobal,
			IsBuiltin:                  true,
			ManagedByWorkspaceIDs:      []string{"workspace-1"},
			Creator:                    "creator",
			CreatedAt:                  createdAt,
			Updater:                    "updater",
			UpdatedAt:                  updatedAt,
			AppCompInstanceCount:       1,
			WorkspaceCompInstanceCount: 2,
		}

		output := new(serializer.ComponentDefOutputObj).FromModel(model)

		Expect(output.Name).To(Equal("custom-comp"))
		Expect(output.Properties).To(Equal([]serializer.PropertyDefOutputObj{{
			Name:    "replicas",
			Type:    "INT",
			Options: []serializer.PropertyOptionOutputObj{},
		}}))
		Expect(output.Patchers).To(Equal(model.Patchers))
		Expect(output.Specs).To(Equal(model.Specs))
		Expect(output.AppCompInstanceCount).To(Equal(int32(1)))
		Expect(output.WorkspaceCompInstanceCount).To(Equal(int32(2)))
	})

	It("maps output template builtin vars from models", func() {
		output := new(serializer.ListBuiltinVarsOutput).FromModels(
			component.BuiltinVars,
		)

		Expect(output.Data).To(HaveLen(6))
		Expect(output.Data[0].Key).To(Equal(component.PropNameAppName))
		Expect(output.Data[0].Description).NotTo(BeEmpty())
		Expect(output.Data).To(ContainElement(
			WithTransform(func(item *serializer.BuiltinVarOutputObj) string {
				return item.Key
			}, Equal(component.PropNameName)),
		))
	})

	It("normalizes empty slices in component definition outputs", func() {
		model := &component.ComponentDef{
			Name:    "custom-comp",
			Version: component.DefaultComponentDefVersion,
			Properties: []component.Property{{
				Name: "replicas",
				Type: component.PropTypeInt,
			}},
			ScopeType: component.ScopeTypeGlobal,
		}

		output := new(serializer.ComponentDefOutputObj).FromModel(model)

		Expect(output.Properties).To(Equal([]serializer.PropertyDefOutputObj{{
			Name:    "replicas",
			Type:    "INT",
			Options: []serializer.PropertyOptionOutputObj{},
		}}))
		Expect(output.ScopeWorkspaceIDs).To(Equal([]string{}))
		Expect(output.ManagedByWorkspaceIDs).To(Equal([]string{}))
		Expect(output.Patchers).To(Equal([]string{}))
		Expect(output.Specs).To(Equal([]string{}))
	})

	It("preserves patcher and spec arrays", func() {
		input := serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			ScopeType:   "global",
			Patchers:    []string{"spec:\n  template: {}\n"},
			Specs:       []string{"apiVersion: v1\nkind: ConfigMap\n"},
		}

		model := input.ToModel("creator")

		Expect(model.Patchers).To(Equal(input.Patchers))
		Expect(model.Specs).To(Equal(input.Specs))
	})

	DescribeTable(
		"validates create input fields",
		func(input serializer.CreateComponentDefInput, expectedErrSubstrings []string) {
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
		Entry("valid input", serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			ScopeType:   "global",
			Patchers:    []string{"spec: {}\n"},
		}, nil),
		Entry("valid input with specs only", serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			ScopeType:   "global",
			Specs:       []string{"apiVersion: v1\nkind: ConfigMap\n"},
		}, nil),
		Entry("missing patchers and specs", serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			ScopeType:   "global",
		}, []string{"component_fragments_required"}),
		Entry("too long component definition name", serializer.CreateComponentDefInput{
			CompDefName: "custom-component-name",
			ScopeType:   "global",
			Patchers:    []string{},
			Specs:       []string{},
		}, []string{
			"CreateComponentDefInput.CompDefName",
			"failed on the 'component_def_name' tag",
		}),
		Entry("invalid property type", serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			ScopeType:   "global",
			Patchers:    []string{},
			Specs:       []string{},
			Properties:  []serializer.PropertyDefInput{{Name: "bad", Type: "UNKNOWN"}},
		}, []string{
			"CreateComponentDefInput.Properties[0].Type",
			"failed on the 'oneof' tag",
		}),
		Entry("invalid scope type", serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			ScopeType:   "environment",
			Patchers:    []string{},
			Specs:       []string{},
		}, []string{
			"CreateComponentDefInput.ScopeType",
			"failed on the 'oneof' tag",
		}),
		Entry("invalid patcher fragment", serializer.CreateComponentDefInput{
			CompDefName: "custom-comp",
			ScopeType:   "global",
			Patchers:    []string{"- invalid\n"},
			Specs:       []string{},
		}, []string{
			"CreateComponentDefInput.Patchers[0]",
			"failed on the 'component_fragment' tag",
		}),
	)
})

func ptr[T any](v T) *T {
	return &v
}
