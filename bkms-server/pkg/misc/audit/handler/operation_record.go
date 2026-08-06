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

// Package handler contains Gin handlers for operation audit APIs.
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

var _ audit.Handler = (*Handler)(nil)

// Handler handles Gin operation audit API requests.
type Handler struct {
	registry *storereg.Registry
}

// New creates a Handler.
func New(registry *storereg.Registry) *Handler {
	return &Handler{registry: registry}
}

// ListOperationRecords 获取操作审计记录列表
//
//	@ID				ListOperationRecords
//	@Summary		获取操作审计记录列表
//	@Tags			operation-audit
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID		path		string	true	"工作空间 ID"
//	@Param			appID			query		string	false	"可选分组参数：AppID"
//	@Param			envName			query		string	false	"可选分组参数：环境名称，如：dev，prod"
//	@Param			startedAt		query		string	false	"可选过滤参数：开始时间，RFC3339"
//	@Param			endedAt			query		string	false	"可选过滤参数：结束时间，RFC3339"
//	@Param			operationType	query		string	false	"可选过滤参数：操作类型，如：create, update, delete"
//	@Param			resourceType	query		string	false	"可选过滤参数：资源类型，如：workspace, app, env"
//	@Param			result			query		string	false	"可选过滤参数：结果，如：success, failed"
//	@Param			username		query		string	false	"可选过滤参数：操作人用户名"
//	@Param			page			query		int		true	"分页参数：页码，从 1 开始"
//	@Param			pageSize		query		int		true	"分页参数：每页数量，支持 5/10/20/50/100"
//	@Success		200				{object}	serializer.ListOperationRecordsOutput
//	@Failure		400				{object}	bkerrs.GinErrorOutput
//	@Failure		404				{object}	bkerrs.GinErrorOutput
//	@Failure		500				{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/operation-records [get]
func (h *Handler) ListOperationRecords(c *gin.Context) {
	var uriInput serializer.WorkspaceURIInput
	var queryInput serializer.ListOperationRecordsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	if _, err := perm.ValidateWorkspaceByID(ctx, h.registry, uriInput.WorkspaceID, perm.TypeView); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	opts, err := queryInput.ToListOptions(uriInput.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "parse query request"))
		return
	}

	records, total, err := h.registry.OperationRecordStore.List(ctx, opts)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list operation records"))
		return
	}

	results := make([]*serializer.OperationRecordOutputObj, 0, len(records))
	for _, record := range records {
		results = append(results, new(serializer.OperationRecordOutputObj).FromModel(record))
	}

	ginutils.OK(c, serializer.ListOperationRecordsOutput{
		Data: &serializer.PaginatedOperationRecordOutputObj{
			Count:   total,
			Results: results,
		},
	})
}

// ListOperationRecordFilterOptions 获取操作记录筛选选项
//
//	@ID				ListOperationRecordFilterOptions
//	@Summary		获取操作记录筛选选项
//	@Tags			operation-audit
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Success		200	{object}	serializer.ListOperationRecordFilterOptionsOutput
//	@Failure		500	{object}	bkerrs.GinErrorOutput
//	@Router			/operation-records/filter-options [get]
func (h *Handler) ListOperationRecordFilterOptions(c *gin.Context) {
	// 可选的资源类型
	resourceTypes := make([]*serializer.FilterOptionOutputObj, 0, len(audit.AllResourceTypes))
	for _, resourceType := range audit.AllResourceTypes {
		resourceTypes = append(resourceTypes, &serializer.FilterOptionOutputObj{
			Value:       string(resourceType),
			DisplayName: resourceType.DisplayName(),
		})
	}
	// 可选的操作类型
	operationTypes := make([]*serializer.FilterOptionOutputObj, 0, len(audit.AllOperationTypes))
	for _, operationType := range audit.AllOperationTypes {
		operationTypes = append(operationTypes, &serializer.FilterOptionOutputObj{
			Value:       string(operationType),
			DisplayName: operationType.DisplayName(),
		})
	}
	// 可选的操作结果
	operationResults := make([]*serializer.FilterOptionOutputObj, 0, len(audit.AllOperationResults))
	for _, result := range audit.AllOperationResults {
		operationResults = append(operationResults, &serializer.FilterOptionOutputObj{
			Value:       string(result),
			DisplayName: result.DisplayName(),
		})
	}

	ginutils.OK(c, serializer.ListOperationRecordFilterOptionsOutput{
		Data: &serializer.OperationRecordFilterOptionsOutputObj{
			ResourceTypes:    resourceTypes,
			OperationTypes:   operationTypes,
			OperationResults: operationResults,
		},
	})
}
