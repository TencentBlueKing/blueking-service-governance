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
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

// DynamicApplyEnqueuer 为单个可动态下发环境投递 asynq 任务。
type DynamicApplyEnqueuer func(ctx context.Context, appID, configName, envName string) error

// ErrClusterSyncFailed 表示配置本身已经保存成功，但 immediate 模式的集群资源同步失败。
//
// 调用方据此区分"操作没生效"和"配置已保存、北极星侧未收敛"：后者的配置变更依然有效，
// 失败原因已写入对应环境的 EnvState.LastError，重新保存一次配置即可重试。
var ErrClusterSyncFailed = errors.New("polaris cluster resources sync failed")

// PolarisConfigService 负责配置管理和北极星平台服务生命周期。
// on_deploy 的动态下发通过 asynq 按环境入队；immediate 配置在请求内同步下发 CR 与 Service。
type PolarisConfigService struct {
	polarisConfigStore  PolarisConfigStore
	platformManager     *PolarisPlatformManager
	envStateManager     *PolarisEnvStateManager
	applier             *CRApplier
	envStore            bkmsenv.EnvironmentStore
	appModelStore       appmodel.AppModelStore
	envVarsReader       dynamicApplyEnvVarsReader
	enqueueDynamicApply DynamicApplyEnqueuer
}

// NewPolarisConfigService 创建北极星配置服务。
func NewPolarisConfigService(
	store PolarisConfigStore,
	platformManager *PolarisPlatformManager,
	envStateManager *PolarisEnvStateManager,
	envStore bkmsenv.EnvironmentStore,
	appModelStore appmodel.AppModelStore,
	envVarsReader dynamicApplyEnvVarsReader,
	enqueue DynamicApplyEnqueuer,
) *PolarisConfigService {
	return &PolarisConfigService{
		polarisConfigStore:  store,
		platformManager:     platformManager,
		envStateManager:     envStateManager,
		applier:             NewCRApplier(),
		envStore:            envStore,
		appModelStore:       appModelStore,
		envVarsReader:       envVarsReader,
		enqueueDynamicApply: enqueue,
	}
}

// Create 创建北极星配置，并按需创建北极星服务实例
func (s *PolarisConfigService) Create(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	createNewService bool,
) error {
	if createNewService {
		result, err := s.platformManager.CreateService(ctx, &CreatePolarisServiceParams{
			PolarisName:      config.PolarisName,
			PolarisNamespace: config.PolarisNamespace,
			Operator:         config.Operator,
			WorkspaceID:      app.WorkspaceID,
			AppID:            app.ID,
			ScopeEnvNames:    config.ScopeEnvNames,
		})
		if err != nil {
			return errors.Wrap(err, "create polaris service")
		}
		config.PolarisToken = result.Token
		config.DepSvcInstID = result.ServiceInstanceID
	}

	// 过滤掉 scope 外且未部署的环境权重，并为 scope 内未设置权重的环境补充默认值
	config.EnvWeights = s.envStateManager.reconcileEnvWeightsForScope(
		config.ScopeEnvNames, config.EnvWeights, nil, config.RegisterMode,
	)

	if err := s.polarisConfigStore.Create(ctx, config); err != nil {
		return err
	}

	// immediate 配置绑定即注册，在请求内同步下发；配置已经落库，失败时由调用方决定是否重试
	if !config.IsImmediateRegister() {
		return nil
	}
	return s.applyImmediately(ctx, app, config, config.ScopeEnvNames)
}

