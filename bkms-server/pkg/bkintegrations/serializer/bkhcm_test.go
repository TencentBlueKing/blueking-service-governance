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

var _ = Describe("BkHCM Serializer", func() {
	Describe("ListBkHCMRegionsOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"id": "region-001",
						"vendor": "tcloud",
						"region_id": "ap-guangzhou",
						"region_name": "广州",
						"status": "AVAILABLE"
					},
					{
						"id": "region-002",
						"vendor": "tcloud",
						"region_id": "ap-shanghai",
						"region_name": "上海",
						"status": "AVAILABLE"
					}
				]
			}`

			var resp serializer.ListBkHCMRegionsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(2))
			Expect(resp.Data[0].ID).To(Equal("region-001"))
			Expect(resp.Data[0].Vendor).To(Equal("tcloud"))
			Expect(resp.Data[0].RegionID).To(Equal("ap-guangzhou"))
			Expect(resp.Data[0].RegionName).To(Equal("广州"))
			Expect(resp.Data[0].Status).To(Equal("AVAILABLE"))
		})

		It("should parse empty data list", func() {
			rawJSON := `{"data": []}`

			var resp serializer.ListBkHCMRegionsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeEmpty())
		})
	})

	Describe("ListBkHCMSubnetsOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"id": "subnet-001",
						"vendor": "tcloud",
						"account_id": "acc-001",
						"cloud_vpc_id": "vpc-12345",
						"cloud_id": "subnet-abc",
						"name": "子网A",
						"region": "ap-guangzhou",
						"zone": "ap-guangzhou-3",
						"ipv4_cidr": ["10.0.1.0/24"],
						"ipv6_cidr": [],
						"memo": "测试子网",
						"vpc_id": "vpc-001",
						"bk_biz_id": 100001
					}
				]
			}`

			var resp serializer.ListBkHCMSubnetsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(1))
			Expect(resp.Data[0].ID).To(Equal("subnet-001"))
			Expect(resp.Data[0].CloudVpcID).To(Equal("vpc-12345"))
			Expect(resp.Data[0].Name).To(Equal("子网A"))
			Expect(resp.Data[0].Zone).To(Equal("ap-guangzhou-3"))
			Expect(resp.Data[0].Ipv4Cidr).To(Equal([]string{"10.0.1.0/24"}))
			Expect(resp.Data[0].BkBizID).To(Equal(int64(100001)))
		})
	})

	Describe("ListBkHCMVPCsOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"id": "vpc-001",
						"vendor": "tcloud",
						"account_id": "acc-001",
						"cloud_id": "vpc-12345",
						"name": "默认VPC",
						"region": "ap-guangzhou",
						"category": "biz",
						"memo": "",
						"bk_biz_id": 100001,
						"extension": {"cidr": "10.0.0.0/16"}
					}
				]
			}`

			var resp serializer.ListBkHCMVPCsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(1))
			Expect(resp.Data[0].ID).To(Equal("vpc-001"))
			Expect(resp.Data[0].CloudID).To(Equal("vpc-12345"))
			Expect(resp.Data[0].Name).To(Equal("默认VPC"))
			Expect(resp.Data[0].Category).To(Equal("biz"))
			Expect(resp.Data[0].BkBizID).To(Equal(int64(100001)))
			Expect(resp.Data[0].Extension).To(HaveKey("cidr"))
		})
	})

	Describe("ListBkHCMZonesOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"id": "zone-001",
						"vendor": "tcloud",
						"cloud_id": "200003",
						"name": "ap-guangzhou-3",
						"name_cn": "广州三区",
						"region": "ap-guangzhou",
						"state": "AVAILABLE"
					},
					{
						"id": "zone-002",
						"vendor": "tcloud",
						"cloud_id": "200004",
						"name": "ap-guangzhou-4",
						"name_cn": "广州四区",
						"region": "ap-guangzhou",
						"state": "AVAILABLE"
					}
				]
			}`

			var resp serializer.ListBkHCMZonesOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(2))
			Expect(resp.Data[0].ID).To(Equal("zone-001"))
			Expect(resp.Data[0].CloudID).To(Equal("200003"))
			Expect(resp.Data[0].Name).To(Equal("ap-guangzhou-3"))
			Expect(resp.Data[0].NameCN).To(Equal("广州三区"))
			Expect(resp.Data[0].State).To(Equal("AVAILABLE"))
		})
	})

	Describe("CreateBkHCMLoadBalancerOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": {
					"id": "lb-application-001"
				}
			}`

			var resp serializer.CreateBkHCMLoadBalancerOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ID).To(Equal("lb-application-001"))
		})

		It("should parse JSON with null data", func() {
			rawJSON := `{"data": null}`

			var resp serializer.CreateBkHCMLoadBalancerOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeNil())
		})
	})
})
