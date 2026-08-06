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

// Package serializer 定义 TAF 应用相关的 Gin input/output 序列化结构和转换方法。
package serializer

import (
	tafapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/taf"
)

// TafSpecInput is the TAF spec input.
type TafSpecInput struct {
	// 配置文件名
	FileName string `json:"fileName" binding:"required"`
	// 配置文件路径
	FilePath string `json:"filePath" binding:"required"`
	// 配置文件内容
	FileContent string `json:"fileContent"`
}

// TafSpecOutputObj is the TAF spec output.
type TafSpecOutputObj struct {
	// 配置文件名
	FileName string `json:"fileName"`
	// 配置文件路径
	FilePath string `json:"filePath"`
	// 配置文件内容
	FileContent string `json:"fileContent"`
}

// ToTafCreateParams 将 AppModelSpecInput 转换为 taf 内部创建参数类型
func (input *AppModelSpecInput) ToTafCreateParams() *tafapp.CreateParams {
	params := &tafapp.CreateParams{
		Command: input.Command,
		Args:    input.Args,
		EnvVars: variableInputsToModel(input.EnvVars),
	}
	if input.TafSpec != nil {
		params.TafConfig = &tafapp.TafConfigParams{
			FileName:    input.TafSpec.FileName,
			FilePath:    input.TafSpec.FilePath,
			FileContent: input.TafSpec.FileContent,
		}
	}
	return params
}

// ToTafUpdateParams 将 AppModelSpecInput 转换为 taf 内部更新参数类型
func (input *AppModelSpecInput) ToTafUpdateParams() *tafapp.UpdateParams {
	// 更新 Spec 时忽略 EnvVars 以兼容旧客户端，应用环境变量应通过独立 CRUD 接口修改。
	params := &tafapp.UpdateParams{
		Command: input.Command,
		Args:    input.Args,
	}
	if input.TafSpec != nil {
		params.TafConfig = &tafapp.TafConfigParams{
			FileName: input.TafSpec.FileName,
			FilePath: input.TafSpec.FilePath,
		}
	}
	return params
}
