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

package component_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

var _ = Describe("entities tests", func() {
	var compDefStore component.ComponentDefStore
	var ctx context.Context

	BeforeEach(func() {
		var err error
		compDefStore, err = component.NewComponentDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
	})

	Context("Property NormalizedDefaultValue", func() {
		It("should work", func() {
			compDef := dbfactory.CompDef(ctx, compDefStore, &dbfactory.ComponentDefOpts{
				Properties: []component.Property{
					{
						Name:         "fruits",
						Type:         "MAP",
						DefaultValue: map[string]any{"apple": int64(3)},
					},
				},
			})

			// dbfactory.CompDef 已落库，再读回以验证 DefaultValue 的序列化/反序列化。
			retrieved, err := compDefStore.Get(ctx, compDef.Name, compDef.Version)
			Expect(err).NotTo(HaveOccurred())

			// NOTE: All numeric values will be converted to float64 by json.Unmarshal
			expectedVal := map[string]any{"apple": float64(3)}
			// Without normalization, the val would be `{"apple":{"$numberLong":"3"}}`
			// It's determined by how MongoDB serializes int64 values in BSON.
			Expect(retrieved.Properties[0].DefaultValue).To(Not(Equal(expectedVal)))
			Expect(retrieved.Properties[0].NormalizedDefaultValue()).To(Equal(expectedVal))
		})
	})
})
