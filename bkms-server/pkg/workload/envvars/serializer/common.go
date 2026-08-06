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
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// AppEnvURIInput is the path input for APIs scoped by app and environment name.
type AppEnvURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
	// 环境名称
	EnvName string `uri:"envName" binding:"required,uri_slug"`
}

// EnvVarOutputObj is the JSON representation of an effective env var.
type EnvVarOutputObj struct {
	// 环境变量 Key
	Key string `json:"key"`
	// 环境变量值
	Value string `json:"value"`
	// 环境变量描述
	Description string `json:"description"`
	// 是否是内置变量
	IsBuiltin bool `json:"isBuiltin"`
	// 是否是敏感变量
	IsSensitive bool `json:"isSensitive"`
}

// FromModel fills output fields from an effective env var model.
func (o *EnvVarOutputObj) FromModel(item envvartypes.EnvVariableObj) *EnvVarOutputObj {
	*o = EnvVarOutputObj{
		Key:         item.Key,
		Value:       item.ValueForDisplay(),
		Description: item.Description,
		IsBuiltin:   item.IsBuiltin,
		IsSensitive: item.IsSensitive,
	}
	return o
}
