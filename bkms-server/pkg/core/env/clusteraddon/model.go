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

// Package clusteraddon 集群 Addon 部署管理
package clusteraddon

import (
	"cmp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	helmrelease "helm.sh/helm/v3/pkg/release"
)

// ClusterAddonDef 集群 Addon 定义
type ClusterAddonDef struct {
	// ID 主键
	ID bson.ObjectID `bson:"_id,omitempty" yaml:"-"`

	// Name 组件名称（唯一标识）
	Name string `bson:"name" yaml:"name"`
	// DisplayName 展示名称
	DisplayName string `bson:"displayName" yaml:"displayName"`
	// Description 组件描述
	Description string `bson:"description" yaml:"description"`

	// ChartInfo Chart 相关信息
	ChartInfo HelmChartInfo `bson:"chartInfo" yaml:"chartInfo"`
	// RequiredForAppTypes 必装该 Addon 的应用类型列表
	RequiredForAppTypes []string `bson:"requiredForAppTypes" yaml:"requiredForAppTypes"`
	// OptionalForAppTypes 可选安装该 Addon 的应用类型列表
	OptionalForAppTypes []string `bson:"optionalForAppTypes" yaml:"optionalForAppTypes"`
	// Creator 创建人
	Creator string `bson:"creator" yaml:"-"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt" yaml:"-"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt" yaml:"-"`
}

// DefaultNamespaceValue 默认命名空间
const DefaultNamespaceValue = "bcs-system"

// GetNamespace 获取命名空间，如果为空则返回默认值
func (d *ClusterAddonDef) GetNamespace(ns string) string {
	return cmp.Or(ns, d.ChartInfo.DefaultNamespace, DefaultNamespaceValue)
}

// ClusterAddonInfo 集群 Addon 信息（领域对象），包含定义、版本、集群状态等完整信息
type ClusterAddonInfo struct {
	// ---- 元数据 ----
	// Name 插件名称
	Name string
	// DisplayName 展示名称
	DisplayName string
	// Description 插件描述
	Description string
	// RequiredForAppTypes 必装该插件的应用类型列表
	RequiredForAppTypes []string
	// OptionalForAppTypes 可选安装该插件的应用类型列表
	OptionalForAppTypes []string
	// SupportedActions 支持的操作列表（如 install, upgrade, uninstall）
	SupportedActions []string

	// ---- HelmChart 信息 ----
	ChartInfo *HelmChartInfo
	// ---- 集群安装信息 ----
	InstallInfo *ClusterInstallInfo
}

// HelmChartInfo HelmChart 相关信息
type HelmChartInfo struct {
	// ChartName Chart 名称
	ChartName string `bson:"chartName" yaml:"chartName"`
	// ReleaseName Release 名称
	ReleaseName string `bson:"releaseName" yaml:"releaseName"`
	// DefaultChartVersion 默认安装时使用的 Chart 版本
	DefaultChartVersion string `bson:"defaultChartVersion" yaml:"defaultChartVersion"`
	// DefaultNamespace 默认安装命名空间
	DefaultNamespace string `bson:"defaultNamespace" yaml:"defaultNamespace"`
	// AvailableVersions 仓库中可用的 Chart 版本列表（运行时填充，不持久化）
	AvailableVersions []string `bson:"-" yaml:"-"`
	// ExampleValues 安装示例参数（YAML 字符串，可包含注释）
	ExampleValues string `bson:"exampleValues,omitempty" yaml:"exampleValues,omitempty"`
}

// AddonStatus 插件安装状态
type AddonStatus = helmrelease.Status

// ClusterInstallInfo 集群中的安装状态信息
type ClusterInstallInfo struct {
	// Status 安装状态（空字符串表示未安装）
	Status AddonStatus
	// Message 状态信息（安装失败时给出提示信息）
	Message string
	// CurrentChartVersion 当前已安装的 Chart 版本
	CurrentChartVersion string
	// CurrentValues 当前已安装的 values 参数（JSON 字符串，未安装时为空）
	CurrentValues string
	// Namespace 插件安装的命名空间
	Namespace string
}

// NewAddonInfoFromDef 从插件定义构造 AddonInfo 基础信息
func NewAddonInfoFromDef(def *ClusterAddonDef, namespace string) *ClusterAddonInfo {
	chartInfo := def.ChartInfo // 值拷贝
	return &ClusterAddonInfo{
		Name:                def.Name,
		DisplayName:         def.DisplayName,
		Description:         def.Description,
		RequiredForAppTypes: def.RequiredForAppTypes,
		OptionalForAppTypes: def.OptionalForAppTypes,
		ChartInfo:           &chartInfo,
		InstallInfo: &ClusterInstallInfo{
			Namespace: namespace,
		},
	}
}
