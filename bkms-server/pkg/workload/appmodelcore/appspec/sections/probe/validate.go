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
	"github.com/go-playground/validator/v10"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// RegisterValidation registers validators used by this section.
func RegisterValidation(v *validator.Validate) {
	v.RegisterStructValidation(validateSpec, Spec{})
}

// validateSpec checks that each configured probe is valid.
func validateSpec(sl validator.StructLevel) {
	spec := sl.Current().Interface().(Spec)

	if spec.Liveness != nil {
		validateProbe(sl, "Liveness", spec.Liveness)
	}
	if spec.Readiness != nil {
		validateProbe(sl, "Readiness", spec.Readiness)
	}
	if spec.Startup != nil {
		validateProbe(sl, "Startup", spec.Startup)
	}
}

// validateProbe validates a single probe with a field name prefix.
func validateProbe(sl validator.StructLevel, fieldName string, p *Probe) {
	if p == nil {
		return
	}
	validateProbeFields(sl, fieldName+".", *p)
}

// validateProbeFields validates probe fields.
func validateProbeFields(sl validator.StructLevel, prefix string, p Probe) {
	if p.Handler == nil {
		sl.ReportError(p.Handler, prefix+"Handler", "", "required", "")
		return
	}
	validateHandlerFields(sl, prefix, *p.Handler)
	if p.InitialDelaySeconds != nil && *p.InitialDelaySeconds < 0 {
		sl.ReportError(*p.InitialDelaySeconds, prefix+"InitialDelaySeconds", "", "gte", "0")
	}
	if p.TimeoutSeconds != nil && *p.TimeoutSeconds < 0 {
		sl.ReportError(*p.TimeoutSeconds, prefix+"TimeoutSeconds", "", "gte", "0")
	}
	if p.PeriodSeconds != nil && *p.PeriodSeconds < 0 {
		sl.ReportError(*p.PeriodSeconds, prefix+"PeriodSeconds", "", "gte", "0")
	}
	if p.SuccessThreshold != nil && *p.SuccessThreshold < 0 {
		sl.ReportError(*p.SuccessThreshold, prefix+"SuccessThreshold", "", "gte", "0")
	}
	if p.FailureThreshold != nil && *p.FailureThreshold < 0 {
		sl.ReportError(*p.FailureThreshold, prefix+"FailureThreshold", "", "gte", "0")
	}
}

// validateHandlerFields validates handler fields.
func validateHandlerFields(sl validator.StructLevel, prefix string, h Handler) {
	switch h.Type {
	case appmodel.ProbeTypeExec:
		hasCmd := len(h.Command) > 0
		hasShCommand := h.ShCommand != ""
		switch {
		case hasCmd && hasShCommand:
			sl.ReportError(h.Type, prefix+"Handler", "", "exec_command_or_sh_command_exclusive", "")
		case !hasCmd && !hasShCommand:
			sl.ReportError(h.Type, prefix+"Handler", "", "exec_command_or_sh_command_required", "")
		}
	case appmodel.ProbeTypeHTTP:
		if h.URL == "" {
			sl.ReportError(h.URL, prefix+"Handler.URL", "", "required_for_http", "")
		}
		if h.Port <= 0 {
			sl.ReportError(h.Port, prefix+"Handler.Port", "", "required_for_http", "")
		}
	case appmodel.ProbeTypeTCP:
		if h.Port <= 0 {
			sl.ReportError(h.Port, prefix+"Handler.Port", "", "required_for_tcp", "")
		}
	default:
		sl.ReportError(h.Type, prefix+"Handler.Type", "", "oneof", "EXEC HTTP TCP")
	}
}
