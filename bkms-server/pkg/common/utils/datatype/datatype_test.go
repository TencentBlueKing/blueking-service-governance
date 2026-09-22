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

package datatype_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/datatype"
)

var _ = Describe("CloneMap", func() {
	It("isolates nested Go containers from the source", func() {
		src := map[string]any{"fileName": "a.yaml", "nested": map[string]any{"x": 1}}

		cloned := datatype.CloneMap(src)
		cloned["fileName"] = "b.yaml"
		cloned["nested"].(map[string]any)["x"] = 2

		Expect(src["fileName"]).To(Equal("a.yaml"))
		Expect(src["nested"].(map[string]any)["x"]).To(Equal(1))
	})

	It("isolates nested BSON containers from the source", func() {
		// bson.D/bson.A are slices, so a copy that only walked the Go containers
		// would hand back a map still aliasing the persisted arrays.
		src := map[string]any{
			"nested": bson.D{{Key: "ok", Value: true}},
			"tags":   bson.A{"x"},
		}

		cloned := datatype.CloneMap(src)
		cloned["nested"].(bson.D)[0].Value = false
		cloned["tags"].(bson.A)[0] = "y"

		Expect(src["nested"].(bson.D)[0].Value).To(Equal(true))
		Expect(src["tags"].(bson.A)[0]).To(Equal("x"))
	})

	It("clones maps of other key and value types", func() {
		slices := map[string][]string{"k": {"v1", "v2"}}
		clonedSlices := datatype.CloneMap(slices)
		clonedSlices["k"][0] = "mutated"
		Expect(slices["k"][0]).To(Equal("v1"))

		nested := map[int]map[string]int{1: {"x": 1}}
		clonedNested := datatype.CloneMap(nested)
		clonedNested[1]["x"] = 99
		Expect(nested[1]["x"]).To(Equal(1))
	})

	It("returns nil for a nil map and an empty map for an empty one", func() {
		Expect(datatype.CloneMap[string, any](nil)).To(BeNil())
		Expect(datatype.CloneMap(map[string]any{})).To(BeEmpty())
	})
})
