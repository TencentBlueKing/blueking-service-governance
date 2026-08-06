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

// Package tkerouteeni implements the tkeRouteEni AppSpec section, which controls whether the
// application uses TKE Route ENI (VPC-CNI) networking by injecting the corresponding Pod annotation.
package tkerouteeni

import "go.mongodb.org/mongo-driver/v2/bson"

// Spec stores the tkeRouteEni configuration.
type Spec struct {
	Enabled *bool `bson:"enabled,omitempty"`
}

// Clone deep-copies the section and collapses empty specs to nil.
func Clone(spec *Spec) *Spec {
	if !HasData(spec) {
		return nil
	}
	v := *spec.Enabled
	return &Spec{Enabled: &v}
}

// HasData returns whether the section carries any explicit configuration.
func HasData(spec *Spec) bool {
	return spec != nil && spec.Enabled != nil
}

// Merge overlays override onto base. The override value takes precedence when present.
func Merge(base, override *Spec) *Spec {
	if base == nil && override == nil {
		return nil
	}
	// override takes precedence
	if HasData(override) {
		return Clone(override)
	}
	return Clone(base)
}

// AppendPatch adds MongoDB $set entries for this section.
func AppendPatch(set *bson.D, spec *Spec) {
	if spec == nil {
		return
	}
	*set = append(*set, bson.E{Key: "tkeRouteEni.enabled", Value: spec.Enabled})
}
