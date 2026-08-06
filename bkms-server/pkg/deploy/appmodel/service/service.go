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

package appmodeldeploy

import (
	"context"

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	deployappmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	scopedenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/promotion"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// DeployParams appmodel 部署动作所需参数
type DeployParams struct {
	EnvName         string
	TrafficLaneName string
	ImageTag        string
	UpdateStrategy  string
	Replicas        int32
}

// Service 负责 appmodel 部署相关编排，不负责权限校验与任务投递
type Service struct {
	appStore                            bkmsapp.ApplicationStore
	envStore                            envmodel.EnvironmentStore
	promotionStore                      promotion.PromotionStore
	snapshotService                     *snapshot.Service
	appModelStore                       appmodel.AppModelStore
	workspaceStore                      workspace.WorkspaceStore
	imageRegistryStore                  registry.ImageRegistryStore
	scopedEnvVarStore                   scopedenvvars.ScopedEnvVarStore
	appDepsVarReader                    *depenvvars.Reader
	polarisVarReader                    *polarisenvvars.Reader
	bscpCfgStore                        bscpcfg.Store
	workspaceCompsStore                 workspace.WorkspaceCompsStore
	polarisConfigStore                  polaris.PolarisConfigStore
	appSpecStore                        appspec.AppSpecStore
	buildConfigStore                    build.ConfigStore
	buildAutoDeployRecordStore          autodeploy.RecordStore
	appModelDeployRecordStore           deployappmodel.RecordStore
	appModelDeployResourceSnapshotStore deployappmodel.ResourceSnapshotStore
	appConfigFileStore                  appcfg.AppConfigFileStore
}

// ServiceDeps 部署服务所需依赖
type ServiceDeps struct {
	AppStore                            bkmsapp.ApplicationStore             `validate:"required"`
	EnvStore                            envmodel.EnvironmentStore            `validate:"required"`
	PromotionStore                      promotion.PromotionStore             `validate:"required"`
	SnapshotService                     *snapshot.Service                    `validate:"required"`
	AppModelStore                       appmodel.AppModelStore               `validate:"required"`
	WorkspaceStore                      workspace.WorkspaceStore             `validate:"required"`
	ImageRegistryStore                  registry.ImageRegistryStore          `validate:"required"`
	ScopedEnvVarStore                   scopedenvvars.ScopedEnvVarStore      `validate:"required"`
	AppDepsVarReader                    *depenvvars.Reader                   `validate:"required"`
	PolarisVarReader                    *polarisenvvars.Reader               `validate:"required"`
	WorkspaceCompsStore                 workspace.WorkspaceCompsStore        `validate:"required"`
	PolarisConfigStore                  polaris.PolarisConfigStore           `validate:"required"`
	BscpCfgStore                        bscpcfg.Store                        `validate:"required"`
	AppSpecStore                        appspec.AppSpecStore                 `validate:"required"`
	BuildConfigStore                    build.ConfigStore                    `validate:"required"`
	BuildAutoDeployRecordStore          autodeploy.RecordStore               `validate:"required"`
	AppModelDeployRecordStore           deployappmodel.RecordStore           `validate:"required"`
	AppModelDeployResourceSnapshotStore deployappmodel.ResourceSnapshotStore `validate:"required"`
	AppConfigFileStore                  appcfg.AppConfigFileStore            `validate:"required"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

// NewService 创建部署服务
func NewService(deps ServiceDeps) (*Service, error) {
	if err := validate.Struct(deps); err != nil {
		return nil, errors.Wrap(err, "validate dependencies")
	}
	return &Service{
		appStore:                            deps.AppStore,
		envStore:                            deps.EnvStore,
		promotionStore:                      deps.PromotionStore,
		snapshotService:                     deps.SnapshotService,
		appModelStore:                       deps.AppModelStore,
		workspaceStore:                      deps.WorkspaceStore,
		imageRegistryStore:                  deps.ImageRegistryStore,
		scopedEnvVarStore:                   deps.ScopedEnvVarStore,
		appDepsVarReader:                    deps.AppDepsVarReader,
		polarisVarReader:                    deps.PolarisVarReader,
		bscpCfgStore:                        deps.BscpCfgStore,
		workspaceCompsStore:                 deps.WorkspaceCompsStore,
		polarisConfigStore:                  deps.PolarisConfigStore,
		appSpecStore:                        deps.AppSpecStore,
		buildConfigStore:                    deps.BuildConfigStore,
		buildAutoDeployRecordStore:          deps.BuildAutoDeployRecordStore,
		appModelDeployRecordStore:           deps.AppModelDeployRecordStore,
		appModelDeployResourceSnapshotStore: deps.AppModelDeployResourceSnapshotStore,
		appConfigFileStore:                  deps.AppConfigFileStore,
	}, nil
}

// NewServiceFromRegistry registry 构建部署服务
func NewServiceFromRegistry(reg *storereg.Registry) (*Service, error) {
	if reg == nil {
		return nil, errors.New("store registry not initialized")
	}
	snapshotSvc := snapshot.NewService(
		reg.SnapshotStore,
		reg.BuildConfigStore,
		reg.AppStore,
	)
	return NewService(ServiceDeps{
		AppStore:                            reg.AppStore,
		EnvStore:                            reg.EnvStore,
		PromotionStore:                      reg.PromotionStore,
		SnapshotService:                     snapshotSvc,
		AppModelStore:                       reg.AppModelStore,
		WorkspaceStore:                      reg.WorkspaceStore,
		ImageRegistryStore:                  reg.ImageRegistryStore,
		ScopedEnvVarStore:                   reg.ScopedEnvVarStore,
		AppDepsVarReader:                    reg.AppDepsVarReader,
		PolarisVarReader:                    reg.PolarisVarReader,
		WorkspaceCompsStore:                 reg.WorkspaceCompsStore,
		PolarisConfigStore:                  reg.PolarisConfigStore,
		BscpCfgStore:                        reg.BscpCfgStore,
		AppSpecStore:                        reg.AppSpecStore,
		BuildConfigStore:                    reg.BuildConfigStore,
		BuildAutoDeployRecordStore:          reg.BuildAutoDeployRecordStore,
		AppModelDeployRecordStore:           reg.AppModelDeployRecordStore,
		AppModelDeployResourceSnapshotStore: reg.AppModelDeployResourceSnapshotStore,
		AppConfigFileStore:                  reg.AppConfigFileStore,
	})
}

// NewGlobalService store service构建工厂
func NewGlobalService() (*Service, error) {
	return NewServiceFromRegistry(storereg.G())
}

// Deploy 执行 appmodel 部署并返回 deployID。
func (s *Service) Deploy(ctx context.Context, app *bkmsapp.Application, params DeployParams) (string, error) {
	env, err := s.envStore.GetByName(ctx, app.WorkspaceID, app.ID, params.EnvName)
	if err != nil {
		return "", errors.Wrap(err, "get env")
	}

	// 执行部署前置检查
	if err = deploy.NewPreDeployChecker(
		s.envStore, s.promotionStore, s.snapshotService,
	).Do(ctx, &deploy.PreDeployCheckParams{
		WorkspaceID:     app.WorkspaceID,
		EnvName:         params.EnvName,
		TrafficLaneName: params.TrafficLaneName,
		AppType:         app.Type,
		AppID:           app.ID,
		ImageTag:        params.ImageTag,
	}); err != nil {
		return "", errors.Wrap(err, "pre deploy check")
	}

	appModel, err := s.appModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		return "", errors.Wrapf(err, "get app %s model", app.ID)
	}
	ws, err := s.workspaceStore.Get(ctx, app.WorkspaceID)
	if err != nil {
		return "", errors.Wrap(err, "get workspace")
	}
	imageRegistry, err := s.imageRegistryStore.GetByWorkspaceAndType(ctx, ws.ID, ws.ImageRegistryType)
	if err != nil {
		return "", errors.Wrap(err, "get image registry")
	}

	updateStrategy := params.UpdateStrategy
	if updateStrategy == "" {
		updateStrategy = string(tkex.RollingGameDeploymentUpdateStrategyType)
	}
	builderService := workload.NewBuilderService(
		s.scopedEnvVarStore,
		s.appDepsVarReader,
		s.polarisVarReader,
		s.workspaceCompsStore,
		s.polarisConfigStore,
		s.bscpCfgStore,
		s.appModelStore,
		s.appSpecStore,
		s.buildConfigStore,
	)
	deployer := deployappmodel.NewDeployer(
		s.appModelDeployRecordStore,
		s.appModelDeployResourceSnapshotStore,
		s.buildAutoDeployRecordStore,
		s.appModelStore,
		builderService,
		s.appSpecStore,
		s.buildConfigStore,
		s.appConfigFileStore,
		polaris.NewPolarisEnvStateManager(s.polarisConfigStore),
		app,
	)
	deployID, err := deployer.Deploy(
		ctx,
		appModel,
		env,
		imageRegistry,
		params.TrafficLaneName,
		params.ImageTag,
		updateStrategy,
		params.Replicas,
	)
	if err != nil {
		return "", errors.Wrap(err, "deploy appmodel")
	}

	return deployID, nil
}

// DeployByAppID 通过 appID 装载应用后执行部署
func (s *Service) DeployByAppID(ctx context.Context, appID string, params DeployParams) (string, error) {
	app, err := s.appStore.GetApp(ctx, appID)
	if err != nil {
		return "", errors.Wrapf(err, "get app %s", appID)
	}
	return s.Deploy(ctx, app, params)
}
