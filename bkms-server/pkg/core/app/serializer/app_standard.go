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

// Package serializer 定义 standard 应用相关的 Gin input/output 序列化结构和转换方法。
package serializer

import (
	standardapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/standard"
)

// ToStandardCreateParams 将 AppModelSpecInput 转换为 standard 内部创建参数类型。
// 语言/框架字段由 handler 直接写入 Application 顶层，不进 CreateParams。
func (input *AppModelSpecInput) ToStandardCreateParams() *standardapp.CreateParams {
	return &standardapp.CreateParams{
		Command: input.Command,
		Args:    input.Args,
		EnvVars: variableInputsToModel(input.EnvVars),
	}
}
