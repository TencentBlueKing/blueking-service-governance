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

package serializer_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
)

var _ = Describe("BKCC Serializer", func() {
	Describe("ListBKCCAuthorizedBusinessesOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"bizID": "100001",
						"bizName": "Test Business A",
						"obsProductID": "op-001",
						"obsProductName": "Product A",
						"level1BizID": "1",
						"level1BizName": "Level 1 Business",
						"level2BizID": "10",
						"level2BizName": "Level 2 Business"
					},
					{
						"bizID": "100002",
						"bizName": "Test Business B",
						"obsProductID": "",
						"obsProductName": "",
						"level1BizID": "",
						"level1BizName": "",
						"level2BizID": "",
						"level2BizName": ""
					}
				]
			}`

			var resp serializer.ListBKCCAuthorizedBusinessesOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(2))
			Expect(resp.Data[0].BizID).To(Equal("100001"))
			Expect(resp.Data[0].BizName).To(Equal("Test Business A"))
			Expect(resp.Data[0].ObsProductID).To(Equal("op-001"))
			Expect(resp.Data[0].ObsProductName).To(Equal("Product A"))
			Expect(resp.Data[0].Level1BizID).To(Equal("1"))
			Expect(resp.Data[0].Level1BizName).To(Equal("Level 1 Business"))
			Expect(resp.Data[0].Level2BizID).To(Equal("10"))
			Expect(resp.Data[0].Level2BizName).To(Equal("Level 2 Business"))

			Expect(resp.Data[1].BizID).To(Equal("100002"))
			Expect(resp.Data[1].BizName).To(Equal("Test Business B"))
			Expect(resp.Data[1].ObsProductID).To(BeEmpty())
		})

		It("should parse empty data list", func() {
			rawJSON := `{"data": []}`

			var resp serializer.ListBKCCAuthorizedBusinessesOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeEmpty())
		})

		It("should parse null data as nil", func() {
			rawJSON := `{"data": null}`

			var resp serializer.ListBKCCAuthorizedBusinessesOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeNil())
		})
	})
})
