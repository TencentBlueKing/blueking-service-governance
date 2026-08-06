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

package devmode

import (
	"slices"

	"github.com/go-playground/validator/v10"
)

// RegisterValidation registers validators used by this section.
func RegisterValidation(v *validator.Validate) {
	v.RegisterStructValidation(validateStruct, Spec{})
}

// validateStruct 检查 Spec 数据的有效性。
// WorkPath 和 MountPath 只允许为空或等于已知的合法路径值（trpc/taf 对应的路径）。
func validateStruct(sl validator.StructLevel) {
	spec := sl.Current().Interface().(Spec)

	if spec.WorkPath != nil && !slices.Contains(allowedWorkPaths, *spec.WorkPath) {
		sl.ReportError(*spec.WorkPath, "WorkPath", "", "oneof", "trpc or taf work path")
	}
	if spec.MountPath != nil && !slices.Contains(allowedMountPaths, *spec.MountPath) {
		sl.ReportError(*spec.MountPath, "MountPath", "", "oneof", "trpc or taf mount path")
	}
}
