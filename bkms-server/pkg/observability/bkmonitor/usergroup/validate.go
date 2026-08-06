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

package usergroup

import (
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// Validate 校验保存告警组请求参数。
func (r *SaveParams) Validate() error {
	if r == nil {
		return errors.New("request is nil")
	}
	if err := validate.Struct(r); err != nil {
		var validationErrs validator.ValidationErrors
		ok := errors.As(err, &validationErrs)
		if !ok || len(validationErrs) == 0 {
			return err
		}

		fe := validationErrs[0]
		return errors.Errorf("field %s failed on %s", fe.Field(), fe.Tag())
	}
	return nil
}
