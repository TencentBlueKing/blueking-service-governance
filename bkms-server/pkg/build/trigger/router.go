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

package trigger

import "github.com/gin-gonic/gin"

// Handler contains views required by build trigger Gin routes.
type Handler interface {
	ListBuildTriggerPolicies(c *gin.Context)
	CreateBuildTriggerPolicy(c *gin.Context)
	UpdateBuildTriggerPolicy(c *gin.Context)
	PatchBuildTriggerPolicyStatus(c *gin.Context)
	DeleteBuildTriggerPolicy(c *gin.Context)
	CheckBuildTriggerPolicyConflict(c *gin.Context)
	ListBuildTriggerPolicyRecords(c *gin.Context)
	HandleBuildTriggerPolicyCallback(c *gin.Context)
}

// Register registers Gin build trigger routes that require user authentication.
//
// 回调路由不在此注册，它走应用独享凭证鉴权，需由 RegisterCallback 挂到免用户票据的路由组上
func Register(rg *gin.RouterGroup, h Handler) {
	apps := rg.Group("/apps/:appID")
	apps.GET("/build-trigger-policies", h.ListBuildTriggerPolicies)
	apps.POST("/build-trigger-policies", h.CreateBuildTriggerPolicy)
	apps.POST("/build-trigger-policies/conflict-check", h.CheckBuildTriggerPolicyConflict)
	apps.PUT("/build-trigger-policies/:policyID", h.UpdateBuildTriggerPolicy)
	apps.PATCH("/build-trigger-policies/:policyID/status", h.PatchBuildTriggerPolicyStatus)
	apps.DELETE("/build-trigger-policies/:policyID", h.DeleteBuildTriggerPolicy)
	apps.GET("/build-trigger-policies/:policyID/records", h.ListBuildTriggerPolicyRecords)
}

// RegisterCallback registers the build trigger callback route.
//
// 该路由由蓝盾触发专用流水线调用，携带应用独享凭证而非用户票据，因此必须挂在
// 不带 auth.Required 的路由组上；鉴权由 handler 自行完成
func RegisterCallback(rg *gin.RouterGroup, h Handler) {
	rg.POST("/apps/:appID/build-trigger-policies/callback", h.HandleBuildTriggerPolicyCallback)
}
