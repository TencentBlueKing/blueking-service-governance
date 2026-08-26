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

// Package serializer defines Gin input and output serializers for HostPort APIs.
package serializer

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"

// AppURIInput binds the appID path parameter.
type AppURIInput struct {
	AppID string `uri:"appID" binding:"required"`
}

// PutHostPortsInput is the request body for replacing hostport declarations.
// An empty ports list clears the declaration (store deletes the config document).
type PutHostPortsInput struct {
	Ports []int32 `json:"ports" binding:"dive,min=1,max=65535"`
}

// HostPortEnvStateOutput is one federated environment's HostPort status.
type HostPortEnvStateOutput struct {
	AppliedPorts       []int32 `json:"appliedPorts"`
	PendingAddPorts    []int32 `json:"pendingAddPorts"`
	PendingRemovePorts []int32 `json:"pendingRemovePorts"`
}

// FromModel fills output fields from an EnvStateView domain model.
func (o *HostPortEnvStateOutput) FromModel(view hostport.EnvStateView) *HostPortEnvStateOutput {
	*o = HostPortEnvStateOutput{
		AppliedPorts:       nonNilPorts(view.AppliedPorts),
		PendingAddPorts:    nonNilPorts(view.PendingAddPorts),
		PendingRemovePorts: nonNilPorts(view.PendingRemovePorts),
	}
	return o
}

// HostPortsOutput is the response for listing / replacing hostports.
type HostPortsOutput struct {
	Ports     []int32                           `json:"ports"`
	EnvStates map[string]HostPortEnvStateOutput `json:"envStates"`
}

// FromModel fills output fields from a HostPorts domain aggregate.
func (o *HostPortsOutput) FromModel(m *hostport.HostPorts) *HostPortsOutput {
	ports := []int32{}
	if m != nil && m.Ports != nil {
		ports = m.Ports
	}
	envStates := map[string]HostPortEnvStateOutput{}
	if m != nil {
		envStates = make(map[string]HostPortEnvStateOutput, len(m.EnvStates))
		for name, view := range m.EnvStates {
			envStates[name] = *new(HostPortEnvStateOutput).FromModel(view)
		}
	}
	*o = HostPortsOutput{
		Ports:     ports,
		EnvStates: envStates,
	}
	return o
}

func nonNilPorts(ports []int32) []int32 {
	if ports == nil {
		return []int32{}
	}
	return ports
}
