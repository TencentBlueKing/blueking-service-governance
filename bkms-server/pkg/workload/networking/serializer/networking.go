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

// Package serializer defines Gin input and output serializers for networking APIs.
package serializer

import (
	"regexp"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking"
)

var (
	// appIDPattern 校验规则
	appIDPattern = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")

	// svcNamePattern 校验规则
	svcNamePattern = regexp.MustCompile("^[a-z]([-a-z0-9]*[a-z0-9])?$")

	// workspaceIDPattern 校验规则
	workspaceIDPattern = regexp.MustCompile("^[a-z]([a-z0-9-]*[a-z0-9])?$")
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("app_id", validateAppID); err != nil {
			panic("failed to register app_id validator: " + err.Error())
		}
		if err := v.RegisterValidation("svc_name", validateSvcName); err != nil {
			panic("failed to register svc_name validator: " + err.Error())
		}
		if err := v.RegisterValidation("workspace_id", validateWorkspaceID); err != nil {
			panic("failed to register workspace_id validator: " + err.Error())
		}
	}
}

// validateAppID 校验器
func validateAppID(fl validator.FieldLevel) bool {
	return appIDPattern.MatchString(fl.Field().String())
}

// validateSvcName 校验器
func validateSvcName(fl validator.FieldLevel) bool {
	return svcNamePattern.MatchString(fl.Field().String())
}

// validateWorkspaceID 校验器
func validateWorkspaceID(fl validator.FieldLevel) bool {
	return workspaceIDPattern.MatchString(fl.Field().String())
}

// --- URI 参数结构体 ---

// AppURIInput 绑定 /apps/:appID 路径参数
type AppURIInput struct {
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// AppServiceURIInput 绑定 /apps/:appID/services/:name 路径参数
type AppServiceURIInput struct {
	AppID string `uri:"appID" binding:"required,uri_slug"`
	Name  string `uri:"name" binding:"required,min=1,max=63,svc_name"`
}

// WorkspaceURIInput 绑定 /workspaces/:workspaceID 路径参数
type WorkspaceURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,min=1,max=27,workspace_id"`
}

// --- Input 结构体 ---

// ServicePortInput 服务端口配置输入
type ServicePortInput struct {
	Name       string `json:"name" binding:"required,min=1,max=63,svc_name"`
	Port       int32  `json:"port"`
	Protocol   string `json:"protocol"`
	TargetPort string `json:"targetPort"`
}

// CreateAppServiceInput 创建应用 Service 的请求体
type CreateAppServiceInput struct {
	Name               string             `json:"name" binding:"required,min=1,max=63,svc_name"`
	Selector           map[string]string  `json:"selector"`
	Ports              []ServicePortInput `json:"ports"`
	TrafficLaneEnabled *bool              `json:"trafficLaneEnabled"`
}

// UpdateAppServiceInput 更新应用 Service 的请求体
type UpdateAppServiceInput struct {
	Selector           map[string]string  `json:"selector"`
	Ports              []ServicePortInput `json:"ports"`
	TrafficLaneEnabled *bool              `json:"trafficLaneEnabled"`
}

// --- Output 结构体 ---

// ServicePortOutput 服务端口配置输出
type ServicePortOutput struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	Protocol   string `json:"protocol"`
	TargetPort string `json:"targetPort"`
}

// AppServiceOutput 应用 Service 输出
type AppServiceOutput struct {
	Name               string              `json:"name"`
	Selector           map[string]string   `json:"selector"`
	Ports              []ServicePortOutput `json:"ports"`
	TrafficLaneEnabled bool                `json:"trafficLaneEnabled"`
	CreatedAt          *time.Time          `json:"createdAt"`
	UpdatedAt          *time.Time          `json:"updatedAt"`
}

// FromModel 从领域模型填充输出字段
func (o *AppServiceOutput) FromModel(svc networking.Service) *AppServiceOutput {
	if o == nil {
		return nil
	}
	ports := make([]ServicePortOutput, 0, len(svc.Ports))
	for _, p := range svc.Ports {
		ports = append(ports, ServicePortOutput{
			Name:       p.Name,
			Port:       p.Port,
			Protocol:   string(p.Protocol),
			TargetPort: p.TargetPort,
		})
	}

	*o = AppServiceOutput{
		Name:               svc.Name,
		Selector:           svc.Selector,
		Ports:              ports,
		TrafficLaneEnabled: svc.TrafficLaneEnabled,
	}
	if !svc.CreatedAt.IsZero() {
		o.CreatedAt = &svc.CreatedAt
	}
	if !svc.UpdatedAt.IsZero() {
		o.UpdatedAt = &svc.UpdatedAt
	}
	return o
}

// ListAppServicesOutput 获取应用 Services 列表的响应
type ListAppServicesOutput struct {
	Data []*AppServiceOutput `json:"data"`
}

// CandidateAppServiceOutput 候选应用 Service 输出
type CandidateAppServiceOutput struct {
	Name               string `json:"name"`
	TrafficLaneEnabled bool   `json:"trafficLaneEnabled"`
}

// TrafficLaneCandidateAppOutput 泳道候选应用输出
type TrafficLaneCandidateAppOutput struct {
	AppName  string                      `json:"appName"`
	Services []CandidateAppServiceOutput `json:"services"`
}

// ListTrafficLaneCandidateAppsOutput 查询泳道候选应用列表的响应
type ListTrafficLaneCandidateAppsOutput struct {
	Data []TrafficLaneCandidateAppOutput `json:"data"`
}

// -----------------------------------------------------------------------------
// Empty output
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}
