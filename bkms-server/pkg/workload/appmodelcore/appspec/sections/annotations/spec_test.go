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

package annotations

import (
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ = Describe("Annotations Spec", func() {
	Describe("Clone", func() {
		It("returns nil for a nil spec", func() {
			Expect(Clone(nil)).To(BeNil())
		})

		It("collapses an empty map to nil", func() {
			Expect(Clone(&Spec{Annotations: map[string]string{}})).To(BeNil())
		})

		It("deep-copies the annotations map", func() {
			src := &Spec{Annotations: map[string]string{"desc": "my-app"}}
			cloned := Clone(src)
			Expect(cloned).NotTo(BeNil())
			Expect(cloned.Annotations).To(Equal(map[string]string{"desc": "my-app"}))

			src.Annotations["desc"] = "changed"
			Expect(cloned.Annotations["desc"]).To(Equal("my-app"))
		})
	})

	Describe("HasData", func() {
		It("is false for nil or empty", func() {
			Expect(HasData(nil)).To(BeFalse())
			Expect(HasData(&Spec{})).To(BeFalse())
			Expect(HasData(&Spec{Annotations: map[string]string{}})).To(BeFalse())
		})

		It("is true when annotations are present", func() {
			Expect(HasData(&Spec{Annotations: map[string]string{"a": "1"}})).To(BeTrue())
		})
	})

	Describe("Merge", func() {
		It("returns nil when both are nil", func() {
			Expect(Merge(nil, nil)).To(BeNil())
		})

		It("merges keys, with override winning on conflicts", func() {
			base := &Spec{Annotations: map[string]string{"a": "1", "b": "2"}}
			override := &Spec{Annotations: map[string]string{"b": "3", "c": "4"}}
			Expect(Merge(base, override)).To(Equal(&Spec{Annotations: map[string]string{
				"a": "1",
				"b": "3",
				"c": "4",
			}}))
		})
	})

	Describe("AppendPatch", func() {
		It("does nothing for a nil spec", func() {
			set := bson.D{}
			AppendPatch(&set, nil)
			Expect(set).To(BeEmpty())
		})

		It("appends annotations.annotations for a non-nil map", func() {
			set := bson.D{}
			AppendPatch(&set, &Spec{Annotations: map[string]string{"a": "1"}})
			Expect(set).To(HaveLen(1))
			Expect(set[0].Key).To(Equal("annotations.annotations"))
		})
	})

	Describe("Validation", func() {
		var v *validator.Validate

		BeforeEach(func() {
			v = validator.New(validator.WithRequiredStructEnabled())
			RegisterValidation(v)
		})

		It("passes for valid annotations", func() {
			spec := Spec{Annotations: map[string]string{"app.kubernetes.io/description": "anything goes here"}}
			Expect(v.Struct(spec)).NotTo(HaveOccurred())
		})

		It("passes for an empty map", func() {
			spec := Spec{Annotations: map[string]string{}}
			Expect(v.Struct(spec)).NotTo(HaveOccurred())
		})

		It("rejects empty key with friendly message", func() {
			err := v.Struct(Spec{Annotations: map[string]string{"": "value"}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is empty after trimming"))
		})

		It("rejects empty value with friendly message", func() {
			err := v.Struct(Spec{Annotations: map[string]string{"valid-key": "   "}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("value is empty after trimming"))
		})

		It("rejects invalid key with friendly message", func() {
			err := v.Struct(Spec{Annotations: map[string]string{"invalid key!": "v"}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid key!"))
			Expect(err.Error()).To(ContainSubstring("is invalid"))
		})

		It("rejects system-reserved keys with friendly message", func() {
			err := v.Struct(Spec{Annotations: map[string]string{
				"controller.kubernetes.io/pod-deletion-cost": "100",
			}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("reserved by the system"))
		})
	})
})
