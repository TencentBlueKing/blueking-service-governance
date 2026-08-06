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

// Package serializer defines Gin input and output serializers for cluster-addon APIs.
package serializer

import (
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
)

// -----------------------------------------------------------------------------
// Path inputs
// -----------------------------------------------------------------------------

// EnvURIInput is the path input for APIs scoped by environment.
type EnvURIInput struct {
	// 环境 ID
	EnvID string `uri:"envID" binding:"required,min=1"`
}

// EnvAddonURIInput is the path input for APIs scoped by environment and addon name.
type EnvAddonURIInput struct {
	// 环境 ID
	EnvID string `uri:"envID" binding:"required,min=1"`
	// 插件名称
	AddonName string `uri:"addonName" binding:"required,min=1"`
}

// -----------------------------------------------------------------------------
// List cluster addons
// -----------------------------------------------------------------------------

// ListClusterAddonsQueryInput is the query input for listing cluster addons.
type ListClusterAddonsQueryInput struct {
	// 命名空间（可选，默认为 bcs-system）
	Namespace string `form:"namespace"`
}

// ListClusterAddonsOutput is the JSON response for listing cluster addons.
type ListClusterAddonsOutput struct {
	// 插件列表
	Addons []*ClusterAddonInfoOutput `json:"addons"`
}

// ClusterAddonInfoOutput is the JSON representation of a cluster addon info.
type ClusterAddonInfoOutput struct {
	// 插件名称
	Name string `json:"name"`
	// 展示名称
	DisplayName string `json:"displayName"`
	// 插件描述
	Description string `json:"description"`
	// 必装该插件的应用类型列表
	RequiredForAppTypes []string `json:"requiredForAppTypes"`
	// 可选安装该插件的应用类型列表
	OptionalForAppTypes []string `json:"optionalForAppTypes"`
	// 支持的操作列表（如 install, upgrade, uninstall）
	SupportedActions []string `json:"supportedActions"`
	// HelmChart 信息
	ChartInfo *HelmChartInfoOutput `json:"chartInfo"`
	// 集群安装信息
	InstallInfo *ClusterInstallInfoOutput `json:"installInfo"`
}

// HelmChartInfoOutput is the JSON representation of helm chart info.
type HelmChartInfoOutput struct {
	// Chart 名称
	ChartName string `json:"chartName"`
	// 默认安装时使用的 Chart 版本
	DefaultChartVersion string `json:"defaultChartVersion"`
	// 仓库中可用的 Chart 版本列表
	AvailableVersions []string `json:"availableVersions"`
	// 安装示例参数（YAML 字符串，可包含注释）
	ExampleValues string `json:"exampleValues"`
}

// ClusterInstallInfoOutput is the JSON representation of cluster install info.
type ClusterInstallInfoOutput struct {
	// 当前安装状态（空字符串表示未安装）
	Status string `json:"status"`
	// 状态信息（安装失败时给出提示信息）
	Message string `json:"message"`
	// 当前已安装的 Chart 版本
	CurrentChartVersion string `json:"currentChartVersion"`
	// 当前已安装的 values 参数（JSON 字符串，未安装时为空）
	CurrentValues string `json:"currentValues"`
	// 插件安装的命名空间
	Namespace string `json:"namespace"`
}

// FromModel fills output fields from a ClusterAddonInfo domain model.
func (o *ClusterAddonInfoOutput) FromModel(info clusteraddon.ClusterAddonInfo) *ClusterAddonInfoOutput {
	*o = ClusterAddonInfoOutput{
		Name:                info.Name,
		DisplayName:         info.DisplayName,
		Description:         info.Description,
		RequiredForAppTypes: info.RequiredForAppTypes,
		OptionalForAppTypes: info.OptionalForAppTypes,
		SupportedActions:    info.SupportedActions,
	}
	if info.ChartInfo != nil {
		o.ChartInfo = &HelmChartInfoOutput{
			ChartName:           info.ChartInfo.ChartName,
			DefaultChartVersion: info.ChartInfo.DefaultChartVersion,
			AvailableVersions:   info.ChartInfo.AvailableVersions,
			ExampleValues:       info.ChartInfo.ExampleValues,
		}
	}
	if info.InstallInfo != nil {
		o.InstallInfo = &ClusterInstallInfoOutput{
			Status:              string(info.InstallInfo.Status),
			Message:             info.InstallInfo.Message,
			CurrentChartVersion: info.InstallInfo.CurrentChartVersion,
			CurrentValues:       info.InstallInfo.CurrentValues,
			Namespace:           info.InstallInfo.Namespace,
		}
	}
	return o
}

// -----------------------------------------------------------------------------
// Upsert cluster addon
// -----------------------------------------------------------------------------

// UpsertClusterAddonInput is the JSON input for deploying/updating a cluster addon.
type UpsertClusterAddonInput struct {
	// 命名空间（可选，默认为插件定义中的 defaultNamespace）
	Namespace string `json:"namespace"`
	// Chart 版本
	ChartVersion string `json:"chartVersion" binding:"required,min=1"`
	// Helm values 参数（JSON 格式）
	Values map[string]any `json:"values"`
}

// -----------------------------------------------------------------------------
// Delete cluster addon
// -----------------------------------------------------------------------------

// DeleteClusterAddonQueryInput is the query input for deleting a cluster addon.
type DeleteClusterAddonQueryInput struct {
	// 命名空间（可选，默认为插件定义中的 defaultNamespace）
	Namespace string `form:"namespace"`
}

// DeleteClusterAddonOutput is the JSON response for deleting a cluster addon.
type DeleteClusterAddonOutput struct {
	// 卸载状态
	Status string `json:"status"`
	// 状态描述
	Message string `json:"message"`
}
