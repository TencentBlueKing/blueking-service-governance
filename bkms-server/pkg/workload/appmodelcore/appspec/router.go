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

package appspec

import "github.com/gin-gonic/gin"

// Handler contains views required by AppSpec Gin routes.
type Handler interface {
	GetAppSpecOverview(c *gin.Context)

	GetAppDefaultAppSpecResources(c *gin.Context)
	SetAppDefaultAppSpecResources(c *gin.Context)
	GetEnvAppSpecResources(c *gin.Context)
	GetEnvEffectiveAppSpecResources(c *gin.Context)
	SetEnvAppSpecResources(c *gin.Context)
	DeleteEnvAppSpecResources(c *gin.Context)

	GetAppDefaultAppSpecUpdateStrategy(c *gin.Context)
	SetAppDefaultAppSpecUpdateStrategy(c *gin.Context)
	GetEnvAppSpecUpdateStrategy(c *gin.Context)
	GetEnvEffectiveAppSpecUpdateStrategy(c *gin.Context)
	SetEnvAppSpecUpdateStrategy(c *gin.Context)
	DeleteEnvAppSpecUpdateStrategy(c *gin.Context)

	GetEnvAppSpecDevMode(c *gin.Context)
	GetEnvEffectiveAppSpecDevMode(c *gin.Context)
	SetEnvAppSpecDevMode(c *gin.Context)
	DeleteEnvAppSpecDevMode(c *gin.Context)

	GetAppDefaultAppSpecLifecycle(c *gin.Context)
	SetAppDefaultAppSpecLifecycle(c *gin.Context)
	GetEnvAppSpecLifecycle(c *gin.Context)
	GetEnvEffectiveAppSpecLifecycle(c *gin.Context)
	SetEnvAppSpecLifecycle(c *gin.Context)
	DeleteEnvAppSpecLifecycle(c *gin.Context)

	GetAppDefaultAppSpecProbe(c *gin.Context)
	SetAppDefaultAppSpecProbe(c *gin.Context)
	GetEnvAppSpecProbe(c *gin.Context)
	GetEnvEffectiveAppSpecProbe(c *gin.Context)
	SetEnvAppSpecProbe(c *gin.Context)
	DeleteEnvAppSpecProbe(c *gin.Context)
	DeleteEnvAppSpecProbeByType(c *gin.Context)

	GetAppDefaultAppSpecLabels(c *gin.Context)
	SetAppDefaultAppSpecLabels(c *gin.Context)
	GetEnvAppSpecLabels(c *gin.Context)
	GetEnvEffectiveAppSpecLabels(c *gin.Context)
	SetEnvAppSpecLabels(c *gin.Context)
	DeleteEnvAppSpecLabels(c *gin.Context)

	GetAppDefaultAppSpecAnnotations(c *gin.Context)
	SetAppDefaultAppSpecAnnotations(c *gin.Context)
	GetEnvAppSpecAnnotations(c *gin.Context)
	GetEnvEffectiveAppSpecAnnotations(c *gin.Context)
	SetEnvAppSpecAnnotations(c *gin.Context)
	DeleteEnvAppSpecAnnotations(c *gin.Context)

	GetAppDefaultAppSpecTkeRouteEni(c *gin.Context)
	SetAppDefaultAppSpecTkeRouteEni(c *gin.Context)
	GetEnvAppSpecTkeRouteEni(c *gin.Context)
	GetEnvEffectiveAppSpecTkeRouteEni(c *gin.Context)
	SetEnvAppSpecTkeRouteEni(c *gin.Context)
	DeleteEnvAppSpecTkeRouteEni(c *gin.Context)
}

