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
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/podquery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// ListEvents 获取指定环境的事件列表。
//
//	@ID			ListEvents
//	@Summary	获取指定环境的事件列表
//	@Tags		instance
//	@Produce	json
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID			path		string	true	"应用 ID"
//	@Param		envName			path		string	true	"部署环境名称"
//	@Param		trafficLaneName	query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param		level			query		string	false	"事件级别（可选过滤参数，可选值：Normal, Warning）"
//	@Param		startedAt		query		int		false	"起始时间戳"
//	@Param		endedAt			query		int		false	"结束时间戳"
//	@Param		page			query		int		true	"页码，从 1 开始"
//	@Param		pageSize		query		int		true	"每页数量"
//	@Success	200				{object}	serializer.ListEventsOutput
//	@Failure	400				{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/events [get]
func (h *Handler) ListEvents(c *gin.Context) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.ListEventsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()
	// 校验 App 查看权限
	app, err := perm.ValidateAppByID(ctx, h.registry, uriInput.AppID, perm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 获取最新部署记录，用于获取集群 ID、命名空间以及工作负载资源信息
	record, err := appmodeldeploy.GetLatest(
		ctx, h.registry.DB, app.ID, uriInput.EnvName, queryInput.TrafficLaneName,
	)
	if err != nil {
		// 如果部署记录不存在，则不会有事件，此时需返回空结果
		if errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound) {
			ginutils.OK(c, serializer.ListEventsOutput{
				Data: &serializer.PaginatedEventsOutputObj{Count: 0, Results: nil},
			})
			return
		}
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get latest deploy record"))
		return
	}

	// 获取环境信息，用于获取 BCS 项目 Code
	env, err := h.envs().Get(ctx, envmodel.AppEnvByName(app.WorkspaceID, app.ID, uriInput.EnvName))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get env by name"))
		return
	}

	// 选择器为空时会列出整个命名空间的 Pod，把无关事件混进来，因此必须先拦下
	if len(record.LabelSelector) == 0 {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInternalServerError, "deploy record label selector is empty"))
		return
	}
	// 工作负载事件与该次部署 Pod 的事件一并查询
	pods, err := podquery.NewPodLister(podquery.Location{
		ClusterID:     record.ClusterID,
		Namespace:     record.Namespace,
		LabelSelector: record.LabelSelector,
	}).List(ctx)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list deploy pods"))
		return
	}
	refs := append(podquery.BuildWorkloadRefs(record.ResourceKeys), podquery.BuildPodRefs(pods.Items)...)

	lister, err := podquery.NewEventLister(ctx, env)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial event lister"))
		return
	}
	paginatedEvents, err := lister.List(ctx, record.Namespace, refs, podquery.EventFilter{
		Level:     queryInput.Level,
		StartedAt: queryInput.StartedAt,
		EndedAt:   queryInput.EndedAt,
		Page:      int(queryInput.Page),
		PageSize:  int(queryInput.PageSize),
	})
	if err != nil {
		bkerrs.AbortWithErr(
			c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list events from cluster resources"),
		)
		return
	}

	// 转换为 API 响应格式
	eventEntries := make([]*serializer.EventEntryOutputObj, 0, len(paginatedEvents.Data))
	for _, e := range paginatedEvents.Data {
		eventEntries = append(eventEntries, &serializer.EventEntryOutputObj{
			ClusterID:     e.ClusterID,
			Namespace:     e.Namespace,
			Level:         e.Level,
			Content:       e.Content,
			Type:          e.Type,
			ComponentName: e.ComponentName,
			ResourceKind:  e.ResourceKind,
			ResourcesName: e.ResourcesName,
			CreatedAt:     e.CreatedAt,
		})
	}
	ginutils.OK(c, serializer.ListEventsOutput{
		Data: &serializer.PaginatedEventsOutputObj{
			Count:   paginatedEvents.Count,
			Results: eventEntries,
		},
	})
}
