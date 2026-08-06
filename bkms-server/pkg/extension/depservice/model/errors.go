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

package model

import (
	"errors"
	"fmt"
)

// NotFoundError represents a resource not found error
type NotFoundError struct {
	target string
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.target)
}

// NewNotFoundError creates a target not found error
func NewNotFoundError(target string) *NotFoundError {
	return &NotFoundError{target: target}
}

// AsNotFoundError 检查给定的错误 err 是否（或其错误链中是否）包含 *NotFoundError 类型的错误
func AsNotFoundError(err error) bool {
	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}
