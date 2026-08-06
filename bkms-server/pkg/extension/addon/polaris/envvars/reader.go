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

// Package envvars 提供 PolarisConfig 产出的环境变量读取能力，产出的富列表
// (envvartypes.EnvVariableRichList) 会被注入到 envvars.UnifiedEnvVarsReader 中以
// 接入统一的 envvars 链路。该包与 depservice/envvars 同构。
package envvars

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// Reader 基于 PolarisConfig 列出某个应用在某个环境下生效的环境变量。
//
// 它产出的富列表会被注入到 envvars.UnifiedEnvVarsReader 中以接入统一的 envvars 链路。
type Reader struct {
	polarisConfigStore polaris.PolarisConfigStore
}

// NewReader 创建一个 PolarisConfig 的环境变量读取器。
func NewReader(polarisConfigStore polaris.PolarisConfigStore) *Reader {
	return &Reader{polarisConfigStore: polarisConfigStore}
}

// ListEnvVarsForApp 返回 attach 到给定 app 且对该 environment 生效的全部 polaris 配置所产出的环境变量。
//
// 过滤规则由 store 层执行 (ListByEnv 内部基于 IsAvailableInEnv 过滤)。
// 每条配置产出两个变量：{instanceKey}_polarisToken (敏感) 与 {instanceKey}_serviceport。
func (r *Reader) ListEnvVarsForApp(
	ctx context.Context,
	environment envmodel.Environment,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	if app == nil {
		return envvartypes.EnvVariableRichList{}, nil
	}

	configs, err := r.polarisConfigStore.ListByEnv(ctx, app.ID, environment.Name)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list polaris configs for app")
	}

	return r.collect(configs), nil
}

// ListAppVarsForConflicts 返回 attach 到该 app 的全部 polaris 配置所产出的环境变量,
// 用于 app 级别冲突检测 (不限定具体 environment)。
func (r *Reader) ListAppVarsForConflicts(
	ctx context.Context,
	workspaceID string,
	app *bkmsapp.Application,
) (envvartypes.EnvVariableRichList, error) {
	_ = workspaceID // 保留与 depservice reader 一致的签名；polaris 配置按 app 聚合即可
	if app == nil {
		return envvartypes.EnvVariableRichList{}, nil
	}

	configs, err := r.polarisConfigStore.ListByApp(ctx, app.ID)
	if err != nil {
		return envvartypes.EnvVariableRichList{}, errors.Wrap(err, "list polaris configs for app in workspace")
	}

	return r.collect(configs), nil
}

// collect 对 configs 中的每条配置产出环境变量，并以 EnvVarSourcePolaris 包装 source 后扁平拼接。
func (r *Reader) collect(configs []*polaris.PolarisConfig) envvartypes.EnvVariableRichList {
	result := envvartypes.EnvVariableRichList{Vars: make([]envvartypes.EnvVariableRichItem, 0, len(configs)*2)}

	for _, cfg := range configs {
		source := envvartypes.ConflictedSource{
			Source:      envvartypes.EnvVarSourcePolaris,
			SourceValue: cfg.Name,
		}
		for _, v := range buildConfigEnvVars(cfg) {
			result.Vars = append(result.Vars, envvartypes.EnvVariableRichItem{
				Obj:    v,
				Source: source,
			})
		}
	}

	return result
}

// buildConfigEnvVars 为单个 PolarisConfig 产出环境变量列表：
//   - {instanceKey}_polarisToken (敏感)
//   - {instanceKey}_serviceport
//
// 与 PolarisConfig.GetVars 保持一致的 key/value 形态。
func buildConfigEnvVars(cfg *polaris.PolarisConfig) envvartypes.EnvVariableList {
	return envvartypes.EnvVariableList{
		{
			Key:         cfg.InstanceKey + "_polarisToken",
			Value:       cfg.PolarisToken,
			IsSensitive: true,
		},
		{
			// TODO: 此处大小写符合项目对于环境变量的规范
			Key:   cfg.InstanceKey + "_serviceport",
			Value: strconv.Itoa(int(cfg.ServicePort)),
		},
	}
}
