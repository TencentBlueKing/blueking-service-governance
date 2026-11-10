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

// Package serializer 定义 tRPC 应用相关的 Gin input/output 序列化结构和转换方法。
package serializer

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// TrpcSpecInput is the tRPC spec input.
type TrpcSpecInput struct {
	// 编程语言
	Language string `json:"language" binding:"required,oneof=go cpp"`
	// 配置文件名，创建应用时必填，写入 framework def，应用模型不保存。
	// todo 兼容：更新时可不传；传了仍同步到 def。更新改走 def 接口后，该字段只用于创建。
	FileName string `json:"fileName"`
	// 配置文件路径，创建应用时必填，写入 framework def 的挂载目录，应用模型不保存。
	// todo 兼容：更新时可不传；传了仍同步到 def。更新改走 def 接口后，该字段只用于创建。
	FilePath string `json:"filePath" binding:"omitempty,mount_dir"`
	// 配置文件内容
	FileContent string `json:"fileContent"`
}

// TrpcSpecOutputObj is the tRPC spec output.
type TrpcSpecOutputObj struct {
	// 编程语言
	Language string `json:"language"`
	// 配置文件名。
	// todo 兼容：详情仍返回该字段，值从 framework def 填回。调用方改读 def 后删除。
	FileName string `json:"fileName"`
	// 配置文件路径。
	// todo 兼容：详情仍返回该字段，值从 framework def 填回。调用方改读 def 后删除。
	FilePath string `json:"filePath"`
	// 框架配置文件的默认内容
	FileContent string `json:"fileContent"`
}

// variableInputsToModel 将 VariableInput 切片转换为 appmodel.Variable 切片
func variableInputsToModel(vars []VariableInput) []appmodel.Variable {
	if len(vars) == 0 {
		return nil
	}
	result := make([]appmodel.Variable, 0, len(vars))
	for _, v := range vars {
		result = append(result, appmodel.Variable{
			Key:         v.Key,
			Value:       v.Value,
			Description: v.Description,
			IsSensitive: v.IsSensitive,
		})
	}
	return result
}
