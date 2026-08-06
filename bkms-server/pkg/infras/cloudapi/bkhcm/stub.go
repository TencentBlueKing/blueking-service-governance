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

package bkhcm

import (
	"context"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

// stubRegions 本地开发时返回的固定地域列表
var stubRegions = []Region{
	{
		ID:         "00000001",
		Vendor:     "tcloud",
		RegionID:   "ap-guangzhou",
		RegionName: "华南地区(广州)",
		Status:     "AVAILABLE",
	},
	{
		ID:         "00000002",
		Vendor:     "tcloud",
		RegionID:   "ap-shanghai",
		RegionName: "华东地区(上海)",
		Status:     "AVAILABLE",
	},
	{
		ID:         "00000003",
		Vendor:     "tcloud",
		RegionID:   "ap-beijing",
		RegionName: "华北地区(北京)",
		Status:     "AVAILABLE",
	},
}

// stubSubnets 本地开发时返回的固定子网列表
var stubSubnets = []Subnet{
	{
		ID:         "00000001",
		Vendor:     "tcloud",
		AccountID:  "00000001",
		CloudVpcID: "vpc-stub001",
		CloudID:    "subnet-stub001",
		Name:       "stub-subnet-default",
		Region:     "ap-guangzhou",
		Zone:       "ap-guangzhou-6",
		Ipv4Cidr:   []string{"10.0.0.0/24"},
		Ipv6Cidr:   []string{},
		Memo:       "本地开发用 stub 子网",
		VpcID:      "00000001",
		BkBizID:    100,
	},
}

// stubVPCs 本地开发时返回的固定 VPC 列表
var stubVPCs = []VPC{
	{
		ID:        "00000001",
		Vendor:    "tcloud",
		AccountID: "00000001",
		CloudID:   "vpc-stub001",
		Name:      "stub-vpc-default",
		Region:    "ap-guangzhou",
		Category:  "biz",
		Memo:      "本地开发用 stub VPC",
		BkBizID:   100,
		Extension: map[string]any{
			"is_default": true,
			"cidr": []map[string]any{
				{"type": "ipv4", "cidr": "10.0.0.0/16", "category": "master"},
			},
		},
	},
}

// stubZones 本地开发时返回的固定可用区列表
var stubZones = []Zone{
	{
		ID:      "000001rn",
		Vendor:  "tcloud",
		CloudID: "100006",
		Name:    "ap-guangzhou-6",
		NameCN:  "广州六区",
		Region:  "ap-guangzhou",
		State:   "AVAILABLE",
	},
	{
		ID:      "000001ro",
		Vendor:  "tcloud",
		CloudID: "100007",
		Name:    "ap-guangzhou-7",
		NameCN:  "广州七区",
		Region:  "ap-guangzhou",
		State:   "AVAILABLE",
	},
}

// StubApiClient 测试用的 bk-hcm API 客户端实现，返回模拟数据
type StubApiClient struct {
	user auth.User
}

// NewStub 创建 StubApiClient
func NewStub(user auth.User) *StubApiClient {
	return &StubApiClient{user: user}
}

// ListRegions 模拟查询地域列表，返回 stubRegions
func (s *StubApiClient) ListRegions(ctx context.Context, _ *Filter, _ *Page) ([]Region, error) {
	log.Infof(ctx, "Stub: ListRegions request: user=%s", s.user.ID)
	return stubRegions, nil
}

// ListSubnets 模拟查询子网列表，返回 stubSubnets
func (s *StubApiClient) ListSubnets(ctx context.Context, bkBizID int64, _ *Filter, _ *Page) ([]Subnet, error) {
	log.Infof(ctx, "Stub: ListSubnets request: bkBizID=%d, user=%s", bkBizID, s.user.ID)
	return stubSubnets, nil
}

// ListVPCs 模拟查询 VPC 列表，返回 stubVPCs
func (s *StubApiClient) ListVPCs(ctx context.Context, bkBizID int64, _ *Filter, _ *Page) ([]VPC, error) {
	log.Infof(ctx, "Stub: ListVPCs request: bkBizID=%d, user=%s", bkBizID, s.user.ID)
	return stubVPCs, nil
}

// ListZones 模拟查询可用区列表，返回 stubZones
func (s *StubApiClient) ListZones(ctx context.Context, region string, _ *Filter, _ *Page) ([]Zone, error) {
	log.Infof(ctx, "Stub: ListZones request: region=%s, user=%s", region, s.user.ID)
	return stubZones, nil
}

// CreateBizApplicationForCreateLoadBalancer 模拟创建负载均衡申请，返回固定单据 ID
func (s *StubApiClient) CreateBizApplicationForCreateLoadBalancer(
	ctx context.Context, req *CreateLoadBalancerReq,
) (string, error) {
	log.Infof(ctx, "Stub: CreateBizApplicationForCreateLoadBalancer request: name=%s, user=%s",
		req.Name, s.user.ID)
	return "stub-application-00000001", nil
}
