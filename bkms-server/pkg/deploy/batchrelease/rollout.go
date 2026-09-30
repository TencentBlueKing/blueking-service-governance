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

// Package batchrelease 基于 BCS Kourse Rollout（Canary）实现应用模型的分批发布：
// 由发布策略生成 Rollout CR、读写集群中的 Rollout CR、修改与解析其 status。
//
// Rollout 的 Go 类型不在当前依赖中，这里按 CRD 定义（bcs-kourse-rollout 1.31.0-alpha.128）
// 使用动态客户端以 unstructured 形式读写。
package batchrelease

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// GVR Rollout 资源的 GroupVersionResource
var GVR = schema.GroupVersionResource{
	Group:    "tkex.tencent.com",
	Version:  "v1alpha1",
	Resource: "rollouts",
}

const (
	// APIVersion Rollout 资源的 apiVersion
	APIVersion = "tkex.tencent.com/v1alpha1"
	// Kind Rollout 资源的 kind
	Kind = "Rollout"

	// TypeCanary 灰度发布；CRD 默认值为 BlueGreen，必须显式指定
	TypeCanary = "Canary"
	// FailurePolicyPresent 失败时暂停在当前步骤等待人工介入；CRD 默认值为 Rollback，
	// 必须显式指定，保证平台不自动回滚
	FailurePolicyPresent = "Present"

	// WorkloadLabelKey 被 Rollout 控制的工作负载必须携带的标签键
	WorkloadLabelKey = "rollout"
	// WorkloadLabelValue 被 Rollout 控制的工作负载必须携带的标签值
	WorkloadLabelValue = "true"
)

// ErrWorkloadNotSupported 主工作负载不支持分批发布（目前仅支持 GameDeployment）
var ErrWorkloadNotSupported = errors.New("workload does not support batch release")

// BuildManifest 由主工作负载与发布策略的批次生成 Rollout CR，CR 与主工作负载同名。
//
// 固定 type=Canary、failurePolicy=Present，不配置 hook 与 routeControl；
// 批次目标原样写入 steps[].replicas（百分比不换算为个数），暂停方式映射为：
//   - none：不写 pause
//   - duration：pause.duration = "<秒数>s"
//   - permanent：pause = {}
func BuildManifest(
	workloadKind, workloadName string, batches []releasestrategy.Batch,
) (map[string]any, error) {
	if workloadKind != k8skind.GameDeploy {
		return nil, errors.Wrapf(ErrWorkloadNotSupported, "workload kind %s", workloadKind)
	}
	if workloadName == "" {
		return nil, errors.New("workload name is empty")
	}
	if len(batches) == 0 {
		return nil, errors.New("release strategy has no batches")
	}

	steps := make([]any, 0, len(batches))
	for idx, batch := range batches {
		step, err := buildStep(batch)
		if err != nil {
			return nil, errors.Wrapf(err, "build step %d", idx)
		}
		steps = append(steps, step)
	}

	return map[string]any{
		"apiVersion": APIVersion,
		"kind":       Kind,
		"metadata":   map[string]any{"name": workloadName},
		"spec": map[string]any{
			"type":          TypeCanary,
			"failurePolicy": FailurePolicyPresent,
			"objectRef": map[string]any{
				"workloadRef": map[string]any{
					"apiVersion": gvr.GameDeploy.GroupVersion().String(),
					"kind":       workloadKind,
					"name":       workloadName,
				},
			},
			"steps": steps,
		},
	}, nil
}

func buildStep(batch releasestrategy.Batch) (map[string]any, error) {
	target, err := releasestrategy.ParseBatchTarget(batch.Target)
	if err != nil {
		return nil, err
	}
	step := map[string]any{"replicas": replicasValue(target)}

	switch batch.PauseMode {
	case releasestrategy.PauseModeNone:
	case releasestrategy.PauseModePermanent:
		step["pause"] = map[string]any{}
	case releasestrategy.PauseModeDuration:
		if batch.PauseSeconds <= 0 {
			return nil, errors.Errorf("invalid pause seconds %d", batch.PauseSeconds)
		}
		step["pause"] = map[string]any{"duration": fmt.Sprintf("%ds", batch.PauseSeconds)}
	default:
		return nil, errors.Errorf("unknown pause mode %q", batch.PauseMode)
	}
	return step, nil
}

// replicasValue 转换为 JSON 兼容的 unstructured 值（整数统一为 int64）
func replicasValue(target intstr.IntOrString) any {
	if target.Type == intstr.String {
		return target.StrVal
	}
	return int64(target.IntVal)
}
