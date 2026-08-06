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

// Package client provides deploy related types
package client

// HelmDeployOptions 部署选项参数
type HelmDeployOptions struct {
	// ImageTag 镜像 Tag
	ImageTag string `yaml:"imageTag" json:"imageTag" validate:"required"`

	// ChartVersion Chart 版本
	ChartVersion string `yaml:"chartVersion" json:"chartVersion"`

	// ValuesFileID Values 文件 ID
	ValuesFileID string `yaml:"valuesFile" json:"valuesFile"`

	// TrafficLaneName 泳道名称（为空表示不区分泳道）
	TrafficLaneName string `yaml:"trafficLane" json:"trafficLane"`
}

// HelmDeployRecord Helm 部署记录
type HelmDeployRecord struct {
	// ID 部署记录 ID
	ID string `json:"id" yaml:"id"`
	// EnvName 环境名称
	EnvName string `json:"envName" yaml:"envName"`
	// ProjectCode 项目 ID
	ProjectCode string `json:"projectCode" yaml:"projectCode"`
	// ClusterID 集群 ID
	ClusterID string `json:"clusterID" yaml:"clusterID"`
	// Namespace 命名空间
	Namespace string `json:"namespace" yaml:"namespace"`
	// ReleaseName Helm Release 名称
	ReleaseName string `json:"releaseName" yaml:"releaseName"`
	// ChartName Chart 名称
	ChartName string `json:"chartName" yaml:"chartName"`
	// ChartVersion Chart 版本
	ChartVersion string `json:"chartVersion" yaml:"chartVersion"`
	// ValuesFileID Values 文件 ID
	ValuesFileID string `json:"valuesFileID" yaml:"valuesFileID"`
	// ImageTag 镜像 Tag
	ImageTag string `json:"imageTag" yaml:"imageTag"`
	// Revision 代码版本
	Revision string `json:"revision" yaml:"revision"`
	// Status 部署状态
	Status string `json:"status" yaml:"status"`
	// Message 部署消息
	Message string `json:"message" yaml:"message"`
	// Operator 触发人
	Operator string `json:"operator" yaml:"operator"`
	// Values 部署参数
	Values string `json:"values" yaml:"values"`
	// CreatedAt 创建时间
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
}

// ListHelmDeployRecordsRespData 获取 helm 部署记录列表返回数据
type ListHelmDeployRecordsRespData struct {
	Data PaginatedHelmDeployRecords `json:"data"`
}

// PaginatedHelmDeployRecords 分页构建记录
type PaginatedHelmDeployRecords struct {
	Count   string             `json:"count"`
	Results []HelmDeployRecord `json:"results"`
}

// AppModelDeployOptions AppModel 部署参数
type AppModelDeployOptions struct {
	// ImageTag 部署的镜像版本
	ImageTag string `json:"imageTag" yaml:"imageTag" validate:"required"`

	// Replicas 副本数/实例数量，发布实例数量必须大于或等于 1（必填）
	Replicas uint64 `json:"replicas" yaml:"replicas" validate:"required,gte=1"`

	// fixme 暂时不支持泳道
	// TrafficLaneName 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName" yaml:"trafficLaneName"`
}

// AppModelDeployRecord 部署记录
type AppModelDeployRecord struct {
	// ID 部署记录 ID
	ID string `json:"id" yaml:"id"`

	// ClusterID 集群 ID
	ClusterID string `json:"clusterID" yaml:"clusterID"`

	// Namespace 命名空间名称
	Namespace string `json:"namespace" yaml:"namespace"`

	// ImageTag 当前 Release 使用的镜像
	ImageTag string `json:"imageTag" yaml:"imageTag"`

	// Replicas pod 数量
	Replicas int32 `json:"replicas" yaml:"replicas"`

	// Message 部署消息
	Message string `json:"message" yaml:"message"`

	// Status 部署状态
	Status string `json:"status" yaml:"status"`

	// Operator 触发人
	Operator string `json:"operator" yaml:"operator"`

	// CreatedAt 创建时间
	CreatedAt string `json:"createdAt" yaml:"createdAt"`

	// UpdatedAt 更新时间
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
}

// AppModelDeployRecordsRespData 部署记录返回数据
type AppModelDeployRecordsRespData struct {
	// Count 总数
	Count string `json:"count"`

	// Results 部署记录
	Results []AppModelDeployRecord `json:"results"`
}

// AppModelDeployRecordsResp 部署记录返回
type AppModelDeployRecordsResp struct {
	// Data 返回数据
	Data AppModelDeployRecordsRespData `json:"data"`
}
