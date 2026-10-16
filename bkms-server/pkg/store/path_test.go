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

package store

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
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

var (
	pathDocType       = reflect.TypeFor[pathDoc]()
	pathAmbiguousType = reflect.TypeFor[pathAmbiguous]()
)

func mustResolvePath(typ reflect.Type, path string) fieldPath {
	p, err := resolvePath(typ, path)
	Expect(err).NotTo(HaveOccurred())
	return p
}

var _ = Describe("Field path", func() {
	Describe("structFields", func() {
		It("expands inline structs and skips fields excluded from encoding", func() {
			keys := lo.Map(structFields(pathDocType), func(f structField, _ int) string { return f.key })
			Expect(keys).To(Equal([]string{
				"_id", "owner", "title", "enabled", "spec", "specPtr", "labels", "specs",
				"items", "tags", "extra", "counts", "createdAt", "notag",
			}))
		})

		It("records the index path through the inline struct", func() {
			owner, found := lo.Find(structFields(pathDocType), func(f structField) bool { return f.key == "owner" })
			Expect(found).To(BeTrue())
			Expect(owner.index).To(Equal([]int{1, 0}))
			Expect(owner.name).To(Equal("Owner"))
		})
	})

	DescribeTable("resolvePath",
		func(path, expected string, known, assignable bool) {
			p := mustResolvePath(pathDocType, path)
			Expect(p.String()).To(Equal(expected))
			Expect(p.known).To(Equal(known), "known")
			Expect(p.assignable).To(Equal(assignable), "assignable")
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
		Entry("path inside an interface is kept as is", "Extra.Foo.Bar", "extra.Foo.Bar", true, false),
		Entry("path inside a non-string-keyed map is kept as is", "counts.1", "counts.1", true, false),
		Entry("unknown top-level field is kept as is", "Missing.X", "Missing.X", false, false),
		Entry("unknown nested field is kept from the first unknown segment", "Spec.Missing.X",
			"spec.Missing.X", false, false),
		Entry("field excluded by bson:\"-\" is unknown", "Ignored", "Ignored", false, false),
		Entry("path into a scalar is unknown", "title.x", "title.x", false, false),
		Entry("path into time.Time is unknown", "CreatedAt.wall", "createdAt.wall", false, false),
	)

	It("records the kind of each step", func() {
		p := mustResolvePath(pathDocType, "Specs.prod.Color")
		Expect(lo.Map(p.steps, func(st pathStep, _ int) stepKind { return st.kind })).
			To(Equal([]stepKind{stepField, stepMapKey, stepField}))
	})

	Describe("ambiguous names", func() {
		It("rejects a segment that is the bson key and the Go name of different fields", func() {
			_, err := resolvePath(pathAmbiguousType, "Label")
			Expect(err).To(MatchError(errAmbiguousField))
		})

		It("accepts a name that matches only one field", func() {
			Expect(mustResolvePath(pathAmbiguousType, "Title").String()).To(Equal("Label"))
			Expect(mustResolvePath(pathAmbiguousType, "title").String()).To(Equal("title"))
		})

		It("accepts a segment that is both the bson key and the Go name of the same field", func() {
			Expect(mustResolvePath(pathAmbiguousType, "Same").String()).To(Equal("Same"))
		})
	})

	DescribeTable("resolveKey",
		func(path, expected string) {
			key, err := resolveKey(pathDocType, path)
			Expect(err).NotTo(HaveOccurred())
			Expect(key).To(Equal(expected))
		},
		Entry("converts Go names to bson keys", "Spec.Color", "spec.color"),
		Entry("keeps unknown fields as is", "legacy.Field", "legacy.Field"),
	)

	DescribeTable("isAncestorOf",
		func(parent, child string, expected bool) {
			Expect(mustResolvePath(pathDocType, parent).isAncestorOf(mustResolvePath(pathDocType, child))).
				To(Equal(expected))
		},
		Entry("parent of a nested field", "spec", "spec.color", true),
		Entry("parent of a deeply nested field", "specs", "specs.prod.color", true),
		Entry("not its own ancestor", "spec", "spec", false),
		Entry("not an ancestor of a field sharing the prefix", "spec", "specPtr.color", false),
		Entry("child is not an ancestor of its parent", "spec.color", "spec", false),
	)

	Describe("dedupePaths", func() {
		resolveAll := func(paths ...string) []fieldPath {
			return lo.Map(paths, func(p string, _ int) fieldPath { return mustResolvePath(pathDocType, p) })
		}

		It("removes the same field given by bson key and Go name", func() {
			paths, err := dedupePaths(resolveAll("title", "Title", "spec.color", "Spec.Color"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(paths, func(p fieldPath, _ int) string { return p.String() })).
				To(Equal([]string{"title", "spec.color"}))
		})

		It("keeps sibling fields", func() {
			paths, err := dedupePaths(resolveAll("spec.color", "spec.size"))
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(2))
		})

		DescribeTable("rejects a field together with its parent",
			func(paths ...string) {
				_, err := dedupePaths(resolveAll(paths...))
				Expect(err).To(MatchError(ContainSubstring("conflicting fields")))
			},
			Entry("parent first", "spec", "spec.color"),
			Entry("child first", "Spec.Color", "Spec"),
		)
	})
})
