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

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/watch"
)

// WatchAppInstances 订阅应用实例投影变更（SSE）
// 本函数只做编排：绑定参数、鉴权取部署记录、从续传位点起推增量
//
// 只推增量，不推首包快照；resourceVersion 必填，来自 List 成功响应
// 鉴权 / 非 AppModel / 无部署记录：与 List 同一套拒绝，建不成连接
// 集群 Watch 建立失败：位点过期返回 409，其他返回 500；已成流后中断则先推 ENDED 再关连接
// MODIFIED 有两个来源：集群 Pod 变更，以及连接期内北极星周期补拉，二者形态一致
//
//	@ID			WatchAppInstances
//	@Summary	订阅应用实例投影变更
//	@Description	SSE 推送 ADDED/MODIFIED/DELETED/ENDED；DELETED 只保证 id，ENDED 时 object 为 null。
//	@Description	MODIFIED 有两个来源：集群 Pod 变更，以及北极星周期补拉（约 15s 一轮，仅 polarisInfos 变化时推），二者形态一致。
//	@Description	北极星拉取失败不阻塞推送：polarisInfos 为空数组，与未注册北极星同形，K8s 字段照常推。
//
//	@Tags		instance
//	@Produce	text/event-stream
//	@Security	BkUserInfo
//	@Security	BkUserCredential
//	@Param		appID				path		string	true	"应用 ID"
//	@Param		envName				path		string	true	"部署环境名称"
//	@Param		trafficLaneName		query		string	false	"部署的泳道名称（空字符串表示不使用泳道）"
//	@Param		resourceVersion		query		string	true	"List 成功响应带回的续传位点"
//	@Success	200					{string}	string	"SSE event stream"
//	@Failure	400					{object}	bkerrs.GinErrorOutput
//	@Failure	500					{object}	bkerrs.GinErrorOutput
//	@Router		/apps/{appID}/envs/{envName}/instances/watch [get]
func (h *Handler) WatchAppInstances(c *gin.Context) {
	// 绑定路径与查询参数；resourceVersion 不用 binding，由 Validate 给出明确错误
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.WatchAppInstancesQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 续传位点必填；缺则参数错误，建不成 Watch
	if err := queryInput.Validate(); err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	ctx := c.Request.Context()

	// 校验 App 查看权限与 AppModel 类型，并取该环境/泳道最新部署记录
	// 与 List 同一套拒绝：鉴权失败 / 非 AppModel / 无部署记录都建不成连接、无事件
	app, record, err := h.validateAppAndGetDeployRecord(ctx, uriInput, queryInput.TrafficLaneName)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return
	}

	// 作用域与 List 一致：部署记录上的集群 / 命名空间 / LabelSelector
	// 从 List 带回的 resourceVersion 起只推增量，不推首包快照
	// 已成流后的集群中断由 Manager 推 ENDED 并收流，这里只处理未成流的失败
	err = h.newWatchManager(record.ClusterID, app.ID, record.EnvName).Run(ctx, c.Writer, watch.RunParams{
		Namespace:       record.Namespace,
		LabelSelector:   labels.SelectorFromSet(record.LabelSelector).String(),
		ResourceVersion: queryInput.ResourceVersion,
		DeployID:        record.ID.Hex(),
	})
	if err != nil {
		// 位点过期必须与其他集群故障分开：前端据此重新 List，而不是当 500 重试旧位点
		if errors.Is(err, watch.ErrResourceVersionGone) {
			bkerrs.AbortWithErr(c, bkerrs.Wrap(
				err, bkerrs.ErrCodeAborted, "resourceVersion expired, re-list required",
			))
			return
		}

		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "watch app instances"))
	}
}

// newWatchManager 绑定目标集群的 Pod Watch 与该应用环境的北极星拉取
// 北极星拉取失败由 Manager 降级为空数组，不拆流
func (h *Handler) newWatchManager(clusterID, appID, envName string) *watch.Manager {
	return watch.NewManager(
		k8sclient.NewPodClient(cluster.NewConfig(clusterID)),
		func(ctx context.Context) ([]*polaris.PolarisServiceInstances, error) {
			mgr := polaris.NewPolarisPlatformManager(
				h.registry.DepSvcStore,
				h.registry.DepSvcInstStore,
				h.registry.PolarisConfigStore,
			)
			return mgr.ListPolarisServiceInstances(ctx, appID, envName)
		},
	)
}
