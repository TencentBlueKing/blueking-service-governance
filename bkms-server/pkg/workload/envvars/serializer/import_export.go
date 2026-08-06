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

// ImportEnvVarOutput is the JSON response for env var import APIs.
type ImportEnvVarOutput struct {
	// 导入结果汇总
	Data *EnvVarImportPreviewSummaryOutputObj `json:"data"`
}

// AppEnvVarsExportQueryInput is the query input for exporting app env vars.
type AppEnvVarsExportQueryInput struct {
	// 导出范围：appDefined（应用直接定义变量）或 effectiveByEnv（按环境导出最终生效变量）
	Scope string `form:"scope" binding:"required,oneof=appDefined effectiveByEnv"`
	// 环境名称；当 scope=effectiveByEnv 时必填
	EnvName string `form:"envName"`
}
