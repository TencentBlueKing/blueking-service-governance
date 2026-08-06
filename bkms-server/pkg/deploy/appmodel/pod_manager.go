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
	"github.com/spf13/cast"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// ClassifyPodsByStatus 查询 Pod 状态并分类为终止态和运行态
// 通过 labelSelector 批量获取 Pod，然后从中筛选出目标 Pod 进行分类
// 返回值：terminatedPods, runningPods、error
func (m *podManager) ClassifyPodsByStatus(
	ctx context.Context,
	clusterID, namespace string,
	podNames []string,
	labelSelector map[string]string,
) (terminatedPods, runningPods []string, err error) {
	// 批量获取所有匹配的 Pod
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.Po)
	pods, err := client.List(ctx, namespace, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelSelector).String(),
	})
	if err != nil {
		return nil, nil, errors.Wrapf(
			err, "list pods in namespace %s by label selector [%s]", namespace, labelSelector,
		)
	}

	// 待过滤的 Pod 名称集合
	podNameSet := set.NewStringSetWithValues(podNames)

	// 逐个 Pod 遍历，并根据是否为终止态分类
	for _, po := range pods.Items {
		// 获取 Pod 名称，并判断是否在待过滤的 Pod 名称集合中
		podName := mapx.GetStr(po.Object, "metadata.name")
		if podName == "" || !podNameSet.Has(podName) {
			continue
		}

		// 判断 Pod 是否处于终止态
		if m.isPodTerminated(po.Object) {
			terminatedPods = append(terminatedPods, podName)
		} else {
			runningPods = append(runningPods, podName)
		}
	}

	return terminatedPods, runningPods, nil
}

// FilterShouldGrayscalePodNames 过滤出应该灰度的 Pod 名称
// 返回值：shouldGrayscalePodNames、error
func (m *podManager) FilterShouldGrayscalePodNames(
	ctx context.Context,
	clusterID, namespace, grayscaleImageTag string,
	podNames []string,
	labelSelector map[string]string,
) ([]string, error) {
	// 批量获取所有匹配的 Pod
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.Po)
	pods, err := client.List(ctx, namespace, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labelSelector).String(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "list pods in namespace %s by label selector [%s]", namespace, labelSelector)
	}

	var shouldGrayscalePodNames []string
	// 待过滤的 Pod 名称集合
	podNameSet := set.NewStringSetWithValues(podNames)

	// 逐个 Pod 遍历，并根据是否为终止态分类
	for _, po := range pods.Items {
		// 获取 Pod 名称，并判断是否在待过滤的 Pod 名称集合中
		podName := mapx.GetStr(po.Object, "metadata.name")
		if podName == "" || !podNameSet.Has(podName) {
			continue
		}

		// 终止态的 Pod 不需要进行灰度
		if m.isPodTerminated(po.Object) {
			continue
		}

		// 判断 Pod 是否应该进行灰度
		if m.isPodShouldGrayscale(po.Object, grayscaleImageTag) {
			shouldGrayscalePodNames = append(shouldGrayscalePodNames, podName)
		}
	}

	return shouldGrayscalePodNames, nil
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

// isPodShouldGrayscale 判断 Pod 是否应该进行灰度
func (m *podManager) isPodShouldGrayscale(po map[string]any, grayscaleImageTag string) bool {
	for _, container := range mapx.GetList(po, "spec.containers") {
		c, ok := container.(map[string]any)
		if !ok {
			continue
		}
		// 只关注主工作负载容器
		if mapx.GetStr(c, "name") != defaults.WorkloadMainContainerName {
			continue
		}
		// 与灰度镜像 TAG 不匹配，说明需要进行灰度
		image := mapx.GetStr(c, "image")
		if _, tag, _ := strings.Cut(image, ":"); tag != grayscaleImageTag {
			return true
		}
	}
	return false
}
