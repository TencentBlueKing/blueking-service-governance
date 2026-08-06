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
	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// ListAppInstanceLogs 获取应用运行实例（Pod）日志。
//
//	@ID			ListAppInstanceLogs
//	@Summary	获取应用运行实例（Pod）日志
//	@Tags		instance
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		instanceID		path		string	true	"实例 ID"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param		previous		query		bool	false	"是否获取重启前日志"
//	@Param		tailLines		query		int		true	"日志行数（从尾部起算），最大 2000"
//	@Success	200				{object}	serializer.ListAppInstanceLogsOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/{instanceID}/logs [get]
func (h *Handler) ListAppInstanceLogs(c *gin.Context) {
	var uriInput serializer.AppInstanceURIInput
	var queryInput serializer.ListAppInstanceLogsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 校验 App 查看权限，并获取当前部署对应的集群环境
	app, env, err := perm.ValidateAppEnvByName(ctx, h.registry, uriInput.AppID, uriInput.EnvName, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	// 创建日志管理器并查询日志
	mgr, err := instancelog.NewLogManager(
		ctx,
		h.registry.AppModelDeployRecordStore,
		app,
		env,
		queryInput.TrafficLaneName,
		uriInput.InstanceID,
	)
	if err != nil {
		bkerrs.AbortWithErr(
			c, instancelog.WrapManagerError(err, uriInput.AppID, uriInput.EnvName, uriInput.InstanceID),
		)
		return
	}
	result, err := mgr.ListLogs(ctx, uriInput.InstanceID, queryInput.Previous, queryInput.TailLines)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list app instance logs"))
		return
	}

	output := make([]*serializer.LogEntryOutputObj, 0, len(result))
	for _, entry := range result {
		output = append(output, new(serializer.LogEntryOutputObj).FromModel(entry))
	}
	ginutils.OK(c, serializer.ListAppInstanceLogsOutput{Data: output})
}
