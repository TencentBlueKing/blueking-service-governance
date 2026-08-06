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

// Package bkhcm provides api client to bk-hcm（蓝鲸海垫）
package bkhcm

// Filter bk-hcm 通用查询过滤条件
type Filter struct {
	Op    string       `json:"op"`
	Rules []FilterRule `json:"rules"`
}

// FilterRule bk-hcm 通用过滤规则
type FilterRule struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// Page bk-hcm 通用分页设置
type Page struct {
	Count bool   `json:"count"`
	Start uint32 `json:"start,omitempty"`
	Limit uint32 `json:"limit,omitempty"`
	Sort  string `json:"sort,omitempty"`
	Order string `json:"order,omitempty"`
}

// Region 云地域信息
type Region struct {
	ID         string `json:"id"`
	Vendor     string `json:"vendor"`
	RegionID   string `json:"region_id"`
	RegionName string `json:"region_name"`
	Status     string `json:"status"`
}

// Subnet 子网信息
type Subnet struct {
	ID         string   `json:"id"`
	Vendor     string   `json:"vendor"`
	AccountID  string   `json:"account_id"`
	CloudVpcID string   `json:"cloud_vpc_id"`
	CloudID    string   `json:"cloud_id"`
	Name       string   `json:"name"`
	Region     string   `json:"region"`
	Zone       string   `json:"zone"`
	Ipv4Cidr   []string `json:"ipv4_cidr"`
	Ipv6Cidr   []string `json:"ipv6_cidr"`
	Memo       string   `json:"memo"`
	VpcID      string   `json:"vpc_id"`
	BkBizID    int64    `json:"bk_biz_id"`
}

// VPC VPC 信息
type VPC struct {
	ID        string         `json:"id"`
	Vendor    string         `json:"vendor"`
	AccountID string         `json:"account_id"`
	CloudID   string         `json:"cloud_id"`
	Name      string         `json:"name"`
	Region    string         `json:"region"`
	Category  string         `json:"category"`
	Memo      string         `json:"memo"`
	BkBizID   int64          `json:"bk_biz_id"`
	Extension map[string]any `json:"extension"`
}

// Zone 可用区信息
type Zone struct {
	ID      string `json:"id"`
	Vendor  string `json:"vendor"`
	CloudID string `json:"cloud_id"`
	Name    string `json:"name"`
	NameCN  string `json:"name_cn"`
	Region  string `json:"region"`
	State   string `json:"state"`
}

// CreateLoadBalancerReq 创建负载均衡申请请求参数（TCloud）
type CreateLoadBalancerReq struct {
	BkBizID                  int64    `json:"bk_biz_id"`
	AccountID                string   `json:"account_id"`
	Region                   string   `json:"region"`
	LoadBalancerType         string   `json:"load_balancer_type"`
	Name                     string   `json:"name"`
	Zones                    []string `json:"zones,omitempty"`
	BackupZones              []string `json:"backup_zones,omitempty"`
	AddressIPVersion         string   `json:"address_ip_version,omitempty"`
	CloudVpcID               string   `json:"cloud_vpc_id"`
	CloudSubnetID            string   `json:"cloud_subnet_id,omitempty"`
	Vip                      string   `json:"vip,omitempty"`
	CloudEipID               string   `json:"cloud_eip_id,omitempty"`
	VipIsp                   string   `json:"vip_isp,omitempty"`
	InternetChargeType       string   `json:"internet_charge_type,omitempty"`
	InternetMaxBandwidthOut  int64    `json:"internet_max_bandwidth_out,omitempty"`
	BandwidthpkgSubType      string   `json:"bandwidthpkg_sub_type,omitempty"`
	BandwidthPackageID       string   `json:"bandwidth_package_id,omitempty"`
	SlaType                  string   `json:"sla_type,omitempty"`
	AutoRenew                bool     `json:"auto_renew,omitempty"`
	RequireCount             int      `json:"require_count"`
	Memo                     string   `json:"memo,omitempty"`
	ZhiTong                  bool     `json:"zhi_tong,omitempty"`
	TgwGroupName             string   `json:"tgw_group_name,omitempty"`
	LoadBalancerPassToTarget bool     `json:"load_balancer_pass_to_target"`
	Remark                   string   `json:"remark,omitempty"`
}

// CreateLoadBalancerResp 创建负载均衡申请响应
type CreateLoadBalancerResp struct {
	ID string `json:"id"`
}
