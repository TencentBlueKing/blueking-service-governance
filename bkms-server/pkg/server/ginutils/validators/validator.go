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

package validators

import (
	"regexp"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var uriSlugPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("uri_slug", validateURISlug); err != nil {
			panic("failed to register uri_slug validator: " + err.Error())
		}
	}
}

// validateURISlug 用于 `uri_slug` 验证标签，它用来匹配常见的 URI 片段格式。
// 也可以被用来快速验证路径中的 AppName、EnvName 等字段（在不需要过强校验需求时推荐）。
func validateURISlug(fl validator.FieldLevel) bool {
	return uriSlugPattern.MatchString(fl.Field().String())
}
