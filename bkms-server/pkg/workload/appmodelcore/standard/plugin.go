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

// Package standard is an internal app type which does not have any special or extra effects; its main
// purpose is to help testing the workload plugin system.
package standard

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/plugin"
)

// Plugin implements the standard workload behavior without adding extra resources or configuration.
type Plugin struct{}

// Type returns the workload type handled by this plugin.
func (p Plugin) Type() string {
	return appmodel.WorkloadTypeStandard
}

// Start initializes a plugin session for this build.
func (p Plugin) Start(
	_ context.Context,
	_ *envmodel.Environment,
	_ *bkmsapp.Application,
	_ *appmodel.AppModel,
	_ plugin.RenderContext,
) (plugin.WorkloadPluginSession, error) {
	return standardSession{}, nil
}

type standardSession struct{}

// Storage returns any extra storage requirements for the standard workload.
// The standard workload does not provide any extra storage.
func (s standardSession) Storage(
	_ context.Context,
) ([]corev1.VolumeMount, []corev1.Volume, error) {
	return nil, nil, nil
}

// ExtraResources returns any extra Kubernetes resources required for the standard workload.
// The standard workload does not provide any extra resources.
func (s standardSession) ExtraResources(
	_ context.Context,
) ([]unstructured.Unstructured, error) {
	return nil, nil
}

// InitContainers returns init containers for the standard workload.
// The standard workload does not require any init containers.
func (s standardSession) InitContainers(
	_ context.Context,
) ([]corev1.Container, error) {
	return nil, nil
}
