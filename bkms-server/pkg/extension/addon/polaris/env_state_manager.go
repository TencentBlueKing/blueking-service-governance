package polaris

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

// PolarisEnvStateManager 管理 PolarisConfig 的环境部署快照、权重生命周期和动态下发结果。
//
// 配置更新后调用 PrepareDynamicApply，清理无效记录并返回允许动态下发的环境；
// 动态下发完成后调用 RecordDynamicApplyResult，记录下发结果；
// 应用部署后调用 ReconcileAfterDeploy，记录本次部署生效的关键字段并清理离域权重；
// 应用卸载后调用 ReconcileAfterUninstall，移除对应环境的记录与离域权重（条件见函数说明）。
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

// reconcileEnvWeightsForScope 根据 scope 变化调整环境权重，与 EnvState 生命周期对齐：
// - 保留 scope 内的权重，并为 scope 内缺失的环境补充默认值；
// - 已部署且离开 scope 的环境权重暂时保留，直至下次部署/卸载清理；
// - 未部署且不在 scope 的权重立即丢弃。
func (*PolarisEnvStateManager) reconcileEnvWeightsForScope(
	scopeEnvNames []string,
	envWeights map[string]int32,
	envStates map[string]PolarisEnvState,
) map[string]int32 {
	weights := make(map[string]int32, len(scopeEnvNames)+len(envWeights))
	for envName, w := range envWeights {
		if lo.Contains(scopeEnvNames, envName) {
			weights[envName] = w
			continue
		}
		if envStates[envName].IsDeployed() {
			weights[envName] = w
		}
	}
	for _, envName := range scopeEnvNames {
		if _, ok := weights[envName]; !ok {
			weights[envName] = DefaultEnvWeight
		}
	}
	return weights
}

// selectEnvNamesForDynamicApply 返回 scope 内满足动态下发条件的环境名称。
func (m *PolarisEnvStateManager) selectEnvNamesForDynamicApply(config *PolarisConfig) []string {
	envNames := make([]string, 0, len(config.ScopeEnvNames))
	for _, envName := range config.ScopeEnvNames {
		if !m.isEnvReadyForDynamicApply(config, envName) {
			continue
		}
		envNames = append(envNames, envName)
	}
	return envNames
}

// isEnvReadyForDynamicApply 判断指定环境是否允许动态下发。
// 动态下发 PolarisConfig CR，仅当三个关键字段的快照与当前配置的一致时，可以更新非关键字段。
// 这三个字段会影响到环境变量的生成和端口等服务配置，若不一致则必须重新部署实例。
func (*PolarisEnvStateManager) isEnvReadyForDynamicApply(config *PolarisConfig, envName string) bool {
	state := config.GetEnvState(envName)
	desiredFields := NewRedeployRequiredFields(config)
	return state.IsDeployed() && *state.AppliedFields == *desiredFields
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
		if err := m.store.RemoveEnvWeights(ctx, appID, config.Name, []string{envName}); err != nil {
			return errors.Wrapf(err, "remove out-of-scope env weight for config %s", config.Name)
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
		// 若环境仍在 scope 内时保留权重供下次部署使用
		if lo.Contains(config.ScopeEnvNames, envName) {
			continue
		}
		// 若环境已不在 scope 内，同步清理此前保留的 envWeights
		if err = m.store.RemoveEnvWeights(ctx, app.ID, config.Name, []string{envName}); err != nil {
			return errors.Wrapf(err, "remove env weight for config %s after uninstall", config.Name)
		}
	}
	return nil
}
