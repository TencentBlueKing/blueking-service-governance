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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store/path"
)

type pathColor string

type pathSpec struct {
	Color pathColor `bson:"color"`
	Size  int       `bson:"size,omitempty"`
}

type PathBase struct {
	Owner string `bson:"owner"`
}

type pathItem struct {
	Name string `bson:"name"`
}

// pathDoc 覆盖字段路径解析需要处理的各种字段形态
type pathDoc struct {
	ID        string `bson:"_id"`
	PathBase  `bson:",inline"`
	Title     string              `bson:"title"`
	Enabled   *bool               `bson:"enabled"`
	Spec      pathSpec            `bson:"spec"`
	SpecPtr   *pathSpec           `bson:"specPtr"`
	Labels    map[string]string   `bson:"labels"`
	Specs     map[string]pathSpec `bson:"specs"`
	Items     []pathItem          `bson:"items"`
	Tags      []string            `bson:"tags"`
	Extra     any                 `bson:"extra"`
	Counts    map[int]int         `bson:"counts"`
	CreatedAt time.Time           `bson:"createdAt"`
	Ignored   string              `bson:"-"`
	NoTag     string
}

// pathAmbiguous 中 "Label" 既是 Title 的 bson key，又是 Label 的 Go 字段名
type pathAmbiguous struct {
	Title string `bson:"Label"`
	Label string `bson:"title"`
	Same  string `bson:"Same"`
}

// pathNested 的嵌套字段中存在歧义的名字
type pathNested struct {
	Inner pathAmbiguous `bson:"inner"`
}

// pathNode 通过指针、数组引用自身
type pathNode struct {
	Name     string     `bson:"name"`
	Parent   *pathNode  `bson:"parent"`
	Children []pathNode `bson:"children"`
}

func newFields[T any]() *path.SchemaFields {
	return lo.Must(path.New(reflect.TypeFor[T]()))
}

func mustResolve(fields *path.SchemaFields, p string) path.Path {
	resolved, err := fields.Resolve(p)
	Expect(err).NotTo(HaveOccurred())
	return resolved
}