// Register registers Gin AppSpec routes.
func Register(rg *gin.RouterGroup, h Handler) {
	// Overview 只返回当前应用有哪些环境已经保存过 AppSpec 覆盖配置，不对应某个具体 section。
	rg.GET("/apps/:appID/app-spec/overview", h.GetAppSpecOverview)

	// Resources 是标准 AppSpec section：支持应用默认值、环境原始覆盖值、环境最终生效值，以及环境覆盖删除。
	rg.GET("/apps/:appID/app-spec/default-resources", h.GetAppDefaultAppSpecResources)
	rg.PUT("/apps/:appID/app-spec/default-resources", h.SetAppDefaultAppSpecResources)
	rg.GET("/apps/:appID/envs/:envName/app-spec/resources", h.GetEnvAppSpecResources)
	rg.GET("/apps/:appID/envs/:envName/app-spec/resources/effective", h.GetEnvEffectiveAppSpecResources)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/resources", h.SetEnvAppSpecResources)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/resources", h.DeleteEnvAppSpecResources)

	// UpdateStrategy 是标准 AppSpec section，环境级 API 的空字段表示继承应用默认值。
	rg.GET("/apps/:appID/app-spec/default-update-strategy", h.GetAppDefaultAppSpecUpdateStrategy)
	rg.PUT("/apps/:appID/app-spec/default-update-strategy", h.SetAppDefaultAppSpecUpdateStrategy)
	rg.GET("/apps/:appID/envs/:envName/app-spec/update-strategy", h.GetEnvAppSpecUpdateStrategy)
	rg.GET("/apps/:appID/envs/:envName/app-spec/update-strategy/effective", h.GetEnvEffectiveAppSpecUpdateStrategy)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/update-strategy", h.SetEnvAppSpecUpdateStrategy)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/update-strategy", h.DeleteEnvAppSpecUpdateStrategy)

	// DevMode 是非标准 section：没有应用默认值 API，路径字段由应用类型决定，客户端传入的路径不会被保存。
	rg.GET("/apps/:appID/envs/:envName/app-spec/dev-mode", h.GetEnvAppSpecDevMode)
	rg.GET("/apps/:appID/envs/:envName/app-spec/dev-mode/effective", h.GetEnvEffectiveAppSpecDevMode)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/dev-mode", h.SetEnvAppSpecDevMode)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/dev-mode", h.DeleteEnvAppSpecDevMode)

	// Lifecycle 是标准 AppSpec section，但为了兼容 proto JSON，int64 时长字段在响应中序列化为字符串。
	rg.GET("/apps/:appID/app-spec/default-lifecycle", h.GetAppDefaultAppSpecLifecycle)
	rg.PUT("/apps/:appID/app-spec/default-lifecycle", h.SetAppDefaultAppSpecLifecycle)
	rg.GET("/apps/:appID/envs/:envName/app-spec/lifecycle", h.GetEnvAppSpecLifecycle)
	rg.GET("/apps/:appID/envs/:envName/app-spec/lifecycle/effective", h.GetEnvEffectiveAppSpecLifecycle)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/lifecycle", h.SetEnvAppSpecLifecycle)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/lifecycle", h.DeleteEnvAppSpecLifecycle)

	// Probe 整体读写流程接近标准 section，但额外支持按 liveness/readiness/startup 删除单个探针。
	rg.GET("/apps/:appID/app-spec/default-probe", h.GetAppDefaultAppSpecProbe)
	rg.PUT("/apps/:appID/app-spec/default-probe", h.SetAppDefaultAppSpecProbe)
	rg.GET("/apps/:appID/envs/:envName/app-spec/probe", h.GetEnvAppSpecProbe)
	rg.GET("/apps/:appID/envs/:envName/app-spec/probe/effective", h.GetEnvEffectiveAppSpecProbe)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/probe", h.SetEnvAppSpecProbe)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/probe", h.DeleteEnvAppSpecProbe)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/probe/:probeType", h.DeleteEnvAppSpecProbeByType)

	// Labels 是应用部署到 Kubernetes 后附加在资源上的自定义 label。
	// 支持应用默认值与环境覆盖值，相同 key 时环境级覆盖默认级。
	rg.GET("/apps/:appID/app-spec/default-labels", h.GetAppDefaultAppSpecLabels)
	rg.PUT("/apps/:appID/app-spec/default-labels", h.SetAppDefaultAppSpecLabels)
	rg.GET("/apps/:appID/envs/:envName/app-spec/labels", h.GetEnvAppSpecLabels)
	rg.GET("/apps/:appID/envs/:envName/app-spec/labels/effective", h.GetEnvEffectiveAppSpecLabels)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/labels", h.SetEnvAppSpecLabels)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/labels", h.DeleteEnvAppSpecLabels)

	// Annotations 与 Labels 同构，区别是 value 不限制格式与长度；
	// 但 key/value 去除首尾空格后都不能为空白。
	rg.GET("/apps/:appID/app-spec/default-annotations", h.GetAppDefaultAppSpecAnnotations)
	rg.PUT("/apps/:appID/app-spec/default-annotations", h.SetAppDefaultAppSpecAnnotations)
	rg.GET("/apps/:appID/envs/:envName/app-spec/annotations", h.GetEnvAppSpecAnnotations)
	rg.GET("/apps/:appID/envs/:envName/app-spec/annotations/effective", h.GetEnvEffectiveAppSpecAnnotations)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/annotations", h.SetEnvAppSpecAnnotations)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/annotations", h.DeleteEnvAppSpecAnnotations)

	// TkeRouteEni 控制应用是否启用 TKE Route ENI (VPC-CNI) 网络模式。
	rg.GET("/apps/:appID/app-spec/default-tke-route-eni", h.GetAppDefaultAppSpecTkeRouteEni)
	rg.PUT("/apps/:appID/app-spec/default-tke-route-eni", h.SetAppDefaultAppSpecTkeRouteEni)
	rg.GET("/apps/:appID/envs/:envName/app-spec/tke-route-eni", h.GetEnvAppSpecTkeRouteEni)
	rg.GET("/apps/:appID/envs/:envName/app-spec/tke-route-eni/effective", h.GetEnvEffectiveAppSpecTkeRouteEni)
	rg.PUT("/apps/:appID/envs/:envName/app-spec/tke-route-eni", h.SetEnvAppSpecTkeRouteEni)
	rg.DELETE("/apps/:appID/envs/:envName/app-spec/tke-route-eni", h.DeleteEnvAppSpecTkeRouteEni)
}
