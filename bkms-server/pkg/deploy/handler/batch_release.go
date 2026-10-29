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
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// GetBatchReleaseProgress 获取应用模型应用在环境 / 泳道下最新一次部署的分批发布进度
//
//	@ID			GetBatchReleaseProgress
//	@Summary	获取分批发布进度
//	@Description	返回环境 / 泳道最新部署记录的分批发布进度，进度由部署轮询任务写入，不实时请求集群；
//	@Description	没有部署记录或最新记录不是分批发布时 data 为 null
//	@Tags		deploy
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Success	200				{object}	serializer.GetBatchReleaseProgressOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/batch-release [get]
func (h *Handler) GetBatchReleaseProgress(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.GetBatchReleaseProgressInput
	if err := ginutils.BindURIQuery(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, _, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeView, false)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	record, err := appmodeldeploy.GetLatest(ctx, h.registry.DB, app.ID, uriInput.EnvName, input.TrafficLaneName)
	if errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
		ginutils.OK(c, serializer.GetBatchReleaseProgressOutput{})
		return
	}
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError, "get latest deploy record for %s",
			genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, input.TrafficLaneName),
		))
		return
	}
	if record.GetReleaseMode() != appmodeldeploy.ReleaseModeCanary {
		ginutils.OK(c, serializer.GetBatchReleaseProgressOutput{})
		return
	}
	ginutils.OK(c, serializer.GetBatchReleaseProgressOutput{
		Data: new(serializer.BatchReleaseProgressOutputObj).FromModel(record),
	})
}

// OperateBatchRelease 对应用模型应用在环境 / 泳道下进行中的分批发布执行手动操作
//
//	@ID			OperateBatchRelease
//	@Summary	分批发布手动操作
//	@Description	对环境 / 泳道最新部署记录对应的进行中分批发布执行继续、暂停、回退上一批、跳过剩余批次或回滚，
//	@Description	权限同部署。前置条件按集群中 Rollout 的实时状态校验；不满足前置条件、或没有进行中的分批发布时返回 400
//	@Tags		deploy
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID	path		string								true	"应用 ID"
//	@Param		envName	path		string								true	"部署环境名称"
//	@Param		body	body		serializer.OperateBatchReleaseInput	true	"分批发布手动操作请求"
//	@Success	200		{object}	serializer.EmptyOutput
//	@Failure	400		{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/batch-release/operations [post]
func (h *Handler) OperateBatchRelease(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var input serializer.OperateBatchReleaseInput
	if err := ginutils.BindURIJSON(c, &uriInput, &input); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	app, _, err := h.validateAppModelDeployAppEnv(ctx, uriInput.AppID, uriInput.EnvName, perm.TypeEdit, true)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	record, err := appmodeldeploy.OperateBatchRelease(
		ctx, h.registry.DB, app.ID, uriInput.EnvName, input.TrafficLaneName, input.Action,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, batchrelease.APIError(err, fmt.Sprintf(
			"%s batch release of %s", input.Action,
			genDeployInfo(app.WorkspaceID, app.ID, uriInput.EnvName, input.TrafficLaneName),
		)))
		return
	}

	recordBatchReleaseActionAudit(ctx, app, uriInput.EnvName, input.TrafficLaneName, input.Action, record)
	ginutils.OK(c, serializer.EmptyOutput{})
}

// batchReleaseActionOperationTypes 分批发布手动操作对应的操作记录类型
var batchReleaseActionOperationTypes = map[batchrelease.Action]audit.OperationType{
	batchrelease.ActionContinue: audit.OperationTypeContinue,
	batchrelease.ActionPause:    audit.OperationTypePause,
	batchrelease.ActionBackward: audit.OperationTypeBackward,
	batchrelease.ActionSkip:     audit.OperationTypeSkip,
	batchrelease.ActionRollback: audit.OperationTypeRollback,
}

// recordBatchReleaseActionAudit 记录分批发布手动操作的操作记录，只在修改 Rollout 成功后调用，
// 通过 extras 中的 deployID 与该次分批发布的下发、终态记录关联
func recordBatchReleaseActionAudit(
	ctx context.Context,
	app *bkmsapp.Application,
	envName, trafficLaneName string,
	action batchrelease.Action,
	record *appmodeldeploy.Record,
) {
	strategy := lo.FromPtr(record.ReleaseStrategy)
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		batchReleaseActionOperationTypes[action],
		audit.ResourceTypeApp,
		app.ID,
		audit.WithResult(audit.ResultSuccess),
		audit.WithAttribute(audit.AttributeBatchRelease),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(envName),
		audit.WithExtras(map[string]string{
			"trafficLaneName":     trafficLaneName,
			"deployID":            record.ID.Hex(),
			"releaseStrategyID":   strategy.ID,
			"releaseStrategyName": strategy.Name,
		}),
	)
}
