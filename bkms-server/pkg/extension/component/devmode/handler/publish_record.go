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

package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	devmode "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// CreateDevModePublishRecords 上报开发模式发布结果。
// CLI 在 publish 完成后调用本接口，将每个实例（pod）的发布结果写入数据库。
//
//	@ID			CreateDevModePublishRecords
//	@Summary	上报开发模式发布结果
//	@Tags		devmode
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string							true	"应用 ID"
//	@Param		envName	path		string							true	"环境名称"
//	@Param		body	body		slz.CreatePublishRecordsInput	true	"发布结果上报请求体"
//	@Success	200		{object}	slz.CreatePublishRecordsOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Failure	404		{object}	bkerrs.GinErrorOutput
//	@Router		/devmode/{appID}/envs/{envName}/publish-records [post]
func (h *Handler) CreateDevModePublishRecords(c *gin.Context) {
	var uriInput slz.PublishRecordURIInput
	var input slz.CreatePublishRecordsInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if err := ginutils.BindJSON(c, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()
	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	operator := auth.MustGetUser(ctx).ID

	records := lo.Map(input.Results, func(result slz.PublishInstanceResult, _ int) devmode.PublishRecord {
		return devmode.PublishRecord{
			AppID:      app.ID,
			EnvName:    uriInput.EnvName,
			Instance:   result.Instance,
			BinaryName: input.BinaryName,
			FileSize:   input.FileSize,
			MD5:        input.MD5,
			Status:     devmode.PublishStatus(result.Status),
			Message:    result.Message,
			Operator:   operator,
		}
	})

	recordIDs, err := h.registry.PublishRecordStore.Create(ctx, records)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "create devmode publish records"))
		return
	}

	// 写入操作审计记录
	recordCtx := context.WithoutCancel(ctx)
	for _, result := range input.Results {
		go audit.AddOperationRecordAsync(
			recordCtx,
			audit.OperationTypePublish,
			audit.ResourceTypeInstance,
			result.Instance,
			audit.WithAttribute(audit.AttributeDevModePublish),
			audit.WithResult(publishAuditResult(result)),
			audit.WithDataAfter(map[string]any{
				"binaryName": input.BinaryName,
				"fileSize":   input.FileSize,
				"md5":        input.MD5,
				"message":    result.Message,
			}),
			audit.WithWorkspaceID(app.WorkspaceID),
			audit.WithAppID(app.ID),
			audit.WithEnvName(uriInput.EnvName),
		)
	}

	ginutils.OK(c, &slz.CreatePublishRecordsOutput{
		Data: &slz.CreatePublishRecordsData{RecordIDs: recordIDs},
	})
}

// ListDevModePublishRecords 获取开发模式发布记录列表。
// 每个实例（pod）对应一条记录，按创建时间倒序返回。
//
//	@ID			ListDevModePublishRecords
//	@Summary	获取开发模式发布记录列表
//	@Tags		devmode
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"环境名称"
//	@Param		keyword		query		string	false	"搜索关键字"
//	@Param		page		query		int		true	"分页页码（从 1 开始）"
//	@Param		pageSize	query		int		true	"分页大小"
//	@Success	200			{object}	slz.ListPublishRecordsOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Failure	404			{object}	bkerrs.GinErrorOutput
//	@Router		/devmode/{appID}/envs/{envName}/publish-records [get]
func (h *Handler) ListDevModePublishRecords(c *gin.Context) {
	var uriInput slz.PublishRecordURIInput
	var input slz.ListPublishRecordsInput
	if err := ginutils.BindURIQuery(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	ctx := c.Request.Context()

	app, _, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	records, total, err := h.registry.PublishRecordStore.List(
		ctx, app.ID, uriInput.EnvName, input.Keyword, input.Page, input.PageSize,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "list devmode publish records for app %s env %s",
			app.ID, uriInput.EnvName,
		))
		return
	}

	outputRecords := lo.Map(records, func(record devmode.PublishRecord, _ int) *slz.PublishRecordOutputObj {
		return new(slz.PublishRecordOutputObj).FromModel(record)
	})
	ginutils.OK(c, slz.ListPublishRecordsOutput{
		Data: slz.PaginatedPublishRecords{Count: total, Results: outputRecords},
	})
}

// publishAuditResult 将单个实例的发布状态映射为审计结果。
func publishAuditResult(result slz.PublishInstanceResult) audit.Result {
	if result.Status == string(devmode.PublishStatusFailed) {
		return audit.ResultFailed
	}
	return audit.ResultSuccess
}
