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

package ginutils

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

// ErrInvalidRequestParams 请求参数无法解析或未通过校验，校验器给出的具体原因保留在错误详情中
var ErrInvalidRequestParams = bkerrs.Define(
	bkerrs.ErrCodeInvalidRequest, "REQUEST_PARAMS_INVALID", "request params invalid",
)

// BindJSON binds a JSON request body and converts binding errors to bkms errors.
func BindJSON(c *gin.Context, obj any) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return ErrInvalidRequestParams.WithCause(errors.Wrap(err, "bind json request"))
	}
	return nil
}

// BindURI binds URI path parameters and converts binding errors to bkms errors.
func BindURI(c *gin.Context, obj any) error {
	if err := c.ShouldBindUri(obj); err != nil {
		return ErrInvalidRequestParams.WithCause(errors.Wrap(err, "bind uri request"))
	}
	return nil
}

// BindQuery binds query parameters and converts binding errors to bkms errors.
func BindQuery(c *gin.Context, obj any) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return ErrInvalidRequestParams.WithCause(errors.Wrap(err, "bind query request"))
	}
	return nil
}

// BindURIJSON binds both URI path parameters and JSON request body, and converts binding
// errors to bkms errors.
func BindURIJSON(c *gin.Context, uriObj, jsonObj any) error {
	if err := BindURI(c, uriObj); err != nil {
		return err
	}
	if err := BindJSON(c, jsonObj); err != nil {
		return err
	}
	return nil
}

// BindURIQuery binds both URI path parameters and query parameters, and converts binding
// errors to bkms errors.
func BindURIQuery(c *gin.Context, uriObj, queryObj any) error {
	if err := BindURI(c, uriObj); err != nil {
		return err
	}
	if err := BindQuery(c, queryObj); err != nil {
		return err
	}
	return nil
}
