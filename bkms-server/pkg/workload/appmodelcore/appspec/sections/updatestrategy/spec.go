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

package updatestrategy

import (
	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Spec stores rolling update settings.
type Spec struct {
	MaxUnavailable *string `bson:"maxUnavailable,omitempty" validate:"omitempty,int_or_percent_gte0"`
	MaxSurge       *string `bson:"maxSurge,omitempty" validate:"omitempty,int_or_percent_gte0"`
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
	return spec != nil && (spec.MaxUnavailable != nil || spec.MaxSurge != nil)
}

// Merge overlays non-nil values from override onto base.
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
	if override.MaxUnavailable != nil {
		merged.MaxUnavailable = override.MaxUnavailable
	}
	if override.MaxSurge != nil {
		merged.MaxSurge = override.MaxSurge
	}
	return Clone(merged)
}

// AppendPatch adds MongoDB $set entries for this section.
func AppendPatch(set *bson.D, spec *Spec) {
	if spec == nil {
		return
	}
	if spec.MaxUnavailable != nil {
		*set = append(*set, bson.E{Key: "updateStrategy.maxUnavailable", Value: spec.MaxUnavailable})
	}
	if spec.MaxSurge != nil {
		*set = append(*set, bson.E{Key: "updateStrategy.maxSurge", Value: spec.MaxSurge})
	}
}
