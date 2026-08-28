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
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// PolarisEnvStateManager 管理 PolarisConfig 的环境部署快照、权重生命周期和动态下发结果。
//
// 配置更新后调用 PrepareDynamicApply，清理无效记录并返回允许动态下发的环境；
// 动态下发完成后调用 RecordDynamicApplyResult，按配置版本记录下发结果，避免旧任务覆盖新配置状态；
// 应用部署后调用 ReconcileAfterDeploy，记录本次部署生效的关键字段并清理离域环境级设置；
// 应用卸载后调用 ReconcileAfterUninstall，移除对应环境的记录与离域环境级设置（条件见函数说明）。
type PolarisEnvStateManager struct {
	store PolarisConfigStore
}

// NewPolarisEnvStateManager 创建环境状态管理器。
func NewPolarisEnvStateManager(store PolarisConfigStore) *PolarisEnvStateManager {
	return &PolarisEnvStateManager{store: store}
}

// 环境部署状态
const (
	// PolarisEnvStatusDeployed 表示环境在 scope 内，且部署关联字段均与部署快照一致。
	PolarisEnvStatusDeployed = "deployed"
	// PolarisEnvStatusPendingCreate 表示环境在 scope 内，但尚无部署快照。
	PolarisEnvStatusPendingCreate = "pendingCreate"
	// PolarisEnvStatusPendingModify 表示环境在 scope 内，且至少一个部署关联字段与部署快照不同。
	PolarisEnvStatusPendingModify = "pendingModify"
	// PolarisEnvStatusPendingDelete 表示环境已移出 scope，但仍保留部署快照等待下次部署删除。
	PolarisEnvStatusPendingDelete = "pendingDelete"
)

// PolarisEnvStatus 计算指定环境的部署状态。
func PolarisEnvStatus(config *PolarisConfig, envName string, state PolarisEnvState) string {
	if !config.IsAvailableInEnv(envName) {
		return PolarisEnvStatusPendingDelete
	}
	if !state.IsDeployed() {
		return PolarisEnvStatusPendingCreate
	}
	if state.AppliedFields.InstanceKey != config.InstanceKey ||
		state.AppliedFields.PolarisToken != config.PolarisToken ||
		state.AppliedFields.ServicePort != config.ServicePort {
		return PolarisEnvStatusPendingModify
	}
	return PolarisEnvStatusDeployed
}

