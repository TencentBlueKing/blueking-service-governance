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

package probe

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// FromAppModel builds the probe section from an AppModel.
func FromAppModel(am *appmodel.AppModel) *Spec {
	if am == nil {
		return nil
	}

	spec := &Spec{
		Liveness:  fromAppModelProbe(am.Workload.LivenessProbe),
		Readiness: fromAppModelProbe(am.Workload.ReadinessProbe),
		Startup:   fromAppModelProbe(am.Workload.StartupProbe),
	}
	return Clone(spec)
}

// ApplyToAppModel applies the probe section back into an AppModel.
func ApplyToAppModel(spec *Spec, am *appmodel.AppModel) *appmodel.AppModel {
	if spec == nil {
		am.Workload.LivenessProbe = nil
		am.Workload.ReadinessProbe = nil
		am.Workload.StartupProbe = nil
		return am
	}

	am.Workload.LivenessProbe = toAppModelProbe(spec.Liveness)
	am.Workload.ReadinessProbe = toAppModelProbe(spec.Readiness)
	am.Workload.StartupProbe = toAppModelProbe(spec.Startup)
	return am
}

// fromAppModelProbe converts an appmodel.Probe to a section Probe.
func fromAppModelProbe(p *appmodel.Probe) *Probe {
	if p == nil {
		return nil
	}

	probe := &Probe{
		Handler: fromAppModelProbeHandler(p.ProbeHandler),
	}

	probe.InitialDelaySeconds = lo.EmptyableToPtr(p.InitialDelaySeconds)
	probe.TimeoutSeconds = lo.EmptyableToPtr(p.TimeoutSeconds)
	probe.PeriodSeconds = lo.EmptyableToPtr(p.PeriodSeconds)
	probe.SuccessThreshold = lo.EmptyableToPtr(p.SuccessThreshold)
	probe.FailureThreshold = lo.EmptyableToPtr(p.FailureThreshold)

	return probe
}

// fromAppModelProbeHandler converts an appmodel.ProbeHandler to a section Handler.
func fromAppModelProbeHandler(h *appmodel.ProbeHandler) *Handler {
	if h == nil {
		return nil
	}

	handler := &Handler{
		Type: h.Type,
	}

	switch h.Type {
	case appmodel.ProbeTypeExec:
		if h.ExecAction != nil {
			handler.Command = h.Command
			handler.ShCommand = h.ShCommand
		}
	case appmodel.ProbeTypeHTTP:
		if h.HTTPGetAction != nil {
			handler.URL = h.URL
			handler.Port = h.HTTPGetAction.Port
			handler.Headers = h.Headers
		}
	case appmodel.ProbeTypeTCP:
		if h.TCPSocketAction != nil {
			handler.Port = h.TCPSocketAction.Port
		}
	}
	return handler
}

// toAppModelProbe converts a section Probe to an appmodel.Probe.
func toAppModelProbe(p *Probe) *appmodel.Probe {
	if p == nil {
		return nil
	}

	probe := &appmodel.Probe{
		ProbeHandler: toAppModelProbeHandler(p.Handler),
	}

	if p.InitialDelaySeconds != nil {
		probe.InitialDelaySeconds = *p.InitialDelaySeconds
	}
	if p.TimeoutSeconds != nil {
		probe.TimeoutSeconds = *p.TimeoutSeconds
	}
	if p.PeriodSeconds != nil {
		probe.PeriodSeconds = *p.PeriodSeconds
	}
	if p.SuccessThreshold != nil {
		probe.SuccessThreshold = *p.SuccessThreshold
	}
	if p.FailureThreshold != nil {
		probe.FailureThreshold = *p.FailureThreshold
	}
	return probe
}

// toAppModelProbeHandler converts a section Handler to an appmodel.ProbeHandler.
func toAppModelProbeHandler(h *Handler) *appmodel.ProbeHandler {
	if h == nil {
		return nil
	}

	handler := &appmodel.ProbeHandler{
		TypeWrapper: appmodel.TypeWrapper{Type: h.Type},
	}

	switch h.Type {
	case appmodel.ProbeTypeExec:
		handler.ExecAction = &appmodel.ExecAction{
			Command:   h.Command,
			ShCommand: h.ShCommand,
		}
	case appmodel.ProbeTypeHTTP:
		handler.HTTPGetAction = &appmodel.HTTPGetAction{
			URL:     h.URL,
			Port:    h.Port,
			Headers: h.Headers,
		}
	case appmodel.ProbeTypeTCP:
		handler.TCPSocketAction = &appmodel.TCPSocketAction{
			Port: h.Port,
		}
	}
	return handler
}
