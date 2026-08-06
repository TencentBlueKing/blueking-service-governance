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

// Package model 定义了应用配置管理相关的纯数据模型。
package model

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/samber/lo"
)

// validate 包级校验器实例（复用，避免重复创建）
var validate = func() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterStructValidation(validateSnapshot, Snapshot{})
	return v
}()

func validateSnapshot(sl validator.StructLevel) {
	d := sl.Current().Interface().(Snapshot)
	if d.Metadata == nil {
		return // 已由 required tag 报错
	}
	if d.Metadata.MountPath == "" {
		sl.ReportError(d.Metadata.MountPath, "Metadata.MountPath", "MountPath", "required", "")
	}
	if d.Metadata.Token == "" {
		sl.ReportError(d.Metadata.Token, "Metadata.Token", "Token", "required", "")
	}
	if d.Metadata.FeedAddr == "" {
		sl.ReportError(d.Metadata.FeedAddr, "Metadata.FeedAddr", "FeedAddr", "required", "")
	}
	if d.Metadata.WorkloadName == "" {
		sl.ReportError(d.Metadata.WorkloadName, "Metadata.WorkloadName", "WorkloadName", "required", "")
	}
	if d.EnvBinding != nil && len(d.EnvBinding.Services) == 0 {
		sl.ReportError(d.EnvBinding.Services, "EnvBinding.Services", "Services", "min", "")
	}
}

// Snapshot Metadata + EnvBinding 的聚合视图（不持久化）。
type Snapshot struct {
	// Metadata 元信息
	Metadata *Metadata `validate:"required"`
	// EnvBinding 环境绑定
	EnvBinding *EnvBinding `validate:"required"`
}

// GetServiceNames 获取绑定的下发服务名称列表（逗号分隔）
func (d *Snapshot) GetServiceNames() string {
	if d.EnvBinding == nil {
		return ""
	}
	names := lo.Map(d.EnvBinding.Services, func(svc ServiceRef, _ int) string {
		return svc.Name
	})
	return strings.Join(names, ",")
}

// Validate 校验聚合配置的必要字段
func (d *Snapshot) Validate() error {
	return validate.Struct(d)
}
