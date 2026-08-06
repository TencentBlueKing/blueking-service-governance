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

package env

import "github.com/samber/lo"

// 内置的环境变量名
const (
	// EnvVarNameApmGRPCAPI 环境对应的 APM gRPC 采集地址
	EnvVarNameApmGRPCAPI = "BKMS_BKAPM_API"
	// EnvVarNameApmHTTPAPI 环境对应的 APM HTTP 采集地址
	EnvVarNameApmHTTPAPI = "BKMS_BKAPM_HTTP_API"
	// EnvVarNameApmToken 环境对应的 APM 采集 token
	EnvVarNameApmToken = "BKMS_BKAPM_TOKEN" //nolint:gosec

	// EnvVarNameEnvType 环境类型
	EnvVarNameEnvType = "BKMS_ENV_TYPE"
	// EnvVarNameEnvName 环境名称
	EnvVarNameEnvName = "BKMS_ENV_NAME"
)

// Type 表示环境的分类，如开发、测试、预发布、生产。
// 此处的环境类型是最主要的定义，会被其他模块复用。
type Type string

const (
	// TypeTest 表示测试类环境
	TypeTest Type = "test"

	// TypeDevelopment 表示开发类环境
	TypeDevelopment Type = "development"

	// TypeStaging 表示预发布环境
	TypeStaging Type = "staging"

	// TypeProduction 表示生产类环境
	TypeProduction Type = "production"
)

var publicTypeOrder = []Type{
	TypeDevelopment,
	TypeTest,
	TypeStaging,
	TypeProduction,
}

// String 返回环境类型的字符串表示。
func (t Type) String() string {
	return string(t)
}

// PublicTypeOrder 返回环境类型的公开默认顺序。
func PublicTypeOrder() []Type {
	return append([]Type(nil), publicTypeOrder...)
}

// IsProductionType 判断环境类型是否为生产环境。
func IsProductionType(envType Type) bool {
	return envType == TypeProduction
}

// IsValidEnvType 检查给定的环境类型字符串是否是有效的 EnvType。
func IsValidEnvType(envType string) bool {
	return lo.Contains(publicTypeOrder, Type(envType))
}