var _ = Describe("SchemaFields", func() {
	docFields := newFields[pathDoc]()

	Describe("New", func() {
		It("rejects a non-struct model", func() {
			_, err := path.New(reflect.TypeFor[string]())
			Expect(err).To(MatchError(ContainSubstring("model must be a struct")))
		})

		It("handles self-referencing types", func() {
			fields := newFields[pathNode]()
			Expect(mustResolve(fields, "Children.Parent.Name").String()).To(Equal("children.parent.name"))
		})
	})

	Describe("Fields", func() {
		It("expands inline structs and skips fields excluded from encoding", func() {
			keys := lo.Map(docFields.Fields(), func(f path.Field, _ int) string { return f.Key })
			Expect(keys).To(Equal([]string{
				"_id", "owner", "title", "enabled", "spec", "specPtr", "labels", "specs",
				"items", "tags", "extra", "counts", "createdAt", "notag",
			}))
		})

		It("records the index path through the inline struct", func() {
			owner, found := lo.Find(docFields.Fields(), func(f path.Field) bool { return f.Key == "owner" })
			Expect(found).To(BeTrue())
			Expect(owner.Index).To(Equal([]int{1, 0}))
			Expect(owner.Name).To(Equal("Owner"))
		})
	})

	Describe("Lookup", func() {
		It("finds a top-level field by bson key or Go name", func() {
			byKey, err := docFields.Lookup("specPtr")
			Expect(err).NotTo(HaveOccurred())
			byName, err := docFields.Lookup("SpecPtr")
			Expect(err).NotTo(HaveOccurred())
			Expect(byName).To(Equal(byKey))
			Expect(byKey.Type).To(Equal(reflect.TypeFor[*pathSpec]()))
		})

		It("reports an unknown field", func() {
			_, err := docFields.Lookup("color")
			Expect(err).To(MatchError(path.ErrUnknownField))
		})
	})

	DescribeTable("Resolve",
		func(p, expected string, known, assignable bool) {
			resolved := mustResolve(docFields, p)
			Expect(resolved.String()).To(Equal(expected))
			Expect(resolved.Known).To(Equal(known), "known")
			Expect(resolved.Assignable).To(Equal(assignable), "assignable")
		},
		Entry("bson key", "title", "title", true, true),
		Entry("Go field name", "Title", "title", true, true),
		Entry("field without a bson tag by Go name", "NoTag", "notag", true, true),
		Entry("field without a bson tag by its default key", "notag", "notag", true, true),
		Entry("nested bson path", "spec.color", "spec.color", true, true),
		Entry("nested Go path", "Spec.Color", "spec.color", true, true),
		Entry("mixed bson key and Go name", "spec.Color", "spec.color", true, true),
		Entry("field of an inline struct", "Owner", "owner", true, true),
		Entry("field behind a pointer", "SpecPtr.Color", "specPtr.color", true, true),
		Entry("entry of a string-keyed map", "labels.env", "labels.env", true, true),
		Entry("field of a struct inside a map", "Specs.prod.Color", "specs.prod.color", true, true),
		Entry("array index", "items.0.Name", "items.0.name", true, false),
		Entry("field of array elements", "Items.Name", "items.name", true, false),
		Entry("index of a scalar array", "tags.0", "tags.0", true, false),
		Entry("positional operator $", "Items.$.Name", "items.$.name", true, false),
		Entry("all positional operator $[]", "items.$[].Name", "items.$[].name", true, false),
		Entry("filtered positional operator $[id]", "Items.$[it].Name", "items.$[it].name", true, false),
		Entry("unknown field after a positional operator", "items.$[it].missing", "items.$[it].missing", false, false),
		Entry("path inside an interface is kept as is", "Extra.Foo.Bar", "extra.Foo.Bar", true, false),
		Entry("path inside a non-string-keyed map is kept as is", "counts.1", "counts.1", true, false),
		Entry("unknown top-level field is kept as is", "Missing.X", "Missing.X", false, false),
		Entry("unknown nested field is kept from the first unknown segment", "Spec.Missing.X",
			"spec.Missing.X", false, false),
		Entry("field excluded by bson:\"-\" is unknown", "Ignored", "Ignored", false, false),
		Entry("path into a scalar is unknown", "title.x", "title.x", false, false),
		Entry("path into time.Time is unknown", "CreatedAt.wall", "createdAt.wall", false, false),
	)

	Describe("ResolveKnown", func() {
		It("returns the same path as Resolve for a known field", func() {
			for _, p := range []string{"Spec.Color", "items.$[it].Name", "Extra.Foo.Bar", "labels.env"} {
				known, err := docFields.ResolveKnown(p)
				Expect(err).NotTo(HaveOccurred())
				Expect(known).To(Equal(mustResolve(docFields, p)))
			}
		})

		DescribeTable(
			"reports which segment is unknown",
			func(p, msg string) {
				_, err := docFields.ResolveKnown(p)
				Expect(err).To(MatchError(path.ErrUnknownField))
				Expect(err).To(MatchError(ContainSubstring(msg)))
			},
			Entry("unknown top-level field", "Missing.X", `path_test.pathDoc has no field "Missing"`),
			Entry("unknown nested field", "Spec.Missing.X", `path_test.pathSpec has no field "Missing"`),
			Entry(
				"unknown field of array elements",
				"items.$[it].missing",
				`path_test.pathItem has no field "missing"`,
			),
			Entry("field excluded by bson:\"-\"", "Ignored", `path_test.pathDoc has no field "Ignored"`),
			Entry("path into a scalar", "title.x", `string has no field "x"`),
			Entry("path into time.Time", "CreatedAt.wall", `time.Time has no field "wall"`),
		)

		It("rejects an ambiguous segment", func() {
			_, err := newFields[pathAmbiguous]().ResolveKnown("Label")
			Expect(err).To(MatchError(path.ErrAmbiguousField))
		})
	})

	It("records the kind of each step", func() {
		resolved := mustResolve(docFields, "Specs.prod.Color")
		Expect(lo.Map(resolved.Steps, func(st path.Step, _ int) path.StepKind { return st.Kind })).
			To(Equal([]path.StepKind{path.StepField, path.StepMapKey, path.StepField}))
	})

	Describe("ambiguous names", func() {
		ambiguousFields := newFields[pathAmbiguous]()

		It("rejects a segment that is the bson key and the Go name of different fields", func() {
			_, err := ambiguousFields.Resolve("Label")
			Expect(err).To(MatchError(path.ErrAmbiguousField))
			_, err = ambiguousFields.Lookup("Label")
			Expect(err).To(MatchError(path.ErrAmbiguousField))
		})

		It("rejects an ambiguous segment in a nested struct", func() {
			_, err := newFields[pathNested]().Resolve("inner.Label")
			Expect(err).To(MatchError(path.ErrAmbiguousField))
		})

		It("accepts a name that matches only one field", func() {
			Expect(mustResolve(ambiguousFields, "Title").String()).To(Equal("Label"))
			Expect(mustResolve(ambiguousFields, "title").String()).To(Equal("title"))
		})

		It("accepts a segment that is both the bson key and the Go name of the same field", func() {
			Expect(mustResolve(ambiguousFields, "Same").String()).To(Equal("Same"))
		})
	})
})
