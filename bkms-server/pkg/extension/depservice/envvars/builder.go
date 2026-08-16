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

// Package envvars 提供依赖服务绑定的环境变量产出能力：
// 用实例 Credentials 渲染绑定上的 EnvVars 模板，供上游 envvars 链路接入。
//
// 该包仅依赖 envvars/types 中的基础类型，不依赖 envvars 主包，从而避免
// envvars 主包与 depservice 之间形成循环依赖。
package envvars

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/render"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// BuildBindingEnvVars 用实例 Credentials 作为上下文，渲染绑定上的 EnvVars 模板。
// EnvVars 为空时不注入任何变量（凭证不再自动直出）。
func BuildBindingEnvVars(
	ctx context.Context,
	envVars map[string]string,
	credentials map[string]any,
) (envvartypes.EnvVariableList, error) {
	if len(envVars) == 0 {
		return envvartypes.EnvVariableList{}, nil
	}

	renderCtx := make(map[string]string, len(credentials))
	for k, v := range credentials {
		renderCtx[k] = fmt.Sprintf("%v", v)
	}

	result := make(envvartypes.EnvVariableList, 0, len(envVars))
	for k, tmpl := range envVars {
		rendered, rerr := render.New(render.SetEnvContext(renderCtx)).Render(tmpl)
		if rerr != nil {
			log.Errorf(ctx, "render binding env var failed, key=%s, template=%q: %s", k, tmpl, rerr)
			return nil, errors.Wrapf(rerr, "render binding env var %s", k)
		}
		result = append(result, envvartypes.EnvVariableObj{
			Key:   k,
			Value: rendered,
		})
	}
	return result, nil
}
