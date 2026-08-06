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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// CreateAppInstanceWebConsole 创建应用运行实例（Pod）WebConsole。
//
//	@ID			CreateAppInstanceWebConsole
//	@Summary	创建应用运行实例（Pod）WebConsole
//	@Tags		instance
//	@Accept		json
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string										true	"应用 ID"
//	@Param		envName		path		string										true	"部署环境名称"
//	@Param		instanceID	path		string										true	"实例 ID"
//	@Param		body		body		serializer.CreateAppInstanceWebConsoleInput	false	"创建 WebConsole 请求"
//	@Success	200			{object}	serializer.CreateAppInstanceWebConsoleOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/{instanceID}/web-console [post]
func (h *Handler) CreateAppInstanceWebConsole(c *gin.Context) {
	var uriInput serializer.AppInstanceURIInput
	var input serializer.CreateAppInstanceWebConsoleInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	if c.Request.ContentLength > 0 {
		if err := ginutils.BindJSON(c, &input); err != nil {
			bkerrs.AbortWithErr(c, err)
			return
		}
	}
	// 兼容 v1 请求体里的 trafficLaneName 字段；创建 WebConsole 当前不需要泳道信息。
	_ = input.TrafficLaneName

	ctx := c.Request.Context()
	// 校验 App 编辑权限
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	workspace, err := h.registry.WorkspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get workspace failed"))
		return
	}
	client, err := bcs.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bcs client"))
		return
	}
	// 获取当前部署对应的集群环境
	env, err := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get env by name"))
		return
	}

	containerName := defaults.WorkloadMainContainerName
	// 非 AppModel 类应用主容器名称不一定是 main
	if !bkmsapp.IsAppModelType(app.Type) {
		podClient := k8sclient.NewPodClient(cluster.NewConfig(env.Cluster.ClusterID))
		containerName, err = podClient.GetFirstContainerName(ctx, env.Cluster.Namespace, uriInput.InstanceID)
		if err != nil {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get first container name"))
			return
		}
	}
	webConsoleURL, err := client.CreateWebConsole(
		ctx,
		workspace.BkSystems.BkBCSProjectCode,
		env.Cluster.ClusterID,
		env.Cluster.Namespace,
		uriInput.InstanceID,
		containerName,
		// 目前固定使用 sh 作为 web console 命令
		"/bin/sh",
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get web console url"))
		return
	}

	// 创建 WebConsole 操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeCreate,
		audit.ResourceTypeInstance,
		uriInput.InstanceID,
		audit.WithAttribute(audit.AttributeWebConsole),
		audit.WithWorkspaceID(app.WorkspaceID),
		audit.WithAppID(app.ID),
		audit.WithEnvName(uriInput.EnvName),
	)
	ginutils.OK(c, serializer.CreateAppInstanceWebConsoleOutput{
		Data: &serializer.WebConsoleInfoOutputObj{URL: webConsoleURL},
	})
}
