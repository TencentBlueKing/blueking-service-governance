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

package appmodeldeploypoll

import (
	"context"

	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// triggerTopologyRefresh 按部署记录的 ResourceKeys / LabelSelector 刷拓扑资源范围
// 由 Handle 在 goroutine 中调用，resourceKeys 为空则跳过；失败只打日志，不影响本 tick
func triggerTopologyRefresh(ctx context.Context, args Args, record *appmodeldeploy.Record) {
	var resourceKeys []topology.ResourceKeyEntry
	for _, rk := range record.ResourceKeys {
		resourceKeys = append(resourceKeys, topology.ResourceKeyEntry{Kind: rk.Kind, Name: rk.Name})
	}
	if len(resourceKeys) == 0 {
		log.Warnf(ctx, "skip topology refresh (app model): resourceKeys is empty")
		return
	}

	store, err := topology.NewResourceSnapshotStoreMongo(database.Client(), database.Name())
	if err != nil {
		log.Errorf(ctx, "topology refresh (app model): create store: %v", err)
		return
	}
	topology.NewRefresher(store).TriggerRefresh(ctx, topology.RefreshArgs{
		AppID:           args.AppID,
		EnvName:         args.EnvName,
		TrafficLaneName: args.TrafficLaneName,
		ClusterID:       record.ClusterID,
		Namespace:       record.Namespace,
		ResourceKeys:    resourceKeys,
		LabelSelector:   record.LabelSelector,
	})
}

// handleDeploySucceeded 部署成功后的后置动作：记录应用与环境关联，并异步同步该环境告警策略
// 仅 StatusDeployed 时由 onStable 调用；任一步失败只打日志，不改部署结果、不让本 tick 失败
func handleDeploySucceeded(ctx context.Context, args Args, record *appmodeldeploy.Record) {
	reg := storereg.G()
	if reg == nil {
		log.Errorf(ctx, "post-deploy hooks: registry is not initialized")
		return
	}

	log.Infof(
		ctx, "deploy succeeded, start post-deploy hooks for workspace=%s app=%s env=%s lane=%s operator=%s",
		args.WorkspaceID, args.AppID, args.EnvName, args.TrafficLaneName, record.Creator,
	)
	// 记录应用到环境的部署关联，envStore 未初始化时只打日志，不阻断后续告警同步
	if reg.EnvStore == nil {
		log.Errorf(ctx, "track env add app: env store is not initialized")
	} else {
		deploy.TrackEnvAddApp(ctx, reg.EnvStore, args.WorkspaceID, args.EnvName, args.AppID)
	}

	// 异步把应用关联的告警策略同步到当前环境，失败只打日志，不改部署结果、不让本 tick 失败
	deploy.SyncAlertStrategiesAfterDeploy(
		ctx, args.WorkspaceID, args.AppID, args.EnvName, args.TrafficLaneName, record.Creator,
	)

	// 部署成功后，回调 apm 关联容器接口，不影响部署结果
	syncApmServiceConfigAfterDeploy(ctx, reg, args, record)
}

// syncApmServiceConfigAfterDeploy 回调 apm 关联容器接口
func syncApmServiceConfigAfterDeploy(
	ctx context.Context,
	reg *storereg.Registry,
	args Args,
	record *appmodeldeploy.Record,
) {
	kind, name := record.MainWorkload()
	if kind == "" || name == "" {
		log.Warnf(ctx, "sync apm service config: main workload not found in deploy record, skip: %s", args)
		return
	}

	app, env, ws, ok := loadApmServiceConfigContext(ctx, reg, args)
	if !ok {
		return
	}

	req := newApmServiceConfigRequest(ctx, reg, app, env, ws, kind, name)
	if req == nil {
		return
	}

	client, err := bkmapi.NewMonitorClient(record.Creator)
	if err != nil {
		log.Errorf(
			ctx, "sync apm service config: new bkmonitor client failed, app=%s env=%s: %v",
			app.ID, env.Name, err,
		)
		return
	}
	if err = client.UpdateApmServiceConfig(ctx, req); err != nil {
		log.Errorf(
			ctx, "sync apm service config: update apm service config failed, app=%s env=%s: %v",
			app.ID, env.Name, err,
		)
		return
	}

	log.Infof(
		ctx, "update apm service config success, app=%s env=%s app_name=%s service_name=%s",
		app.ID, env.Name, req.AppName, req.ServiceName,
	)
}

// loadApmServiceConfigContext 加载回调所需的 app/env/workspace，任一失败仅记录日志并返回 false。
func loadApmServiceConfigContext(
	ctx context.Context,
	reg *storereg.Registry,
	args Args,
) (*bkmsapp.Application, *envmodel.Environment, *workspace.Workspace, bool) {
	app, err := reg.AppStore.GetApp(ctx, args.AppID)
	if err != nil {
		log.Errorf(ctx, "sync apm service config: get app %s: %v", args.AppID, err)
		return nil, nil, nil, false
	}
	env, err := reg.EnvStore.GetByName(ctx, args.WorkspaceID, args.AppID, args.EnvName)
	if err != nil {
		log.Errorf(ctx, "sync apm service config: get env %s: %v", args.EnvName, err)
		return nil, nil, nil, false
	}
	ws, err := reg.WorkspaceStore.Get(ctx, args.WorkspaceID)
	if err != nil {
		log.Errorf(ctx, "sync apm service config: get workspace %s: %v", args.WorkspaceID, err)
		return nil, nil, nil, false
	}
	return app, env, ws, true
}

// newApmServiceConfigRequest 获取 app_name / service_name / owners 并构建请求
func newApmServiceConfigRequest(
	ctx context.Context,
	reg *storereg.Registry,
	app *bkmsapp.Application,
	env *envmodel.Environment,
	ws *workspace.Workspace,
	kind, name string,
) *bkmapi.UpdateApmServiceConfigReq {
	// 仅 trpc / taf 应用需要回调 apm 关联容器接口
	if !bkmsapp.IsAppModelType(app.Type) {
		log.Infof(ctx, "app %s type %s not trpc/taf, skip sync apm service config", app.ID, app.Type)
		return nil
	}
	// 查询环境绑定的 APM 名称
	apm, err := reg.ApmInstConfigStore.GetByEnvID(ctx, env.ID)
	if err != nil {
		if errors.Is(err, bkmmodel.ErrApmInstConfigNotFound) {
			log.Infof(ctx, "env %s not bound to any apm, skip sync apm service config, app=%s", env.Name, app.ID)
			return nil
		}
		log.Errorf(ctx, "sync apm service config: get apm by env failed, app=%s env=%s: %v", app.ID, env.Name, err)
		return nil
	}
	// 获取 service_name
	serviceName := resolveApmServiceName(ctx, reg, app, env)
	if serviceName == "" {
		return nil
	}
	// owners：admin/sre 成员
	owners, err := workspace.ListRoleMembers(ctx, ws.ID, perm.RoleCodeAdmin, perm.RoleCodeSre)
	if err != nil {
		owners = nil
		log.Warnf(ctx, "sync apm service config: list owners failed, app=%s env=%s: %v", app.ID, env.Name, err)
	}
	// 构建请求
	bkMonitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		log.Errorf(
			ctx, "sync apm service config: resolve bk monitor project id failed, app=%s env=%s: %v",
			app.ID, env.Name, err,
		)
		return nil
	}

	return bkmapi.NewUpdateApmServiceConfigReq(
		bkMonitorProjectID, apm.Name, serviceName, owners,
		[]bkmapi.ApmServiceK8sRelation{{
			BcsClusterID: env.Cluster.ClusterID,
			Namespace:    env.Cluster.Namespace,
			Kind:         kind,
			Name:         name,
		}},
	)
}