// Update 更新北极星配置
func (s *PolarisConfigService) Update(
	ctx context.Context,
	app *bkmsapp.Application,
	oldConfig *PolarisConfig,
	updateData *ConfigUpdateData,
) (*PolarisConfig, error) {
	// 处理对负责人字段的更新
	if updateData.Operator != nil {
		if strings.TrimSpace(*updateData.Operator) == "" {
			return nil, ErrOperatorEmpty
		}
		if oldConfig.DepSvcInstID.IsZero() {
			return nil, ErrOperatorNotManaged
		}
		if err := s.platformManager.UpdateServiceOwners(ctx, oldConfig, *updateData.Operator); err != nil {
			return nil, err
		}
	}

	if updateData.ScopeEnvNames != nil {
		// scope 变化时保留仍有效的权重，并为新增环境补充默认值。
		updateData.envWeights = s.envStateManager.reconcileEnvWeightsForScope(
			updateData.ScopeEnvNames,
			oldConfig.EnvWeights,
			oldConfig.EnvStates,
			oldConfig.RegisterMode,
		)
	}

	if err := s.polarisConfigStore.Update(ctx, app.ID, oldConfig.Name, updateData); err != nil {
		return nil, errors.Wrap(err, "update polaris config")
	}

	newConfig, err := s.polarisConfigStore.Get(ctx, app.ID, oldConfig.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get updated polaris config")
	}
	if !updateData.affectsWorkload() {
		return newConfig, nil
	}

	if newConfig.IsImmediateRegister() {
		reconcileErr := s.reconcileImmediately(ctx, app, newConfig)
		// 同步下发会改写环境记录，重新读取以便调用方拿到本次下发后的状态
		syncedConfig, getErr := s.polarisConfigStore.Get(ctx, app.ID, oldConfig.Name)
		if getErr != nil {
			return newConfig, stderrors.Join(
				reconcileErr, errors.Wrap(getErr, "get polaris config after immediate apply"),
			)
		}
		return syncedConfig, reconcileErr
	}

	envNames, err := s.envStateManager.PrepareDynamicApply(ctx, newConfig)
	if err != nil {
		return newConfig, errors.Wrap(err, "prepare dynamic polaris apply")
	}
	// 请求结束后仍需保证任务投递，避免客户端断开导致配置更新成功但任务丢失。
	if err = s.triggerDynamicApply(context.WithoutCancel(ctx), newConfig, envNames); err != nil {
		return newConfig, errors.Wrap(err, "enqueue polaris dynamic apply")
	}
	return newConfig, nil
}

// Delete 删除北极星配置，并按需删除北极星服务实例
func (s *PolarisConfigService) Delete(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
) error {
	// immediate 配置的集群资源不会随部署清理，必须在删除配置前先摘除，否则实例会一直留在北极星上。
	// 删除失败时不继续删配置，用户重试即可收敛。
	if config.IsImmediateRegister() {
		if err := s.releaseFromEnvs(ctx, app, config, config.TrackedEnvNames()); err != nil {
			return err
		}
	}

	if !config.DepSvcInstID.IsZero() {
		if err := s.platformManager.DeleteService(ctx, &DeleteServiceParams{
			ServiceInstanceID: config.DepSvcInstID,
			AppID:             app.ID,
		}); err != nil {
			return errors.Wrap(err, "delete polaris service")
		}
	}
	if err := s.polarisConfigStore.Delete(ctx, app.ID, config.Name); err != nil {
		return errors.Wrap(err, "delete polaris config")
	}
	return nil
}

// reconcileImmediately 同步收敛 immediate 配置的集群资源：先清理离域环境，再下发全部 scope 环境。
func (s *PolarisConfigService) reconcileImmediately(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
) error {
	releaseErr := s.releaseFromEnvs(
		ctx, app, config, config.EnvNamesOutsideScope(),
	)
	applyErr := s.applyImmediately(ctx, app, config, config.ScopeEnvNames)
	if releaseErr != nil {
		return releaseErr
	}
	return applyErr
}

// applyImmediately 逐环境同步下发 CR 与 Service，并记录每个环境的结果。
// 单个环境失败不影响其余环境，全部处理完后汇总返回。
func (s *PolarisConfigService) applyImmediately(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envNames []string,
) error {
	if len(envNames) == 0 {
		return nil
	}

	appModel, err := s.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		applyErr := errors.Wrap(err, "get app model for polaris resources apply")
		log.Errorf(ctx, "get app model for polaris resources apply failed, app=%s: %v", app.ID, applyErr)
		return errors.Wrap(ErrClusterSyncFailed, applyErr.Error())
	}

	failures := make([]string, 0, len(envNames))
	for _, envName := range envNames {
		applyErr := s.applyResourcesToEnv(ctx, app, appModel, config, envName)
		s.recordImmediateApplyResult(ctx, config, envName, applyErr)
		if applyErr == nil {
			continue
		}
		log.Errorf(ctx, "apply polaris resources failed, app=%s config=%s env=%s: %v",
			app.ID, config.Name, envName, applyErr)
		failures = append(failures, fmt.Sprintf("%s: %v", envName, applyErr))
	}
	if len(failures) > 0 {
		return errors.Wrapf(ErrClusterSyncFailed, "apply failed in env(s) [%s]", strings.Join(failures, "; "))
	}
	return nil
}

// applyResourcesToEnv 读取单个环境的构建输入并下发该配置的全部资源。
func (s *PolarisConfigService) applyResourcesToEnv(
	ctx context.Context,
	app *bkmsapp.Application,
	appModel *appmodel.AppModel,
	config *PolarisConfig,
	envName string,
) error {
	env, err := s.envStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return errors.Wrapf(err, "get env %s", envName)
	}
	envVars, err := s.envVarsReader.ListVars(ctx, *env, app, appModel)
	if err != nil {
		return errors.Wrapf(err, "build env vars for %s", envName)
	}
	return s.applier.Apply(ctx, app, env, config, envVars.ToMap())
}

