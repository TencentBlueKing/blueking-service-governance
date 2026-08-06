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

package polaris

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// PolarisEnvStateManager 管理 PolarisConfig 的环境部署快照和动态下发结果。
//
// 配置更新后调用 PrepareDynamicApply，清理无效记录并返回允许动态下发的环境；
// 动态下发完成后调用 RecordDynamicApplyResult，记录下发结果；
// 应用部署后调用 ReconcileAfterDeploy，记录本次部署生效的关键字段；
// 应用卸载后调用 ReconcileAfterUninstall，移除对应环境的记录。
type PolarisEnvStateManager struct {
	store PolarisConfigStore
}

// NewPolarisEnvStateManager 创建环境状态管理器。
func NewPolarisEnvStateManager(store PolarisConfigStore) *PolarisEnvStateManager {
	return &PolarisEnvStateManager{store: store}
}

// PrepareDynamicApply 清理不再需要的环境状态，并返回可以动态下发的环境名称。
func (m *PolarisEnvStateManager) PrepareDynamicApply(
	ctx context.Context,
	config *PolarisConfig,
) ([]string, error) {
	if err := m.removeUnappliedEnvStatesOutsideScope(ctx, config); err != nil {
		return nil, errors.Wrap(err, "remove unapplied env states outside scope")
	}
	return m.selectEnvNamesForDynamicApply(config), nil
}

// removeUnappliedEnvStatesOutsideScope 移除不在 scope 中且从未部署过的环境记录。
func (m *PolarisEnvStateManager) removeUnappliedEnvStatesOutsideScope(
	ctx context.Context,
	config *PolarisConfig,
) error {
	orphanEnvNames := make([]string, 0)
	for envName, state := range config.EnvStates {
		if lo.Contains(config.ScopeEnvNames, envName) || state.AppliedFields != nil {
			continue
		}
		orphanEnvNames = append(orphanEnvNames, envName)
	}

	if err := m.store.RemoveEnvStates(ctx, config.AppID, config.Name, orphanEnvNames); err != nil {
		return err
	}
	for _, envName := range orphanEnvNames {
		delete(config.EnvStates, envName)
	}
	return nil
}

// selectEnvNamesForDynamicApply 返回当前 scope 中部署快照与配置关键字段一致的环境。
func (m *PolarisEnvStateManager) selectEnvNamesForDynamicApply(config *PolarisConfig) []string {
	desiredFields := NewRedeployRequiredFields(config)
	envNames := make([]string, 0, len(config.ScopeEnvNames))
	for _, envName := range config.ScopeEnvNames {
		state := config.GetEnvState(envName)
		if state.AppliedFields == nil || *state.AppliedFields != *desiredFields {
			continue
		}
		envNames = append(envNames, envName)
	}
	return envNames
}

// RecordDynamicApplyResult 记录一次动态下发结果；成功时清空 LastError。
func (m *PolarisEnvStateManager) RecordDynamicApplyResult(
	ctx context.Context,
	appID, configName, envName string,
	applyErr error,
) error {
	lastError := ""
	if applyErr != nil {
		lastError = applyErr.Error()
	}
	if err := m.store.UpsertEnvState(
		ctx,
		appID,
		configName,
		envName,
		PolarisEnvStateUpdate{LastError: &lastError},
	); err != nil {
		return errors.Wrapf(err, "record dynamic apply result for env %s", envName)
	}
	return nil
}

// ReconcileAfterDeploy 记录应用部署已生效的关键字段，并清理已离开 scope 的环境记录。
func (m *PolarisEnvStateManager) ReconcileAfterDeploy(
	ctx context.Context,
	app *bkmsapp.Application,
	env *bkmsenv.Environment,
) error {
	configs, err := m.store.ListByApp(ctx, app.ID)
	if err != nil {
		return errors.Wrap(err, "list polaris configs after deploy")
	}

	for _, config := range configs {
		if err = m.reconcileEnvStateAfterDeploy(ctx, app.ID, env.Name, config); err != nil {
			return err
		}
	}
	return nil
}

// reconcileEnvStateAfterDeploy 更新单条配置的关键字段快照，或移除其离域环境记录。
func (m *PolarisEnvStateManager) reconcileEnvStateAfterDeploy(
	ctx context.Context,
	appID, envName string,
	config *PolarisConfig,
) error {
	if !lo.Contains(config.ScopeEnvNames, envName) {
		if err := m.store.RemoveEnvStates(ctx, appID, config.Name, []string{envName}); err != nil {
			return errors.Wrapf(err, "remove out-of-scope env state for config %s", config.Name)
		}
		return nil
	}

	if err := m.store.UpsertEnvState(
		ctx,
		appID,
		config.Name,
		envName,
		PolarisEnvStateUpdate{
			AppliedFields: NewRedeployRequiredFields(config),
			LastError:     lo.ToPtr(""),
		},
	); err != nil {
		return errors.Wrapf(err, "record applied fields for config %s", config.Name)
	}
	return nil
}

// ReconcileAfterUninstall 清理当前卸载环境的全部 PolarisConfig 状态。
func (m *PolarisEnvStateManager) ReconcileAfterUninstall(
	ctx context.Context,
	app *bkmsapp.Application,
	envName string,
) error {
	configs, err := m.store.ListByApp(ctx, app.ID)
	if err != nil {
		return errors.Wrap(err, "list polaris configs after uninstall")
	}
	for _, config := range configs {
		if err = m.store.RemoveEnvStates(ctx, app.ID, config.Name, []string{envName}); err != nil {
			return errors.Wrapf(err, "remove env state for config %s after uninstall", config.Name)
		}
	}
	return nil
}
