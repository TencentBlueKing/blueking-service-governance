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

package appspec

import (
	stderrors "errors"

	pkgerrors "github.com/pkg/errors"
)

// ErrAppSpecNotFound is returned when app spec is not found in store.
var ErrAppSpecNotFound = stderrors.New("app spec not found")

// ErrAppSpecValidation is a sentinel error for validation failures.
var ErrAppSpecValidation = stderrors.New("app spec validation failed")

func wrapValidationErr(err error) error {
	return pkgerrors.Wrap(stderrors.Join(ErrAppSpecValidation, err), "validating app spec")
}
