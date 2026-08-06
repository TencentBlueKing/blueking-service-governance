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
	"fmt"
	"strings"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/TencentBlueKing/gopkg/collection/set"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// GameDeploymentJSONPatchBuilder GameDeploy Patch 构建器
type GameDeploymentJSONPatchBuilder struct {
	gameDeploy *tkex.GameDeployment
}

// NewGameDeploymentJSONPatchBuilder 创建 GameDeployment Patch 构建器
func NewGameDeploymentJSONPatchBuilder(gd *tkex.GameDeployment) *GameDeploymentJSONPatchBuilder {
	return &GameDeploymentJSONPatchBuilder{gameDeploy: gd}
}

// BuildReplicasPatch 构建副本数 Patch，用于扩缩容
func (b *GameDeploymentJSONPatchBuilder) BuildReplicasPatch(replicas int32) map[string]any {
	return map[string]any{
		"op":    "replace",
		"path":  "/spec/replicas",
		"value": replicas,
	}
}

// BuildPodsToDeletePatch 构建删除 Pod Patch，用于删除指定的一批 Pod（按名称）
func (b *GameDeploymentJSONPatchBuilder) BuildPodsToDeletePatch(podNames []string) map[string]any {
	// 与已有的待删除的 Pod 名称做合并（防止覆盖上次指定删除，但还未删除的 Pod）
	mergedPodNames := set.NewStringSetWithValues(podNames)
	for _, podName := range b.gameDeploy.Spec.ScaleStrategy.PodsToDelete {
		mergedPodNames.Add(podName)
	}

	return map[string]any{
		"op":    "replace",
		"path":  "/spec/scaleStrategy/podsToDelete",
		"value": mergedPodNames.ToSlice(),
	}
}

// BuildMainContainerImagePatch 构建主容器镜像 Patch，用于更新主容器镜像
//
// GameDeployment 原地更新镜像（也叫原地重启），它仅会更新目标容器的镜像，替换为新镜像，但是它不会重建 pod 实例。
// 重建 pod 与 仅更新镜像的区别在于：
//   - 重建 pod 会重建 pod 实例，重建 pod 实例会触发 kubelet 重新拉取镜像，pod 实例名称、pod IP 都会被重新分配。
//   - 原地更新不会重建 pod 实例，也会触发 kubelet 拉取新镜像，但是 pod 实例名称、pod IP 不会发生变化。
//
// 使用场景: 业务需要仅更新镜像，但是不希望重建 pod 实例。一般用于有状态服务，或者不方便重启的 pod 实例。
func (b *GameDeploymentJSONPatchBuilder) BuildMainContainerImagePatch(imageTag string) map[string]any {
	// 逐个容器检查，按名称找到主容器
	for idx, c := range b.gameDeploy.Spec.Template.Spec.Containers {
		if c.Name == defaults.WorkloadMainContainerName {
			repo, _, _ := strings.Cut(c.Image, ":")
			return map[string]any{
				"op":    "replace",
				"path":  fmt.Sprintf("/spec/template/spec/containers/%d/image", idx),
				"value": fmt.Sprintf("%s:%s", repo, imageTag),
			}
		}
	}

	return nil
}

// BuildInplaceUpdatePatch 构建原地更新 Patch
// 场景：全量更新 or 灰度更新（持续灰度，通过控制 partition 值，保留多少个旧 pod 数量）
//
// 用法：
// - 为 0 表示不限制 Pod 的数量，直接更新所有 Pod。
// - 假设有 10 个 pod 待更新，partition 设置为 1，那么更新时，只会保留 1 个 Pod 为旧版本，其余 9 个 Pod 为新版本。
//
// GameDeployment 内部实现：
// partition 指在更新过程中，有多少个 Pod 需要保持为旧版本，类似于灰度更新。这样在滚动更新过程中，就可以先更新一部分 Pod 为新版本，
// 观察没问题后，再修改 Partition，继续更新，直至更新完所有的 Pod。可设置为整数值或百分比，默认值为 0。
//
// Q: 为什么是返回两个 Patch，而不是 Patch 整个 /spec/updateStrategy?
// A: 因为 /spec/updateStrategy 是 Map 结构，直接 Patch 会导致其他字段被覆盖（如 maxAvailable 变成默认值 25%）。
func (b *GameDeploymentJSONPatchBuilder) BuildInplaceUpdatePatch(partition int) []map[string]any {
	return []map[string]any{
		{
			"op":    "replace",
			"path":  "/spec/updateStrategy/type",
			"value": "InplaceUpdate",
		},
		{
			"op":    "replace",
			"path":  "/spec/updateStrategy/partition",
			"value": partition,
		},
	}
}

// BuildRollingUpdatePatch 构建滚动更新 Patch，用于支持全量更新（镜像+配置 / 仅配置）场景
func (b *GameDeploymentJSONPatchBuilder) BuildRollingUpdatePatch() map[string]any {
	return map[string]any{
		"op":    "replace",
		"path":  "/spec/updateStrategy/type",
		"value": "RollingUpdate",
	}
}
