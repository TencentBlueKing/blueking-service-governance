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

// Package serializer 提供 arrangement 模块 Gin v2 API 的请求和响应结构。
package serializer

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement"

// PlaceholderVarOutputObj 是占位符变量的输出对象。
type PlaceholderVarOutputObj struct {
	// 占位符变量的 key, 如 IMAGE, IMAGE_TAG 等
	Key string `json:"key"`
	// 占位符变量的描述信息
	Description string `json:"description"`
}

// FromModel 把领域层占位符变量转换为兼容 v1 响应字段名的输出对象。
func (o *PlaceholderVarOutputObj) FromModel(item arrangement.PlaceholderVar) *PlaceholderVarOutputObj {
	*o = PlaceholderVarOutputObj{
		Key:         item.Key,
		Description: item.Description,
	}
	return o
}

// ListPlaceholderVarsOutput 是获取占位符变量列表接口的响应。
type ListPlaceholderVarsOutput struct {
	Data []*PlaceholderVarOutputObj `json:"data"`
}

// FromModels 把领域层占位符变量列表转换为兼容 v1 的 data 列表，保留原有顺序和空值行为。
func (o *ListPlaceholderVarsOutput) FromModels(items []arrangement.PlaceholderVar) *ListPlaceholderVarsOutput {
	o.Data = make([]*PlaceholderVarOutputObj, 0, len(items))
	for _, item := range items {
		o.Data = append(o.Data, new(PlaceholderVarOutputObj).FromModel(item))
	}
	return o
}
