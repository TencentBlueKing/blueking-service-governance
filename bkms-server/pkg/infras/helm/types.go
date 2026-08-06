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

import helmrelease "helm.sh/helm/v3/pkg/release"

const (
	// StatusUnknown indicates that a release is in an uncertain state.
	StatusUnknown = helmrelease.StatusUnknown
	// StatusDeployed indicates that the release has been pushed to Kubernetes.
	StatusDeployed = helmrelease.StatusDeployed
	// StatusUninstalled indicates that a release has been uninstalled from Kubernetes.
	StatusUninstalled = helmrelease.StatusUninstalled
	// StatusSuperseded indicates that this release object is outdated and a newer one exists.
	StatusSuperseded = helmrelease.StatusSuperseded
	// StatusFailed indicates that the release was not successfully deployed.
	StatusFailed = helmrelease.StatusFailed
	// StatusUninstalling indicates that an uninstall operation is underway.
	StatusUninstalling = helmrelease.StatusUninstalling
	// StatusPendingInstall indicates that an install operation is underway.
	StatusPendingInstall = helmrelease.StatusPendingInstall
	// StatusPendingUpgrade indicates that an upgrade operation is underway.
	StatusPendingUpgrade = helmrelease.StatusPendingUpgrade
	// StatusPendingRollback indicates that a rollback operation is underway.
	StatusPendingRollback = helmrelease.StatusPendingRollback

	// ---------- 以下为特殊状态（Helm SDK 不支持，为自定义扩展） ----------

	// StatusPollingTimeout 轮询超时
	StatusPollingTimeout helmrelease.Status = "polling-timeout"
	// StatusPollingBroken 轮询中断
	StatusPollingBroken helmrelease.Status = "polling-broken"
	// StatusNotFound 表示 Release 不存在
	StatusNotFound helmrelease.Status = "not-found"
)

// IsStable 判断状态是否为稳定态（不再变化，直到下次人工操作）
func IsStable(s helmrelease.Status) bool {
	switch s {
	case StatusDeployed,
		StatusUninstalled,
		StatusSuperseded,
		StatusFailed:
		return true
	case StatusPollingTimeout,
		StatusPollingBroken:
		return true
	default:
		return false
	}
}

// Chart ...
type Chart struct {
	// Chart 名称
	Name string
	// Chart 版本
	Version string
	// App 版本
	AppVersion string
	// Chart 描述
	Description string
}

// DeployResult 部署结果
type DeployResult struct {
	// 部署状态
	Status helmrelease.Status
	// 部署详情
	Description string
	// 部署时间
	CreatedAt string
}

// Release ...
type Release struct {
	// release 名称
	Name string
	// 部署的命名空间
	Namespace string
	// release 版本（Revision）
	Version string
	// chart 信息
	Chart Chart
	// 部署信息
	DeployResult DeployResult
	// 部署配置信息
	Values map[string]any
	// 部署的 k8s 资源信息
	Resources []map[string]any
	// 存储 release secret 名称
	SecretName string
}
