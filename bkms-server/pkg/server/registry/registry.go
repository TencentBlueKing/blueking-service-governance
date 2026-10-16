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

package storereg

import (
	"context"
	"reflect"
	"sync"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	helmchartbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/semver"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	appcfghooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/hooks"
	apphooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/hooks"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	depinit "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model/initdata"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
	alertstrategyhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy/hooks"
	appdefaultshooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/hooks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	scopedenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvarhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/hooks"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
)

// Registry 是 web 或 worker 进程入口持有的 Store 注册中心。
// 它只负责在数据库客户端完成初始化后创建并暴露所有数据库访问 Store，便于
// 进程层面组装 handler、task 或 service。Registry 不持有业务 Service。
//
// 除 handler（application/controller）层外，各领域核心层不应该直接依赖 Registry
// 或 GlobalRegistry，而应该通过构造函数显式依赖自身需要的 Store 接口。
type Registry struct {
	// DB 供基于 GenericStore 的模型使用：通过 store.Of(DB, Meta) 获取对应模型的 Store
	DB *mongo.Database
	// 工作空间类
	WorkspaceStore      workspace.WorkspaceStore
	WorkspaceCompsStore workspace.WorkspaceCompsStore
	// 环境类
	EnvStore          envmodel.EnvironmentStore
	ScopedEnvVarStore scopedenvvars.ScopedEnvVarStore
	// 应用配置类
	ComponentDefStore         component.ComponentDefStore
	AppStore                  bkmsapp.ApplicationStore
	AppModelStore             appmodel.AppModelStore
	AppSpecStore              appspec.AppSpecStore
	AppConfigFileStore        appcfg.AppConfigFileStore
	AppConfigFileVersionStore appcfg.AppConfigFileVersionStore
	PolarisConfigStore        polaris.PolarisConfigStore
	HostPortStore             hostport.HostPortStore
	// 构建 & 部署类
	BuildRecordStore                    build.RecordStore
	BuildAutoDeployRecordStore          autodeploy.RecordStore
	HelmDeployRecordStore               helmdeploy.RecordStore
	AppModelDeployRecordStore           appmodeldeploy.RecordStore
	AppModelDeployResourceSnapshotStore appmodeldeploy.ResourceSnapshotStore
	// 开发模式类
	PublishRecordStore devmode.PublishRecordStore
	// Helm Chart 构建类
	HelmChartBuildRecordStore   helmchartbuild.RecordStore
	HelmChartSemverCounterStore semver.CounterStore
	// 服务类
	DepSvcInstStore depsvcmodel.ServiceInstanceStore
	// AppDepsVarReader 基于 ServiceInstance 产出环境变量的读取器,
	// 用于注入到 envvars.UnifiedEnvVarsReader, 接入依赖服务实例变量。
	AppDepsVarReader *depenvvars.Reader
	// PolarisVarReader 基于 PolarisConfig 产出环境变量的读取器,
	// 用于注入到 envvars.UnifiedEnvVarsReader, 接入北极星配置变量。
	PolarisVarReader *polarisenvvars.Reader
	// 蓝鲸监控类
	ApmInstConfigStore bkmmodel.ApmInstConfigStore
	AlertStrategyStore alertstrategy.Store
	// 操作审计类
	OperationRecordStore audit.OperationRecordStore
	// 镜像快照类
	CustomRuntimeImageStore customruntime.Store
	SnapshotStore           snapshot.SnapshotStore
	// 拓扑类
	ResourceSnapshotStore topology.ResourceSnapshotStore
}

var (
	globalMu       sync.Mutex
	globalInitOnce sync.Once
	// GlobalRegistry 全局 registry
	GlobalRegistry *Registry
)

// Init 初始化进程内可复用的 Store 注册中心。
func Init(ctx context.Context) {
	globalInitOnce.Do(func() {
		reg := &Registry{}
		reg.initStores(database.Client(), database.Name())
		reg.registerStoreHooks()
		reg.initStoreData()
		GlobalRegistry = reg
		log.Info(ctx, "stores initialized successfully")
	})
}

// G 返回进程内全局 Store 注册中心。
//
// IMPORTANT: 它只适合在项目上层如 handler、task 层使用；领域核心层应直接依赖
// 明确的 Store 接口，而不是直接访问全局 Registry。
func G() *Registry {
	if GlobalRegistry == nil {
		panic("store registry is not initialized")
	}

	return GlobalRegistry
}

// Reset 重置当前的 GlobalRegistry，主要用于测试场景，确保用例执行完后状态重置。
func Reset() {
	globalMu.Lock()
	defer globalMu.Unlock()

	GlobalRegistry = nil
	globalInitOnce = sync.Once{}
	bkmsenv.ResetHooksForTest()
	workspace.ResetLifecycleHooksForTest()
}

