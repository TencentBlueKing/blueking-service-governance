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

// Package status 定义 K8s 资源综合状态枚举，作为项目节点状态的唯一常量来源
package status

// 节点状态枚举，用于拓扑图中 Node.Status 字段，由 resolveNodeStatus 根据 K8s 资源条件计算得出
const (
	// Running Pod 正在运行（status.phase=Running）
	Running = "Running"
	// Healthy Service/ConfigMap/Secret/Ingress 等稳态资源，条件就绪
	Healthy = "Healthy"
	// Degraded 任意资源出现 Degraded 条件（condition Degraded=True）
	Degraded = "Degraded"
	// NotFound 资源在 ResourceSnapshot 中存在但集群中未找到
	NotFound = "NotFound"
	// Unknown 无法判定状态（条件缺失或未识别的资源类型）
	Unknown = "Unknown"
	// Available Deployment/StatefulSet/DaemonSet/ReplicaSet 满足 Available 条件
	Available = "Available"
	// Progressing Deployment/StatefulSet/DaemonSet/ReplicaSet 正在滚动更新（Progressing=True）
	Progressing = "Progressing"
	// Active 虚拟根节点的固定状态
	Active = "Active"
	// Suspended 资源被挂起或暂停（如 GameDeployment paused、CronJob suspended）
	Suspended = "Suspended"
	// Missing 资源在集群中缺失（调用方获取资源失败后的外部事实，区别于资源自身 manifest 能解析出的 NotFound）
	Missing = "Missing"
)

// Result 资源状态评估结果，作为所有 status parser 的统一返回类型
type Result struct {
	// Code 状态码
	Code string
	// Message 对状态的补充说明（parser 无详细说明时为空）
	Message string
}
