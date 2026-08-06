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

package serializer

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	componentNamePattern = regexp.MustCompile("^[a-z](?:[a-z0-9-]*[a-z0-9])?$")
	workspaceIDPattern   = regexp.MustCompile("^[a-z](?:[a-z0-9-]*[a-z0-9])?$")
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("workspace_id", validateWorkspaceID); err != nil {
			panic("failed to register workspace_id validator: " + err.Error())
		}
		if err := v.RegisterValidation("component_name", validateComponentName); err != nil {
			panic("failed to register component_name validator: " + err.Error())
		}
	}
}

func validateWorkspaceID(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	if len(input) < 1 || len(input) > 27 {
		return false
	}
	return workspaceIDPattern.MatchString(input)
}

func validateComponentName(fl validator.FieldLevel) bool {
	input := fl.Field().String()
	if len(input) < 1 || len(input) > 20 {
		return false
	}
	return componentNamePattern.MatchString(input)
}
