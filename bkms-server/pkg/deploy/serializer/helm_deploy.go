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

// Package serializer 定义部署相关 Gin API 的输入和输出结构。
package serializer

import (
	"time"

	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// HelmDeployURIInput 是 Helm 应用部署 API 共用的路径参数。
type HelmDeployURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// HelmDeployRecordURIInput 是按部署记录访问 Helm 应用部署 API 的路径参数。
type HelmDeployRecordURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 部署环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
	// 部署记录 ID
	DeployID string `uri:"deployID" binding:"required,min=1"`
}

// ListHelmDeployRecordsQueryInput 是查询 Helm 应用部署记录列表的 query 输入。
//
// [bkms-cli 使用] 避免破坏性修改。keyword 按 Chart 版本、Image Tag、操作人模糊匹配。
type ListHelmDeployRecordsQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
	// 搜索关键字
	Keyword string `form:"keyword"`
	// 分页页码（从 1 开始）
	Page int64 `form:"page" binding:"required,gte=1"`
	// 分页大小
	PageSize int64 `form:"pageSize" binding:"required,oneof=1 5 10 20 50 100"`
}

// PreviewHelmDeployQueryInput 是预览 Helm 应用部署的 query 输入。
type PreviewHelmDeployQueryInput struct {
	// 目标镜像 TAG
	ImageTag string `form:"imageTag" binding:"required,min=1"`
	// 指定的部署的 Chart 版本，目前要求必须提供版本（前端获取最新版本并提交）
	ChartVersion string `form:"chartVersion" binding:"required,min=1"`
	// 部署使用的 ValuesFile ID
	ValuesFileID string `form:"valuesFileID" binding:"required,min=1"`
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
}

// CreateHelmDeployInput 是部署 Helm 应用的 JSON 输入。
//
// [bkms-cli 使用] 避免破坏性修改。appID 和 envName 由路径参数提供，其余字段保持旧 proto 请求语义。
type CreateHelmDeployInput struct {
	// 目标镜像 TAG
	ImageTag string `json:"imageTag" binding:"required,min=1"`
	// 指定的部署的 Chart 版本，目前要求必须提供版本（前端获取最新版本并提交）
	ChartVersion string `json:"chartVersion" binding:"required,min=1"`
	// 部署使用的 ValuesFile ID
	ValuesFileID string `json:"valuesFileID" binding:"required,min=1"`
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
}

// HelmDeployTrafficLaneQueryInput 是 DELETE 等无请求体接口使用的 query 输入。
type HelmDeployTrafficLaneQueryInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `form:"trafficLaneName"`
}

// RollbackHelmDeployInput 是回滚 Helm 部署的 JSON 输入。
type RollbackHelmDeployInput struct {
	// 部署的泳道名称（空字符串表示不使用泳道）
	TrafficLaneName string `json:"trafficLaneName"`
}

// PreviewHelmDeployOutput 是 Helm 部署预览和回滚预览的 JSON 响应。
type PreviewHelmDeployOutput struct {
	// 目前部署的 manifest
	Current string `json:"current"`
	// 部署或回滚操作下发的 manifest
	Target string `json:"target"`
	// MissingVars values 中引用但未定义的非 env 命名空间变量（以 "ns.var" 形式，如 bkms.BAR）
	MissingVars []string `json:"missingVars,omitempty"`
	// MissingEnvVars values 中引用但未定义的 env 命名空间变量
	MissingEnvVars []string `json:"missingEnvVars,omitempty"`
}

// EmptyOutput 是无数据接口的 JSON 响应。
type EmptyOutput struct{}

// HelmDeployRecordOutputObj 是 Helm 部署记录的 JSON 表示。
//
// [bkms-cli 使用] 避免破坏性修改。
type HelmDeployRecordOutputObj struct {
	// 部署记录 ID
	ID string `json:"id"`
	// 部署环境名称
	EnvName string `json:"envName"`
	// 蓝盾项目 ID
	ProjectCode string `json:"projectCode"`
	// 集群 ID
	ClusterID string `json:"clusterID"`
	// 命名空间
	Namespace string `json:"namespace"`
	// Helm Release 名称
	ReleaseName string `json:"releaseName"`
	// Chart 名称
	ChartName string `json:"chartName"`
	// Chart 版本
	ChartVersion string `json:"chartVersion"`
	// 部署使用的 ValuesFile ID
	ValuesFileID string `json:"valuesFileID"`
	// 镜像 TAG
	ImageTag string `json:"imageTag"`
	// Helm Revision
	Revision string `json:"revision"`
	// 部署状态
	Status string `json:"status"`
	// 部署消息
	Message string `json:"message"`
	// 操作人
	Operator string `json:"operator"`
	// Helm Release Values；受 Helm 历史存储限制，历史久远的数据可能为空
	Values string `json:"values"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel 将 Helm 部署记录模型转换为 Gin v2 输出对象。
func (o *HelmDeployRecordOutputObj) FromModel(record helmdeploy.Record) *HelmDeployRecordOutputObj {
	*o = HelmDeployRecordOutputObj{
		ID:           record.ID.Hex(),
		EnvName:      record.EnvName,
		ProjectCode:  record.ProjectCode,
		ClusterID:    record.ClusterID,
		Namespace:    record.Namespace,
		ReleaseName:  record.ReleaseName,
		ChartName:    record.ChartName,
		ChartVersion: record.ChartVersion,
		ValuesFileID: record.ValuesFileID,
		ImageTag:     record.ImageTag,
		Revision:     record.Revision,
		Status:       string(record.Status),
		Message:      record.Message,
		Operator:     record.Operator,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
	return o
}

// PaginatedHelmDeployRecordOutputObjs 是分页 Helm 部署记录列表。
//
// [bkms-cli 使用] 避免破坏性修改。
type PaginatedHelmDeployRecordOutputObjs struct {
	// 总记录数
	Count int64 `json:"count,string"`
	// 当前页 Helm 部署记录列表
	Results []*HelmDeployRecordOutputObj `json:"results"`
}

// ListHelmDeployRecordsOutput 是获取 Helm 应用部署记录列表的 JSON 响应。
//
// [bkms-cli 使用] 避免破坏性修改。
type ListHelmDeployRecordsOutput struct {
	// 分页 Helm 部署记录列表
	Data *PaginatedHelmDeployRecordOutputObjs `json:"data"`
}
