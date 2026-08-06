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

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
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
	record, err := h.registry.AppModelDeployRecordStore.GetLatest(
		ctx, app.ID, uriInput.EnvName, queryInput.TrafficLaneName,
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
	env, err := h.registry.EnvStore.GetByName(ctx, app.WorkspaceID, app.ID, uriInput.EnvName)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "get env by name"))
		return
	}

	// 从部署记录的 ResourceKeys 中提取工作负载的 Kind 和 Name
	var resourceKinds, resourceNames []string
	for _, rk := range record.ResourceKeys {
		resourceKinds = append(resourceKinds, rk.Kind)
		resourceNames = append(resourceNames, rk.Name)
	}
	// 通过部署记录的 LabelSelector 从集群获取关联的 Pod，将 Pod 事件一并查询
	if len(record.LabelSelector) == 0 {
		bkerrs.AbortWithErr(c, bkerrs.New(bkerrs.ErrCodeInternalServerError, "deploy record label selector is empty"))
		return
	}
	podClient := k8sclient.NewPodClient(cluster.NewConfig(record.ClusterID))
	labelSelector := labels.SelectorFromSet(record.LabelSelector).String()
	pods, err := podClient.List(ctx, record.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError,
			"list namespace %s labelSelector [%s] pods", record.Namespace, labelSelector,
		))
		return
	}
	for _, pod := range pods.Items {
		podName := mapx.GetStr(pod.Object, "metadata.name")
		if podName != "" {
			resourceKinds = append(resourceKinds, k8skind.Po)
			resourceNames = append(resourceNames, podName)
		}
	}

	// 创建 ClusterResources API 客户端
	client, err := clusterresources.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial cluster resources client"))
		return
	}
	// 调用 ClusterResources API 获取事件列表
	paginatedEvents, err := client.ListEvents(
		ctx,
		env.Cluster.ProjectCode,
		env.Cluster.ClusterID,
		clusterresources.ListEventParams{
			Namespace:     record.Namespace,
			ResourceKinds: resourceKinds,
			ResourceNames: resourceNames,
			Level:         queryInput.Level,
			StartedAt:     queryInput.StartedAt,
			EndedAt:       queryInput.EndedAt,
			Page:          int(queryInput.Page),
			PageSize:      int(queryInput.PageSize),
		},
	)
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
