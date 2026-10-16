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

package path_test

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var _ = Describe("Field path values", func() {
	var doc *pathDoc
	docFields := newFields[pathDoc]()

	BeforeEach(func() {
		doc = &pathDoc{}
	})

	set := func(field string, val any) error {
		return mustResolve(docFields, field).Set(reflect.ValueOf(doc).Elem(), val)
	}

	Describe("Set", func() {
		It("sets a top-level field", func() {
			Expect(set("Title", "demo")).To(Succeed())
			Expect(doc.Title).To(Equal("demo"))
		})

		It("sets a field of an inline struct", func() {
			Expect(set("owner", "alice")).To(Succeed())
			Expect(doc.Owner).To(Equal("alice"))
		})

		It("sets a nested field without touching its siblings", func() {
			doc.Spec = pathSpec{Color: "red", Size: 2}
			Expect(set("spec.size", 3)).To(Succeed())
			Expect(doc.Spec).To(Equal(pathSpec{Color: "red", Size: 3}))
		})

		It("allocates nil pointers on the way", func() {
			Expect(set("specPtr.color", "blue")).To(Succeed())
			Expect(doc.SpecPtr).To(Equal(&pathSpec{Color: "blue"}))
		})

		It("creates a nil map and adds the entry", func() {
			Expect(set("labels.env", "prod")).To(Succeed())
			Expect(doc.Labels).To(Equal(map[string]string{"env": "prod"}))
		})

		It("updates a field of a struct inside a map and keeps its other fields", func() {
			doc.Specs = map[string]pathSpec{"prod": {Color: "red"}, "dev": {Color: "gray"}}
			Expect(set("specs.prod.size", 3)).To(Succeed())
			Expect(doc.Specs).To(Equal(map[string]pathSpec{"prod": {Color: "red", Size: 3}, "dev": {Color: "gray"}}))
		})

		It("creates the struct inside a map when the entry is absent", func() {
			Expect(set("specs.prod.color", "red")).To(Succeed())
			Expect(doc.Specs).To(Equal(map[string]pathSpec{"prod": {Color: "red"}}))
		})

		It("deletes a map entry for nil", func() {
			doc.Specs = map[string]pathSpec{"prod": {Color: "red"}, "dev": {Color: "gray"}}
			Expect(set("specs.prod", nil)).To(Succeed())
			Expect(doc.Specs).To(Equal(map[string]pathSpec{"dev": {Color: "gray"}}))
		})

		It("sets the zero value for nil on a field inside a map entry", func() {
			doc.Specs = map[string]pathSpec{"prod": {Color: "red", Size: 1}}
			Expect(set("specs.prod.color", nil)).To(Succeed())
			Expect(doc.Specs).To(Equal(map[string]pathSpec{"prod": {Size: 1}}))
		})

		It("returns the conversion error with the target type", func() {
			err := set("title", []int{1})
			Expect(err).To(MatchError(ContainSubstring("convert []int to string")))
		})

		It("rejects a path that is not assignable", func() {
			err := set("items.0.name", "x")
			Expect(err).To(MatchError(ContainSubstring("is not assignable")))
			Expect(doc.Items).To(BeNil())
		})
	})

	Describe("value conversion", func() {
		It("sets a value of the same type directly", func() {
			Expect(set("tags", []string{"a"})).To(Succeed())
			Expect(doc.Tags).To(Equal([]string{"a"}))
		})

		It("sets the zero value for nil", func() {
			doc.Enabled, doc.Tags = lo.ToPtr(true), []string{"a"}
			Expect(set("enabled", nil)).To(Succeed())
			Expect(set("tags", nil)).To(Succeed())
			Expect(doc.Enabled).To(BeNil())
			Expect(doc.Tags).To(BeNil())
		})

		It("wraps the element value for a pointer field", func() {
			Expect(set("enabled", false)).To(Succeed())
			Expect(doc.Enabled).To(Equal(lo.ToPtr(false)))
		})

		It("converts a scalar to a named type with the same kind", func() {
			Expect(set("spec.color", "green")).To(Succeed())
			Expect(doc.Spec.Color).To(Equal(pathColor("green")))
		})

		It("converts between integer kinds through bson", func() {
			Expect(set("spec.size", int64(5))).To(Succeed())
			Expect(doc.Spec.Size).To(Equal(5))
		})

		It("converts a generic array to a typed slice through bson", func() {
			Expect(set("tags", bson.A{"x", "y"})).To(Succeed())
			Expect(doc.Tags).To(Equal([]string{"x", "y"}))
		})

		It("converts a map to a struct through bson", func() {
			Expect(set("spec", bson.M{"color": "red", "size": 2})).To(Succeed())
			Expect(doc.Spec).To(Equal(pathSpec{Color: "red", Size: 2}))
		})

		It("keeps a map assigned to an interface field as is", func() {
			Expect(set("extra", map[string]any{"a": 1})).To(Succeed())
			Expect(doc.Extra).To(Equal(map[string]any{"a": 1}))
		})
	})
})
