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

// Package serializer defines Gin input and output serializers for bkintegrations APIs.
package serializer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"

	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
)

var (
	// appIDPattern 应用 ID 校验规则：以小写字母开头，可包含小写字母、数字、中划线，不能以中划线结尾
	appIDPattern = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")

	// workspaceIDPattern 工作空间 ID 校验规则
	workspaceIDPattern = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")

	// envNamePattern 环境名称 校验规则
	envNamePattern = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("app_id", validateAppID); err != nil {
			panic("failed to register app_id validator: " + err.Error())
		}
		if err := v.RegisterValidation("workspace_id", validateWorkspaceID); err != nil {
			panic("failed to register workspace_id validator: " + err.Error())
		}
		if err := v.RegisterValidation("env_name", validateEnvName); err != nil {
			panic("failed to register env_name validator: " + err.Error())
		}
		v.RegisterStructValidation(validateGetInstanceTimeSeriesQueryInput, GetInstanceTimeSeriesQueryInput{})
	}
}

// validateGetInstanceTimeSeriesQueryInput 结构体级别校验：跨字段校验逻辑
func validateGetInstanceTimeSeriesQueryInput(sl validator.StructLevel) {
	input := sl.Current().Interface().(GetInstanceTimeSeriesQueryInput)

	// StartTime 必须 <= EndTime
	if input.StartTime > input.EndTime {
		sl.ReportError(input.EndTime, "EndTime", "EndTime",
			"start_time must be less than or equal to end_time", "")
	}

	// 校验指标标识是否合法
	if input.MetricKey != "" && !lo.ContainsBy(bkmmodel.MetricDefinitions, func(def bkmmodel.MetricDefinition) bool {
		return def.Key == input.MetricKey
	}) {
		sl.ReportError(input.MetricKey, "MetricKey", "MetricKey",
			fmt.Sprintf("invalid metric key: %s", input.MetricKey), "")
	}

	// 过滤空白后，Instances 不能为空
	hasNonEmpty := lo.ContainsBy(input.Instances, func(inst string) bool {
		return strings.TrimSpace(inst) != ""
	})
	if !hasNonEmpty {
		sl.ReportError(input.Instances, "Instances", "Instances",
			"instances must contain at least one non-empty value", "")
	}
}

func validateAppID(fl validator.FieldLevel) bool {
	return appIDPattern.MatchString(fl.Field().String())
}

func validateWorkspaceID(fl validator.FieldLevel) bool {
	return workspaceIDPattern.MatchString(fl.Field().String())
}

func validateEnvName(fl validator.FieldLevel) bool {
	return envNamePattern.MatchString(fl.Field().String())
}

// -----------------------------------------------------------------------------
// Empty output
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}
