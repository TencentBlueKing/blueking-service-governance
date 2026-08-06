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

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// PolarisConfigService 负责配置管理、北极星平台服务生命周期和动态下发编排。
type PolarisConfigService struct {
	polarisConfigStore PolarisConfigStore
	platformManager    *PolarisPlatformManager
	envStateManager    *PolarisEnvStateManager
	applier            *polarisCRApplier
	envStore           bkmsenv.EnvironmentStore
	appModelStore      appmodel.AppModelStore
	envVarsReader      envVarsReader
}

// NewPolarisConfigService 创建北极星配置服务。
func NewPolarisConfigService(
	store PolarisConfigStore,
	platformManager *PolarisPlatformManager,
	envStateManager *PolarisEnvStateManager,
	envStore bkmsenv.EnvironmentStore,
	appModelStore appmodel.AppModelStore,
	envVarsReader envVarsReader,
) *PolarisConfigService {
	return &PolarisConfigService{
		polarisConfigStore: store,
		platformManager:    platformManager,
		envStateManager:    envStateManager,
		applier:            newPolarisCRApplier(),
		envStore:           envStore,
		appModelStore:      appModelStore,
		envVarsReader:      envVarsReader,
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

	return s.polarisConfigStore.Create(ctx, config)
}

// Update 更新北极星配置
func (s *PolarisConfigService) Update(
	ctx context.Context,
	app *bkmsapp.Application,
	oldConfig *PolarisConfig,
	updateData *ConfigUpdateData,
) (*PolarisConfig, error) {
	if err := s.polarisConfigStore.Update(ctx, app.ID, oldConfig.Name, updateData); err != nil {
		return nil, errors.Wrap(err, "update polaris config")
	}

	newConfig, err := s.polarisConfigStore.Get(ctx, app.ID, oldConfig.Name)
	if err != nil {
		return nil, errors.Wrap(err, "get updated polaris config")
	}
	envNames, err := s.envStateManager.PrepareDynamicApply(ctx, newConfig)
	if err != nil {
		return newConfig, errors.Wrap(err, "prepare dynamic polaris apply")
	}
	s.triggerDynamicApply(ctx, app, newConfig, envNames)
	return newConfig, nil
}

// Delete 删除北极星配置，并按需删除北极星服务实例
func (s *PolarisConfigService) Delete(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
) error {
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

type envVarsReader interface {
	ListVars(
		ctx context.Context,
		environment bkmsenv.Environment,
		app *bkmsapp.Application,
		appModel *appmodel.AppModel,
	) (envvartypes.EnvVariableList, error)
}

// triggerDynamicApply 异步下发允许直接生效的 PolarisConfig CR。
func (s *PolarisConfigService) triggerDynamicApply(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envNames []string,
) {
	if len(envNames) == 0 {
		return
	}
	// TODO: asynq 引入后，切换为异步任务队列。
	go s.applyToEnvs(context.WithoutCancel(ctx), app, config, envNames)
}

// applyToEnvs 准备各环境的构建上下文，触发资源下发并记录结果。
func (s *PolarisConfigService) applyToEnvs(
	ctx context.Context,
	app *bkmsapp.Application,
	config *PolarisConfig,
	envNames []string,
) {
	appModel, err := s.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		applyErr := errors.Wrap(err, "get app model for polaris CR apply")
		for _, envName := range envNames {
			s.recordDynamicApplyResult(ctx, app.ID, config.Name, envName, applyErr)
		}
		log.Errorf(ctx, "get app model for polaris CR apply failed, app=%s: %v", app.ID, applyErr)
		return
	}

	for _, envName := range envNames {
		applyErr := s.applyToEnv(ctx, app, appModel, config, envName)
		s.recordDynamicApplyResult(ctx, app.ID, config.Name, envName, applyErr)
		if applyErr != nil {
			log.Errorf(ctx, "apply polaris CR failed, app=%s config=%s env=%s: %v",
				app.ID, config.Name, envName, applyErr)
		}
	}
}

// applyToEnv 读取单个环境的构建输入并调用资源下发器。
func (s *PolarisConfigService) applyToEnv(
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
	return s.applier.apply(ctx, app, env, config, envVars.ToMap())
}

func (s *PolarisConfigService) recordDynamicApplyResult(
	ctx context.Context,
	appID, configName, envName string,
	applyErr error,
) {
	if err := s.envStateManager.RecordDynamicApplyResult(
		ctx, appID, configName, envName, applyErr,
	); err != nil {
		log.Errorf(ctx, "record polaris CR apply result failed, app=%s config=%s env=%s: %v",
			appID, configName, envName, err)
	}
}
