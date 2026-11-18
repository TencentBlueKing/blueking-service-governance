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

// Package sanitizers 集中登记各业务模型写入操作记录前的脱敏函数。
//
// audit 包被这些模型包依赖，无法反向引用它们，因此脱敏函数放在本子包中，通过 audit.RegisterSanitizer
// 在 init 时登记。程序入口（cmd）以 blank import 引入本包；新增含敏感字段、且会写入操作记录的模型时，
// 在这里补充登记，并同步更新 scripts/mask-sensitive-operation-records.js 的规则。
package sanitizers

import (
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

func init() {
	// 环境变量：敏感变量的值
	audit.RegisterSanitizer(func(v appmodel.Variable) any { return maskVariable(v) })
	audit.RegisterSanitizer(maskScopedEnvVar)
	audit.RegisterSanitizer(maskAppModel)

	// 组件属性：按 key 后缀打码
	audit.RegisterSanitizer(func(c component.Component) any { return maskComponent(c) })
	audit.RegisterSanitizer(maskWorkspaceComponent)
	audit.RegisterSanitizer(maskAppComponentUpdateData)
	audit.RegisterSanitizer(maskWorkspaceComponentUpdateData)

	// 其它密钥字段
	audit.RegisterSanitizer(maskWorkspace)
	audit.RegisterSanitizer(maskApplication)
	audit.RegisterSanitizer(func(s bkmsapp.HelmSource) any { return maskHelmSource(s) })
	audit.RegisterSanitizer(maskBuildConfig)
	audit.RegisterSanitizer(maskPolarisConfig)
}

func maskVariable(v appmodel.Variable) appmodel.Variable {
	if v.IsSensitive {
		v.Value = audit.SensitiveValueMask
	}
	return v
}

func maskScopedEnvVar(v envvars.ScopedEnvVar) any {
	if v.IsSensitive {
		v.Value = audit.SensitiveValueMask
	}
	return v
}

// maskAppModel 打码环境变量与组件属性。切片新建，不能原地改：原对象还要用于接口响应。
func maskAppModel(m appmodel.AppModel) any {
	if m.Workload.EnvVars != nil {
		envVars := make([]appmodel.Variable, len(m.Workload.EnvVars))
		for i, v := range m.Workload.EnvVars {
			envVars[i] = maskVariable(v)
		}
		m.Workload.EnvVars = envVars
	}
	if m.Components != nil {
		comps := make([]*component.Component, len(m.Components))
		for i, c := range m.Components {
			if c != nil {
				masked := maskComponent(*c)
				comps[i] = &masked
			}
		}
		m.Components = comps
	}
	return m
}

func maskComponent(c component.Component) component.Component {
	c.Properties = audit.MaskSensitiveProperties(c.Properties)
	return c
}

func maskWorkspaceComponent(c workspace.Component) any {
	c.Properties = audit.MaskSensitiveProperties(c.Properties)
	return c
}

func maskAppComponentUpdateData(d appmodel.ComponentUpdateData) any {
	d.Properties = audit.MaskSensitiveProperties(d.Properties)
	return d
}

func maskWorkspaceComponentUpdateData(d workspace.ComponentUpdateData) any {
	d.Properties = audit.MaskSensitiveProperties(d.Properties)
	return d
}

func maskWorkspace(w workspace.Workspace) any {
	w.BkSystems.BscpToken = audit.MaskString(w.BkSystems.BscpToken)
	return w
}

func maskApplication(a bkmsapp.Application) any {
	if a.HelmSpec != nil && a.HelmSpec.HelmSource != nil {
		source := maskHelmSource(*a.HelmSpec.HelmSource)
		a.HelmSpec = &bkmsapp.HelmSpec{HelmSource: &source}
	}
	return a
}

func maskHelmSource(s bkmsapp.HelmSource) bkmsapp.HelmSource {
	if s.HelmRepoConfig != nil {
		cfg := *s.HelmRepoConfig
		cfg.Password = audit.MaskString(cfg.Password)
		s.HelmRepoConfig = &cfg
	}
	return s
}

func maskBuildConfig(c build.Config) any {
	if c.Image != nil {
		img := *c.Image
		img.Password = audit.MaskString(img.Password)
		c.Image = &img
	}
	return c
}

// maskPolarisConfig 打码 token，包括各环境已生效字段中的 token。
func maskPolarisConfig(c polaris.PolarisConfig) any {
	c.PolarisToken = audit.MaskString(c.PolarisToken)
	if c.EnvStates != nil {
		states := make(map[string]polaris.PolarisEnvState, len(c.EnvStates))
		for env, state := range c.EnvStates {
			if state.AppliedFields != nil {
				fields := *state.AppliedFields
				fields.PolarisToken = audit.MaskString(fields.PolarisToken)
				state.AppliedFields = &fields
			}
			states[env] = state
		}
		c.EnvStates = states
	}
	return c
}
