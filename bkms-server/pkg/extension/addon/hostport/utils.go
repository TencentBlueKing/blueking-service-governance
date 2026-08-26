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

package hostport

import (
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

const (
	// AnnotationEnabledKey enables BCS random HostPort webhook injection.
	AnnotationEnabledKey = "randhostport.webhook.bkbcs.tencent.com"
	// AnnotationEnabledVal is the value that enables the webhook.
	AnnotationEnabledVal = "true"
	// AnnotationPortsKey declares container ports that need random HostPort mapping.
	AnnotationPortsKey = "ports.randhostport.webhook.bkbcs.tencent.com"
)

// NormalizePorts returns a sorted unique copy of ports.
func NormalizePorts(ports []int32) []int32 {
	if len(ports) == 0 {
		return []int32{}
	}
	uniq := lo.Uniq(ports)
	slices.Sort(uniq)
	return uniq
}

// DiffPorts returns ports in desired but not applied (add) and in applied but not desired (remove).
func DiffPorts(desired, applied []int32) (add, remove []int32) {
	desiredSet := lo.SliceToMap(NormalizePorts(desired), func(p int32) (int32, struct{}) {
		return p, struct{}{}
	})
	appliedSet := lo.SliceToMap(NormalizePorts(applied), func(p int32) (int32, struct{}) {
		return p, struct{}{}
	})

	for p := range desiredSet {
		if _, ok := appliedSet[p]; !ok {
			add = append(add, p)
		}
	}
	for p := range appliedSet {
		if _, ok := desiredSet[p]; !ok {
			remove = append(remove, p)
		}
	}
	return NormalizePorts(add), NormalizePorts(remove)
}

// ComputeEnvState builds the pending-deploy view for one env.
// state == nil means the env has never been reconciled after a HostPort-aware deploy.
func ComputeEnvState(desired []int32, state *HostPortEnvState) EnvStateView {
	desired = NormalizePorts(desired)
	if state == nil {
		if len(desired) == 0 {
			return EnvStateView{
				AppliedPorts:       []int32{},
				PendingAddPorts:    []int32{},
				PendingRemovePorts: []int32{},
			}
		}
		return EnvStateView{
			AppliedPorts:       []int32{},
			PendingAddPorts:    desired,
			PendingRemovePorts: []int32{},
		}
	}

	applied := NormalizePorts(state.AppliedPorts)
	add, remove := DiffPorts(desired, applied)
	return EnvStateView{
		AppliedPorts:       applied,
		PendingAddPorts:    add,
		PendingRemovePorts: remove,
	}
}

// FormatPortsAnnotationValue joins ports as a comma-separated ascending string.
func FormatPortsAnnotationValue(ports []int32) string {
	ports = NormalizePorts(ports)
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		parts = append(parts, strconv.FormatInt(int64(p), 10))
	}
	return strings.Join(parts, ",")
}

// BuildPodAnnotations returns HostPort webhook annotations for the given ports.
// Returns nil when ports is empty.
func BuildPodAnnotations(ports []int32) map[string]string {
	ports = NormalizePorts(ports)
	if len(ports) == 0 {
		return nil
	}
	return map[string]string{
		AnnotationEnabledKey: AnnotationEnabledVal,
		AnnotationPortsKey:   FormatPortsAnnotationValue(ports),
	}
}
