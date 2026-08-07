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

	// 过滤掉 scope 外且未部署的环境权重，并为 scope 内未设置权重的环境补充默认值
	config.EnvWeights = s.envStateManager.reconcileEnvWeightsForScope(
		config.ScopeEnvNames, config.EnvWeights, nil,
	)

	return s.polarisConfigStore.Create(ctx, config)
}

// Update 更新北极星配置
func (s *PolarisConfigService) Update(
	ctx context.Context,
	app *bkmsapp.Application,
	oldConfig *PolarisConfig,
	updateData *ConfigUpdateData,
) (*PolarisConfig, error) {
	if updateData.ScopeEnvNames != nil {
		// scope 变化时保留仍有效的权重，并为新增环境补充默认值。
		updateData.envWeights = s.envStateManager.reconcileEnvWeightsForScope(
			updateData.ScopeEnvNames,
			oldConfig.EnvWeights,
			oldConfig.EnvStates,
		)
	}

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
	return s.applier.patchWeight(ctx, app, env, config, weight)
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
