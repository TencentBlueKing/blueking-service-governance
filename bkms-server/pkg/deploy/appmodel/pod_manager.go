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

package appmodel

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/TencentBlueKing/gopkg/collection/set"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// podManager Pod 管理器，负责 Pod 的查询、分类和删除操作
type podManager struct{}

// SetPodDeletionCost 批量设置 Pod 的 deletion cost 值
// deletion cost 值较小的 Pod 会被优先更新/删除
func (m *podManager) SetPodDeletionCost(
	ctx context.Context,
	clusterID, namespace string,
	podNames []string,
	deletionCost int64,
) error {
	// 构建 JSON Patch
	patchesByte, err := json.Marshal([]map[string]any{
		{
			"op": "replace",
			// 在 json patch 的模式下 / 是特殊字符，需要写成 ~1
			"path":  "/metadata/annotations/controller.kubernetes.io~1pod-deletion-cost",
			"value": cast.ToString(deletionCost),
		},
	})
	if err != nil {
		return errors.Wrap(err, "marshal pod deletion cost patches")
	}

	// 批量 Patch Pod
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.Po)
	for _, podName := range podNames {
		if _, err = client.Patch(
			ctx, namespace, podName, types.JSONPatchType, patchesByte, metav1.PatchOptions{},
		); err != nil {
			return errors.Wrapf(err, "patch pod %s/%s deletion cost to %d", namespace, podName, deletionCost)
		}
	}

	log.Infof(ctx, "successfully set deletion cost to %d for %d pods in namespace %s",
		deletionCost, len(podNames), namespace,
	)
	return nil
}

// SetPodPolarisAnnotations 批量设置 Pod 的北极星相关注解（权重 / 隔离）
// 使用 MergePatch 模式，支持注解不预存在的场景
func (m *podManager) SetPodPolarisAnnotations(
	ctx context.Context,
	clusterID, namespace string,
	podNames []string,
	annotations map[string]string,
) error {
	patchBytes, err := json.Marshal(map[string]any{
		"metadata": map[string]any{
			"annotations": annotations,
		},
	})
	if err != nil {
		return errors.Wrap(err, "marshal polaris annotation patches")
	}

	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.Po)
	for _, podName := range podNames {
		if _, err = client.Patch(
			ctx, namespace, podName, types.MergePatchType, patchBytes, metav1.PatchOptions{},
		); err != nil {
			return errors.Wrapf(err, "patch pod %s/%s polaris annotations", namespace, podName)
		}
	}
	return nil
}

// ClassifyPodsByStatus 将指定的 Pod 按终止态 / 运行态分类
func (m *podManager) ClassifyPodsByStatus(
	ctx context.Context,
	clusterID, namespace string,
	podNames []string,
	labelSelector map[string]string,
) (terminatedPods, runningPods []string, err error) {
	workloadPods, err := m.ListPods(ctx, clusterID, namespace, labelSelector)
	if err != nil {
		return nil, nil, err
	}
	allTerminated, allRunning := m.splitPodNamesByStatus(workloadPods)
	return lo.Intersect(allTerminated, podNames), lo.Intersect(allRunning, podNames), nil
}

// splitPodNamesByStatus 将 Pod 按是否处于终止态分类，返回终止态与运行态的 Pod 名称
func (m *podManager) splitPodNamesByStatus(pods []unstructured.Unstructured) (terminated, running []string) {
	named := lo.Filter(pods, func(po unstructured.Unstructured, _ int) bool { return getPodName(po) != "" })
	terminatedPods, runningPods := lo.FilterReject(named, func(po unstructured.Unstructured, _ int) bool {
		return m.isPodTerminated(po.Object)
	})
	return lo.Map(terminatedPods, func(po unstructured.Unstructured, _ int) string { return getPodName(po) }),
		lo.Map(runningPods, func(po unstructured.Unstructured, _ int) string { return getPodName(po) })
}

// FilterShouldGrayscalePodNames 从已查询的 Pod 中过滤出应该灰度的 Pod 名称
func (m *podManager) FilterShouldGrayscalePodNames(
	workloadPods []unstructured.Unstructured,
	podNames []string,
	grayscaleImageTag string,
) []string {
	podNameSet := set.NewStringSetWithValues(podNames)
	// 仅指定的、运行态的、镜像 Tag 与灰度 Tag 不同的 Pod 需要灰度
	return lo.FilterMap(workloadPods, func(po unstructured.Unstructured, _ int) (string, bool) {
		name := getPodName(po)
		return name, name != "" && podNameSet.Has(name) &&
			!m.isPodTerminated(po.Object) && m.isPodShouldGrayscale(po.Object, grayscaleImageTag)
	})
}

// ListPods 通过 labelSelector 批量获取 Pod
func (m *podManager) ListPods(
	ctx context.Context,
	clusterID, namespace string,
	labelSelector map[string]string,
) ([]unstructured.Unstructured, error) {
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.Po)
	pods, err := client.List(ctx, namespace, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelSelector).String(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "list pods in namespace %s by label selector [%s]", namespace, labelSelector)
	}
	return pods.Items, nil
}

// coversAllRunningPods 判断指定的 Pod 名称是否包含工作负载全部运行态的 Pod（终止态的 Pod 不参与判断）
// 不存在运行态的 Pod 时返回 false
func (m *podManager) coversAllRunningPods(workloadPods []unstructured.Unstructured, specifiedPodNames []string) bool {
	_, runningPodNames := m.splitPodNamesByStatus(workloadPods)
	return len(runningPodNames) > 0 && lo.Every(specifiedPodNames, runningPodNames)
}

// BatchDeleteTerminatedPods 批量直接删除终止态的 Pod
func (m *podManager) BatchDeleteTerminatedPods(
	ctx context.Context,
	clusterID, namespace string,
	podNames []string,
) error {
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.Po)
	for _, podName := range podNames {
		if err := client.Delete(ctx, namespace, podName, metav1.DeleteOptions{}); err != nil {
			return errors.Wrapf(err, "delete terminated pod %s/%s", namespace, podName)
		}
		log.Infof(ctx, "successfully deleted terminated pod %s/%s", namespace, podName)
	}
	return nil
}

// isPodTerminated 判断 Pod 是否处于终止态
func (m *podManager) isPodTerminated(po map[string]any) bool {
	phase := corev1.PodPhase(mapx.GetStr(po, "status.phase"))
	return phase == corev1.PodFailed || phase == corev1.PodSucceeded
}

// isPodShouldGrayscale 判断 Pod 是否需要灰度：主容器镜像 Tag 与灰度 Tag 不一致
func (m *podManager) isPodShouldGrayscale(po map[string]any, grayscaleImageTag string) bool {
	return lo.SomeBy(mapx.GetList(po, "spec.containers"), func(container any) bool {
		c, ok := container.(map[string]any)
		if !ok || mapx.GetStr(c, "name") != defaults.WorkloadMainContainerName {
			return false
		}
		_, tag, _ := strings.Cut(mapx.GetStr(c, "image"), ":")
		return tag != grayscaleImageTag
	})
}

// getPodName 获取 Pod 名称
func getPodName(po unstructured.Unstructured) string {
	return mapx.GetStr(po.Object, "metadata.name")
}
