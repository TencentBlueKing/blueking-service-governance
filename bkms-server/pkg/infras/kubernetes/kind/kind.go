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

// Package kind include k8s resource kinds,
// use SHORTNAMES column from `kubectl api-resources`
package kind

// --- Core Resources ---

const (
	// Node ...
	Node = "Node"
	// NS ...
	NS = "Namespace"
)

// --- Workloads ---

const (
	// Deploy ...
	Deploy = "Deployment"
	// RS ...
	RS = "ReplicaSet"
	// DS ...
	DS = "DaemonSet"
	// STS ...
	STS = "StatefulSet"
	// CJ ...
	CJ = "CronJob"
	// Job ...
	Job = "Job"
	// Po ...
	Po = "Pod"

	// GameDeploy 蓝鲸提供的游戏场景用的 Deployment
	GameDeploy = "GameDeployment"
	// GameSTS 蓝鲸提供的游戏场景用的 StatefulSet
	GameSTS = "GameStatefulSet"
)

// --- Networks ---

const (
	// Ing ...
	Ing = "Ingress"
	// SVC ...
	SVC = "Service"
	// EP ...
	EP = "Endpoints"
	// PolarisCfg 北极星组件
	PolarisCfg = "PolarisConfig"
)

// --- Storage ---

const (
	// CM ...
	CM = "ConfigMap"
	// Secret ...
	Secret = "Secret"
	// PV ...
	PV = "PersistentVolume"
	// PVC ...
	PVC = "PersistentVolumeClaim"
	// SC ...
	SC = "StorageClass"
)

// --- RBAC ---

const (
	// SA ...
	SA = "ServiceAccount"
	// ClusterRole ...
	ClusterRole = "ClusterRole"
	// ClusterRoleBinding ...
	ClusterRoleBinding = "ClusterRoleBinding"
)

// --- Autoscaling ---

const (
	// HPA ...
	HPA = "HorizontalPodAutoscaler"
	// GPA 蓝鲸自研扩缩容组件
	GPA = "GeneralPodAutoscaler"
)

// --- Custom ---

const (
	// CRD 自定义资源定义
	CRD = "CustomResourceDefinition"
	// CObj 自定义资源对象
	CObj = "CustomObject"
)

// IsClusterScoped 判断给定的 Kubernetes 资源 Kind 是否为集群级别资源（不需要 Namespace）
func IsClusterScoped(kindName string) bool {
	switch kindName {
	case Node, NS, PV, SC, ClusterRole, ClusterRoleBinding, CRD:
		return true
	default:
		return false
	}
}

// IsWorkload 判断给定的 Kubernetes 资源 Kind 是否为工作负载类型（会引用容器镜像）
func IsWorkload(kindName string) bool {
	switch kindName {
	case Deploy, DS, STS, CJ, Job, Po, GameDeploy, GameSTS:
		return true
	default:
		return false
	}
}
