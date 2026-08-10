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

package env

import "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"

// EnvVarDetailedTableRow 用于 table 展示的扁平结构体，将 ScopedEnvVarDetailed 中的嵌套字段平级展开。
type EnvVarDetailedTableRow struct {
	ID               string `json:"id" yaml:"id"`
	ScopeType        string `json:"scopeType" yaml:"scopeType"`
	ScopeValue       string `json:"scopeValue" yaml:"scopeValue"`
	Key              string `json:"key" yaml:"key"`
	Value            string `json:"value" yaml:"value"`
	Description      string `json:"description" yaml:"description"`
	IsSensitive      bool   `json:"isSensitive" yaml:"isSensitive"`
	ConflictedDetail string `json:"conflictedDetail" yaml:"conflictedDetail"`
}

// ToTableRows 将 []ScopedEnvVarDetailed 转换为 []EnvVarDetailedTableRow，用于 table 格式输出。
func ToTableRows(items []client.ScopedEnvVarDetailed) []EnvVarDetailedTableRow {
	rows := make([]EnvVarDetailedTableRow, 0, len(items))
	for i := range items {
		row := EnvVarDetailedTableRow{}
		if items[i].ScopedEnvVar != nil {
			row.ID = items[i].ScopedEnvVar.ID
			row.ScopeType = items[i].ScopedEnvVar.ScopeType
			row.ScopeValue = items[i].ScopedEnvVar.ScopeValue
			row.Key = items[i].ScopedEnvVar.Key
			row.Value = items[i].ScopedEnvVar.Value
			row.Description = items[i].ScopedEnvVar.Description
			row.IsSensitive = items[i].ScopedEnvVar.IsSensitive
		}
		if items[i].ConflictedInfo != nil {
			row.ConflictedDetail = items[i].ConflictedInfo.ConflictedDetail
		}
		rows = append(rows, row)
	}
	return rows
}
