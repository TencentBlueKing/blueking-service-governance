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

import envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"

// ListAppEnvVarsOutput is the JSON response for listing app env vars.
type ListAppEnvVarsOutput struct {
	// 应用部署到某个环境后可用的环境变量列表
	Data []*EnvVarOutputObj `json:"data"`
}

// BgEnvVarOutputObj is the JSON representation of a background env var.
type BgEnvVarOutputObj struct {
	// 环境变量 Key
	Key string `json:"key"`
	// 环境变量值
	Value string `json:"value"`
	// 描述
	Description string `json:"description"`
	// 来源，如 builtin、scopedWorkspace、scopedEnvType、scopedEnv、app
	Source string `json:"source"`
}

// FromModel fills output fields from a background env var model.
func (o *BgEnvVarOutputObj) FromModel(item envvartypes.EnvVariableRichItem) *BgEnvVarOutputObj {
	*o = BgEnvVarOutputObj{
		Key:         item.Obj.Key,
		Value:       item.Obj.ValueForDisplay(),
		Description: item.Obj.Description,
		Source:      string(item.Source.Source),
	}
	return o
}

// ListEnvBgEnvVarsOutput is the JSON response for listing environment background env vars.
type ListEnvBgEnvVarsOutput struct {
	// 指定环境的背景环境变量列表
	Data []*BgEnvVarOutputObj `json:"data"`
}

// ListAppBgEnvVarsOutput is the JSON response for listing app background env vars.
type ListAppBgEnvVarsOutput struct {
	// 应用在某个环境下的背景环境变量列表
	Data []*BgEnvVarOutputObj `json:"data"`
}
