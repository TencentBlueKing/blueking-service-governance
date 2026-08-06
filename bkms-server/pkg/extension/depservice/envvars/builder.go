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

// Package envvars 提供基于 ServiceInstance 的环境变量产出能力：
//   - 将 Credentials 中的键值对作为环境变量输出（IsSensitive=true）
//   - 以 Credentials 为上下文渲染用户自定义的 CustomEnvVars 模板
//   - 输出 envvartypes.EnvVariableObj 列表，供上游 envvars 链路接入
//
// 该包仅依赖 envvars/types 中的基础类型，不依赖 envvars 主包，从而避免
// envvars 主包与 depservice 之间形成循环依赖。
package envvars

import (
	"context"
	"fmt"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// skipCredentialsServices 中的 service 不从 Credentials 生成环境变量。
// 这些服务通过其他链路（如 component）注入变量，避免重复输出。
var skipCredentialsServices = map[string]struct{}{
	"polaris": {},
}

// BuildInstanceEnvVars 为单个 ServiceInstance 产出环境变量列表：
//  1. 若该实例的 ServiceName 不在跳过名单中，将 Credentials 中的键值对作为内置变量输出（IsSensitive=true）。
//  2. 把内置变量作为唯一上下文，渲染 inst.CustomEnvVars 中的 ${{env.KEY}} 模板。
//  3. 内置变量与衍生变量一并输出。
func BuildInstanceEnvVars(
	ctx context.Context,
	inst *model.ServiceInstance,
) (envvartypes.EnvVariableList, error) {
	result := make(envvartypes.EnvVariableList, 0, len(inst.Credentials)+len(inst.CustomEnvVars))

	// 1. 输出 Credentials 作为内置变量，并构建渲染上下文
	renderCtx := make(map[string]string, len(inst.Credentials))
	_, skip := skipCredentialsServices[inst.ServiceName]
	if !skip {
		for k, v := range inst.Credentials {
			val := fmt.Sprintf("%v", v)
			renderCtx[k] = val
			result = append(result, envvartypes.EnvVariableObj{
				Key:         k,
				Value:       val,
				IsSensitive: true,
			})
		}
	}

	// 2. 渲染 CustomEnvVars 模板
	for k, tmpl := range inst.CustomEnvVars {
		rendered, rerr := render.New(render.SetEnvContext(renderCtx)).Render(tmpl)
		if rerr != nil {
			// 不打印实际 value，避免暴露敏感数据
			log.Errorf(
				ctx, "render custom env var of dep service instance %s failed, key=%s, template=%q: %s",
				inst.Name, k, tmpl, rerr,
			)
			return nil, rerr
		}
		// NOTE: 这里没有继承 IsSensitive 标记
		result = append(result, envvartypes.EnvVariableObj{
			Key:   k,
			Value: rendered,
		})
	}

	return result, nil
}
