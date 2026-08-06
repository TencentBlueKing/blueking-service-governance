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

package component

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ComponentDef validation", func() {
	var cd ComponentDef

	BeforeEach(func() {
		cd = ComponentDef{
			Name:       "demo",
			Version:    "v1.0.0",
			Properties: nil,
			Patchers:   []string{"k: v\n"},
		}
	})

	It("accepts a minimal valid component def", func() {
		Expect(ValidateComponentDef(&cd)).To(Succeed())
	})

	Context("required fields", func() {
		It("rejects when Name is empty", func() {
			cd.Name = ""
			Expect(ValidateComponentDef(&cd)).To(HaveOccurred())
		})

		It("rejects when Version is empty", func() {
			cd.Version = ""
			Expect(ValidateComponentDef(&cd)).To(HaveOccurred())
		})
	})

	Context("prop_type", func() {
		It("rejects invalid property type", func() {
			cd.Properties = []Property{{Name: "p", Type: "NOT_A_TYPE"}}
			err := ValidateComponentDef(&cd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid property type"))
		})
	})

	Context("component fragments", func() {
		It("rejects empty patcher and spec arrays", func() {
			cd.Patchers = []string{}
			cd.Specs = []string{}
			err := ValidateComponentDef(&cd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one patcher or spec"))
		})

		It("accepts a patcher without specs", func() {
			cd.Patchers = []string{"spec: {}\n"}
			cd.Specs = nil
			Expect(ValidateComponentDef(&cd)).To(Succeed())
		})

		It("rejects empty fragments", func() {
			cd.Patchers = []string{""}
			err := ValidateComponentDef(&cd)
			Expect(err).To(HaveOccurred())
		})

		It("rejects a non-mapping fragment", func() {
			cd.Patchers = []string{"- only\n- root\n"}
			err := ValidateComponentDef(&cd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("YAML mapping"))
		})

		It("accepts specs without patchers", func() {
			cd.Patchers = nil
			cd.Specs = []string{"apiVersion: v1\nkind: Pod\n"}
			Expect(ValidateComponentDef(&cd)).To(Succeed())
		})

		It("rejects invalid template syntax", func() {
			cd.Patchers = []string{"{{ unclosed"}
			Expect(ValidateComponentDef(&cd)).To(HaveOccurred())
		})

		It("accepts a template that renders to a mapping", func() {
			cd.Properties = []Property{{Name: "replicas", Type: PropTypeInt}}
			cd.Patchers = []string{"replicas: {{ .replicas }}\n"}
			Expect(ValidateComponentDef(&cd)).To(Succeed())
		})
	})

	Context("SELECT property", func() {
		It("requires options when type is SELECT", func() {
			cd.Properties = []Property{{
				Name: "mode",
				Type: PropTypeSelect,
			}}
			err := ValidateComponentDef(&cd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("non-empty options"))
		})

		It("rejects option with empty value", func() {
			cd.Properties = []Property{{
				Name:    "mode",
				Type:    PropTypeSelect,
				Options: []PropertyOption{{Label: "A", Value: ""}},
			}}
			Expect(ValidateComponentDef(&cd)).To(HaveOccurred())
		})

		It("rejects option with empty label", func() {
			cd.Properties = []Property{{
				Name:    "mode",
				Type:    PropTypeSelect,
				Options: []PropertyOption{{Label: "", Value: "a"}},
			}}
			Expect(ValidateComponentDef(&cd)).To(HaveOccurred())
		})

		It("rejects non-string defaultValue", func() {
			cd.Properties = []Property{{
				Name:         "mode",
				Type:         PropTypeSelect,
				Options:      []PropertyOption{{Label: "A", Value: "a"}},
				DefaultValue: int64(1),
			}}
			err := ValidateComponentDef(&cd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be string"))
		})

		It("rejects defaultValue not in options", func() {
			cd.Properties = []Property{{
				Name:         "mode",
				Type:         PropTypeSelect,
				Options:      []PropertyOption{{Label: "A", Value: "a"}},
				DefaultValue: "b",
			}}
			err := ValidateComponentDef(&cd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("one of options values"))
		})

		It("accepts SELECT with default in options", func() {
			cd.Properties = []Property{{
				Name:         "mode",
				Type:         PropTypeSelect,
				Options:      []PropertyOption{{Label: "A", Value: "a"}, {Label: "B", Value: "b"}},
				DefaultValue: "b",
			}}
			Expect(ValidateComponentDef(&cd)).To(Succeed())
		})

		It("accepts SELECT with nil default", func() {
			cd.Properties = []Property{{
				Name:    "mode",
				Type:    PropTypeSelect,
				Options: []PropertyOption{{Label: "A", Value: "a"}},
			}}
			Expect(ValidateComponentDef(&cd)).To(Succeed())
		})

		It("accepts SELECT with empty string default", func() {
			cd.Properties = []Property{{
				Name:         "mode",
				Type:         PropTypeSelect,
				Options:      []PropertyOption{{Label: "A", Value: "a"}},
				DefaultValue: "",
			}}
			Expect(ValidateComponentDef(&cd)).To(Succeed())
		})
	})
})

var _ = Describe("formatValidationError", func() {
	It("returns the original error when not validator.ValidationErrors", func() {
		err := errors.New("plain")
		Expect(formatValidationError(err)).To(Equal(err))
	})
})
