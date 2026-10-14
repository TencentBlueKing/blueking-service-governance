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

package store_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("Mongo document utilities", func() {
	DescribeTable("documentHasKey",
		func(doc bson.D, key string, expected bool) {
			Expect(store.DocumentHasKeyForTest(doc, key)).To(Equal(expected))
		},
		Entry("finds a direct field",
			bson.D{{Key: "_id", Value: "id"}}, "_id", true,
		),
		Entry("finds a field inside $and encoded with bson.M",
			bson.D{{Key: "$and", Value: bson.A{bson.M{"_id": "id"}}}}, "_id", true,
		),
		Entry("finds a field inside nested operators encoded with bson.D",
			bson.D{{
				Key: "$or",
				Value: bson.A{bson.D{{
					Key:   "$and",
					Value: bson.A{bson.D{{Key: "_id", Value: "id"}}},
				}}},
			}}, "_id", true,
		),
		Entry("ignores the same key in an ordinary nested document",
			bson.D{{Key: "payload", Value: bson.M{"_id": "nested"}}}, "_id", false,
		),
		Entry("ignores the same key below an ordinary field inside an operator",
			bson.D{{
				Key: "$and",
				Value: bson.A{bson.M{
					"payload": bson.M{"_id": "nested"},
				}},
			}}, "_id", false,
		),
		Entry("returns false when the key is absent",
			bson.D{{Key: "name", Value: "demo"}}, "_id", false,
		),
	)
})
