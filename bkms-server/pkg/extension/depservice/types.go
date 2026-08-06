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

package depservice

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/types"
)

// CreateServiceInstanceParams 创建服务实例的参数
type CreateServiceInstanceParams struct {
	// Name 服务实例名称
	Name string `validate:"required"`

	// ServiceName 依赖服务注册的服务名, 对应 model.Service.Name. 如北极星的服务名是 Polaris
	ServiceName string `validate:"required"`
	// PlanName 服务方案名. 一个 Service 可能有多个生成资源实例的方案, 如自注册, 或者第三方接口提供. 具体对应 model.Service.Plans[].Name
	PlanName string `validate:"required"`

	ScopeType   model.ScopeType `validate:"required"`
	WorkspaceID string          `validate:"required"`
	// ScopeValue 作用域值:
	// - 当 ScopeType 为 workspace 时，固定为空字符串
	// - 当 ScopeType 为 envType 时，可选值为 development、test、staging 或 production
	// - 当 ScopeType 为 env 时，值为具体的环境名称
	ScopeValue  string
	Description string

	// AttachedApps 记录当前实例分配给哪些应用
	AttachedApps []string

	// CustomEnvVars 用户自定义衍生环境变量模板, 详见 model.ServiceInstance.CustomEnvVars
	CustomEnvVars map[string]string

	Operator string `validate:"required"`

	// Params 是创建服务实例所需的业务参数，需实现 types.ProvisionParams 接口。
	// 其具体类型由对应的 ServiceProvider 实现决定：
	//   - Polaris: 传入 *polaris.CreateParams
	//   - 其他 Provider: 传入对应的强类型参数
	Params types.ProvisionParams `validate:"required"`
}