// releaseFromEnvs 逐环境删除集群资源，成功后清理该环境的记录与权重。
// 删除失败时保留记录，下次保存配置会重新进入清理列表。
func (s *PolarisConfigService) releaseFromEnvs(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envNames []string,
) error {
	failures := make([]string, 0, len(envNames))
	for _, envName := range envNames {
		releaseErr := s.releaseFromEnv(ctx, app, config, envName)
		if releaseErr == nil {
			continue
		}
		log.Errorf(ctx, "release polaris resources failed, app=%s config=%s env=%s: %v",
			app.ID, config.Name, envName, releaseErr)
		s.recordImmediateApplyResult(ctx, config, envName, releaseErr)
		failures = append(failures, fmt.Sprintf("%s: %v", envName, releaseErr))
	}
	if len(failures) > 0 {
		return errors.Wrapf(ErrClusterSyncFailed, "release failed in env(s) [%s]", strings.Join(failures, "; "))
	}
	return nil
}

func (s *PolarisConfigService) releaseFromEnv(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envName string,
) error {
	env, err := s.envStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return errors.Wrapf(err, "get env %s", envName)
	}
	if err = s.applier.DeleteResources(ctx, app, env, config); err != nil {
		return err
	}
	return s.envStateManager.ReleaseEnv(ctx, app.ID, config.Name, envName)
}

func (s *PolarisConfigService) recordImmediateApplyResult(
	ctx context.Context,
	config *PolarisConfig,
	envName string,
	applyErr error,
) {
	if err := s.envStateManager.RecordImmediateApplyResult(ctx, config, envName, applyErr); err != nil {
		log.Errorf(ctx, "record polaris resources apply result failed, app=%s config=%s env=%s: %v",
			config.AppID, config.Name, envName, err)
	}
}

// triggerDynamicApply 为每个可下发环境投递一条 asynq 任务。
// 某个环境投递失败会写入该环境 LastError，不阻止其余环境入队；返回全部投递错误的汇总。
func (s *PolarisConfigService) triggerDynamicApply(
	ctx context.Context,
	config *PolarisConfig,
	envNames []string,
) error {
	var errs []error
	for _, envName := range envNames {
		if err := s.enqueueDynamicApply(ctx, config.AppID, config.Name, envName); err != nil {
			enqueueErr := errors.Wrapf(err, "enqueue polaris dynamic apply for env %s", envName)
			if recErr := s.envStateManager.RecordDynamicApplyResult(
				ctx, config.AppID, config.Name, envName, config.UpdatedAt, enqueueErr,
			); recErr != nil {
				log.Errorf(ctx, "record polaris enqueue failure failed, app=%s config=%s env=%s: %v",
					config.AppID, config.Name, envName, recErr)
			}
			errs = append(errs, enqueueErr)
		}
	}
	return stderrors.Join(errs...)
}

func (s *PolarisConfigService) patchEnvWeight(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envName string,
	weight int32,
) error {
	env, err := s.envStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return errors.Wrapf(err, "get env %s", envName)
	}
	return s.applier.PatchWeight(ctx, app, env, config, weight)
}

// UpdateEnvWeight 更新指定环境的北极星实例权重；已部署环境会先同步 Patch 集群资源，成功后再持久化。
func (s *PolarisConfigService) UpdateEnvWeight(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envName string,
	weight int32,
) (*PolarisConfig, error) {
	isDeployed := config.GetEnvState(envName).IsDeployed()
	if isDeployed {
		if err := s.patchEnvWeight(ctx, app, config, envName, weight); err != nil {
			log.Errorf(ctx, "patch polaris CR weight failed, app=%s config=%s env=%s: %v",
				app.ID, config.Name, envName, err)
			return nil, errors.Wrap(err, "patch env weight")
		}
	}

	if err := s.polarisConfigStore.UpsertEnvWeight(ctx, app.ID, config.Name, envName, weight); err != nil {
		if isDeployed {
			log.Errorf(ctx, "persist polaris env weight after cluster patch failed, app=%s config=%s env=%s: %v",
				app.ID, config.Name, envName, err)
		}
		return nil, errors.Wrap(err, "update env weight")
	}

	// 重新读取最新配置
	newConfig, err := s.polarisConfigStore.Get(ctx, app.ID, config.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get updated polaris config")
	}

	return newConfig, nil
}
