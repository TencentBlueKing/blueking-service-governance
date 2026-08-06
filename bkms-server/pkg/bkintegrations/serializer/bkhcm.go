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

package serializer

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkhcm"

// --- bk-hcm URI 参数 ---

// BkHCMRegionURIInput 包含 region 路径参数（用于 list_zone）
type BkHCMRegionURIInput struct {
	Region string `uri:"region" binding:"required"`
}

// BkHCMBizURIInput 包含 bkBizID 路径参数（用于 list_subnet、list_vpc）
type BkHCMBizURIInput struct {
	BkBizID int64 `uri:"bkBizID" binding:"required"`
}

// --- bk-hcm JSON Body 参数 ---

// BkHCMListInput 通用的 bk-hcm 列表查询请求体
type BkHCMListInput struct {
	Filter *bkhcm.Filter `json:"filter"`
	Page   *bkhcm.Page   `json:"page" binding:"required"`
}

// BkHCMCreateLoadBalancerInput 创建负载均衡申请请求体
type BkHCMCreateLoadBalancerInput struct {
	BkBizID                  int64    `json:"bk_biz_id" binding:"required"`
	AccountID                string   `json:"account_id" binding:"required"`
	Region                   string   `json:"region" binding:"required"`
	LoadBalancerType         string   `json:"load_balancer_type" binding:"required"`
	Name                     string   `json:"name" binding:"required"`
	Zones                    []string `json:"zones,omitempty"`
	BackupZones              []string `json:"backup_zones,omitempty"`
	AddressIPVersion         string   `json:"address_ip_version,omitempty"`
	CloudVpcID               string   `json:"cloud_vpc_id" binding:"required"`
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
	RequireCount             int      `json:"require_count" binding:"required"`
	Memo                     string   `json:"memo,omitempty"`
	ZhiTong                  bool     `json:"zhi_tong,omitempty"`
	TgwGroupName             string   `json:"tgw_group_name,omitempty"`
	LoadBalancerPassToTarget bool     `json:"load_balancer_pass_to_target"`
	Remark                   string   `json:"remark,omitempty"`
}

// ToReq 将 Input 转换为 cloudapi 层的请求结构体
func (i *BkHCMCreateLoadBalancerInput) ToReq() *bkhcm.CreateLoadBalancerReq {
	return &bkhcm.CreateLoadBalancerReq{
		BkBizID:                  i.BkBizID,
		AccountID:                i.AccountID,
		Region:                   i.Region,
		LoadBalancerType:         i.LoadBalancerType,
		Name:                     i.Name,
		Zones:                    i.Zones,
		BackupZones:              i.BackupZones,
		AddressIPVersion:         i.AddressIPVersion,
		CloudVpcID:               i.CloudVpcID,
		CloudSubnetID:            i.CloudSubnetID,
		Vip:                      i.Vip,
		CloudEipID:               i.CloudEipID,
		VipIsp:                   i.VipIsp,
		InternetChargeType:       i.InternetChargeType,
		InternetMaxBandwidthOut:  i.InternetMaxBandwidthOut,
		BandwidthpkgSubType:      i.BandwidthpkgSubType,
		BandwidthPackageID:       i.BandwidthPackageID,
		SlaType:                  i.SlaType,
		AutoRenew:                i.AutoRenew,
		RequireCount:             i.RequireCount,
		Memo:                     i.Memo,
		ZhiTong:                  i.ZhiTong,
		TgwGroupName:             i.TgwGroupName,
		LoadBalancerPassToTarget: i.LoadBalancerPassToTarget,
		Remark:                   i.Remark,
	}
}

// --- bk-hcm Output ---

// RegionOutput 地域输出
type RegionOutput struct {
	ID         string `json:"id"`
	Vendor     string `json:"vendor"`
	RegionID   string `json:"region_id"`
	RegionName string `json:"region_name"`
	Status     string `json:"status"`
}

// ListBkHCMRegionsOutput 查询地域列表的响应
type ListBkHCMRegionsOutput struct {
	Data []*RegionOutput `json:"data"`
}

// SubnetOutput 子网输出
type SubnetOutput struct {
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

// ListBkHCMSubnetsOutput 查询子网列表的响应
type ListBkHCMSubnetsOutput struct {
	Data []*SubnetOutput `json:"data"`
}

// VPCOutput VPC 输出
type VPCOutput struct {
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

// ListBkHCMVPCsOutput 查询 VPC 列表的响应
type ListBkHCMVPCsOutput struct {
	Data []*VPCOutput `json:"data"`
}

// ZoneOutput 可用区输出
type ZoneOutput struct {
	ID      string `json:"id"`
	Vendor  string `json:"vendor"`
	CloudID string `json:"cloud_id"`
	Name    string `json:"name"`
	NameCN  string `json:"name_cn"`
	Region  string `json:"region"`
	State   string `json:"state"`
}

// ListBkHCMZonesOutput 查询可用区列表的响应
type ListBkHCMZonesOutput struct {
	Data []*ZoneOutput `json:"data"`
}

// CreateBkHCMLoadBalancerOutput 创建负载均衡申请的响应
type CreateBkHCMLoadBalancerOutput struct {
	Data *CreateBkHCMLoadBalancerData `json:"data"`
}

// CreateBkHCMLoadBalancerData 创建负载均衡申请的响应数据
type CreateBkHCMLoadBalancerData struct {
	ID string `json:"id"`
}
