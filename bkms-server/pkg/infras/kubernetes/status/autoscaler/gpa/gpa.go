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

// Package gpa 提供 GeneralPodAutoscaler 资源的状态解析能力
//
// 首期 GPA 与 HPA 的状态判定规则完全一致，本包通过委托调用 hpa.Parse 实现；
// TODO 未来 GPA 出现特有状态字段，可在此文件中扩展差异逻辑，无需改动 HPA 分支
package gpa

import (
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/autoscaler/hpa"
)

// Parse 解析 GeneralPodAutoscaler 的综合状态，当前实现直接复用 HPA 判定规则
func Parse(manifest map[string]any) *k8sstatus.Result {
	return hpa.Parse(manifest)
}
