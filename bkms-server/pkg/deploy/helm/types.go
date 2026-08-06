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

package helm

import "time"

// ReleaseSpec Helm Chart Release 相关配置
type ReleaseSpec struct {
	ProjectCode     string
	ClusterID       string
	Namespace       string
	ReleaseName     string
	ChartRepoName   string
	ChartName       string
	TrafficLaneName string
}

// PreviewResult Manifest 预览结果
type PreviewResult struct {
	// CurrentManifests 当前已部署的 Manifest 内容
	CurrentManifests string
	// TargetManifests 目标版本的 Manifest 内容
	TargetManifests string
	// MissingVars values 中引用但未定义的非 env 命名空间变量（以 "ns.var" 形式，如 bkms.BAR）
	MissingVars []string
	// MissingEnvVars values 中引用但未定义的 env 命名空间变量
	MissingEnvVars []string
}

// DeployResult 部署操作结果
type DeployResult struct {
	// ProjectCode 蓝盾项目 ID
	ProjectCode string
	// ClusterID 集群 ID
	ClusterID string
	// Name Release 名称
	Name string
	// Namespace 命名空间
	Namespace string
	// Revision Release 版本号
	Revision string
	// Status 部署状态
	Status string
	// Chart Chart 名称
	Chart string
	// ChartVersion Chart 版本号
	ChartVersion string
}

// ReleaseHistory Release 历史版本
type ReleaseHistory struct {
	// Name Release 名称
	Name string
	// Namespace 命名空间
	Namespace string
	// Revision Release 版本号
	Revision string
	// Status 部署状态
	Status string
	// Message 状态描述信息
	Message string
	// Chart Chart 名称
	Chart string
	// ChartVersion Chart 版本号
	ChartVersion string
	// Values 部署配置
	Values string
	// UpdatedAt 更新时间
	UpdatedAt time.Time
}

// LintResult Chart 校验结果
type LintResult struct {
	// Errors 错误列表（校验不通过）
	Errors []string
	// Warnings 警告列表（校验通过但有隐患）
	Warnings []string
	// Infos 信息列表（校验通过）
	Infos []string
}

// HasErrors 判断 Lint 结果是否包含错误
func (r *LintResult) HasErrors() bool {
	return len(r.Errors) > 0
}
