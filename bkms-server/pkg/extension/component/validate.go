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

package component

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

func init() {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		// Register custom validations: ComponentDef
		_ = validate.RegisterValidation("prop_type", validatePropType)
		_ = validate.RegisterValidation("comp_fragment", validateCompFragment)
		validate.RegisterStructValidation(validateComponentDefFragments, ComponentDef{})
		validate.RegisterStructValidation(validateProperty, Property{})
	})
}

// ValidateComponentDef validates a component definition using the domain validation rules.
func ValidateComponentDef(componentDef *ComponentDef) error {
	if err := validate.Struct(componentDef); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// ValidateFragmentTemplate checks a patcher/spec template renders to a YAML mapping.
func ValidateFragmentTemplate(fragment string) error {
	if strings.TrimSpace(fragment) == "" {
		return errors.New("template is empty")
	}

	// The fragment uses Go template syntax, so render it with empty data before YAML validation.
	// because the template expressions such as {{ .replicas }} are illegal YAML values.
	// As a workaround, we render the template with empty data first, unmarshal the content
	// as YAML later.
	tmpl, err := template.New("comp_fragment").Option("missingkey=zero").Parse(fragment)
	if err != nil {
		return errors.Wrap(err, "parse Go template")
	}
	var buf bytes.Buffer
	data := make(map[string]any)
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return errors.Wrap(err, "execute Go template")
	}

	// Unmarshal the rendered content as YAML to check if it's a valid mapping.
	var payload map[string]any
	if err = yaml.Unmarshal(buf.Bytes(), &payload); err != nil {
		return errors.Wrap(err, "parse rendered YAML")
	}
	if payload == nil {
		return errors.New("rendered YAML is not a mapping")
	}
	return nil
}

func validateComponentDefFragments(sl validator.StructLevel) {
	componentDef := sl.Current().Interface().(ComponentDef)
	if len(componentDef.Patchers)+len(componentDef.Specs) == 0 {
		sl.ReportError(componentDef.Patchers, "Patchers", "patchers", "component_fragments_required", "")
	}
}

func validatePropType(fl validator.FieldLevel) bool {
	propType, ok := fl.Field().Interface().(PropType)
	if !ok {
		return false
	}
	return propType.IsValid()
}

// validateCompFragment checks a patcher/spec template renders to a YAML mapping.
func validateCompFragment(fl validator.FieldLevel) bool {
	return ValidateFragmentTemplate(fl.Field().String()) == nil
}

func validateProperty(sl validator.StructLevel) {
	prop, ok := sl.Current().Interface().(Property)
	if !ok || prop.Type != PropTypeSelect {
		return
	}

	if len(prop.Options) == 0 {
		sl.ReportError(prop.Options, "Options", "options", "select_options_required", "")
		return
	}

	// 检查所有 option 的 value 和 label 是否非空
	for _, option := range prop.Options {
		if strings.TrimSpace(option.Value) == "" || strings.TrimSpace(option.Label) == "" {
			sl.ReportError(option, "Options", "options", "select_option_empty_value_or_label", "")
			return
		}
	}

	// defaultValue 可选
	if prop.DefaultValue == nil {
		return
	}

	defaultValue, ok := prop.DefaultValue.(string)
	if !ok {
		sl.ReportError(prop.DefaultValue, "DefaultValue", "defaultValue", "select_default_not_string", "")
		return
	}

	// 如果 defaultValue 是空字符串，则认为没有默认值
	if defaultValue == "" {
		return
	}

	// 检查 defaultValue 是否在 options 中
	if _, ok := lo.Find(prop.Options, func(opt PropertyOption) bool { return opt.Value == defaultValue }); !ok {
		sl.ReportError(defaultValue, "DefaultValue", "defaultValue", "select_default_not_in_options", "")
	}
}

// formatValidationError 格式化 validator 模块返回的错误信息
func formatValidationError(err error) error {
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err
	}

	messages := make([]string, 0, len(validationErrs))
	for _, fe := range validationErrs {
		switch fe.Tag() {
		case "prop_type":
			messages = append(messages, fmt.Sprintf("%s has invalid property type %v", fe.Namespace(), fe.Value()))
		case "comp_fragment":
			messages = append(messages, fmt.Sprintf("%s is not a valid YAML mapping template", fe.Namespace()))
		case "component_fragments_required":
			messages = append(messages, "at least one patcher or spec is required")
		case "select_options_required":
			messages = append(messages, fmt.Sprintf("%s requires non-empty options for SELECT type", fe.Namespace()))
		case "select_option_empty_value_or_label":
			messages = append(messages, fmt.Sprintf("%s contains option with empty value", fe.Namespace()))
		case "select_default_not_string":
			messages = append(messages, fmt.Sprintf("%s defaultValue must be string for SELECT type", fe.Namespace()))
		case "select_default_not_in_options":
			messages = append(
				messages,
				fmt.Sprintf("%s defaultValue must be one of options values for SELECT type", fe.Namespace()),
			)
		default:
			messages = append(messages, fe.Error())
		}
	}
	return errors.New(strings.Join(messages, "; "))
}
