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
	"sync"

	"go.mongodb.org/mongo-driver/v2/mongo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	appcfghooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/hooks"
	apphooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/hooks"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	releasestrategyhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy/hooks"
	polarisenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/envvars"
	depenvvars "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/envvars"
	depinit "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model/initdata"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	alertstrategyhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy/hooks"
	appdefaultshooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/hooks"
	envvarhooks "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/hooks"
)

// Registry 是 web 或 worker 进程入口持有的 Store 注册中心。
// 它只负责在数据库客户端完成初始化后暴露数据库句柄与少量环境变量读取器，便于
// 进程层面组装 handler、task 或 service。Registry 不持有业务 Service。
//
// 除 handler（application/controller）层外，各领域核心层不应该直接依赖 Registry
// 或 GlobalRegistry，而应该通过构造函数或函数参数显式接收 *mongo.Database，
// 再用 store.Of(db, Meta) 获取所需模型的 Store。
type Registry struct {
	// DB 供基于 GenericStore 的模型使用：通过 store.Of(DB, Meta) 获取对应模型的 Store
	DB *mongo.Database
	// AppDepsVarReader 基于 ServiceInstance 产出环境变量的读取器,
	// 用于注入到 envvars.UnifiedEnvVarsReader, 接入依赖服务实例变量。
	AppDepsVarReader *depenvvars.Reader
	// PolarisVarReader 基于 PolarisConfig 产出环境变量的读取器,
	// 用于注入到 envvars.UnifiedEnvVarsReader, 接入北极星配置变量。
	PolarisVarReader *polarisenvvars.Reader
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
// IMPORTANT: 它只适合在项目上层如 handler、task 层使用；领域核心层应显式接收
// *mongo.Database 并用 store.Of(db, Meta) 获取 Store，而不是直接访问全局 Registry。
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
	r.AppDepsVarReader = depenvvars.NewReader(r.DB)
	r.PolarisVarReader = polarisenvvars.NewReader(r.DB)
}

// registerStoreHooks 注册钩子函数。
// 钩子函数主要指那些当由特定动作触发时需要调用的回调函数。
func (r *Registry) registerStoreHooks() {
	envvarhooks.RegisterDeleteHooks(r.DB)
	apphooks.RegisterDeleteHooks(r.DB)
	appcfghooks.RegisterDeleteHooks(r.DB)
	alertstrategyhooks.RegisterUpdateHooks(r.DB)
	appdefaultshooks.RegisterPreDeleteHooks(r.DB)
	releasestrategyhooks.RegisterPreDeleteHooks(r.DB)
}

// initStoreData 初始化 Store 中的基础数据，例如依赖服务的数据。
func (r *Registry) initStoreData() {
	if err := depinit.Do(r.DB); err != nil {
		log.Fatalf("failed to initialize dep service data: %v", err)
	}
}