// PolarisTokenChanged 比较配置期望 Token 与该环境最近一次部署快照中的 Token 是否不同。
func PolarisTokenChanged(config *PolarisConfig, envName string, state PolarisEnvState) bool {
	return config.IsAvailableInEnv(envName) &&
		state.IsDeployed() && state.AppliedFields.PolarisToken != config.PolarisToken
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
		if lo.Contains(config.ScopeEnvNames, envName) || state.IsDeployed() {
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

// reconcileEnvSettingsForScope 根据 scope 变化调整环境权重与动态权重开关，与 EnvState 生命周期对齐：
// - 保留仍应存在的环境条目，丢弃未部署且离域的条目；
// - 为 scope 内缺失的环境补充默认权重。
//
// 动态权重开关不补默认值：缺省即关闭，新环境自动按关闭处理比预建 false 更安全。
func (*PolarisEnvStateManager) reconcileEnvSettingsForScope(
	scopeEnvNames []string,
	envWeights map[string]int32,
	envDynamicWeights map[string]bool,
	envStates map[string]PolarisEnvState,
	registerMode string,
) (map[string]int32, map[string]bool) {
	weights := make(map[string]int32, len(scopeEnvNames)+len(envWeights))
	for envName, w := range envWeights {
		if keepEnvSettingForScope(envName, scopeEnvNames, envStates, registerMode) {
			weights[envName] = w
		}
	}
	for _, envName := range scopeEnvNames {
		if _, ok := weights[envName]; !ok {
			weights[envName] = DefaultEnvWeight
		}
	}

	dynamicWeights := make(map[string]bool, len(envDynamicWeights))
	for envName, enabled := range envDynamicWeights {
		if keepEnvSettingForScope(envName, scopeEnvNames, envStates, registerMode) {
			dynamicWeights[envName] = enabled
		}
	}
	return weights, dynamicWeights
}

// keepEnvSettingForScope 判断某环境的环境级设置在 scope 变化后是否保留。
//
// on_deploy（含空值）会暂时保留已部署且离开 scope 的设置，直到下次部署/卸载清理。
// immediate 离域时会同步删除集群资源，没有"等下次部署清理"的阶段，因此不保留离域设置。
func keepEnvSettingForScope(
	envName string,
	scopeEnvNames []string,
	envStates map[string]PolarisEnvState,
	registerMode string,
) bool {
	if lo.Contains(scopeEnvNames, envName) {
		return true
	}
	return registerMode != RegisterModeImmediate && envStates[envName].IsDeployed()
}

// selectEnvNamesForDynamicApply 返回 scope 内满足动态下发条件的环境名称。
func (m *PolarisEnvStateManager) selectEnvNamesForDynamicApply(config *PolarisConfig) []string {
	envNames := make([]string, 0, len(config.ScopeEnvNames))
	for _, envName := range config.ScopeEnvNames {
		if !m.IsEnvReadyForDynamicApply(config, envName) {
			continue
		}
		envNames = append(envNames, envName)
	}
	return envNames
}

// IsEnvReadyForDynamicApply 判断指定环境是否允许动态下发。
// 动态下发 PolarisConfig CR，仅当三个关键字段的快照与当前配置的一致时，可以更新非关键字段。
// 这三个字段会影响到环境变量的生成和端口等服务配置，若不一致则必须重新部署实例。
func (*PolarisEnvStateManager) IsEnvReadyForDynamicApply(config *PolarisConfig, envName string) bool {
	state := config.GetEnvState(envName)
	desiredFields := NewRedeployRequiredFields(config)
	return state.IsDeployed() && *state.AppliedFields == *desiredFields
}

// RecordImmediateApplyResult 记录一次即时下发结果。
//
// 与 RecordDynamicApplyResult 的区别在于成功时会一并刷新部署快照：immediate 配置不参与
// Workload 渲染，Pod 侧没有任何待生效内容，CR 与 Service 下发成功就代表配置已完全生效。
func (m *PolarisEnvStateManager) RecordImmediateApplyResult(
	ctx context.Context,
	config *PolarisConfig,
	envName string,
	applyErr error,
) error {
	update := PolarisEnvStateUpdate{LastError: lo.ToPtr("")}
	if applyErr != nil {
		update.LastError = lo.ToPtr(applyErr.Error())
	} else {
		update.AppliedFields = NewRedeployRequiredFields(config)
	}
	if err := m.store.UpsertEnvState(ctx, config.AppID, config.Name, envName, update); err != nil {
		return errors.Wrapf(err, "record immediate apply result for env %s", envName)
	}
	return nil
}

// ReleaseEnv 在 immediate 配置的集群资源删除成功后，移除该环境的记录与环境级设置。
func (m *PolarisEnvStateManager) ReleaseEnv(
	ctx context.Context,
	appID, configName, envName string,
) error {
	envNames := []string{envName}
	if err := m.store.RemoveEnvStates(ctx, appID, configName, envNames); err != nil {
		return errors.Wrapf(err, "remove env state for env %s after release", envName)
	}
	if err := m.store.RemoveEnvSettings(ctx, appID, configName, envNames); err != nil {
		return errors.Wrapf(err, "remove env settings for env %s after release", envName)
	}
	return nil
}

// RecordDynamicApplyResult 仅在配置顶层 UpdatedAt 仍为 expectedUpdatedAt 时记录结果。
// 配置版本已变化时视为旧任务，跳过写入且不返回错误。
func (m *PolarisEnvStateManager) RecordDynamicApplyResult(
	ctx context.Context,
	appID, configName, envName string,
	expectedUpdatedAt time.Time,
	applyErr error,
) error {
	lastError := ""
	if applyErr != nil {
		lastError = applyErr.Error()
	}
	if _, err := m.store.UpsertEnvStateIfUpdatedAtMatch(
		ctx,
		appID,
		configName,
		envName,
		expectedUpdatedAt,
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

// reconcileEnvStateAfterDeploy 更新单条配置的关键字段快照，或移除其离域环境记录与保留权重。
func (m *PolarisEnvStateManager) reconcileEnvStateAfterDeploy(
	ctx context.Context,
	appID, envName string,
	config *PolarisConfig,
) error {
	if !lo.Contains(config.ScopeEnvNames, envName) {
		if err := m.store.RemoveEnvStates(ctx, appID, config.Name, []string{envName}); err != nil {
			return errors.Wrapf(err, "remove out-of-scope env state for config %s", config.Name)
		}
		if err := m.store.RemoveEnvSettings(ctx, appID, config.Name, []string{envName}); err != nil {
			return errors.Wrapf(err, "remove out-of-scope env settings for config %s", config.Name)
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
		// 若环境仍在 scope 内时保留权重与动态权重开关，供下次部署使用
		if lo.Contains(config.ScopeEnvNames, envName) {
			continue
		}
		// 若环境已不在 scope 内，同步清理此前保留的环境级设置
		if err = m.store.RemoveEnvSettings(ctx, app.ID, config.Name, []string{envName}); err != nil {
			return errors.Wrapf(err, "remove env settings for config %s after uninstall", config.Name)
		}
	}
	return nil
}
