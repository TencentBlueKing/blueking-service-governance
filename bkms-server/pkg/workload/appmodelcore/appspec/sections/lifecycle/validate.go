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

import (
	"github.com/go-playground/validator/v10"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// RegisterValidation registers validators used by this section.
func RegisterValidation(v *validator.Validate) {
	v.RegisterStructValidation(validateSpec, Spec{})
}

// validateSpec checks that each configured handler is valid.
func validateSpec(sl validator.StructLevel) {
	spec := sl.Current().Interface().(Spec)

	if spec.PostStart != nil {
		validateLifecycleHandler(sl, "PostStart", spec.PostStart)
	}
	if spec.PreStop != nil {
		validateLifecycleHandler(sl, "PreStop", spec.PreStop)
	}
}

// validateLifecycleHandler validates a single lifecycle handler with a field prefix.
func validateLifecycleHandler(sl validator.StructLevel, fieldName string, h *Handler) {
	if h == nil {
		return
	}

	switch h.Type {
	case appmodel.LifecycleTypeExec:
		hasCmd := len(h.Command) > 0
		hasShCommand := h.ShCommand != ""
		if hasCmd && hasShCommand {
			sl.ReportError(h.Type, fieldName, "", "exec_command_or_sh_command_exclusive", "")
		}
		if !hasCmd && !hasShCommand && h.SleepSeconds == nil {
			sl.ReportError(h.Command, fieldName+".Command", "", "required_command_or_sh_command_or_sleep_seconds", "")
		}
		if h.SleepSeconds != nil && *h.SleepSeconds < 0 {
			sl.ReportError(*h.SleepSeconds, fieldName+".SleepSeconds", "", "gte", "0")
		}
	case appmodel.LifecycleTypeHTTP:
		if h.URL == "" {
			sl.ReportError(h.URL, fieldName+".URL", "", "required_for_http", "")
		}
	default:
		sl.ReportError(h.Type, fieldName+".Type", "", "oneof", "EXEC HTTP")
	}
}
