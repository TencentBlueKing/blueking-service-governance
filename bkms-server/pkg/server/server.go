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

// Package server 包主要包含 bkms 服务的主 API Server 服务相关的内容，包含主 Gin 路由创建、Gin 工具函数，等等
package server

import (
	"context"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/account/user"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations"
	bkintegrationshandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	autodeployhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build"
	buildhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/handler"
	helmchart "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	helmcharthandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/handler"
	buildtrigger "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger"
	buildtriggerhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/trigger/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	appcfghandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/handler"
	apphandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	clusteraddonhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon/handler"
	envhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	workspacehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	deployhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa"
	gpahandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/gpa/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarishandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg"
	bscpcfghandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/bscpcfg/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	componenthandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/portpool"
	portpoolhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/portpool/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice"
	depservicehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/usertoken"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	audithandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm"
	bkmalert "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert"
	bkmalerthandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/handler"
	bkmusergroup "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup"
	bkmusergrouphandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog"
	instanceloghandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/admin"
	adminhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/admin/handler"
	platmgtworkspace "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace"
	platworkspaceadmin "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/admin"
	platworkspaceadminhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/admin/handler"
	platmgtworkspacehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/platmgt/workspace/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/basic"
	basichandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/basic/handler"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults"
	appdefaultshandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appdefaults/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	appspechandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvarhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement"
	arrangementhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/arrangement/handler"
	imageapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image"
	imagehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance"
	instancehandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking"
	networkinghandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology"
	topologyhandler "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/topology/handler"
)

// RegisterRouter 创建 Gin router，并注册 bkms-server 的所有 HTTP 路由
//
// serverRole 用于区分调用方进程角色（如 "webserver"），供 APM Middleware 生成默认服务名使用
func RegisterRouter(ctx context.Context, cfg config.Config, serverRole string) *gin.Engine {
	gin.SetMode(cfg.HTTPServer.Mode)
	r := gin.New()
	r.Use(
		apm.OTelMiddleware(cfg.BkMonitor, serverRole),
		apm.HTTPLogMiddleware(),
		gin.Recovery(),
		apm.ResponseTraceMiddleware(),
		metrics.Middleware(),
	)

	// Enable the Swagger UI if configured to do so.
	if cfg.HTTPServer.EnableSwaggerPath {
		log.Info(ctx, "enabling Swagger UI at /swagger/*")
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}
	// Health and version endpoints are available at the root without authentication.
	basic.Register(r.Group(""), basichandler.New(storereg.G()))

	// 初始化用户账号 account 相关 API
	// 这批 API 不要求请求必须携带有效身份信息
	tokenClient := usertoken.NewAPIGatewayTokenClient(cfg.Account.AuthBaseURL, cfg.BkApp.Code, cfg.BkApp.Secret)
	accountHandler := account.NewHandler(account.Config{
		AuthEnvName: cfg.Account.AuthEnvName,
		LoginURL:    cfg.Account.LoginURL,
	}, tokenClient)
	authConfig := auth.Config{
		BackendType:          cfg.Account.BackendType,
		LoginURL:             cfg.Account.LoginURL,
		AllowSetUserInHeader: cfg.Development.AllowSetUserInHeader,
	}
	account.Register(r.Group(""), accountHandler, auth.Optional(authConfig, tokenClient))

	// 构建触发回调由蓝盾触发专用流水线调用，携带应用独享凭证而非用户票据，
	// 因此单独挂在不带 auth.Required 的路由组上。
	// FIXME W1 仅做 header 存在性检查；凭证内容比对与限流由后续子需求实现，合入生产前勿依赖本路由做真实鉴权
	buildTriggerHandler := buildtriggerhandler.New()
	callbackGroup := r.Group("/bkms/v1/bkms-server")
	callbackGroup.Use(bkerrs.ErrorHandler())
	buildtrigger.RegisterCallback(callbackGroup, buildTriggerHandler)

	// Register authenticated business APIs under /v1.
	// 以下 Group 的所有 API 均要求请求必须携带有效身份信息
	v1 := r.Group("/bkms/v1/bkms-server")
	v1.Use(bkerrs.ErrorHandler(), auth.Required(authConfig, tokenClient))
	app.Register(v1, apphandler.New(storereg.G()))
	deploy.Register(v1, deployhandler.New(storereg.G()))
	workspace.Register(v1, workspacehandler.New(storereg.G()))
	component.Register(v1, componenthandler.New(storereg.G()))
	networking.Register(v1, networkinghandler.New(storereg.G()))
	autodeploy.Register(v1, autodeployhandler.New(storereg.G()))
	build.Register(v1, buildhandler.New(storereg.G()))
	buildtrigger.Register(v1, buildTriggerHandler)
	imageapi.Register(v1, imagehandler.New(storereg.G()))
	appcfg.Register(v1, appcfghandler.New(storereg.G()))
	helmchart.Register(v1, helmcharthandler.New(storereg.G()))
	instancelog.Register(v1, instanceloghandler.New(storereg.G()))
	appdefaults.Register(v1, appdefaultshandler.New(storereg.G()))
	appspec.Register(v1, appspechandler.New(storereg.G()))
	arrangement.Register(v1, arrangementhandler.New(storereg.G()))
	instance.Register(v1, instancehandler.New(storereg.G()))
	envvars.Register(v1, envvarhandler.New(storereg.G()))
	topology.Register(v1, topologyhandler.New(storereg.G()))
	env.Register(v1, envhandler.New(storereg.G()))
	audit.Register(v1, audithandler.New(storereg.G()))
	bkintegrations.Register(v1, bkintegrationshandler.New(storereg.G()))
	bkmusergroup.Register(v1, bkmusergrouphandler.New(storereg.G(), bkmusergroup.New()))
	bkmalert.Register(v1, bkmalerthandler.New(storereg.G()))
	clusteraddon.Register(v1, clusteraddonhandler.New(storereg.G()))
	polaris.Register(v1, polarishandler.New(storereg.G()))
	depservice.Register(v1, depservicehandler.New(storereg.G()))
	gpa.Register(v1, gpahandler.New(storereg.G()))
	portpool.Register(v1, portpoolhandler.New(storereg.G()))
	bscpcfg.Register(v1, bscpcfghandler.New(storereg.G()))
	user.Register(v1, user.New(storereg.G()))
	admin.Register(
		v1,
		adminhandler.New(storereg.G().PlatAdminStore),
		platmgt.RequirePlatformRole(storereg.G().PlatAdminStore, admin.RoleCodeAdmin),
	)
	platmgtworkspace.Register(
		v1,
		platmgtworkspacehandler.New(storereg.G()),
		platmgt.RequirePlatformRole(storereg.G().PlatAdminStore, admin.RoleCodeAdmin),
	)
	platworkspaceadmin.Register(
		v1,
		platworkspaceadminhandler.New(storereg.G()),
		platmgt.RequirePlatformRole(storereg.G().PlatAdminStore, admin.RoleCodeAdmin),
	)

	return r
}
