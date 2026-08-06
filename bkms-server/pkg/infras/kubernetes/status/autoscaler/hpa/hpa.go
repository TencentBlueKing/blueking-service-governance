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

// Package hpa 提供 HorizontalPodAutoscaler 资源的状态解析能力
package hpa

import (
	"github.com/TencentBlueKing/gopkg/mapx"

	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// HPA manifest 字段与 condition 常量
const (
	// fieldStatusConditions HPA conditions 列表字段路径
	fieldStatusConditions = "status.conditions"
	// condKeyType condition 的 type 字段 key
	condKeyType = "type"
	// condKeyStatus condition 的 status 字段 key
	condKeyStatus = "status"
	// conditionAbleToScale HPA 是否能触发扩缩容的 condition 类型
	conditionAbleToScale = "AbleToScale"
	// conditionScalingActive HPA 是否正在基于指标执行扩缩容的 condition 类型
	conditionScalingActive = "ScalingActive"
	// conditionStatusTrue condition.status 的 True 取值
	conditionStatusTrue = "True"
)

// Parse 解析 HPA 的综合状态
//
// 判定规则：
//  1. AbleToScale=True 且 ScalingActive=True -> Healthy
//  2. AbleToScale=False 或 ScalingActive=False -> Degraded
//  3. 两个关键 condition 任一缺失（或 status.conditions 为空/缺失）-> Unknown
//  4. ScalingLimited 不参与异常判定（达到 min/max 边界属正常状态）
//  5. condition.status 仅 "True" 视为 True，其余（含 "Unknown"）按 False 处理
func Parse(manifest map[string]any) *k8sstatus.Result {
	if manifest == nil {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}

	conditions := mapx.GetList(manifest, fieldStatusConditions)
	if len(conditions) == 0 {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}

	ableToScaleFound, ableToScaleTrue := false, false
	scalingActiveFound, scalingActiveTrue := false, false

	for _, c := range conditions {
		cond, ok := c.(map[string]any)
		if !ok {
			continue
		}
		isTrue := mapx.GetStr(cond, condKeyStatus) == conditionStatusTrue
		switch mapx.GetStr(cond, condKeyType) {
		case conditionAbleToScale:
			ableToScaleFound = true
			ableToScaleTrue = isTrue
		case conditionScalingActive:
			scalingActiveFound = true
			scalingActiveTrue = isTrue
		}
	}

	if !ableToScaleFound || !scalingActiveFound {
		return &k8sstatus.Result{Code: k8sstatus.Unknown}
	}
	if !ableToScaleTrue || !scalingActiveTrue {
		return &k8sstatus.Result{Code: k8sstatus.Degraded}
	}
	return &k8sstatus.Result{Code: k8sstatus.Healthy}
}