func (r *Registry) initStores(mongoClient *mongo.Client, dbName string) {
	r.DB = mongoClient.Database(dbName)

	// 组件定义
	r.ComponentDefStore = mustInit(component.NewComponentDefStoreMongo(mongoClient, dbName))
	// 工作空间类
	r.WorkspaceStore = mustInit(workspace.NewWorkspaceStoreMongo(mongoClient, dbName))
	r.WorkspaceCompsStore = mustInit(workspace.NewWorkspaceCompsStoreMongo(mongoClient, dbName))
	// 环境类
	r.EnvStore = mustInit(envmodel.NewEnvironmentStoreMongo(mongoClient, dbName))
	r.ScopedEnvVarStore = mustInit(scopedenvvars.NewScopedEnvVarStoreMongo(mongoClient, dbName))
	// 应用配置类
	r.AppStore = mustInit(bkmsapp.NewApplicationStoreMongo(mongoClient, dbName))
	r.AppModelStore = mustInit(appmodel.NewAppModelStoreMongo(mongoClient, dbName))
	r.AppSpecStore = mustInit(appspec.NewAppSpecStoreMongo(mongoClient, dbName))
	r.AppConfigFileStore = mustInit(appcfg.NewAppConfigFileStoreMongo(mongoClient, dbName))
	r.AppConfigFileVersionStore = mustInit(appcfg.NewAppConfigFileVersionStoreMongo(mongoClient, dbName))
	r.PolarisConfigStore = mustInit(polaris.NewPolarisConfigStoreMongo(mongoClient, dbName))
	r.HostPortStore = mustInit(hostport.NewHostPortStoreMongo(mongoClient, dbName))
	// 构建 & 部署类
	r.BuildRecordStore = mustInit(build.NewRecordStoreMongo(mongoClient, dbName))
	r.BuildAutoDeployRecordStore = mustInit(autodeploy.NewRecordStoreMongo(mongoClient, dbName))
	r.AppModelDeployRecordStore = mustInit(appmodeldeploy.NewRecordStoreMongo(mongoClient, dbName))
	r.AppModelDeployResourceSnapshotStore = mustInit(appmodeldeploy.NewResourceSnapshotStoreMongo(mongoClient, dbName))
	r.PublishRecordStore = mustInit(devmode.NewPublishRecordStoreMongo(mongoClient, dbName))
	r.HelmDeployRecordStore = mustInit(helmdeploy.NewRecordStoreMongo(mongoClient, dbName))
	r.HelmChartBuildRecordStore = mustInit(helmchartbuild.NewRecordStoreMongo(mongoClient, dbName))
	r.HelmChartSemverCounterStore = mustInit(semver.NewCounterStoreMongo(mongoClient, dbName))
	// 服务类
	r.DepSvcInstStore = mustInit(depsvcmodel.NewServiceInstanceStoreMongo(mongoClient, dbName))
	r.AppDepsVarReader = depenvvars.NewReader(r.DepSvcInstStore, r.DB)
	r.PolarisVarReader = polarisenvvars.NewReader(r.PolarisConfigStore)
	// 蓝鲸监控类
	r.ApmInstConfigStore = mustInit(bkmmodel.NewApmInstConfigStoreMongo(mongoClient, dbName))
	r.AlertStrategyStore = mustInit(alertstrategy.NewStoreMongo(mongoClient, dbName))
	// 操作审计类
	r.OperationRecordStore = mustInit(audit.NewOperationRecordStoreMongo(mongoClient, dbName))
	// 镜像快照类
	r.CustomRuntimeImageStore = mustInit(customruntime.NewStoreMongo(mongoClient, dbName))
	r.SnapshotStore = mustInit(snapshot.NewSnapshotStoreMongo(mongoClient, dbName))
	// 拓扑类
	r.ResourceSnapshotStore = mustInit(topology.NewResourceSnapshotStoreMongo(mongoClient, dbName))
}

// registerStoreHooks 注册钩子函数。
// 钩子函数主要指那些当由特定动作触发时需要调用的回调函数。
func (r *Registry) registerStoreHooks() {
	r.WorkspaceCompsStore.SetComponentHooks(workspace.NewComponentRefCountHooks(r.ComponentDefStore))
	r.AppModelStore.SetComponentHooks(appmodel.NewComponentRefCountHooks(r.ComponentDefStore))
	envvarhooks.RegisterDeleteHooks(r.ScopedEnvVarStore)
	apphooks.RegisterDeleteHooks(r.AppStore)
	appcfghooks.RegisterDeleteHooks(
		r.AppStore,
		r.AppConfigFileStore,
		r.DB,
		r.AppConfigFileVersionStore,
	)
	alertstrategyhooks.RegisterUpdateHooks(
		r.WorkspaceStore,
		r.AlertStrategyStore,
		r.EnvStore,
		r.AppStore,
		r.ResourceSnapshotStore,
	)
	appdefaultshooks.RegisterPreDeleteHooks(r.DB)
}

// initStoreData 初始化 Store 中的基础数据，例如依赖服务的数据。
func (r *Registry) initStoreData() {
	if err := depinit.Do(r.DB); err != nil {
		log.Fatalf("failed to initialize dep service data: %v", err)
	}
}

func mustInit[T any](store T, err error) T {
	if err != nil {
		log.Fatalf("failed to initialize %s: %v", typeNameOf(store), err)
	}
	return store
}

func typeNameOf(value any) string {
	typ := reflect.TypeOf(value)
	if typ == nil {
		return "<nil>"
	}
	return typ.String()
}
