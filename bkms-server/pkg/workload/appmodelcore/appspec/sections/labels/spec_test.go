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

package labels

import (
	"strings"

	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ = Describe("Labels Spec", func() {
	Describe("Clone", func() {
		It("returns nil for a nil spec", func() {
			Expect(Clone(nil)).To(BeNil())
		})

		It("collapses an empty map to nil", func() {
			Expect(Clone(&Spec{Labels: map[string]string{}})).To(BeNil())
		})

		It("deep-copies the labels map", func() {
			src := &Spec{Labels: map[string]string{"team": "sre"}}
			cloned := Clone(src)
			Expect(cloned).NotTo(BeNil())
			Expect(cloned.Labels).To(Equal(map[string]string{"team": "sre"}))

			// Mutating the source must not affect the clone.
			src.Labels["team"] = "dev"
			Expect(cloned.Labels["team"]).To(Equal("sre"))
		})
	})

	Describe("HasData", func() {
		It("is false for nil or empty", func() {
			Expect(HasData(nil)).To(BeFalse())
			Expect(HasData(&Spec{})).To(BeFalse())
			Expect(HasData(&Spec{Labels: map[string]string{}})).To(BeFalse())
		})

		It("is true when labels are present", func() {
			Expect(HasData(&Spec{Labels: map[string]string{"a": "1"}})).To(BeTrue())
		})
	})

	Describe("Merge", func() {
		It("returns nil when both are nil", func() {
			Expect(Merge(nil, nil)).To(BeNil())
		})

		It("returns a clone of override when base is nil", func() {
			Expect(Merge(nil, &Spec{Labels: map[string]string{"a": "1"}})).
				To(Equal(&Spec{Labels: map[string]string{"a": "1"}}))
		})

		It("returns a clone of base when override is nil", func() {
			Expect(Merge(&Spec{Labels: map[string]string{"a": "1"}}, nil)).
				To(Equal(&Spec{Labels: map[string]string{"a": "1"}}))
		})

		It("merges keys, with override winning on conflicts", func() {
			base := &Spec{Labels: map[string]string{"a": "1", "b": "2"}}
			override := &Spec{Labels: map[string]string{"b": "3", "c": "4"}}
			Expect(Merge(base, override)).To(Equal(&Spec{Labels: map[string]string{
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

		It("appends labels.labels for a non-nil map", func() {
			set := bson.D{}
			AppendPatch(&set, &Spec{Labels: map[string]string{"a": "1"}})
			Expect(set).To(HaveLen(1))
			Expect(set[0].Key).To(Equal("labels.labels"))
		})
	})

	Describe("Validation", func() {
		var v *validator.Validate

		BeforeEach(func() {
			v = validator.New(validator.WithRequiredStructEnabled())
			RegisterValidation(v)
		})

		It("passes for valid labels", func() {
			spec := Spec{Labels: map[string]string{"team": "sre", "app.kubernetes.io/version": "v1"}}
			Expect(v.Struct(spec)).NotTo(HaveOccurred())
		})

		It("passes for an empty map", func() {
			spec := Spec{Labels: map[string]string{}}
			Expect(v.Struct(spec)).NotTo(HaveOccurred())
		})

		It("rejects empty key with friendly message", func() {
			err := v.Struct(Spec{Labels: map[string]string{"": "value"}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is empty after trimming"))
		})

		It("rejects empty value with friendly message", func() {
			err := v.Struct(Spec{Labels: map[string]string{"key": "   "}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("value is empty after trimming"))
		})

		It("rejects invalid key with friendly message", func() {
			err := v.Struct(Spec{Labels: map[string]string{"invalid key!": "v"}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid key!"))
			Expect(err.Error()).To(ContainSubstring("is invalid"))
		})

		It("rejects invalid value with friendly message", func() {
			longValue := strings.Repeat("x", 64) // >63 chars
			err := v.Struct(Spec{Labels: map[string]string{"key": longValue}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("value"))
			Expect(err.Error()).To(ContainSubstring("is invalid"))
		})

		It("rejects system-reserved keys with friendly message", func() {
			err := v.Struct(Spec{Labels: map[string]string{"app.kubernetes.io/name": "myapp"}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("reserved by the system"))
		})
	})
})
