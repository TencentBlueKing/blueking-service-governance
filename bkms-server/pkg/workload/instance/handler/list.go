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
	"log/slog"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

// 以下 helper 一律只返回 error，abort 由入口 ListAppInstances 统一做
// 这样 gin 上下文不下沉到每个步骤，Watch 等其他入口才能复用同一批步骤

// bindListAppInstancesQuery 绑定路径与查询参数，并校验全量/分页互斥
func (h *Handler) bindListAppInstancesQuery(
	c *gin.Context,
) (serializer.AppEnvURIInput, serializer.ListAppInstancesQueryInput, error) {
	var uriInput serializer.AppEnvURIInput
	var queryInput serializer.ListAppInstancesQueryInput

	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		return uriInput, queryInput, err
	}

	if err := queryInput.Validate(); err != nil {
		return uriInput, queryInput, err
	}

	return uriInput, queryInput, nil
}

// validateAppAndGetDeployRecord 依次校验 App 查看权限与 AppModel 类型，再取该环境/泳道的最新部署记录
// 三步中任一步不通过都直接返回错误，不返回半成品；泳道单独传入，不依赖 List 专属的查询参数类型
func (h *Handler) validateAppAndGetDeployRecord(
	ctx context.Context,
	uriInput serializer.AppEnvURIInput,
	trafficLaneName string,
) (*bkmsapp.Application, *appmodeldeploy.Record, error) {
	// 校验 App 查看权限
	app, err := ginperm.ValidateAppByID(ctx, h.registry, uriInput.AppID, ginperm.TypeView)
	if err != nil {
		return nil, nil, err
	}

	// 目前只支持查看 AppModel 类型应用实例
	if !bkmsapp.IsAppModelType(app.Type) {
		return nil, nil, bkerrs.Errorf(bkerrs.ErrCodeInvalidArgument, "invalid app type: %s", app.Type)
	}

	// 获取应用部署记录
	record, err := h.registry.AppModelDeployRecordStore.GetLatest(ctx, app.ID, uriInput.EnvName, trafficLaneName)
	if err != nil {
		return nil, nil, bkerrs.Wrap(err, bkerrs.ErrCodeNotFound, "deploy record not found")
	}

	return app, record, nil
}

// listMatchingAppInstancePods 按部署记录的命名空间 + LabelSelector 拉取匹配 Pod
func (h *Handler) listMatchingAppInstancePods(
	ctx context.Context,
	record *appmodeldeploy.Record,
) ([]unstructured.Unstructured, error) {
	// 集群与命名空间以部署记录为准，不从请求参数推断
	client := k8sclient.NewPodClient(cluster.NewConfig(record.ClusterID))

	// 只拉该次部署写入的 LabelSelector 匹配的 Pod，后续再按全量/分页窗口投影
	labelSelector := labels.SelectorFromSet(record.LabelSelector).String()
	pods, err := client.List(ctx, record.Namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		// 拉 Pod 失败则整次 List 失败，不分页/全量都不会降级为空列表
		return nil, bkerrs.Wrapf(
			err, bkerrs.ErrCodeInternalServerError,
			"list namespace %s labelsSelector [%s] pods", record.Namespace, labelSelector,
		)
	}

	return pods.Items, nil
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

// attachPolarisToListedAppInstances 合并该应用环境北极星实例
// 北极星不可用不阻塞 Pod 输出：降级为空 polarisInfos，与「未注册北极星」同形，由前端统一展示为未知
// 没有会导致整次 List 失败的分支，因此不返回 error
func (h *Handler) attachPolarisToListedAppInstances(
	ctx context.Context,
	appID, envName string,
	appInstances []*serializer.AppInstanceOutputObj,
) {
	// 先统一写成空数组，避免 JSON 出现 polarisInfos: null；未命中北极星的实例也保持这个形态
	for _, instance := range appInstances {
		instance.PolarisInfos = []*serializer.PolarisInstanceInfoOutputObj{}
	}

	// 按应用 + 环境拉北极星实例，分页与全量共用同一条失败口径
	mgr := polaris.NewPolarisPlatformManager(
		h.registry.DepSvcStore,
		h.registry.DepSvcInstStore,
		h.registry.PolarisConfigStore,
	)

	svcInstances, err := mgr.ListPolarisServiceInstances(ctx, appID, envName)
	if err != nil {
		// 北极星是旁路信息，拉不到只降级这一列，不牵连整份 Pod 列表；仅告警供排查
		log.WarnAttrs(ctx, "list polaris service instances failed, fallback to unknown polaris state",
			slog.String("app_id", appID),
			slog.String("env_name", envName),
			slog.String("err", err.Error()),
		)

		return
	}

	// 按实例标识把北极星信息挂到已投影结果上，不改变 results 顺序与条数
	serializer.MergePolarisInfoToAppInstances(appInstances, svcInstances)
}