// resolveApmServiceName 解析应用在当前环境下的 APM 服务名称，失败仅记录日志并返回空串。
func resolveApmServiceName(
	ctx context.Context,
	reg *storereg.Registry,
	app *bkmsapp.Application,
	env *envmodel.Environment,
) string {
	appModel, err := reg.AppModelStore.GetAppModel(ctx, app.ID)
	if err != nil {
		log.Errorf(ctx, "sync apm service config: get app model failed, app=%s env=%s: %v", app.ID, env.Name, err)
		return ""
	}
	cfgProvider := appcfg.NewMountableFileProvider(
		reg.AppConfigFileStore,
		reg.AppConfigFileDefStore,
		reg.AppConfigFileVersionStore,
	)
	frameworkFile, err := cfgProvider.GetFrameworkMountableFile(ctx, app.ID, env.Name)
	if err != nil {
		log.Errorf(
			ctx,
			"sync apm service config: get framework config failed, app=%s env=%s: %v",
			app.ID,
			env.Name,
			err,
		)
		return ""
	}
	appEnvVars, err := envvars.BuildAppEnvVars(
		ctx, app, appModel, env,
		envvars.NewUnifiedEnvVarsReader(reg.ScopedEnvVarStore, reg.AppDepsVarReader, reg.PolarisVarReader),
	)
	if err != nil {
		log.Errorf(ctx, "sync apm service config: build app env vars failed, app=%s env=%s: %v", app.ID, env.Name, err)
		return ""
	}
	serviceName, err := bkmmodel.GetApmServiceName(app.Type, frameworkFile.Content, appEnvVars.ToMap())
	if err != nil {
		if errors.Is(err, bkmmodel.ErrAPMConfigMissing) {
			log.Infof(ctx, "apm config missing, skip sync apm service config, app=%s env=%s", app.ID, env.Name)
			return ""
		}
		log.Errorf(
			ctx, "sync apm service config: resolve apm service name failed, app=%s env=%s: %v",
			app.ID, env.Name, err,
		)
		return ""
	}
	return serviceName
}
