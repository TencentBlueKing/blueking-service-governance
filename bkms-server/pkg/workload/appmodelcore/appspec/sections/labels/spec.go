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

// Package labels implements the labels AppSpec section, which stores user-defined Kubernetes
// labels for an application and supports both app-default and env-level overrides.
package labels

import (
	"maps"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Spec stores user-defined Kubernetes labels.
type Spec struct {
	Labels map[string]string `bson:"labels,omitempty"`
}

// Clone deep-copies the section and collapses empty specs to nil.
func Clone(spec *Spec) *Spec {
	if spec == nil {
		return nil
	}

	cloned := &Spec{Labels: maps.Clone(spec.Labels)}
	if !HasData(cloned) {
		return nil
	}
	return cloned
}

// HasData returns whether the section carries any explicit configuration.
func HasData(spec *Spec) bool {
	return spec != nil && len(spec.Labels) > 0
}

// Merge overlays override onto base using key-level merge semantics: keys in override override the
// same keys in base, while keys only present in base are preserved.
func Merge(base, override *Spec) *Spec {
	switch {
	case base == nil && override == nil:
		return nil
	case base == nil:
		return Clone(override)
	case override == nil:
		return Clone(base)
	}

	merged := make(map[string]string, len(base.Labels)+len(override.Labels))
	maps.Copy(merged, base.Labels)
	maps.Copy(merged, override.Labels)
	return Clone(&Spec{Labels: merged})
}

// AppendPatch adds MongoDB $set entries for this section.
func AppendPatch(set *bson.D, spec *Spec) {
	if spec == nil {
		return
	}
	if spec.Labels != nil {
		*set = append(*set, bson.E{Key: "labels.labels", Value: spec.Labels})
	}
}
