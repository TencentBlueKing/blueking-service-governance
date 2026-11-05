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

package cfgfile

import (
	"context"
	"slices"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// UpdateOptions 更新配置文件定义的参数，nil / 零值字段表示不修改。
type UpdateOptions struct {
	// NewName 新文件名。
	NewName *string
	// NewMountDir 新挂载目录。
	NewMountDir *string
	// MountEnvNames 新的挂载环境列表，与 MountAllEnvs 互斥。
	MountEnvNames []string
	// MountAllEnvs 挂载到应用当前的所有环境。
	MountAllEnvs bool
	// IsUnifiedConfig 切换统一配置 / 按环境独立配置。
	IsUnifiedConfig *bool
	// EnableEnvVarRender 是否启用环境变量渲染。
	EnableEnvVarRender *bool
	// EnvContentStrategy 环境内容策略（仅 framework）：overlay 或 overwrite。
	EnvContentStrategy *string
}

// UpdatePlan 是一次更新的执行计划，执行前可据此向用户展示将被删除的数据。
type UpdatePlan struct {
	// Def 待更新的配置文件定义。
	Def *client.AppConfigFileDef
	// Body 将提交给后端的更新内容。
	Body client.UpdateAppConfigFileDefBody
	// RemovedEnvInstances 执行后将被删除的环境独立实例（含其版本历史）。
	RemovedEnvInstances []string
}

// HasChanges 表示计划中是否存在需要提交的修改。
func (p *UpdatePlan) HasChanges() bool {
	return p.Body != client.UpdateAppConfigFileDefBody{}
}

// PlanUpdate 生成更新计划，并计算切回统一配置、缩小挂载范围时会被后端级联删除的环境实例。
func PlanUpdate(
	ctx context.Context,
	cli client.Client,
	appID string,
	ref FileRef,
	opts UpdateOptions,
) (*UpdatePlan, error) {
	def, err := ResolveDef(ctx, cli, appID, ref)
	if err != nil {
		return nil, err
	}

	plan := &UpdatePlan{
		Def: def,
		Body: client.UpdateAppConfigFileDefBody{
			Name:               opts.NewName,
			MountDir:           opts.NewMountDir,
			EnableEnvVarRender: opts.EnableEnvVarRender,
			EnvContentStrategy: opts.EnvContentStrategy,
		},
	}
	if opts.IsUnifiedConfig != nil && *opts.IsUnifiedConfig != def.IsUnifiedConfig {
		plan.Body.IsUnifiedConfig = opts.IsUnifiedConfig
	}

	envNames, err := resolveUpdateMountEnvNames(ctx, cli, appID, def, opts)
	if err != nil {
		return nil, err
	}
	if envNames != nil {
		plan.Body.MountedEnvNames = &envNames
	}

	if plan.Body.IsUnifiedConfig != nil || plan.Body.MountedEnvNames != nil {
		instances, err := listEnvInstanceNames(ctx, cli, appID, def.ID)
		if err != nil {
			return nil, err
		}
		plan.RemovedEnvInstances = removedEnvInstances(instances, plan.Body)
	}
	return plan, nil
}

// ApplyUpdate 执行更新计划。
func ApplyUpdate(
	ctx context.Context,
	cli client.Client,
	appID string,
	plan *UpdatePlan,
) (*client.AppConfigFileDef, error) {
	def, err := cli.UpdateAppConfigFileDef(ctx, appID, plan.Def.ID, plan.Body)
	if err != nil {
		return nil, errors.Wrap(err, "update app config file")
	}
	return def, nil
}

// resolveUpdateMountEnvNames 计算要提交的挂载环境列表，返回 nil 表示不修改。
//
// 后端无法把挂载范围改回「所有环境」（nil），--mount-all-envs 只能提交应用当前所有环境的快照，
// 之后新增的环境不会自动挂载。文件原本已挂载到所有环境时不提交，保留原语义。
func resolveUpdateMountEnvNames(
	ctx context.Context,
	cli client.Client,
	appID string,
	def *client.AppConfigFileDef,
	opts UpdateOptions,
) ([]string, error) {
	switch {
	case opts.MountAllEnvs && len(def.MountedEnvNames) > 0:
		envNames, err := listAppEnvNames(ctx, cli, appID)
		if err != nil {
			return nil, err
		}
		if envNames == nil {
			envNames = []string{}
		}
		return envNames, nil
	case opts.MountEnvNames != nil:
		return resolveMountEnvNames(ctx, cli, appID, opts.MountEnvNames)
	default:
		return nil, nil
	}
}

// removedEnvInstances 与后端行为保持一致：切回统一配置删除所有环境实例；
// 修改挂载范围删除不在新范围内的环境实例。
func removedEnvInstances(instances []string, body client.UpdateAppConfigFileDefBody) []string {
	if lo.FromPtr(body.IsUnifiedConfig) {
		return instances
	}
	if body.MountedEnvNames == nil {
		return nil
	}
	return lo.Filter(instances, func(envName string, _ int) bool {
		return !slices.Contains(*body.MountedEnvNames, envName)
	})
}
