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
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	deployappmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// DeployParams appmodel 部署动作所需参数
type DeployParams struct {
	EnvName             string
	TrafficLaneName     string
	ImageTag            string
	UpdateStrategy      string
	Replicas            int32
	BuildAutoDeployInfo *deployappmodel.BuildAutoDeployInfo
	// ReleaseStrategy 按该发布策略分批更新；为 nil 时一次更新全部实例
	ReleaseStrategy *releasestrategy.ReleaseStrategy
}

// Service 负责 appmodel 部署相关编排，不负责权限校验与任务投递
type Service struct {
	db               *mongo.Database
	appStore         *bkmsapp.ApplicationStore
	envStore         *envmodel.EnvironmentStore
	snapshotService  *snapshot.Service
	appDepsVarReader *depenvvars.Reader
	polarisVarReader *polarisenvvars.Reader
}

// ServiceDeps 部署服务所需依赖
type ServiceDeps struct {
	// DB 供基于 GenericStore 的模型使用
	DB               *mongo.Database        `validate:"required"`
	SnapshotService  *snapshot.Service      `validate:"required"`
	AppDepsVarReader *depenvvars.Reader     `validate:"required"`
	PolarisVarReader *polarisenvvars.Reader `validate:"required"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

// NewService 创建部署服务
func NewService(deps ServiceDeps) (*Service, error) {
	if err := validate.Struct(deps); err != nil {
		return nil, errors.Wrap(err, "validate dependencies")
	}
	return &Service{
		db:               deps.DB,
		appStore:         store.Of(deps.DB, bkmsapp.AppMeta),
		envStore:         store.Of(deps.DB, envmodel.EnvMeta),
		snapshotService:  deps.SnapshotService,
		appDepsVarReader: deps.AppDepsVarReader,
		polarisVarReader: deps.PolarisVarReader,
	}, nil
}

// NewServiceFromRegistry registry 构建部署服务
func NewServiceFromRegistry(reg *storereg.Registry) (*Service, error) {
	if reg == nil {
		return nil, errors.New("store registry not initialized")
	}
	snapshotSvc := snapshot.NewService(reg.DB)
	return NewService(ServiceDeps{
		DB:               reg.DB,
		SnapshotService:  snapshotSvc,
		AppDepsVarReader: reg.AppDepsVarReader,
		PolarisVarReader: reg.PolarisVarReader,
	})
}

// NewGlobalService store service构建工厂
func NewGlobalService() (*Service, error) {
	return NewServiceFromRegistry(storereg.G())
}

// Deploy 执行 appmodel 部署并返回 deployID。
func (s *Service) Deploy(ctx context.Context, app *bkmsapp.Application, params DeployParams) (string, error) {
	env, err := s.envStore.Get(ctx, envmodel.AppEnvByName(app.WorkspaceID, app.ID, params.EnvName))
	if err != nil {
		return "", errors.Wrap(err, "get env")
	}

	if err = s.runPreDeployChecks(ctx, app, env, params); err != nil {
		return "", err
	}

	appModel, err := store.Of(s.db, appmodel.AppModelMeta).Get(ctx, appmodel.ByApp(app.ID))
	if err != nil {
		return "", errors.Wrapf(err, "get app %s model", app.ID)
	}
	ws, err := store.Of(s.db, workspace.Meta).Get(ctx, workspace.ByWorkspaceID(app.WorkspaceID))
	if err != nil {
		return "", errors.Wrap(err, "get workspace")
	}
	imageRegistry, err := store.Of(s.db, registry.ImageRegistryMeta).Get(
		ctx, registry.ByWorkspaceAndType(ws.ID, ws.ImageRegistryType),
	)
	if err != nil {
		return "", errors.Wrap(err, "get image registry")
	}

	updateStrategy := params.UpdateStrategy
	if updateStrategy == "" {
		updateStrategy = string(tkex.RollingGameDeploymentUpdateStrategyType)
	}
	builderService := workload.NewBuilderService(
		s.appDepsVarReader,
		s.polarisVarReader,
		s.db,
		appcfg.NewMountableFileProvider(s.db),
	)
	deployer := deployappmodel.NewDeployer(
		builderService,
		s.db,
		polaris.NewPolarisEnvStateManager(s.db),
		hostport.NewEnvStateManager(s.db),
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
		params.BuildAutoDeployInfo,
		params.ReleaseStrategy,
	)
	if err != nil {
		return "", errors.Wrap(err, "deploy appmodel")
	}

	return deployID, nil
}

func (s *Service) runPreDeployChecks(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	params DeployParams,
) error {
	if err := s.checkRequiredClusterAddons(ctx, app, env); err != nil {
		return errors.Wrap(err, "check required cluster addons")
	}
	if err := deploy.NewPreDeployChecker(s.db, s.snapshotService).Do(ctx, &deploy.PreDeployCheckParams{
		WorkspaceID:     app.WorkspaceID,
		EnvName:         params.EnvName,
		TrafficLaneName: params.TrafficLaneName,
		AppType:         app.Type,
		AppID:           app.ID,
		ImageTag:        params.ImageTag,
	}); err != nil {
		return errors.Wrap(err, "pre deploy check")
	}
	return nil
}

func (s *Service) checkRequiredClusterAddons(
	ctx context.Context,
	app *bkmsapp.Application,
	env *envmodel.Environment,
) error {
	defs, err := clusteraddon.ListDefs(ctx, store.Of(s.db, clusteraddon.ClusterAddonDefMeta))
	if err != nil {
		return err
	}
	missing, err := clusteraddon.InspectRequiredAddons(
		ctx, defs, app.Type, env, "",
	)
	if err != nil {
		return errors.Wrap(err, "inspect required cluster addons")
	}
	if len(missing) > 0 {
		return &clusteraddon.RequiredAddonsNotInstalledError{Missing: missing}
	}
	return nil
}

// DeployByAppID 通过 appID 装载应用后执行部署
func (s *Service) DeployByAppID(ctx context.Context, appID string, params DeployParams) (string, error) {
	app, err := s.appStore.Get(ctx, bkmsapp.ByID(appID))
	if err != nil {
		return "", errors.Wrapf(err, "get app %s", appID)
	}
	return s.Deploy(ctx, app, params)
}
