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
	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Spec stores replicas and CPU/memory requests/limits in a structured form.
type Spec struct {
	Replicas       *int32  `bson:"replicas,omitempty" validate:"omitempty,gte=0"`
	CPURequests    *string `bson:"cpuRequests,omitempty" validate:"omitempty,resource_quantity"`
	CPULimits      *string `bson:"cpuLimits,omitempty" validate:"omitempty,resource_quantity"`
	MemoryRequests *string `bson:"memoryRequests,omitempty" validate:"omitempty,resource_quantity"`
	MemoryLimits   *string `bson:"memoryLimits,omitempty" validate:"omitempty,resource_quantity"`
}

// Clone deep-copies the section and collapses empty specs to nil.
func Clone(spec *Spec) *Spec {
	if spec == nil {
		return nil
	}

	cloned := new(Spec)
	_ = copier.CopyWithOption(cloned, spec, copier.Option{DeepCopy: true})
	if !HasData(cloned) {
		return nil
	}
	return cloned
}

// HasData returns whether the section carries any explicit configuration.
func HasData(spec *Spec) bool {
	return spec != nil && (spec.Replicas != nil ||
		spec.CPURequests != nil ||
		spec.CPULimits != nil ||
		spec.MemoryRequests != nil ||
		spec.MemoryLimits != nil)
}

// Merge overlays non-nil values from override onto base.
// A nil field on override means the field is not set and the base value is kept.
func Merge(base, override *Spec) *Spec {
	switch {
	case base == nil && override == nil:
		return nil
	case base == nil:
		return Clone(override)
	case override == nil:
		return Clone(base)
	}

	merged := Clone(base)
	if override.Replicas != nil {
		merged.Replicas = override.Replicas
	}
	if override.CPURequests != nil {
		merged.CPURequests = override.CPURequests
	}
	if override.CPULimits != nil {
		merged.CPULimits = override.CPULimits
	}
	if override.MemoryRequests != nil {
		merged.MemoryRequests = override.MemoryRequests
	}
	if override.MemoryLimits != nil {
		merged.MemoryLimits = override.MemoryLimits
	}
	return Clone(merged)
}

// AppendPatch adds MongoDB $set entries for this section.
func AppendPatch(set *bson.D, spec *Spec) {
	if spec == nil {
		return
	}

	if spec.Replicas != nil {
		*set = append(*set, bson.E{Key: "resources.replicas", Value: spec.Replicas})
	}
	if spec.CPURequests != nil {
		*set = append(*set, bson.E{Key: "resources.cpuRequests", Value: spec.CPURequests})
	}
	if spec.CPULimits != nil {
		*set = append(*set, bson.E{Key: "resources.cpuLimits", Value: spec.CPULimits})
	}
	if spec.MemoryRequests != nil {
		*set = append(*set, bson.E{Key: "resources.memoryRequests", Value: spec.MemoryRequests})
	}
	if spec.MemoryLimits != nil {
		*set = append(*set, bson.E{Key: "resources.memoryLimits", Value: spec.MemoryLimits})
	}
}
