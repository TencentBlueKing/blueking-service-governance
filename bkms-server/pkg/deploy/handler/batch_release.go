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
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
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
