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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
)

// ResetPlan 是恢复环境为默认配置前收集的信息。
type ResetPlan struct {
	// Def 目标配置文件定义。
	Def *client.AppConfigFileDef
	// EnvName 目标环境。
	EnvName string
	// HasEnvInstance 环境是否存在独立实例；不存在时无需恢复。
	HasEnvInstance bool
}

// PlanReset 定位配置文件，并确认目标环境存在可删除的独立实例。
func PlanReset(ctx context.Context, cli client.Client, appID string, ref FileRef, envName string) (*ResetPlan, error) {
	def, err := ResolveDef(ctx, cli, appID, ref)
	if err != nil {
		return nil, err
	}
	if err = ensureEnvConfigurable(ctx, cli, appID, envName); err != nil {
		return nil, err
	}
	if def.IsUnifiedConfig {
		return nil, errors.Errorf(
			"config file %s uses unified config, env %s already shares the default",
			def.Name,
			envName,
		)
	}
	instances, err := listEnvInstanceNames(ctx, cli, appID, def.ID)
	if err != nil {
		return nil, err
	}
	return &ResetPlan{Def: def, EnvName: envName, HasEnvInstance: slices.Contains(instances, envName)}, nil
}

// ApplyReset 删除环境独立实例及其版本历史，使该环境回到默认配置。
func ApplyReset(ctx context.Context, cli client.Client, appID string, plan *ResetPlan) error {
	if err := cli.ResetAppConfigFileDefEnv(ctx, appID, plan.Def.ID, plan.EnvName); err != nil {
		return errors.Wrap(err, "reset app config file env")
	}
	return nil
}
