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

package lifecycle

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"

// FromAppModel builds the lifecycle section from an AppModel.
func FromAppModel(am *appmodel.AppModel) *Spec {
	if am == nil {
		return nil
	}

	var spec Spec
	if lc := am.Workload.Lifecycle; lc != nil {
		spec.PostStart = fromAppModelHandler(lc.PostStart)
		spec.PreStop = fromAppModelHandler(lc.PreStop)
	}
	spec.TerminationGracePeriodSeconds = am.Workload.TerminationGracePeriodSeconds
	if !HasData(&spec) {
		return nil
	}
	return Clone(&spec)
}

// ApplyToAppModel applies the lifecycle section back into an AppModel.
func ApplyToAppModel(spec *Spec, am *appmodel.AppModel) *appmodel.AppModel {
	if spec == nil {
		am.Workload.Lifecycle = nil
		am.Workload.TerminationGracePeriodSeconds = nil
		return am
	}

	postStart := toAppModelHandler(spec.PostStart)
	preStop := toAppModelHandler(spec.PreStop)

	if postStart == nil && preStop == nil {
		am.Workload.Lifecycle = nil
	} else {
		am.Workload.Lifecycle = &appmodel.Lifecycle{
			PostStart: postStart,
			PreStop:   preStop,
		}
	}
	am.Workload.TerminationGracePeriodSeconds = spec.TerminationGracePeriodSeconds
	return am
}

// fromAppModelHandler converts an appmodel.LifecycleHandler to a section Handler.
func fromAppModelHandler(h *appmodel.LifecycleHandler) *Handler {
	if h == nil {
		return nil
	}

	handler := &Handler{
		Type: h.Type,
	}

	switch h.Type {
	case appmodel.LifecycleTypeExec:
		if h.ExecAction != nil {
			handler.Command = h.Command
			handler.ShCommand = h.ShCommand
			handler.SleepSeconds = h.SleepSeconds
		}
	case appmodel.LifecycleTypeHTTP:
		if h.HTTPGetAction != nil {
			handler.URL = h.URL
			handler.Port = h.Port
			handler.Headers = h.Headers
		}
	}
	return handler
}

// toAppModelHandler converts a section Handler to an appmodel.LifecycleHandler.
func toAppModelHandler(h *Handler) *appmodel.LifecycleHandler {
	if h == nil {
		return nil
	}

	handler := &appmodel.LifecycleHandler{
		TypeWrapper: appmodel.TypeWrapper{Type: h.Type},
	}

	switch h.Type {
	case appmodel.LifecycleTypeExec:
		handler.ExecAction = &appmodel.ExecAction{
			Command:      h.Command,
			ShCommand:    h.ShCommand,
			SleepSeconds: h.SleepSeconds,
		}
	case appmodel.LifecycleTypeHTTP:
		handler.HTTPGetAction = &appmodel.HTTPGetAction{
			URL:     h.URL,
			Port:    h.Port,
			Headers: h.Headers,
		}
	}
	return handler
}
