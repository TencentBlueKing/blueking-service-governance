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

// Package bkcc provides api client to bkcc（蓝鲸配置平台）
package bkcc

import (
	"context"
	"time"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/bk-apigateway-sdks/core/define"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// ListBusinesses  查询 SRE 有权限的业务信息
func (c *ApiClient) ListBusinesses(ctx context.Context) ([]Business, error) {
	results := make([]Business, 0)

	fields := []string{
		"bk_biz_id",
		"bk_biz_name",
		"bs2_name_id",
	}
	bizPropertyFilter := map[string]any{
		"condition": "AND",
		"rules": []map[string]any{
			{
				"field":    "bk_biz_maintainer",
				"operator": "contains",
				"value":    c.user.ID,
			},
		},
	}

	for start := 0; start < maxScrollPages; start += pageLimit {
		resp, err := c.searchBusiness(ctx, searchBusinessParams{
			Fields:            fields,
			BizPropertyFilter: bizPropertyFilter,
			Page:              &PageParam{Start: start, Limit: pageLimit},
		})
		if err != nil {
			return nil, err
		}
		results = append(results, resp...)

		if len(resp) < pageLimit {
			return results, nil
		}

		time.Sleep(10 * time.Millisecond)
	}

	return nil, errors.Errorf("list business details: exceeded max scroll pages (%d)", maxScrollPages)
}

// GetBusinessByID 按 bk_biz_id 精确查询单个业务；查询不到时返回 ErrBusinessNotFound
func (c *ApiClient) GetBusinessByID(ctx context.Context, bizID int64) (*Business, error) {
	businesses, err := c.searchBusiness(ctx, buildGetBusinessByIDParams(bizID))
	if err != nil {
		return nil, errors.Wrapf(err, "get business by id %d", bizID)
	}
	if len(businesses) == 0 {
		return nil, ErrBusinessNotFound
	}
	return &businesses[0], nil
}

// searchBusiness 查询业务列表
func (c *ApiClient) searchBusiness(ctx context.Context, params searchBusinessParams) ([]Business, error) {
	result, err := c.handleOperation(ctx, c.newSearchBusinessOperation(params))
	if err != nil {
		return nil, err
	}

	// 解析 data.info 中的业务列表
	var businesses []Business
	for _, d := range mapx.GetList(result, "data.info") {
		item, ok := d.(map[string]any)
		if !ok {
			continue
		}
		businesses = append(businesses, Business{
			BizName:     mapx.GetStr(item, "bk_biz_name"),
			BizID:       cast.ToString(mapx.Get(item, "bk_biz_id", 0)),
			Level2BizID: cast.ToString(mapx.Get(item, "bs2_name_id", 0)),
		})
	}

	return businesses, nil
}

// newSearchBusinessOperation 返回 operation
func (c *ApiClient) newSearchBusinessOperation(params searchBusinessParams) define.Operation {
	// 路径参数：bk_supplier_account 为空时默认使用 "0"
	supplierAccount := params.SupplierAccount
	if supplierAccount == "" {
		supplierAccount = "0"
	}

	// 按非空字段构造请求体
	body := map[string]any{}
	if len(params.Fields) > 0 {
		body["fields"] = params.Fields
	}
	if len(params.BizPropertyFilter) > 0 {
		body["biz_property_filter"] = params.BizPropertyFilter
	}
	if len(params.TimeCondition) > 0 {
		body["time_condition"] = params.TimeCondition
	}
	if params.Page != nil {
		body["page"] = params.Page
	}

	return c.NewOperation(
		bkapi.OperationConfig{
			Name:   "open_search_business",
			Method: "POST",
			Path:   "/api/v3/open/biz/search/{bk_supplier_account}",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"bk_supplier_account": supplierAccount,
		}),
		bkapi.OptSetRequestBody(body),
	)
}

// buildGetBusinessByIDParams 构造按 bk_biz_id 精确查询的参数
func buildGetBusinessByIDParams(bizID int64) searchBusinessParams {
	return searchBusinessParams{
		Fields: []string{
			"bk_biz_id",
			"bk_biz_name",
			"bs2_name_id",
		},
		BizPropertyFilter: map[string]any{
			"condition": "AND",
			"rules": []map[string]any{
				{
					"field":    "bk_biz_id",
					"operator": "equal",
					"value":    bizID,
				},
			},
		},
		Page: &PageParam{Start: 0, Limit: 1},
	}
}
