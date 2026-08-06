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

package gpa

import (
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
)

// validate 是 gpa 包内独立的结构体校验实例。
var validate = validator.New(validator.WithRequiredStructEnabled())

// cronParser 标准 5 段 Crontab parser（分 时 日 月 周），用于校验 schedule 语法合法性。
// 语义与 GPA 文档引用的标准 Crontab（https://en.wikipedia.org/wiki/Cron）一致。
var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
)

func init() {
	// 注册 crontab 自定义校验：校验标准 5 段 Crontab 语法合法性。
	_ = validate.RegisterValidation("crontab", validateCrontab)

	// 指标资源不重复（cpu、memory 各至多一条）属于跨字段约束，
	// 无法用单字段 tag 表达，注册为 struct-level 校验。
	validate.RegisterStructValidation(validateMetricsUnique, GPAConfig{})
	// 至少配置一种扩缩容模式（指标或定时），同为跨字段约束。
	validate.RegisterStructValidation(validateScaleMode, GPAConfig{})
}

// validateCrontab 是 crontab tag 的字段级校验函数。
func validateCrontab(fl validator.FieldLevel) bool {
	return isValidCrontab(fl.Field().String())
}

// isValidCrontab 校验 expr 是否为合法的标准 5 段 Crontab 表达式（分 时 日 月 周）。
// 仅校验语法合法性，不区分「时间段」与「时间点」。
func isValidCrontab(expr string) bool {
	_, err := cronParser.Parse(expr)
	return err == nil
}

// validateMetricsUnique 校验 GPAConfig.Metrics 中的 resource 不重复
func validateMetricsUnique(sl validator.StructLevel) {
	config, ok := sl.Current().Interface().(GPAConfig)
	if !ok {
		return
	}

	seen := make(map[ResourceName]struct{}, len(config.Metrics))
	for _, m := range config.Metrics {
		if _, dup := seen[m.Resource]; dup {
			sl.ReportError(config.Metrics, "Metrics", "metrics", "duplicate_resource", string(m.Resource))
			return
		}
		seen[m.Resource] = struct{}{}
	}
}

// validateScaleMode 校验至少配置了一种扩缩容模式（指标或定时）。
func validateScaleMode(sl validator.StructLevel) {
	config, ok := sl.Current().Interface().(GPAConfig)
	if !ok {
		return
	}

	if len(config.Metrics) == 0 && len(config.TimeRanges) == 0 {
		sl.ReportError(config.Metrics, "Metrics", "metrics", "at_least_one_mode", "")
	}
}

// formatValidationError 将 validator 的校验错误转换为友好的错误信息。
// 仅返回首条错误，与原有「遇错即返回」的语义保持一致。
func formatValidationError(err error) error {
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok || len(validationErrs) == 0 {
		return err
	}

	fe := validationErrs[0]
	switch {
	case fe.Field() == "MinReplicas":
		return errors.Errorf("minReplicas must be >= %d", MinReplicasLowerBound)
	case fe.Field() == "MaxReplicas":
		return errors.New("maxReplicas must be >= minReplicas")
	case fe.Field() == "Metrics" && fe.Tag() == "at_least_one_mode":
		return errors.New("at least one of metrics or timeRanges is required")
	case fe.Field() == "Metrics" && fe.Tag() == "max":
		return errors.Errorf("at most %d metrics (cpu, memory) are allowed", MaxMetricsCount)
	case fe.Field() == "Metrics" && fe.Tag() == "duplicate_resource":
		return errors.Errorf("duplicate metric resource %q", fe.Value())
	case fe.Field() == "Resource":
		return errors.Errorf("metric resource must be cpu or memory, got %q", fe.Value())
	case fe.Field() == "AverageUtilization":
		return errors.Errorf(
			"averageUtilization must be between %d and %d, got %v",
			UtilizationLowerBound, UtilizationUpperBound, fe.Value(),
		)
	case fe.Field() == "DesiredReplicas":
		return errors.New("desiredReplicas must be >= 1")
	case fe.Field() == "Schedule" && fe.Tag() == "crontab":
		return errors.Errorf("schedule must be a valid 5-field crontab expression, got %q", fe.Value())
	case fe.Field() == "Schedule":
		return errors.New("schedule is required")
	default:
		return errors.New(fe.Error())
	}
}
