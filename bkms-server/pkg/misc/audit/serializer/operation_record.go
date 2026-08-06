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

// Package serializer defines Gin input and output serializers for operation audit APIs.
package serializer

import (
	"fmt"
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

// WorkspaceURIInput is the path input for APIs scoped by workspace.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required"`
}

// ListOperationRecordsQueryInput is the query input for listing operation records.
type ListOperationRecordsQueryInput struct {
	// 可选分组参数：AppID
	AppID string `form:"appID"`
	// 可选分组参数：环境名称，如：dev，prod
	EnvName string `form:"envName"`
	// 可选过滤参数：开始时间
	StartedAt string `form:"startedAt"`
	// 可选过滤参数：结束时间
	EndedAt string `form:"endedAt"`
	// 可选过滤参数：操作类型，如：create, update, delete
	OperationType string `form:"operationType"`
	// 可选过滤参数：资源类型，如：workspace, app, env
	ResourceType string `form:"resourceType"`
	// 可选过滤参数：结果，如：success, failed
	Result string `form:"result"`
	// 可选过滤参数：操作人用户名
	Username string `form:"username"`
	// 页码，从 1 开始
	Page int64 `form:"page" binding:"required,gte=1"`
	// 每页数量，仅支持固定枚举值
	PageSize int64 `form:"pageSize" binding:"required,oneof=5 10 20 50 100"`
}

// ToListOptions converts query input into store list options.
func (i ListOperationRecordsQueryInput) ToListOptions(workspaceID string) (audit.ListOptions, error) {
	startedAt, err := parseOptionalRFC3339("startedAt", i.StartedAt)
	if err != nil {
		return audit.ListOptions{}, err
	}
	endedAt, err := parseOptionalRFC3339("endedAt", i.EndedAt)
	if err != nil {
		return audit.ListOptions{}, err
	}

	return audit.ListOptions{
		WorkspaceID: workspaceID,
		AppID:       i.AppID,
		EnvName:     i.EnvName,
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		OpType:      audit.OperationType(i.OperationType),
		ResType:     audit.ResourceType(i.ResourceType),
		Result:      audit.Result(i.Result),
		Username:    i.Username,
		Page:        i.Page,
		PageSize:    i.PageSize,
	}, nil
}

func parseOptionalRFC3339(fieldName, raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339 timestamp: %w", fieldName, err)
	}
	return parsed, nil
}

// FilterOptionOutputObj is one selectable filter option.
type FilterOptionOutputObj struct {
	Value       string `json:"value"`
	DisplayName string `json:"displayName"`
}

// OperationDataOutputObj is the before/after payload diff of one operation.
type OperationDataOutputObj struct {
	// 变更前数据
	Before string `json:"before"`
	// 变更后数据
	After string `json:"after"`
}

// OperationGroupOutputObj is the grouping metadata used for filtering.
type OperationGroupOutputObj struct {
	WorkspaceID string `json:"workspaceID"`
	AppID       string `json:"appID"`
	EnvName     string `json:"envName"`
}

// OperationRecordOutputObj is one operation audit record in list results.
type OperationRecordOutputObj struct {
	// 操作人用户名
	Username string `json:"username"`
	// 访问类型，如：web, api
	AccessType string `json:"accessType"`
	// 操作类型，如：create, update, delete
	OperationType string `json:"operationType"`
	// 操作类型展示用名称，如：创建、更新、删除
	OperationTypeDisplayName string `json:"operationTypeDisplayName"`
	// 资源类型，如：workspace, app, env
	ResourceType string `json:"resourceType"`
	// 资源类型展示用名称，如：工作空间、应用、环境
	ResourceTypeDisplayName string `json:"resourceTypeDisplayName"`
	// 资源 ID
	ResourceID string `json:"resourceID"`
	// 资源属性，如：name
	Attribute string `json:"attribute"`
	// 资源属性展示用名称，如：名称
	AttributeDisplayName string `json:"attributeDisplayName"`
	// 操作结果，如：success, failed
	Result string `json:"result"`
	// 操作数据（前后数据对比）
	Data *OperationDataOutputObj `json:"data"`
	// 操作分组（聚合数据，用于关联到特定的分类）
	Group *OperationGroupOutputObj `json:"group"`
	// 操作时间
	CreatedAt time.Time `json:"createdAt"`
}

// FromModel fills output fields from an operation record model.
func (o *OperationRecordOutputObj) FromModel(record audit.OperationRecord) *OperationRecordOutputObj {
	*o = OperationRecordOutputObj{
		Username:                 record.Username,
		AccessType:               string(record.AccessType),
		OperationType:            string(record.OperationType),
		OperationTypeDisplayName: record.OperationType.DisplayName(),
		ResourceType:             string(record.ResourceType),
		ResourceTypeDisplayName:  record.ResourceType.DisplayName(),
		ResourceID:               record.ResourceID,
		Attribute:                string(record.Attribute),
		AttributeDisplayName:     record.Attribute.DisplayName(),
		Result:                   string(record.Result),
		Data: &OperationDataOutputObj{
			Before: string(record.Data.Before),
			After:  string(record.Data.After),
		},
		Group: &OperationGroupOutputObj{
			WorkspaceID: record.Group.WorkspaceID,
			AppID:       record.Group.AppID,
			EnvName:     record.Group.EnvName,
		},
		CreatedAt: record.CreatedAt,
	}
	return o
}

// PaginatedOperationRecordOutputObj is the paginated list payload.
type PaginatedOperationRecordOutputObj struct {
	// 结果数量
	Count int64 `json:"count,string"`
	// 查询结果
	Results []*OperationRecordOutputObj `json:"results"`
}

// ListOperationRecordsOutput is the JSON response for listing operation records.
type ListOperationRecordsOutput struct {
	Data *PaginatedOperationRecordOutputObj `json:"data"`
}

// OperationRecordFilterOptionsOutputObj contains all filter option groups.
type OperationRecordFilterOptionsOutputObj struct {
	// 资源类型选项
	ResourceTypes []*FilterOptionOutputObj `json:"resourceTypes"`
	// 操作类型选项
	OperationTypes []*FilterOptionOutputObj `json:"operationTypes"`
	// 操作结果选项
	OperationResults []*FilterOptionOutputObj `json:"operationResults"`
}

// ListOperationRecordFilterOptionsOutput is the JSON response for listing filter options.
type ListOperationRecordFilterOptionsOutput struct {
	Data *OperationRecordFilterOptionsOutputObj `json:"data"`
}
