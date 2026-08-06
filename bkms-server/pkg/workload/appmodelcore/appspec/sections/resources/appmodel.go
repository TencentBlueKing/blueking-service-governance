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

package resources

import (
	"maps"
	"strings"

	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

const (
	resourceCPU    = "cpu"
	resourceMemory = "memory"
)

// FromAppModel builds the section from an AppModel.
func FromAppModel(appModel *appmodel.AppModel) *Spec {
	if appModel == nil {
		return nil
	}

	spec := &Spec{Replicas: appModel.Replicas}
	if cpuRaw, ok := appModel.Workload.Resources[resourceCPU]; ok && cpuRaw != "" {
		spec.CPURequests, spec.CPULimits = parseResourceValue(cpuRaw)
	}
	if memoryRaw, ok := appModel.Workload.Resources[resourceMemory]; ok && memoryRaw != "" {
		spec.MemoryRequests, spec.MemoryLimits = parseResourceValue(memoryRaw)
	}
	return Clone(spec)
}

// ApplyToAppModel applies the section into AppModel.
func ApplyToAppModel(spec *Spec, appModel *appmodel.AppModel) *appmodel.AppModel {
	appModel.Replicas = nil
	if spec != nil {
		appModel.Replicas = spec.Replicas
	}
	appModel.Workload.Resources = buildAppModelResources(appModel.Workload.Resources, spec)
	return appModel
}

// buildAppModelResources strictly synchronizes appspec-managed resource keys while preserving unrelated ones.
func buildAppModelResources(existing map[string]string, spec *Spec) map[string]string {
	resources := maps.Clone(existing)
	if resources == nil {
		resources = map[string]string{}
	}

	delete(resources, resourceCPU)
	delete(resources, resourceMemory)

	if spec != nil && spec.CPURequests != nil {
		resources[resourceCPU] = buildResourceValue(spec.CPURequests, spec.CPULimits)
	}
	if spec != nil && spec.MemoryRequests != nil {
		resources[resourceMemory] = buildResourceValue(spec.MemoryRequests, spec.MemoryLimits)
	}
	if len(resources) == 0 {
		return nil
	}
	return resources
}

// parseResourceValue splits a "requests-limits" string (e.g. "100m-200m") into two pointers.
// A single value without "-" (e.g. "256Mi") is treated as both requests and limits.
func parseResourceValue(raw string) (*string, *string) {
	before, after, found := strings.Cut(raw, "-")
	if !found {
		return lo.ToPtr(before), lo.ToPtr(before)
	}
	return lo.ToPtr(before), lo.ToPtr(after)
}

func buildResourceValue(requests, limits *string) string {
	if limits == nil || *limits == "" {
		limits = requests
	}
	return *requests + "-" + *limits
}
