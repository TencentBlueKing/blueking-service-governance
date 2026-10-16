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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store/path"
)

var _ = Describe("Path", func() {
	docFields := newFields[pathDoc]()
	resolveAll := func(paths ...string) []path.Path {
		return lo.Map(paths, func(p string, _ int) path.Path { return mustResolve(docFields, p) })
	}

	DescribeTable("IsAncestorOf",
		func(parent, child string, expected bool) {
			Expect(mustResolve(docFields, parent).IsAncestorOf(mustResolve(docFields, child))).To(Equal(expected))
		},
		Entry("parent of a nested field", "spec", "spec.color", true),
		Entry("parent of a deeply nested field", "specs", "specs.prod.color", true),
		Entry("not its own ancestor", "spec", "spec", false),
		Entry("not an ancestor of a field sharing the prefix", "spec", "specPtr.color", false),
		Entry("child is not an ancestor of its parent", "spec.color", "spec", false),
	)

	DescribeTable("Positional",
		func(p string, expected bool) {
			Expect(mustResolve(docFields, p).Positional()).To(Equal(expected))
		},
		Entry("positional operator $", "items.$.name", true),
		Entry("all positional operator $[]", "items.$[].name", true),
		Entry("filtered positional operator $[id]", "items.$[it].name", true),
		Entry("array index", "items.0.name", false),
		Entry("plain field", "spec.color", false),
	)

	Describe("Dedupe", func() {
		It("removes the same field given by bson key and Go name", func() {
			paths, err := path.Dedupe(resolveAll("title", "Title", "spec.color", "Spec.Color"))
			Expect(err).NotTo(HaveOccurred())
			Expect(lo.Map(paths, func(p path.Path, _ int) string { return p.String() })).
				To(Equal([]string{"title", "spec.color"}))
		})

		It("keeps sibling fields", func() {
			paths, err := path.Dedupe(resolveAll("spec.color", "spec.size"))
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(2))
		})

		DescribeTable("rejects a field together with its parent",
			func(paths ...string) {
				_, err := path.Dedupe(resolveAll(paths...))
				Expect(err).To(MatchError(ContainSubstring("conflicting fields")))
			},
			Entry("parent first", "spec", "spec.color"),
			Entry("child first", "Spec.Color", "Spec"),
		)
	})
})
