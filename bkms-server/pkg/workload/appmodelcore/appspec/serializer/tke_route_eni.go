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

package serializer

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"

// AppSpecTkeRouteEniInput is the input structure of the tkeRouteEni section.
type AppSpecTkeRouteEniInput struct {
	// 是否启用 TKE Route ENI (VPC-CNI) 网络模式
	Enabled bool `json:"enabled"`
}

// ToModel converts input to an AppSpec tkeRouteEni section.
func (i *AppSpecTkeRouteEniInput) ToModel() *appspec.TkeRouteEniSpec {
	if i == nil {
		return nil
	}
	return &appspec.TkeRouteEniSpec{Enabled: &i.Enabled}
}

// AppSpecTkeRouteEniOutput is the JSON representation of the tkeRouteEni section.
type AppSpecTkeRouteEniOutput struct {
	// 是否启用 TKE Route ENI (VPC-CNI) 网络模式
	Enabled *bool `json:"enabled"`
}

// FromModel fills output fields from an AppSpec tkeRouteEni section.
func (o *AppSpecTkeRouteEniOutput) FromModel(spec *appspec.TkeRouteEniSpec) *AppSpecTkeRouteEniOutput {
	if spec == nil {
		return nil
	}
	*o = AppSpecTkeRouteEniOutput{Enabled: spec.Enabled}
	return o
}

// AppSpecTkeRouteEniSectionOutput is the JSON response for querying tkeRouteEni.
type AppSpecTkeRouteEniSectionOutput struct {
	Data *AppSpecTkeRouteEniOutput `json:"data"`
}
