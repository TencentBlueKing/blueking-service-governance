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
	trpcapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/trpc"
)

// TrpcSpecInput is the tRPC spec input.
type TrpcSpecInput struct {
	// 编程语言
	Language string `json:"language" binding:"required,oneof=go cpp"`
	// 配置文件名
	FileName string `json:"fileName" binding:"required"`
	// 配置文件路径
	FilePath string `json:"filePath" binding:"required"`
	// 配置文件内容
	FileContent string `json:"fileContent"`
}

// TrpcSpecOutputObj is the tRPC spec output.
type TrpcSpecOutputObj struct {
	// 编程语言
	Language string `json:"language"`
	// 配置文件名
	FileName string `json:"fileName"`
	// 配置文件路径
	FilePath string `json:"filePath"`
	// 配置文件内容
	FileContent string `json:"fileContent"`
}

// ToTrpcCreateParams 将 AppModelSpecInput 转换为 trpc 内部创建参数类型
func (input *AppModelSpecInput) ToTrpcCreateParams() *trpcapp.CreateParams {
	params := &trpcapp.CreateParams{
		Command: input.Command,
		Args:    input.Args,
		EnvVars: variableInputsToModel(input.EnvVars),
	}
	if input.TrpcSpec != nil {
		params.TrpcConfig = &trpcapp.TrpcConfigParams{
			Language:    input.TrpcSpec.Language,
			FileName:    input.TrpcSpec.FileName,
			FilePath:    input.TrpcSpec.FilePath,
			FileContent: input.TrpcSpec.FileContent,
		}
	}
	return params
}

// ToTrpcUpdateParams 将 AppModelSpecInput 转换为 trpc 内部更新参数类型
func (input *AppModelSpecInput) ToTrpcUpdateParams() *trpcapp.UpdateParams {
	// 更新 Spec 时忽略 EnvVars 以兼容旧客户端，应用环境变量应通过独立 CRUD 接口修改。
	params := &trpcapp.UpdateParams{
		Command: input.Command,
		Args:    input.Args,
	}
	if input.TrpcSpec != nil {
		params.TrpcConfig = &trpcapp.TrpcConfigParams{
			Language: input.TrpcSpec.Language,
			FileName: input.TrpcSpec.FileName,
			FilePath: input.TrpcSpec.FilePath,
		}
	}
	return params
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
