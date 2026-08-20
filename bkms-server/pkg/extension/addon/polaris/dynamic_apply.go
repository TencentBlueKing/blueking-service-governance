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

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

var (
	// ErrDynamicApplyNotReady 表示环境当前不满足动态下发条件；在下一次部署前无需重试。
	ErrDynamicApplyNotReady = errors.New("polaris environment is not ready for dynamic apply")
	// ErrDynamicApplyConfigChanged 表示渲染或下发期间配置发生变化，任务应使用最新配置重试。
	ErrDynamicApplyConfigChanged = errors.New("polaris config changed during dynamic apply")
)

// dynamicApplyEnvVarsReader 是动态下发所需的最小环境变量读取接口。
// 接口放在 Polaris 包内，避免引入顶层 envvars 包造成循环依赖。
type dynamicApplyEnvVarsReader interface {
	ListVars(
		ctx context.Context,
		environment bkmsenv.Environment,
		app *bkmsapp.Application,
		appModel *appmodel.AppModel,
	) (envvartypes.EnvVariableList, error)
}

// DynamicApplyService 编排一次 PolarisConfig CR 动态下发。
//
// 设计目的：任务只携带业务主键，执行时读取最新数据；下发后复核配置版本，
// 发现并发更新就重试，并按版本记录结果，避免旧任务覆盖新状态。
// CR 更新与配置库不在同一事务中，依靠版本复核和重试在正常情况下收敛到最新配置。
type DynamicApplyService struct {
	appStore        bkmsapp.ApplicationStore
	configStore     PolarisConfigStore
	envStore        bkmsenv.EnvironmentStore
	appModelStore   appmodel.AppModelStore
	envVarsReader   dynamicApplyEnvVarsReader
	envStateManager *PolarisEnvStateManager
	applier         *CRApplier
}

// NewDynamicApplyService 创建 Polaris 动态下发服务。
func NewDynamicApplyService(
	appStore bkmsapp.ApplicationStore,
	configStore PolarisConfigStore,
	envStore bkmsenv.EnvironmentStore,
	appModelStore appmodel.AppModelStore,
	envVarsReader dynamicApplyEnvVarsReader,
	envStateManager *PolarisEnvStateManager,
) *DynamicApplyService {
	return &DynamicApplyService{
		appStore:        appStore,
		configStore:     configStore,
		envStore:        envStore,
		appModelStore:   appModelStore,
		envVarsReader:   envVarsReader,
		envStateManager: envStateManager,
		applier:         NewCRApplier(),
	}
}

// Apply 读取最新下发输入并更新目标 CR。
// 返回下发开始时观察到的配置 UpdatedAt，供调用方按版本记录任务结果。
func (s *DynamicApplyService) Apply(
	ctx context.Context,
	appID, configName, envName string,
) (time.Time, error) {
	app, err := s.appStore.GetApp(ctx, appID)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "get app for polaris CR apply")
	}

	config, err := s.configStore.Get(ctx, appID, configName)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "get polaris config for dynamic apply")
	}
	configUpdatedAt := config.UpdatedAt
	if !s.envStateManager.IsEnvReadyForDynamicApply(config, envName) {
		return configUpdatedAt, errors.Wrapf(
			ErrDynamicApplyNotReady,
			"env %s is not ready for dynamic apply",
			envName,
		)
	}

	appModel, err := s.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return configUpdatedAt, errors.Wrap(err, "get app model for polaris CR apply")
	}

	env, err := s.envStore.GetByName(ctx, app.WorkspaceID, app.ID, envName)
	if err != nil {
		return configUpdatedAt, errors.Wrapf(err, "get env %s", envName)
	}
	vars, err := s.envVarsReader.ListVars(ctx, *env, app, appModel)
	if err != nil {
		return configUpdatedAt, errors.Wrapf(err, "build env vars for %s", envName)
	}
	// 先下发、后校验版本，才能覆盖从读取配置到下发完成的并发更新窗口；
	// 若版本已变化，当前结果作废并重试最新配置，因此这是最终收敛而非原子一致。
	if err = s.applier.Apply(ctx, app, env, config, vars.ToMap()); err != nil {
		return configUpdatedAt, err
	}

	// 下发期间可能发生配置更新；完成后复读版本，避免旧渲染结果被当成最新结果。
	latestConfig, err := s.configStore.Get(ctx, appID, configName)
	if err != nil {
		return configUpdatedAt, errors.Wrap(err, "get polaris config after dynamic apply")
	}
	if !latestConfig.UpdatedAt.Equal(configUpdatedAt) {
		return configUpdatedAt, errors.Wrapf(
			ErrDynamicApplyConfigChanged,
			"updatedAt changed from %s to %s",
			configUpdatedAt.Format(time.RFC3339Nano),
			latestConfig.UpdatedAt.Format(time.RFC3339Nano),
		)
	}

	return configUpdatedAt, nil
}

// RecordDynamicApplyResult 按配置版本记录一次动态下发结果。
func (s *DynamicApplyService) RecordDynamicApplyResult(
	ctx context.Context,
	appID, configName, envName string,
	expectedUpdatedAt time.Time,
	applyErr error,
) error {
	return s.envStateManager.RecordDynamicApplyResult(
		ctx, appID, configName, envName, expectedUpdatedAt, applyErr,
	)
}
