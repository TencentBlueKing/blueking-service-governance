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

package appmodel

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

// use a single instance of Validate, it caches struct info
func init() {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		// Register struct-level validation: Component
		validate.RegisterStructValidation(validateComponentStruct, component.Component{})
		validate.RegisterStructValidation(validateWorkloadStruct, Workload{})
	})
}

// validateComponentStruct 校验 Component 结构体
// 规则：
// 1. Name 必须包含小写字母、数字、中划线，必须以字母开头，必须以字母或数字结尾，长度限制20位以内
// 2. ComponentInst.Type 和 ComponentRef.RefWorkspaceCompName 两者必须有且仅有一个有值
func validateComponentStruct(sl validator.StructLevel) {
	comp := sl.Current().Interface().(component.Component)

	// NOTE: 存量数据（如 UpdateStrategy、ImagePullSecret）等命名使用了大写，这些组件将在迁移后下掉。
	// 这里先关闭 DB 层的校验，通过 API 入口（组件的增改）进行校验，待处理完存量数据后打开 DB 层的校验。

	// 验证 Name 字段格式
	// if !compNameRegex.MatchString(comp.Name) {
	// 	sl.ReportError(comp.Name, "Name", "Name", "comp_name", comp.Name)
	// }

	// 验证 Type 和 RefWorkspaceCompName 的互斥关系
	hasType := comp.Type != ""
	hasRef := comp.RefWorkspaceCompName != ""

	if !hasType && !hasRef {
		sl.ReportError(comp, "Component", "", "comp_type_or_ref_required", "")
	}
	if hasType && hasRef {
		sl.ReportError(comp, "Component", "", "comp_type_xor_ref", "")
	}
}

// validateWorkloadStruct 校验 Workload 结构体
// 规则：
// 1. EnvVars 中的 Key 在同一 app 内必须唯一
func validateWorkloadStruct(sl validator.StructLevel) {
	workload := sl.Current().Interface().(Workload)
	if len(workload.EnvVars) == 0 {
		return
	}

	seen := make(map[string]struct{}, len(workload.EnvVars))
	for _, envVar := range workload.EnvVars {
		if err := envvartypes.ValidateEnvVarKey(envVar.Key); err != nil {
			sl.ReportError(workload.EnvVars, "EnvVars", "EnvVars", "env_var_key", envVar.Key)
			return
		}
		if _, ok := seen[envVar.Key]; ok {
			sl.ReportError(workload.EnvVars, "EnvVars", "EnvVars", "env_var_key_unique", envVar.Key)
			return
		}
		seen[envVar.Key] = struct{}{}
	}
}

// formatValidationError 格式化 validator 模块返回的错误信息
func formatValidationError(err error) error {
	var validationErrs validator.ValidationErrors
	ok := errors.As(err, &validationErrs)
	if !ok {
		return err
	}

	messages := make([]string, 0, len(validationErrs))
	for _, fe := range validationErrs {
		switch fe.Tag() {
		case "comp_name":
			messages = append(messages, fmt.Sprintf(
				"%s '%v' is invalid: must start with lowercase letter, contain only lowercase letters/numbers/hyphens, end with letter/number, max 20 chars",
				fe.Namespace(),
				fe.Value(),
			))
		case "comp_type_or_ref_required", "comp_type_xor_ref":
			messages = append(
				messages,
				fmt.Sprintf("%s must have only one of comp_type or ref_workspace_comp_name", fe.Namespace()),
			)
		case "env_var_key_unique":
			messages = append(
				messages,
				fmt.Sprintf("%s contains duplicate env var key '%v'", fe.Namespace(), fe.Param()),
			)
		case "env_var_key":
			messages = append(
				messages,
				fmt.Sprintf(
					"%s contains invalid env var key '%v': must start with a letter or underscore and contain only letters, numbers, and underscores",
					fe.Namespace(),
					fe.Param(),
				),
			)
		default:
			messages = append(messages, fe.Error())
		}
	}
	return errors.New(strings.Join(messages, "; "))
}
