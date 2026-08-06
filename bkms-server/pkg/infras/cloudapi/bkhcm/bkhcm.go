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

// Package bkhcm 封装了对蓝鲸 HCM 服务的调用，提供地域、可用区等基础资源查询能力
package bkhcm

import (
	"context"
	"fmt"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

// ListRegions 查询地域列表
//
// 使用固定厂商 tcloud-ziyan 查询可用地域，不需要用户认证。
func (c *ApiClient) ListRegions(ctx context.Context, filter *Filter, page *Page) ([]Region, error) {
	body := map[string]any{"page": page}
	if filter != nil {
		body["filter"] = filter
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_region",
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/cloud/vendors/%s/regions/list", VendorTCloudZiYan),
		},
		bkapi.OptSetRequestBody(body),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	regions := make([]Region, 0)
	for _, item := range mapx.GetList(result, "data.details") {
		if v, ok := item.(map[string]any); ok {
			regions = append(regions, Region{
				ID:         mapx.GetStr(v, "id"),
				Vendor:     mapx.GetStr(v, "vendor"),
				RegionID:   mapx.GetStr(v, "region_id"),
				RegionName: mapx.GetStr(v, "region_name"),
				Status:     mapx.GetStr(v, "status"),
			})
		}
	}
	return regions, nil
}

// ListSubnets 查询子网列表
//
// 按业务 ID 查询子网信息，需要用户认证（bk_ticket）。
func (c *ApiClient) ListSubnets(ctx context.Context, bkBizID int64, filter *Filter, page *Page) ([]Subnet, error) {
	body := map[string]any{"page": page}
	if filter != nil {
		body["filter"] = filter
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_subnet",
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/cloud/bizs/%d/subnets/list", bkBizID),
		},
		bkapi.OptSetRequestBody(body),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	subnets := make([]Subnet, 0)
	for _, item := range mapx.GetList(result, "data.detail") {
		if v, ok := item.(map[string]any); ok {
			subnets = append(subnets, Subnet{
				ID:         mapx.GetStr(v, "id"),
				Vendor:     mapx.GetStr(v, "vendor"),
				AccountID:  mapx.GetStr(v, "account_id"),
				CloudVpcID: mapx.GetStr(v, "cloud_vpc_id"),
				CloudID:    mapx.GetStr(v, "cloud_id"),
				Name:       mapx.GetStr(v, "name"),
				Region:     mapx.GetStr(v, "region"),
				Zone:       mapx.GetStr(v, "zone"),
				Ipv4Cidr:   fromAnySlice[string](mapx.GetList(v, "ipv4_cidr")),
				Ipv6Cidr:   fromAnySlice[string](mapx.GetList(v, "ipv6_cidr")),
				Memo:       mapx.GetStr(v, "memo"),
				VpcID:      mapx.GetStr(v, "vpc_id"),
				BkBizID:    cast.ToInt64(v["bk_biz_id"]),
			})
		}
	}
	return subnets, nil
}

// ListVPCs 查询 VPC 列表
//
// 按业务 ID 查询 VPC 信息（使用固定厂商 tcloud-ziyan），需要用户认证（bk_ticket）。
func (c *ApiClient) ListVPCs(ctx context.Context, bkBizID int64, filter *Filter, page *Page) ([]VPC, error) {
	body := map[string]any{"page": page}
	if filter != nil {
		body["filter"] = filter
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_vpc",
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/cloud/bizs/%d/vendors/%s/vpcs/list", bkBizID, VendorTCloudZiYan),
		},
		bkapi.OptSetRequestBody(body),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	vpcs := make([]VPC, 0)
	for _, item := range mapx.GetList(result, "data.detail") {
		if v, ok := item.(map[string]any); ok {
			var extension map[string]any
			if ext, ok := v["extension"].(map[string]any); ok {
				extension = ext
			}
			vpcs = append(vpcs, VPC{
				ID:        mapx.GetStr(v, "id"),
				Vendor:    mapx.GetStr(v, "vendor"),
				AccountID: mapx.GetStr(v, "account_id"),
				CloudID:   mapx.GetStr(v, "cloud_id"),
				Name:      mapx.GetStr(v, "name"),
				Region:    mapx.GetStr(v, "region"),
				Category:  mapx.GetStr(v, "category"),
				Memo:      mapx.GetStr(v, "memo"),
				BkBizID:   cast.ToInt64(v["bk_biz_id"]),
				Extension: extension,
			})
		}
	}
	return vpcs, nil
}

// ListZones 查询可用区列表
//
// 按地域查询可用区（使用固定厂商 tcloud-ziyan），不需要用户认证。
func (c *ApiClient) ListZones(ctx context.Context, region string, filter *Filter, page *Page) ([]Zone, error) {
	body := map[string]any{"page": page}
	if filter != nil {
		body["filter"] = filter
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "list_zone",
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/cloud/vendors/%s/regions/%s/zones/list", VendorTCloudZiYan, region),
		},
		bkapi.OptSetRequestBody(body),
	)

	result, err := c.handleOperation(ctx, op)
	if err != nil {
		return nil, err
	}

	zones := make([]Zone, 0)
	for _, item := range mapx.GetList(result, "data.details") {
		if v, ok := item.(map[string]any); ok {
			zones = append(zones, Zone{
				ID:      mapx.GetStr(v, "id"),
				Vendor:  mapx.GetStr(v, "vendor"),
				CloudID: mapx.GetStr(v, "cloud_id"),
				Name:    mapx.GetStr(v, "name"),
				NameCN:  mapx.GetStr(v, "name_cn"),
				Region:  mapx.GetStr(v, "region"),
				State:   mapx.GetStr(v, "state"),
			})
		}
	}
	return zones, nil
}

// fromAnySlice 是 lo.FromAnySlice 的简单包装，忽略 ok 返回值。
func fromAnySlice[T any](items []any) []T {
	result, _ := lo.FromAnySlice[T](items)
	return result
}
