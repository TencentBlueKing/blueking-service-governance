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

// Package gvr defines Kubernetes GroupVersionResource (GVR) mappings for common resources.
// These values can be determined by resource kind, and short names from `kubectl api-resources`.
//
// Note: For resources like CronJob, Ingress, or HPA, which may belong to different API groups
// in different clusters, use a dynamic client to fetch their GVR dynamically.
package gvr

import "k8s.io/apimachinery/pkg/runtime/schema"

// --- Core Resources ---

// NS represents Kubernetes Namespace resource.
var NS = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}

// --- Workloads ---

// Po represents Kubernetes Pod resource.
var Po = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "pods",
}

// GameDeploy represents Kubernetes GameDeployment resource.
var GameDeploy = schema.GroupVersionResource{
	Group:    "tkex.tencent.com",
	Version:  "v1alpha1",
	Resource: "gamedeployments",
}

// --- Network ---

// SVC represents Kubernetes Service resource.
var SVC = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "services",
}

// --- Storage ---

var (
	// CM represents Kubernetes ConfigMap resource.
	CM = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}

	// Secret represents Kubernetes Secret resource.
	Secret = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}
)
