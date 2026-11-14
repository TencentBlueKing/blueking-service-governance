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
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
)

// ListWorkspaceClusterAddons 查询空间集群的集群插件列表及各插件在集群上的安装状态。
//
//	@ID				ListWorkspaceClusterAddons
//	@Summary		查询空间集群的集群插件及安装状态
//	@Description	返回适用于该集群的全部插件（含未安装的），每项带安装状态、当前 chart 版本与 values。
//	@Description	集群必须是空间绑定的 BCS 项目下的非共享集群
//	@Tags			cluster-addon
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string	true	"工作空间 ID"
//	@Param			clusterID	path		string	true	"BCS 集群 ID"
//	@Param			namespace	query		string	false	"命名空间，默认为插件定义中的 defaultNamespace"
//	@Success		200			{object}	serializer.ListClusterAddonsOutput
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/clusters/{clusterID}/cluster-addons [get]
func (h *Handler) ListWorkspaceClusterAddons(c *gin.Context) {
	var uriInput serializer.WorkspaceClusterURIInput
	var queryInput serializer.ListClusterAddonsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if _, err := h.validateWorkspaceCluster(
		c.Request.Context(), uriInput.WorkspaceID, uriInput.ClusterID, perm.TypeView,
	); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	h.respondAddonList(c, clusteraddon.TargetOfCluster(uriInput.ClusterID), queryInput.Namespace)
}

// UpsertWorkspaceClusterAddon 在空间集群上部署/更新集群插件。
//
//	@ID				UpsertWorkspaceClusterAddon
//	@Summary		在空间集群上部署/更新集群插件
//	@Description	集群必须是空间绑定的 BCS 项目下的非共享集群，可以没有环境。
//	@Description	与环境维度的接口共用安装锁，同一集群命名空间内的同一插件同时只允许一个安装或卸载操作。
//	@Tags			cluster-addon
//	@Accept			json
//	@Produce		json
//	@Security		BkUserInfo
//	@Security		BkUserCredential
//	@Param			workspaceID	path		string								true	"工作空间 ID"
//	@Param			clusterID	path		string								true	"BCS 集群 ID"
//	@Param			addonName	path		string								true	"插件名称"
//	@Param			body		body		serializer.UpsertClusterAddonInput	true	"请求体"
//	@Success		201			{object}	nil
//	@Failure		400			{object}	bkerrs.GinErrorOutput
//	@Router			/workspaces/{workspaceID}/clusters/{clusterID}/cluster-addons/{addonName} [post]
func (h *Handler) UpsertWorkspaceClusterAddon(c *gin.Context) {
	var uriInput serializer.WorkspaceClusterAddonURIInput
	var jsonInput serializer.UpsertClusterAddonInput
	if err := ginutils.BindURIJSON(c, &uriInput, &jsonInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	ws, err := h.validateWorkspaceCluster(ctx, uriInput.WorkspaceID, uriInput.ClusterID, perm.TypeEdit)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	if err = h.upsertAddon(
		ctx, clusteraddon.TargetOfCluster(uriInput.ClusterID), uriInput.AddonName, jsonInput,
	); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	go audit.AddOperationRecordAsync(
		c.Request.Context(),
		audit.OperationTypeDeploy,
		audit.ResourceTypeClusterAddon,
		uriInput.AddonName,
		audit.WithDataAfter(map[string]any{
			"clusterID":    uriInput.ClusterID,
			"chartVersion": jsonInput.ChartVersion,
			"values":       jsonInput.Values,
		}),
		audit.WithWorkspaceID(ws.ID),
	)

	ginutils.JSON(c, 201, nil)
}

// validateWorkspaceCluster 校验空间权限，并校验集群是空间绑定的 BCS 项目下的非共享集群，返回的错误已包装为 API 错误
func (h *Handler) validateWorkspaceCluster(
	ctx context.Context,
	workspaceID, clusterID string,
	permType perm.Type,
) (*workspace.Workspace, error) {
	ws, err := perm.ValidateWorkspaceByID(ctx, h.registry, workspaceID, permType)
	if err != nil {
		return nil, err
	}
	if _, err = clusteraddon.GetWorkspaceCluster(ctx, ws.BkSystems.BkBCSProjectID, clusterID); err != nil {
		if errors.Is(err, clusteraddon.ErrClusterNotInWorkspace) {
			return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeNotFound, "cluster %s not found in workspace %s",
				clusterID, workspaceID)
		}
		return nil, bkerrs.Wrapf(err, bkerrs.ErrCodeInternalServerError, "get cluster %s of workspace %s",
			clusterID, workspaceID)
	}
	return ws, nil
}
