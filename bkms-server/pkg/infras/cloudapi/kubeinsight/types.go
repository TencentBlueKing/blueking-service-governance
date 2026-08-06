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

package kubeinsight

// GetLatestClusterReportResponse 获取集群最新检查报告响应
type GetLatestClusterReportResponse struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    *ClusterReport `json:"data"`
	PDFData []byte         `json:"pdfData"`
}

// ClusterReport 集群报告
type ClusterReport struct {
	// ClusterID 集群ID
	ClusterID string `json:"clusterId"`
	// ClusterInfo 集群信息
	ClusterInfo *ClusterInfo `json:"clusterInfo"`
	// AbnormalItems 异常检查项列表
	AbnormalItems []CheckItem `json:"abnormalItems"`
	// Score 健康度评分
	Score int32 `json:"score"`
	// StartTime 报告检查开始时间
	StartTime string `json:"startTime"`
	// EndTime 报告检查结束时间
	EndTime string `json:"endTime"`
}

// ClusterInfo 集群信息
type ClusterInfo struct {
	// ClusterID 集群ID
	ClusterID string `json:"clusterId"`
	// ClusterName 集群名称
	ClusterName string `json:"clusterName"`
	// ClusterType 集群类型, single（独立集群）, federation（联邦集群）
	ClusterType string `json:"clusterType"`
	// Provider 提供商
	Provider string `json:"provider"`
	// ManageType 管理类型
	ManageType string `json:"manageType"`
	// Creator 创建者
	Creator string `json:"creator"`
	// BusinessID 业务ID
	BusinessID string `json:"businessId"`
	// ProjectID 项目ID
	ProjectID string `json:"projectId"`
	// ClusterVersion 集群版本
	ClusterVersion string `json:"clusterVersion"`
	// NetworkParams 集群网络相关参数
	NetworkParams *NetworkParams `json:"networkParams"`
	// OSRuntimeInfo 集群操作系统/镜像/容器运行时信息
	OSRuntimeInfo *OSRuntimeInfo `json:"osRuntimeInfo"`
	// Region 区域, 如 ap-nanjing
	Region string `json:"region"`
	// VpcID VPC ID
	VpcID string `json:"vpcId"`
	// NodeCount 节点数量
	NodeCount int32 `json:"nodeCount"`
	// MasterCount Master节点数量
	MasterCount int32 `json:"masterCount"`
}

// NetworkParams 网络参数
type NetworkParams struct {
	// Cidrs CIDR列表
	Cidrs []string `json:"cidrs"`
	// MaxNodePodNum 每个节点最大Pod数
	MaxNodePodNum int32 `json:"maxNodePodNum"`
	// MaxServiceNum 最大Service数
	MaxServiceNum int32 `json:"maxServiceNum"`
}

// OSRuntimeInfo OS/镜像/运行时信息
type OSRuntimeInfo struct {
	// OSImage 操作系统/镜像
	OSImage string `json:"osImage"`
	// Runtime 容器运行时
	Runtime string `json:"runtime"`
	// RuntimeVersion 运行时版本
	RuntimeVersion string `json:"runtimeVersion"`
}

// CheckItem 检查项
type CheckItem struct {
	// Timestamp 检查项产生时间
	Timestamp string `json:"timestamp"`
	// LastUpdateTimestamp 检查项最后一次更新时间
	LastUpdateTimestamp string `json:"lastUpdateTimestamp"`
	// Key 检查项标识，如 "timecheck_drift"
	Key string `json:"key"`
	// Description 检查项描述 如 "节点产生时间漂移"
	Description string `json:"description"`
	// Category 检查项分类 如 "节点异常"
	Category string `json:"category"`
	// ContextMsg 异常相关的上下文信息
	ContextMsg string `json:"contextMsg"`
	// Level 错误等级: RISK 严重, WARN 警告, INFO 提醒
	Level string `json:"level"`
	// ResourceKey 资源唯一标识符， 如 resourceType 为 node 时， 对应节点IP
	ResourceKey string `json:"resourceKey"`
	// ResourceType 资源类型
	ResourceType string `json:"resourceType"`
	// ErrorDetail 错误详情
	ErrorDetail string `json:"errorDetail"`
	// Solutions 建议处理方式
	Solutions string `json:"solutions"`
	// Recovered 标记该项异常是否已经恢复
	Recovered bool `json:"recovered"`
	// RecordCount 标记该项异常出现的次数
	RecordCount int32 `json:"recordCount"`
}
