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
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// CreateAppInstanceWeTerm 获取应用运行实例（Pod）WeTerm 登录链接。
//
//	@ID			CreateAppInstanceWeTerm
//	@Summary	获取应用运行实例（Pod）WeTerm 登录链接
//	@Description	未配置 WeTerm 环境 ID（如社区版）时返回 NOT_IMPLEMENTED。
//	@Tags		instance
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID		path		string	true	"应用 ID"
//	@Param		envName		path		string	true	"部署环境名称"
//	@Param		instanceID	path		string	true	"实例 ID"
//	@Success	200			{object}	serializer.CreateAppInstanceWeTermOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/{instanceID}/weterm [post]
func (h *Handler) CreateAppInstanceWeTerm(c *gin.Context) {
	var uriInput serializer.AppInstanceURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	envID := config.G.BCS.WeTermEnvID
	if envID == "" {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeNotImplemented, "weterm is not supported"))
		return
	}

	ctx := c.Request.Context()
	target, err := h.resolveConsoleTarget(ctx, uriInput)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}
	weTermURL := buildWeTermURL(weTermParams{
		EnvID:         envID,
		ProjectID:     target.workspace.BkSystems.BkBCSProjectID,
		ProjectCode:   target.workspace.BkSystems.BkBCSProjectCode,
		ClusterID:     target.env.Cluster.ClusterID,
		Namespace:     target.env.Cluster.Namespace,
		PodName:       uriInput.InstanceID,
		ContainerName: target.containerName,
	})

	// 创建 WeTerm 操作记录
	go audit.AddOperationRecordAsync(
		context.WithoutCancel(ctx),
		audit.OperationTypeCreate,
		audit.ResourceTypeInstance,
		uriInput.InstanceID,
		audit.WithAttribute(audit.AttributeWeTerm),
		audit.WithWorkspaceID(target.app.WorkspaceID),
		audit.WithAppID(target.app.ID),
		audit.WithEnvName(uriInput.EnvName),
	)
	ginutils.OK(c, serializer.CreateAppInstanceWeTermOutput{
		Data: &serializer.WeTermInfoOutputObj{URL: weTermURL},
	})
}

// weTermParams 生成 WeTerm 登录容器链接所需的参数
type weTermParams struct {
	// EnvID BCS 侧分配的环境 ID（bkBcsEnvID）
	EnvID string
	// ProjectID BCS 项目 ID
	ProjectID string
	// ProjectCode BCS 项目 Code
	ProjectCode string
	// ClusterID 集群 ID
	ClusterID string
	// Namespace 命名空间
	Namespace string
	// PodName Pod 名称
	PodName string
	// ContainerName 容器名称
	ContainerName string
}

// buildWeTermURL 生成 WeTerm 登录容器的链接，由用户本地安装的 WeTerm 客户端打开，格式由 BCS 侧约定：
//
//	weterm://session/open/bcs?ns={namespace}&pod={podName}&container={containerName}&type=login
//	&clusterId={clusterId}&envId={bkBcsEnvID}&projectID={projectID}&projectCode={projectCode}
//
// 不使用 url.Values.Encode，因为它会按 key 排序，打乱 BCS 约定的参数顺序。
func buildWeTermURL(p weTermParams) string {
	query := []struct{ key, value string }{
		{"ns", p.Namespace},
		{"pod", p.PodName},
		{"container", p.ContainerName},
		{"type", "login"},
		{"clusterId", p.ClusterID},
		{"envId", p.EnvID},
		{"projectID", p.ProjectID},
		{"projectCode", p.ProjectCode},
	}
	pairs := make([]string, 0, len(query))
	for _, q := range query {
		pairs = append(pairs, q.key+"="+url.QueryEscape(q.value))
	}
	return "weterm://session/open/bcs?" + strings.Join(pairs, "&")
}
