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

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// bindListAppInstancesQuery 绑定路径与查询参数，并校验全量/分页互斥
func (h *Handler) bindListAppInstancesQuery(
	c *gin.Context,
) (serializer.AppEnvURIInput, serializer.ListAppInstancesQueryInput, bool) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.ListAppInstancesQueryInput

	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return uriInput, queryInput, false
	}

	if err := queryInput.Validate(); err != nil {
		bkerrs.AbortWithErr(c, err)
		return uriInput, queryInput, false
	}

	return uriInput, queryInput, true
}

// getAppAndLatestDeployRecord 校验 App 查看权限、AppModel 类型，并取最新部署记录
func (h *Handler) getAppAndLatestDeployRecord(
	c *gin.Context,
	ctx context.Context,
	uriInput serializer.AppEnvURIInput,
	queryInput serializer.ListAppInstancesQueryInput,
) (*bkmsapp.Application, *appmodeldeploy.Record, bool) {
	// 校验 App 查看权限
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		bkerrs.AbortWithErr(c, err)
		return nil, nil, false
	}

	// 目前只支持查看 AppModel 类型应用实例
	if !bkmsapp.IsAppModelType(app.Type) {
		bkerrs.AbortWithErr(c, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid app type: %s", app.Type))
		return nil, nil, false
	}

	// 获取应用部署记录
	record, err := h.registry.AppModelDeployRecordStore.GetLatest(
		ctx, app.ID, uriInput.EnvName, queryInput.TrafficLaneName,
	)
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "deploy record not found"))
		return nil, nil, false
	}

	return app, record, true
}

// listMatchingAppInstancePods 按部署记录的命名空间 + LabelSelector 拉取匹配 Pod
func (h *Handler) listMatchingAppInstancePods(
	c *gin.Context,
	ctx context.Context,
	record *appmodeldeploy.Record,
) ([]unstructured.Unstructured, bool) {
	// 集群与命名空间以部署记录为准，不从请求参数推断
	client := k8sclient.NewPodClient(cluster.NewConfig(record.ClusterID))

	// 只拉该次部署写入的 LabelSelector 匹配的 Pod，后续再按全量/分页窗口投影
	labelSelector := labels.SelectorFromSet(record.LabelSelector).String()
	pods, err := client.List(ctx, record.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		// 拉 Pod 失败则整次 List 失败，不分页/全量都不会降级为空列表
		bkerrs.AbortWithErr(c, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError,
			"list namespace %s labelsSelector [%s] pods", record.Namespace, labelSelector,
		))
		return nil, false
	}

	return pods.Items, true
}

// projectListedAppInstances 将窗口内 Pod 投影为 AppInstanceOutputObj
// 全量模式投影全部，单个 Pod 无法投影时跳过并记入 skipped
// 分页模式只解析当前页，解析失败则整次请求失败（与改造前一致，避免 CLI 中断）
func (h *Handler) projectListedAppInstances(
	queryInput serializer.ListAppInstancesQueryInput,
	items []unstructured.Unstructured,
	deployID string,
) (
	results []*serializer.AppInstanceOutputObj,
	skipped []*serializer.SkippedAppInstanceObj,
	count int64,
	err error,
) {
	total := int64(len(items))

	// 全量投影 [0,total)，分页只投影当前页；越界时 start/end 会被截到 total
	start, end := queryInput.ProjectionRange(total)
	results = make([]*serializer.AppInstanceOutputObj, 0, end-start)

	// 预分配空切片，无跳过时响应仍是 [] 而不是 null
	skipped = make([]*serializer.SkippedAppInstanceObj, 0)

	for i := start; i < end; i++ {
		p := items[i]
		instance, pErr := new(serializer.AppInstanceOutputObj).FromPodManifest(p.Object, deployID)
		if pErr != nil {
			podName := mapx.GetStr(p.Object, "metadata.name")

			// 分页必须与改造前一致：当前页任一 Pod 解析失败则整次 500，避免 CLI 中断
			if !queryInput.All {
				return nil, nil, 0, bkerrs.Wrapf(pErr, bkerrs.ErrCodeInternalServerError, "parse pod %s", podName)
			}

			// 全量不因单个坏 Pod 失败整次请求，记入 skipped 后继续
			skipped = append(skipped, &serializer.SkippedAppInstanceObj{ID: podName, Reason: pErr.Error()})
			continue
		}

		results = append(results, instance)
	}

	// 分页 count 是 LabelSelector 匹配总数；全量 count 是成功投影数，不含跳过项
	count = total
	if queryInput.All {
		count = int64(len(results))
	}

	return results, skipped, count, nil
}

// attachPolarisToListedAppInstances 合并该应用环境北极星实例；拉取失败则整次 List 失败
func (h *Handler) attachPolarisToListedAppInstances(
	c *gin.Context,
	ctx context.Context,
	appID, envName string,
	appInstances []*serializer.AppInstanceOutputObj,
) bool {
	// 按应用 + 环境拉北极星实例，分页与全量共用同一条失败口径
	mgr := polaris.NewPolarisPlatformManager(
		h.registry.DepSvcStore,
		h.registry.DepSvcInstStore,
		h.registry.PolarisConfigStore,
	)

	svcInstances, err := mgr.ListPolarisServiceInstances(ctx, appID, envName)
	if err != nil {
		// 北极星失败不降级为空 polarisInfos，整次 List 失败
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "list polaris service instances"))
		return false
	}

	// 按实例标识把北极星信息挂到已投影结果上，不改变 results 顺序与条数
	serializer.MergePolarisInfoToAppInstances(appInstances, svcInstances)
	return true
}
