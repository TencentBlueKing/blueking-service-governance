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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store/path"
)

type selDoc struct {
	ID        string            `bson:"_id"`
	Name      string            `bson:"name"`
	Enabled   bool              `bson:"enabled"`
	Spec      schemaSpec        `bson:"spec"`
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
	pathsOf := func(opts ...UpdateOpt) ([]string, error) {
		paths, err := selection(opts...).paths(sc)
		return lo.Map(paths, func(p path.Path, _ int) string { return p.String() }), err
	}

	DescribeTable("selects fields",
		func(opts []UpdateOpt, expected []string) {
			Expect(pathsOf(opts...)).To(Equal(expected))
		},
		Entry("all top-level fields except _id and timestamps without UpdateSelected",
			nil, []string{"name", "enabled", "spec", "labels"},
		),
		Entry("all top-level fields with \"*\"",
			[]UpdateOpt{UpdateSelected("*")}, []string{"name", "enabled", "spec", "labels"},
		),
		Entry("all top-level fields when merged opts repeat \"*\"",
			[]UpdateOpt{UpdateSelected("*"), UpdateSelected("*")}, []string{"name", "enabled", "spec", "labels"},
		),
		Entry("selected fields by bson key or Go name, in the given order",
			[]UpdateOpt{UpdateSelected("Spec.Color", "name")}, []string{"spec.color", "name"},
		),
		Entry("updatedAt when it is selected explicitly",
			[]UpdateOpt{UpdateSelected("updatedAt")}, []string{"updatedAt"},
		),
		Entry("all fields except the omitted ones",
			[]UpdateOpt{UpdateOmit("Enabled", "labels")}, []string{"name", "spec"},
		),
		Entry("selected fields except the omitted ones",
			[]UpdateOpt{UpdateSelected("name", "enabled"), UpdateOmit("enabled")}, []string{"name"},
		),
		Entry("each field once when merged opts repeat it",
			[]UpdateOpt{UpdateSelected("name"), UpdateSelected("Name")}, []string{"name"},
		),
	)

	DescribeTable("rejects invalid selections",
		func(opts []UpdateOpt, errMatcher any) {
			_, err := pathsOf(opts...)
			Expect(err).To(MatchError(errMatcher))
		},
		Entry("UpdateSelected without fields", []UpdateOpt{UpdateSelected()}, ErrNoFieldsToUpdate),
		Entry("all fields omitted",
			[]UpdateOpt{UpdateOmit("name", "enabled", "spec", "labels")}, ErrNoFieldsToUpdate,
		),
		Entry("\"*\" with other fields",
			[]UpdateOpt{UpdateSelected("*", "name")}, ContainSubstring("cannot be combined"),
		),
		Entry("\"*\" with other fields across merged opts",
			[]UpdateOpt{UpdateSelected("*"), UpdateSelected("*", "name")}, ContainSubstring("cannot be combined"),
		),
		Entry("unknown selected field", []UpdateOpt{UpdateSelected("missing")}, path.ErrUnknownField),
		Entry("unknown omitted field", []UpdateOpt{UpdateOmit("missing")}, path.ErrUnknownField),
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
