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

const (
	// apiName bk-cmdb API 名称
	apiName = "bk-cmdb"

	// pageLimit 分页大小
	pageLimit = 200

	// maxScrollPages 全量拉取时的最大分页数，防止死循环
	maxScrollPages = 1000
)

// Business bkcc 业务信息
type Business struct {
	// BizID 业务 ID
	BizID string

	// BizName 业务名称
	BizName string

	// Level2BizID 二级业务 ID
	Level2BizID string
}

// PageParam 分页参数
type PageParam struct {
	// Start 起始记录索引，从 0 开始
	Start int `json:"start"`

	// Limit 每页记录数
	Limit int `json:"limit"`

	// Sort 排序字段
	Sort string `json:"sort,omitempty"`
}

// searchBusinessParams 接口请求参数
type searchBusinessParams struct {
	// SupplierAccount 开发商账号，路径参数，为空时默认使用 "0"
	SupplierAccount string

	// Fields 指定返回的字段列表，为空时返回所有字段
	Fields []string `json:"fields,omitempty"`

	// BizPropertyFilter 业务属性过滤器
	BizPropertyFilter map[string]any `json:"biz_property_filter,omitempty"`

	// TimeCondition 按时间过滤条件
	TimeCondition map[string]any `json:"time_condition,omitempty"`

	// Page 分页参数
	Page *PageParam `json:"page,omitempty"`
}
