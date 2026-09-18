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

// Package bscp api client，BSCP 服务 API 入参校验
package bscp

import (
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

var (
	validateOnce sync.Once

	validate *validator.Validate
)

func init() {
	validateOnce.Do(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
	})
}

// Validate 通用校验方法
func Validate(v any) error {
	if v == nil {
		return errors.New("request is nil")
	}
	return validate.Struct(v)
}

// Validate 校验创建环境请求
func (r *CreateEnvironmentReq) Validate() error {
	if err := Validate(r); err != nil {
		return err
	}
	switch r.Type {
	case "prod", "staging", "test", "dev":
	default:
		return errors.Errorf("invalid environment type: %s", r.Type)
	}
	return nil
}

// Validate 校验创建 App 请求
func (r *CreateAppReq) Validate() error {
	if err := Validate(r); err != nil {
		return err
	}
	switch r.ConfigType {
	case ConfigTypeFile, ConfigTypeKV:
	default:
		return errors.Errorf("invalid config_type: %s", r.ConfigType)
	}
	switch r.DataType {
	case DataTypeAny, DataTypeString, DataTypeNumber, DataTypeText,
		DataTypeJSON, DataTypeXML, DataTypeYAML, DataTypeSecret:
	default:
		return errors.Errorf("invalid data_type: %s", r.DataType)
	}
	if !r.IsApprove {
		return nil
	}
	switch r.ApproveType {
	case ApproveTypeCountSign, ApproveTypeOrSign:
	default:
		return errors.Errorf("invalid approve_type: %s", r.ApproveType)
	}
	if r.Approver == "" {
		return errors.New("approver is required when is_approve is true")
	}
	return nil
}

// Validate 校验创建客户端密钥请求
func (r *CreateCredentialReq) Validate() error {
	return Validate(r)
}

// Validate 校验更新密钥关联服务规则请求
func (r *UpdateCredentialScopeReq) Validate() error {
	return Validate(r)
}

// Validate 校验创建脚本请求
func (r *CreateHookReq) Validate() error {
	return Validate(r)
}

// Validate 校验更新 App 绑定的前后置脚本请求
func (r *UpdateConfigHookReq) Validate() error {
	return Validate(r)
}
