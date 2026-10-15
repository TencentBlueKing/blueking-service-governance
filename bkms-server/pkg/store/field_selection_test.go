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

type selDoc struct {
	ID        string            `bson:"_id"`
	Name      string            `bson:"name"`
	Enabled   bool              `bson:"enabled"`
	Spec      pathSpec          `bson:"spec"`
	Labels    map[string]string `bson:"labels"`
	CreatedAt time.Time         `bson:"createdAt"`
	UpdatedAt time.Time         `bson:"updatedAt"`
}

var _ = Describe("Field selection", func() {
	sc := lo.Must(newSchema(reflect.TypeFor[selDoc]()))

	// selection 合并 opts，与 parseOpts 中的行为一致
	selection := func(opts ...UpdateOpt) *fieldSelection {
		q := &query{}
		for _, opt := range opts {
			opt.apply(q)
		}
		if q.fields == nil {
			return &fieldSelection{}
		}
		return q.fields
	}
	pathsOf := func(obj selDoc, opts ...UpdateOpt) ([]string, error) {
		paths, err := selection(opts...).paths(sc, reflect.ValueOf(&obj).Elem())
		return lo.Map(paths, func(p fieldPath, _ int) string { return p.String() }), err
	}

	DescribeTable("selects fields",
		func(obj selDoc, opts []UpdateOpt, expected []string) {
			Expect(pathsOf(obj, opts...)).To(Equal(expected))
		},
		Entry("all top-level fields except _id and timestamps without UpdateSelected",
			selDoc{}, nil, []string{"name", "enabled", "spec", "labels"},
		),
		Entry("all top-level fields with \"*\"",
			selDoc{}, []UpdateOpt{UpdateSelected("*")}, []string{"name", "enabled", "spec", "labels"},
		),
		Entry("selected fields by bson key or Go name, in the given order",
			selDoc{}, []UpdateOpt{UpdateSelected("Spec.Color", "name")}, []string{"spec.color", "name"},
		),
		Entry("updatedAt when it is selected explicitly",
			selDoc{}, []UpdateOpt{UpdateSelected("updatedAt")}, []string{"updatedAt"},
		),
		Entry("all fields except the omitted ones",
			selDoc{}, []UpdateOpt{UpdateOmit("Enabled", "labels")}, []string{"name", "spec"},
		),
		Entry("selected fields except the omitted ones",
			selDoc{}, []UpdateOpt{UpdateSelected("name", "enabled"), UpdateOmit("enabled")}, []string{"name"},
		),
		Entry("non-zero top-level fields, judging nested structs as a whole",
			selDoc{Name: "a", Spec: pathSpec{Size: 1}}, []UpdateOpt{UpdateNonZero()}, []string{"name", "spec"},
		),
		Entry("non-zero selected nested fields",
			selDoc{Spec: pathSpec{Color: "red"}},
			[]UpdateOpt{UpdateSelected("spec.color", "spec.size"), UpdateNonZero()},
			[]string{"spec.color"},
		),
		Entry("non-zero map entries",
			selDoc{Labels: map[string]string{"env": "prod"}},
			[]UpdateOpt{UpdateSelected("labels.env", "labels.team"), UpdateNonZero()},
			[]string{"labels.env"},
		),
		Entry("each field once when merged opts repeat it",
			selDoc{}, []UpdateOpt{UpdateSelected("name"), UpdateSelected("Name")}, []string{"name"},
		),
	)

	DescribeTable("rejects invalid selections",
		func(opts []UpdateOpt, errMatcher any) {
			_, err := pathsOf(selDoc{}, opts...)
			Expect(err).To(MatchError(errMatcher))
		},
		Entry("UpdateSelected without fields", []UpdateOpt{UpdateSelected()}, ErrNoFieldsToUpdate),
		Entry("all fields omitted",
			[]UpdateOpt{UpdateOmit("name", "enabled", "spec", "labels")}, ErrNoFieldsToUpdate,
		),
		Entry("all fields are zero with UpdateNonZero", []UpdateOpt{UpdateNonZero()}, ErrNoFieldsToUpdate),
		Entry("\"*\" with other fields",
			[]UpdateOpt{UpdateSelected("*", "name")}, ContainSubstring("cannot be combined"),
		),
		Entry("unknown selected field", []UpdateOpt{UpdateSelected("missing")}, errUnknownField),
		Entry("unknown omitted field", []UpdateOpt{UpdateOmit("missing")}, errUnknownField),
		Entry("_id", []UpdateOpt{UpdateSelected("ID")}, ContainSubstring("cannot be updated")),
		Entry("createdAt", []UpdateOpt{UpdateSelected("createdAt")}, ContainSubstring("cannot be updated")),
		Entry("update operator", []UpdateOpt{UpdateSelected("$inc")}, ContainSubstring("not supported")),
		Entry("a field together with its parent",
			[]UpdateOpt{UpdateSelected("spec", "spec.color")}, ContainSubstring("conflicting fields"),
		),
		Entry("omitting a child of a field selected as a whole",
			[]UpdateOpt{UpdateOmit("spec.color")}, ContainSubstring("its parent \"spec\" is selected as a whole"),
		),
	)
})
