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
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"

	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// ListBCSAuthorizedProjects 获取有权限的 BCS 项目列表
//
//	@ID			ListBCSAuthorizedProjects
//	@Summary	获取有权限的 BCS 项目列表
//	@Tags		bkintegrations-bcs
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Success	200	{object}	serializer.ListBCSAuthorizedProjectsOutput
//	@Failure	400	{object}	bkerrs.GinErrorOutput
//	@Router		/bcs/projects/authorized [get]
func (h *Handler) ListBCSAuthorizedProjects(c *gin.Context) {
	ctx := c.Request.Context()
	client, err := bcs.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bcs client"))
		return
	}

	bcsProjects, err := client.ListAuthorizedProjects(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bcs authorized projects"))
		return
	}

	bkciClient, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return
	}
	bkciProjects, err := bkciClient.ListProjects(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list bkci managed projects"))
		return
	}
	bkciProjects = lo.Filter(bkciProjects, func(item bkci.Project, _ int) bool {
		return item.HasManagePerm
	})
	bkciProjectsMap := lo.SliceToMap(bkciProjects, func(item bkci.Project) (string, bool) {
		return item.Code, true
	})

	workspaces, err := h.registry.WorkspaceStore.List(ctx, &workspace.ListOptions{})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list workspaces"))
		return
	}
	boundProjectsMap := lo.SliceToMap(workspaces, func(item workspace.Workspace) (string, bool) {
		return item.BkSystems.BkBCSProjectCode, true
	})

	projects := lo.Filter(bcsProjects, func(item bcs.Project, _ int) bool {
		return item.Kind == "k8s" && bkciProjectsMap[item.Code]
	})

	ginutils.OK(
		c,
		&slz.ListBCSAuthorizedProjectsOutput{
			Data: lo.Map(projects, func(item bcs.Project, _ int) *slz.BCSProjectOutput {
				return &slz.BCSProjectOutput{
					ID:               item.ID,
					Name:             item.Name,
					Code:             item.Code,
					Description:      item.Description,
					BizID:            item.BizID,
					IsOffline:        item.IsOffline,
					IsBoundWorkspace: boundProjectsMap[item.Code],
				}
			}),
		},
	)
}

// GetBCSProject 根据项目 ID 获取项目详情
//
//	@ID			GetBCSProject
//	@Summary	根据项目 ID 获取 BCS 项目详情
//	@Tags		bkintegrations-bcs
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		projectID	path		string	true	"BCS 项目 ID"
//	@Success	200			{object}	serializer.GetBCSProjectOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/bcs/projects/{projectID} [get]
func (h *Handler) GetBCSProject(c *gin.Context) {
	var uriInput slz.BCSProjectURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bcs.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bcs client"))
		return
	}

	project, err := client.GetProject(ctx, uriInput.ProjectID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get bcs project"))
		return
	}

	projectObj := new(slz.BCSProjectOutput)
	if err = mapstructure.Decode(project, projectObj); err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "decode bcs project"))
		return
	}

	ginutils.OK(c, &slz.GetBCSProjectOutput{Data: projectObj})
}

// ListClustersByProject 获取项目下的集群列表
//
//	@ID			ListClustersByProject
//	@Summary	获取 BCS 项目下的集群列表
//	@Tags		bkintegrations-bcs
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		projectID	path		string	true	"BCS 项目 ID"
//	@Success	200			{object}	serializer.ListClustersByProjectOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/bcs/projects/{projectID}/clusters [get]
func (h *Handler) ListClustersByProject(c *gin.Context) {
	var uriInput slz.BCSProjectClustersURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bcs.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bcs client"))
		return
	}

	clusters, err := client.ListClustersByProject(ctx, uriInput.ProjectID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list clusters by bcs project"))
		return
	}

	ginutils.OK(
		c,
		&slz.ListClustersByProjectOutput{
			Data: lo.Map(clusters, func(item bcs.Cluster, _ int) *slz.ClusterOutput {
				return &slz.ClusterOutput{
					ID:          item.ID,
					Name:        item.Name,
					Type:        item.Type,
					Environment: item.Environment,
					IsShared:    item.IsShared,
					Description: item.Description,
					Status:      item.Status,
				}
			}),
		},
	)
}

// ListNamespacesByCluster 获取集群下的命名空间列表
//
//	@ID			ListNamespacesByCluster
//	@Summary	获取 BCS 集群下的命名空间列表
//	@Tags		bkintegrations-bcs
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		projectID	path		string	true	"BCS 项目 ID"
//	@Param		clusterID	path		string	true	"集群 ID"
//	@Success	200			{object}	serializer.ListNamespacesByClusterOutput
//	@Failure	400			{object}	bkerrs.GinErrorOutput
//	@Router		/bcs/projects/{projectID}/clusters/{clusterID}/namespaces [get]
func (h *Handler) ListNamespacesByCluster(c *gin.Context) {
	var uriInput slz.BCSClusterNamespacesURIInput
	if err := ginutils.BindURI(c, &uriInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	client, err := bcs.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bcs client"))
		return
	}

	namespaces, err := client.ListNamespacesByCluster(ctx, uriInput.ProjectID, uriInput.ClusterID)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list namespaces by cluster"))
		return
	}

	ginutils.OK(
		c,
		&slz.ListNamespacesByClusterOutput{
			Data: lo.Map(namespaces, func(item bcs.Namespace, _ int) *slz.NamespaceOutput {
				return &slz.NamespaceOutput{
					Name:   item.Name,
					Status: item.Status,
				}
			}),
		},
	)
}
