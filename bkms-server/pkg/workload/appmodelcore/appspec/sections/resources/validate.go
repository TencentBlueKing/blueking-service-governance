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

package resources

import (
	"reflect"

	"github.com/go-playground/validator/v10"
	"k8s.io/apimachinery/pkg/api/resource"
)

// RegisterValidation registers validators used by this section.
func RegisterValidation(v *validator.Validate) {
	_ = v.RegisterValidation("resource_quantity", validateResourceQuantity)
	v.RegisterStructValidation(validateStruct, Spec{})
}

// validateResourceQuantity checks if a string is a valid Kubernetes resource quantity.
func validateResourceQuantity(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}
	_, err := resource.ParseQuantity(field.String())
	return err == nil
}

// validateStruct performs cross-field validation for the Spec struct, ensuring that if limits are set,
// requests must also be set, and that requests do not exceed limits.
func validateStruct(sl validator.StructLevel) {
	spec := sl.Current().Interface().(Spec)

	validateResourcePair(sl, "CPURequests", "CPULimits", spec.CPURequests, spec.CPULimits)
	validateResourcePair(sl, "MemoryRequests", "MemoryLimits", spec.MemoryRequests, spec.MemoryLimits)
}

func validateResourcePair(
	sl validator.StructLevel,
	requestField, limitField string,
	request, limit *string,
) {
	if request == nil && limit == nil {
		return
	}
	if request == nil && limit != nil {
		sl.ReportError(limit, limitField, "", "resource_limit_requires_request", "")
		return
	}
	if request == nil || limit == nil {
		return
	}

	requestQuantity, err := resource.ParseQuantity(*request)
	if err != nil {
		return
	}
	limitQuantity, err := resource.ParseQuantity(*limit)
	if err != nil {
		return
	}
	if requestQuantity.Cmp(limitQuantity) > 0 {
		sl.ReportError(request, requestField, "", "resource_request_lte_limit", *limit)
	}
}

// ValidateReplicas validates a replicas pointer in isolation.
func ValidateReplicas(replicas *int32) error {
	if replicas == nil {
		return nil
	}
	return validator.New().Var(*replicas, "gte=0")
}
