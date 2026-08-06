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
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
)

// AppSpecDevModeOutput is the JSON representation of the devMode section.
type AppSpecDevModeOutput struct {
	// 是否启用开发模式
	Enabled bool `json:"enabled"`
	// 开发模式根目录
	WorkPath string `json:"workPath"`
	// 脚本挂载路径
	MountPath string `json:"mountPath"`
}

// FromModel fills output fields from an AppSpec devMode section.
func (o *AppSpecDevModeOutput) FromModel(spec *appspec.DevModeSpec, appType string) *AppSpecDevModeOutput {
	if spec == nil {
		return nil
	}
	// 非标准行为：devMode 的路径字段不是用户配置项，而是由应用类型推导出来，响应前统一补齐。
	spec.SetPathsByAppType(appType)

	*o = AppSpecDevModeOutput{}
	if spec.Enabled != nil {
		o.Enabled = *spec.Enabled
	}
	if spec.WorkPath != nil {
		o.WorkPath = *spec.WorkPath
	}
	if spec.MountPath != nil {
		o.MountPath = *spec.MountPath
	}
	return o
}

// AppSpecDevModeInput is the input structure of the devMode section.
type AppSpecDevModeInput struct {
	// 是否启用开发模式
	Enabled bool `json:"enabled"`
	// 开发模式根目录，由应用类型决定，不接受自定义修改
	WorkPath *string `json:"workPath"`
	// 脚本挂载路径，由应用类型决定，不接受自定义修改
	MountPath *string `json:"mountPath"`
}

// ToModel converts input to an AppSpec devMode section.
func (i *AppSpecDevModeInput) ToModel(appType string) *appspec.DevModeSpec {
	if i == nil {
		return nil
	}
	spec := &appspec.DevModeSpec{
		Enabled: lo.ToPtr(i.Enabled),
	}
	// 非标准行为：兼容旧 API 的入参结构，但不允许客户端自定义 workPath/mountPath。
	spec.SetPathsByAppType(appType)
	return spec
}

// SetEnvAppSpecDevModeInput is the JSON body for setting env devMode.
type SetEnvAppSpecDevModeInput struct {
	// 待设置的 devMode section 值
	AppSpecDevMode *AppSpecDevModeInput `json:"appSpecDevMode" binding:"required"`
}

// AppSpecDevModeSectionOutput is the JSON response for querying devMode.
type AppSpecDevModeSectionOutput struct {
	Data *AppSpecDevModeOutput `json:"data"`
}
