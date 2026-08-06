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

// Package txcmdb provides api client to tx-cmdb
package txcmdb

const (
	// apiName Tx CMDB API 名称，用于 metrics 上报
	apiName = "tx-cmdb" //nolint:unused

	// defaultPageSize 默认分页大小
	defaultPageSize = 200 //nolint:unused

	// maxScrollPages 全量拉取时的最大分页数，防止死循环
	maxScrollPages = 1000 //nolint:unused
)

const (
	// queryLevel2BusinessDetailPath 查询二级业务明细接口路径
	queryLevel2BusinessDetailPath = "/cmdb-service-business-domain/queryBusinessLevel2DetailInfo" //nolint:unused
)

// Level2BusinessDetail 二级业务明细信息
type Level2BusinessDetail struct {
	// Level1BizID 一级业务 ID
	Level1BizID string

	// Level1BizName 一级业务名称
	Level1BizName string

	// Level2BizID 二级业务 ID
	Level2BizID string

	// Level2BizName 二级业务名称
	Level2BizName string

	// ObsProductID 运营产品 ID
	ObsProductID string

	// ObsProductName 运营产品名称
	ObsProductName string
}

// queryLevel2BusinessDetailResult 查询二级业务明细接口响应结果
type queryLevel2BusinessDetailResult struct { //nolint:unused
	// List 二级业务明细列表
	List []Level2BusinessDetail

	// Total 总记录数
	Total int

	// HasMore 是否还有更多数据
	HasMore bool

	// ScrollID 下一页游标，为空或与当前相同时表示已到末页
	ScrollID string
}

// queryLevel2BusinessDetailParams 查询二级业务明细接口请求参数
type queryLevel2BusinessDetailParams struct { //nolint:unused
	// ResultColumn 指定返回的字段列表，为空时返回所有字段
	ResultColumn []string `json:"resultColumn,omitempty"`

	// Condition 过滤条件
	Condition map[string]any `json:"condition,omitempty"`

	// Size 每页记录数
	Size int `json:"size,omitempty"`

	// ScrollID 游标
	// 首次请求传 "0"
	ScrollID string `json:"scrollId,omitempty"`
}
