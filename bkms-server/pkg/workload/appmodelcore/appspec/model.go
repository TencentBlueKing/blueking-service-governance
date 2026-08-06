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

package appspec

import (
	annotationssection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/annotations"
	devmodesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/devmode"
	labelssection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/labels"
	lifecyclesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/lifecycle"
	probesection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/probe"
	resourcessection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/resources"
	tkerouteenisection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/tke_route_eni"
	updatestrategysection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/updatestrategy"
)

const (
	// DefaultEnvName 用来表示默认配置，代表应用在没有特定环境配置时的默认行为。
	DefaultEnvName = ""
)

// AppSpec stores application spec overrides for an application.
type AppSpec struct {
	// AppID is the application ID.
	AppID string `bson:"appID" validate:"required"`

	// EnvName is the environment name, e.g., "stag", "prod". Name is used instead of EnvID for
	// better readability. EnvName="" represents the default spec for the application, which applies
	// when there is no environment-specific override.
	EnvName string `bson:"envName"`

	// Different sections of the app spec. All fields are optional.
	// INFO: Add new sections here when needed, and implement the corresponding logic in sub-packages under sections/.
	Resources      *ResourcesSpec      `bson:"resources,omitempty"`
	UpdateStrategy *UpdateStrategySpec `bson:"updateStrategy,omitempty"`
	DevMode        *DevModeSpec        `bson:"devMode,omitempty"`
	Lifecycle      *LifecycleSpec      `bson:"lifecycle,omitempty"`
	Probes         *ProbeSpec          `bson:"probes,omitempty"`
	Labels         *LabelsSpec         `bson:"labels,omitempty"`
	Annotations    *AnnotationsSpec    `bson:"annotations,omitempty"`
	TkeRouteEni    *TkeRouteEniSpec    `bson:"tkeRouteEni,omitempty"`
}

// Create type aliases

// ResourcesSpec stores replicas and CPU/memory requests/limits in a structured form.
type ResourcesSpec = resourcessection.Spec

// UpdateStrategySpec stores rolling update settings.
type UpdateStrategySpec = updatestrategysection.Spec

// DevModeSpec stores dev mode settings.
type DevModeSpec = devmodesection.Spec

// LifecycleSpec stores container lifecycle hook settings.
type LifecycleSpec = lifecyclesection.Spec

// ProbeSpec stores container probe settings.
type ProbeSpec = probesection.Spec

// LabelsSpec stores user-defined Kubernetes labels.
type LabelsSpec = labelssection.Spec

// AnnotationsSpec stores user-defined Kubernetes annotations.
type AnnotationsSpec = annotationssection.Spec

// TkeRouteEniSpec stores TKE Route ENI (VPC-CNI) networking configuration.
type TkeRouteEniSpec = tkerouteenisection.Spec

// HasConfiguredSections reports whether the app spec still has any section data.
func (s *AppSpec) HasConfiguredSections() bool {
	return s != nil &&
		(s.Resources != nil || s.UpdateStrategy != nil || s.DevMode != nil ||
			s.Lifecycle != nil || s.Probes != nil || s.Labels != nil || s.Annotations != nil ||
			s.TkeRouteEni != nil)
}

// Merge overlays non-nil values from override onto base.
func Merge(base, override *AppSpec) *AppSpec {
	switch {
	case base == nil && override == nil:
		return nil
	case base == nil:
		return cloneSpec(override)
	case override == nil:
		return cloneSpec(base)
	}

	merged := &AppSpec{
		AppID:   override.AppID,
		EnvName: override.EnvName,
	}
	for _, section := range registeredSections {
		section.mergeTo(base, override, merged)
	}
	return merged
}

// cloneSpec creates a deep copy of the given AppSpec.
func cloneSpec(spec *AppSpec) *AppSpec {
	if spec == nil {
		return nil
	}

	cloned := &AppSpec{
		AppID:   spec.AppID,
		EnvName: spec.EnvName,
	}
	for _, section := range registeredSections {
		section.cloneTo(spec, cloned)
	}
	return cloned
}
