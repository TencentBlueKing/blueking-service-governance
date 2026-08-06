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

package updatestrategy

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// RegisterValidation registers validators used by this section.
func RegisterValidation(v *validator.Validate) {
	_ = v.RegisterValidation("int_or_percent_gte0", validateIntOrPercentGTE0)
}

// validateIntOrPercentGTE0 checks that a string field is either an integer or a percentage, and that
// the value is greater than or equal to 0.
//
// Examples of valid values: "0", "5", "100", "0%", "50%".
func validateIntOrPercentGTE0(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.String {
		return false
	}

	raw := field.String()
	if strings.HasSuffix(raw, "%") {
		raw = strings.TrimSuffix(raw, "%")
		if raw == "" {
			return false
		}
	}

	val, err := strconv.Atoi(raw)
	if err != nil {
		return false
	}
	return val >= 0
}
