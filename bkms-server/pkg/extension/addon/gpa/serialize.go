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

package gpa

import (
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

const (
	// gpaKind GeneralPodAutoscaler CR 的 Kind
	gpaKind = "GeneralPodAutoscaler"
	// gpaGroupVersion GeneralPodAutoscaler CR 的 API GroupVersion
	gpaGroupVersion = "autoscaling.tkex.tencent.com/v1alpha1"

	// metricTypeResource 指标类型：基于 Pod resource
	metricTypeResource = "Resource"
	// targetTypeUtilization 目标类型：使用率百分比
	targetTypeUtilization = "Utilization"

	// --- GPA CR labels ---

	// LabelKeyWorkspaceID 标记所属工作空间 ID
	LabelKeyWorkspaceID = "io.tencent.bkms.workspace-id"
	// LabelKeyEnvName 标记所属环境名称
	LabelKeyEnvName = "io.tencent.bkms.env-name"
	// LabelKeyAppID 标记所属应用 ID
	LabelKeyAppID = "io.tencent.bkms.app-id"

	// --- GPA CR annotations ---

	// annotationKeyComputeByLimits 指示 GPA 以 limits（而非默认 requests）为基准计算利用率。
	// 值固定为字符串 "true"
	annotationKeyComputeByLimits = "compute-by-limits"
	// annotationValueTrue compute-by-limits 开启时的固定字符串值
	annotationValueTrue = "true"
)

// gpaConditionRaw 对应 GPA CR status.conditions[] 的一项。
type gpaConditionRaw struct {
	Type    string `mapstructure:"type"`
	Status  string `mapstructure:"status"`
	Message string `mapstructure:"message"`
	Reason  string `mapstructure:"reason"`
}

// gpaStatusRaw 对应 GPA CR 的 status 段，用于 mapstructure 解码
type gpaStatusRaw struct {
	CurrentReplicas int32             `mapstructure:"currentReplicas"`
	DesiredReplicas int32             `mapstructure:"desiredReplicas"`
	LastScaleTime   string            `mapstructure:"lastScaleTime"`
	Conditions      []gpaConditionRaw `mapstructure:"conditions"`
}

// buildGPAManifest 将 GPAConfig 转换为 GeneralPodAutoscaler CR 的 k8s manifest。
// scaleTargetName 为该应用在对应环境部署生成的工作负载名（固定 kind=GameDeployment）。
// 指标模式（spec.metric）与定时模式（spec.time）按是否配置分别输出。
func buildGPAManifest(config *GPAConfig, workspaceID, envName, scaleTargetName string) map[string]any {
	spec := map[string]any{
		"minReplicas": config.MinReplicas,
		"maxReplicas": config.MaxReplicas,
		"scaleTargetRef": map[string]any{
			"apiVersion": gvr.GameDeploy.GroupVersion().String(),
			"kind":       k8skind.GameDeploy,
			"name":       scaleTargetName,
		},
	}

	if len(config.Metrics) > 0 {
		metrics := make([]any, 0, len(config.Metrics))
		for _, m := range config.Metrics {
			metrics = append(metrics, map[string]any{
				"type": metricTypeResource,
				"resource": map[string]any{
					"name": string(m.Resource),
					"target": map[string]any{
						"type":               targetTypeUtilization,
						"averageUtilization": m.AverageUtilization,
					},
				},
			})
		}
		spec["metric"] = map[string]any{
			"metrics": metrics,
		}
	}

	if len(config.TimeRanges) > 0 {
		ranges := make([]any, 0, len(config.TimeRanges))
		for _, r := range config.TimeRanges {
			// 仅启用的定时规则写入 CR；Remark 仅用于展示，不写入 CR。
			if !r.Enabled {
				continue
			}
			ranges = append(ranges, map[string]any{
				"desiredReplicas": r.DesiredReplicas,
				"schedule":        r.Schedule,
			})
		}
		if len(ranges) > 0 {
			spec["time"] = map[string]any{
				"ranges": ranges,
			}
		}
	}

	return map[string]any{
		"apiVersion": gpaGroupVersion,
		"kind":       gpaKind,
		"metadata":   buildGPAMetadata(config, workspaceID, envName),
		"spec":       spec,
	}
}

// buildGPAMetadata 构造 GPA CR 的 metadata 段，含固定 labels 及可选 annotations。
// 当 ComputeByLimits 为 true 时写入 compute-by-limits="true" annotation（与是否配置 metrics 解耦）。
func buildGPAMetadata(config *GPAConfig, workspaceID, envName string) map[string]any {
	metadata := map[string]any{
		"name": config.Name,
		"labels": map[string]any{
			LabelKeyWorkspaceID: workspaceID,
			LabelKeyEnvName:     envName,
			LabelKeyAppID:       config.AppID,
		},
	}

	if config.ComputeByLimits {
		metadata["annotations"] = map[string]any{
			annotationKeyComputeByLimits: annotationValueTrue,
		}
	}

	return metadata
}

// parseGPAStatusFromUnstructured 从 GPA CR 的 Unstructured 解析运行状态。
func parseGPAStatusFromUnstructured(obj *unstructured.Unstructured) (*GPAStatus, error) {
	status := &GPAStatus{
		Name:        obj.GetName(),
		AppID:       obj.GetLabels()[LabelKeyAppID],
		WorkspaceID: obj.GetLabels()[LabelKeyWorkspaceID],
		EnvName:     obj.GetLabels()[LabelKeyEnvName],
	}

	statusMap, ok := obj.Object["status"].(map[string]any)
	if !ok {
		// status 段整体缺失：CR 刚下发、controller 尚未处理，属正常过渡态。
		status.Phase = phaseInitializing
		return status, nil
	}

	var raw gpaStatusRaw
	if err := mapstructure.Decode(statusMap, &raw); err != nil {
		return nil, errors.Wrapf(err, "invalid GPA CR %s: decode status", status.Name)
	}

	status.CurrentReplicas = raw.CurrentReplicas
	status.DesiredReplicas = raw.DesiredReplicas
	status.LastScaleTime = raw.LastScaleTime
	status.Phase, status.StatusMessage = derivePhaseAndMessage(status.Name, raw.Conditions)
	return status, nil
}

// phase 枚举值
const (
	// phaseActive 扩缩容正常运作，副本数在 minReplicas 与 maxReplicas 范围内。
	// 触发条件：AbleToScale=True 且 ScalingActive=True 且 ScalingLimited=False。
	phaseActive = "Active"
	// phasePaused 指标获取失败或无效，扩缩被暂停，副本数保持不变。
	// 触发条件：AbleToScale=True 且 ScalingActive=False。
	phasePaused = "Paused"
	// phaseLimited 扩缩容逻辑正常，但副本数已触达 minReplicas 或 maxReplicas 边界，无法继续同向调整。
	// 触发条件：AbleToScale=True 且 ScalingActive=True 且 ScalingLimited=True。
	phaseLimited = "Limited"
	// phaseFailed GPA 无法访问 scale 子资源（目标工作负载不存在、API 不可达、权限不足等），扩缩能力不可用。
	// 触发条件：AbleToScale 存在且 status=False（优先级最高，无论其他 condition 状态）。
	phaseFailed = "Failed"
	// phaseInitializing CR 刚下发，controller 尚未写入 status.conditions，属正常过渡态。
	// 触发条件：status 段缺失或 status.conditions 为空。稍候 controller 处理后即会转为其他状态。
	phaseInitializing = "Initializing"
	// phaseUnknown conditions 存在但无法解析出关键 condition（AbleToScale/ScalingActive/ScalingLimited 均缺失或不完整），
	// 可能是旧版本 GPA 或异常状态。注意：conditions 为空属 Initializing 而非 Unknown。
	phaseUnknown = "Unknown"
)

// 关注的 condition type
const (
	condAbleToScale    = "AbleToScale"
	condScalingActive  = "ScalingActive"
	condScalingLimited = "ScalingLimited"
	condTrue           = "True"
)

// derivePhaseAndMessage 按 K8s HPA 语义将 conditions 推导为 phase 枚举与汇总消息。
//
// 规则（按优先级从高到低）：
//   - Failed:  AbleToScale 存在且 status=False（无论其他 condition）
//   - Paused:  AbleToScale=True 且 ScalingActive=False
//   - Limited: AbleToScale=True 且 ScalingActive=True 且 ScalingLimited=True
//   - Active:  AbleToScale=True 且 ScalingActive=True 且 ScalingLimited=False
//   - Initializing: conditions 为空（CR 刚下发，controller 未处理）
//   - Unknown: conditions 存在但关键 condition 缺失/不完整
//
// statusMessage 汇总所有 status != "True" 的 condition 的 message（跳过空，用 "; " 连接）。
func derivePhaseAndMessage(crName string, conditions []gpaConditionRaw) (string, string) {
	if len(conditions) == 0 {
		// CR 刚下发、controller 尚未写入 conditions，属正常过渡态，不视为异常。
		log.InfoNoContextf("gpa CR %s: status.conditions not ready, treated as Initializing", crName)
		return phaseInitializing, ""
	}

	// 收集关键 condition 状态：未出现视为缺失
	var ableToScale, scalingActive, scalingLimited *bool
	var messages []string

	for _, c := range conditions {
		isTrue := c.Status == condTrue
		switch c.Type {
		case condAbleToScale:
			ableToScale = &isTrue
		case condScalingActive:
			scalingActive = &isTrue
		case condScalingLimited:
			scalingLimited = &isTrue
		}
		// 所有非 True 的 condition 都计入 statusMessage
		if !isTrue {
			if msg := strings.TrimSpace(c.Message); msg != "" {
				messages = append(messages, msg)
			}
		}
	}

	phase := classifyPhase(ableToScale, scalingActive, scalingLimited)
	return phase, strings.Join(messages, "; ")
}

// classifyPhase 按优先级推导 phase。未出现的关键 condition 视为缺失（→ Unknown）。
func classifyPhase(ableToScale, scalingActive, scalingLimited *bool) string {
	// 关键 condition 均缺失 → Unknown
	if ableToScale == nil && scalingActive == nil && scalingLimited == nil {
		return phaseUnknown
	}

	// AbleToScale=False → Failed
	if ableToScale != nil && !*ableToScale {
		return phaseFailed
	}

	// AbleToScale=True 且 ScalingActive=False → Paused
	if ableToScale != nil && *ableToScale {
		if scalingActive != nil && !*scalingActive {
			return phasePaused
		}
		// 需要 ScalingActive=True 才能判定 Limited/Active
		if scalingActive != nil && *scalingActive {
			if scalingLimited != nil && *scalingLimited {
				return phaseLimited
			}
			return phaseActive
		}
	}

	// 关键 condition 不完整（如仅有 ScalingActive）→ Unknown
	return phaseUnknown
}

// parseGPAStatusListFromUnstructured 批量解析 UnstructuredList 为 GPAStatus 列表。
func parseGPAStatusListFromUnstructured(list *unstructured.UnstructuredList) ([]*GPAStatus, error) {
	statuses := make([]*GPAStatus, 0, len(list.Items))
	for i := range list.Items {
		s, err := parseGPAStatusFromUnstructured(&list.Items[i])
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, s)
	}
	return statuses, nil
}
