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

// Package serializer defines Gin input and output serializers for app APIs.
package serializer

import (
	"time"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// -----------------------------------------------------------------------------
// Path inputs
//
// These serializers only bind Gin URI parameters. Resource existence and
// permission checks stay in handler / perm helpers because they need registry and
// request context.
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// WorkspaceURIInput is the path input for APIs scoped by workspace.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,uri_slug"`
}

// -----------------------------------------------------------------------------
// CreateApp
// -----------------------------------------------------------------------------

// CreateAppInput is the JSON body for creating an application.
type CreateAppInput struct {
	// 应用名称. 长度范围 1~63 个字符，以小写字母开头，可包含小写字母、数字、中划线，不能以中划线结尾
	Name string `json:"name" binding:"required,app_id"`
	// 应用 ID. 应用在全局范围内的唯一标识，默认情况下基于 name 自动生成，在 name 后追加后缀。
	// 长度范围 1~63 个字符，以小写字母开头，可包含小写字母、数字、中划线，不能以中划线结尾
	ID string `json:"id" binding:"required,app_id"`
	// 应用类型
	Type string `json:"type" binding:"required,oneof=trpc taf helm agones"`
	// 构建配置
	BuildConfig *BuildConfigInput `json:"buildConfig" binding:"required"`
	// Helm 应用描述规范（type 为 helm/agones 时需要）
	HelmSpec *HelmSpecInput `json:"helmSpec"`
	// 应用模型规范（type 为 trpc/taf 时需要）
	AppModelSpec *AppModelSpecInput `json:"appModelSpec"`
}

// CreateAppOutput is the response for creating an application.
type CreateAppOutput struct {
	Data *AppOutputObj `json:"data"`
}

// AppOutputObj is the basic app output after creation.
type AppOutputObj struct {
	// 应用 ID
	ID string `json:"id"`
	// 工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 应用名称
	Name string `json:"name"`
	// 应用类型
	Type string `json:"type"`
	// 应用显示名称
	DisplayName string `json:"displayName"`
}

// FromModel fills output fields from an Application model.
func (o *AppOutputObj) FromModel(app *bkmsapp.Application) *AppOutputObj {
	*o = AppOutputObj{
		ID:          app.ID,
		WorkspaceID: app.WorkspaceID,
		Name:        app.Name,
		Type:        app.Type,
		DisplayName: app.DisplayName,
	}
	return o
}

// -----------------------------------------------------------------------------
// GetAppIDAutoSuffix
// -----------------------------------------------------------------------------

// GetAppIDAutoSuffixOutput is the response for getting app ID auto suffix.
type GetAppIDAutoSuffixOutput struct {
	// 后缀字符串
	Suffix string `json:"suffix"`
}

// -----------------------------------------------------------------------------
// GetApp
// -----------------------------------------------------------------------------

// GetAppOutput is the response for getting app detail.
type GetAppOutput struct {
	Data *AppDetailOutputObj `json:"data"`
}

// AppDetailOutputObj is the detailed app output.
type AppDetailOutputObj struct {
	// 应用 ID
	ID string `json:"id"`
	// 工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 应用名称
	Name string `json:"name"`
	// 应用类型
	Type string `json:"type"`
	// 应用显示名称
	DisplayName string `json:"displayName"`
	// 创建人
	Creator string `json:"creator"`
	// 构建配置
	BuildConfig *BuildConfigOutputObj `json:"buildConfig,omitempty"`
	// Helm 应用描述规范
	HelmSpec *HelmSpecOutputObj `json:"helmSpec,omitempty"`
	// 应用模型规范
	AppModelSpec *AppModelSpecOutputObj `json:"appModelSpec,omitempty"`
}

// FromModel fills output fields from app detail models.
func (o *AppDetailOutputObj) FromModel(
	app *bkmsapp.Application,
	buildConfig *build.Config,
	appModel *appmodel.AppModel,
	components []ComponentOutputObj,
) *AppDetailOutputObj {
	*o = AppDetailOutputObj{
		ID:          app.ID,
		WorkspaceID: app.WorkspaceID,
		Name:        app.Name,
		Type:        app.Type,
		DisplayName: app.DisplayName,
		Creator:     app.Creator,
		BuildConfig: new(BuildConfigOutputObj).FromModel(buildConfig),
		HelmSpec:    new(HelmSpecOutputObj).FromModel(app.HelmSpec),
	}
	if appModel != nil {
		o.AppModelSpec = new(AppModelSpecOutputObj).FromModel(appModel, components)
	}
	return o
}

// -----------------------------------------------------------------------------
// AppModelSpec (shared between trpc/taf)
// -----------------------------------------------------------------------------

// AppModelSpecInput is the app model spec input (for trpc/taf apps).
type AppModelSpecInput struct {
	// 容器启动命令
	Command []string `json:"command"`
	// 容器启动参数
	Args []string `json:"args"`
	// 容器环境变量，仅创建应用时生效；更新 tRPC/TAF Spec 时为兼容旧客户端接收但忽略
	EnvVars []VariableInput `json:"envVars"`
	// tRPC 框架配置
	TrpcSpec *TrpcSpecInput `json:"trpcSpec"`
	// TAF 框架配置
	TafSpec *TafSpecInput `json:"tafSpec"`
}

// VariableInput is the env var input.
type VariableInput struct {
	// 变量名
	Key string `json:"key" binding:"required,envvar_key"`
	// 变量值
	Value string `json:"value"`
	// 变量描述
	Description string `json:"description"`
	// 是否敏感
	IsSensitive bool `json:"isSensitive"`
}

// AppModelSpecOutputObj is the app model spec output.
type AppModelSpecOutputObj struct {
	// 容器启动命令
	Command []string `json:"command"`
	// 容器启动参数
	Args []string `json:"args"`
	// 容器环境变量
	EnvVars []VariableOutputObj `json:"envVars"`
	// 应用组件
	Components []ComponentOutputObj `json:"components"`
	// tRPC 框架配置
	TrpcSpec *TrpcSpecOutputObj `json:"trpcSpec,omitempty"`
	// TAF 框架配置
	TafSpec *TafSpecOutputObj `json:"tafSpec,omitempty"`
}

// VariableOutputObj is the env var output.
type VariableOutputObj struct {
	// 变量名
	Key string `json:"key"`
	// 变量值
	Value string `json:"value"`
	// 变量描述
	Description string `json:"description"`
	// 是否敏感
	IsSensitive bool `json:"isSensitive"`
}

// ComponentOutputObj is the component output.
type ComponentOutputObj struct {
	// 组件名称
	Name string `json:"name"`
	// 组件类型
	Type string `json:"type"`
	// 组件版本
	Version string `json:"version"`
	// 组件属性
	Properties map[string]any `json:"properties"`
	// 引用的空间组件名
	RefWorkspaceCompName string `json:"refWorkspaceCompName,omitempty"`
	// 组件生效范围类型
	ScopeType string `json:"scopeType"`
	// 组件生效的环境列表
	ScopeEnvNames []string `json:"scopeEnvNames"`
}

// FromModel fills output fields from appmodel data.
func (o *AppModelSpecOutputObj) FromModel(
	am *appmodel.AppModel,
	components []ComponentOutputObj,
) *AppModelSpecOutputObj {
	if am == nil {
		return nil
	}
	*o = AppModelSpecOutputObj{
		Command: emptySliceIfNil(am.Workload.Command),
		Args:    emptySliceIfNil(am.Workload.Args),
	}
	o.Components = normalizeComponentOutputs(components)
	o.EnvVars = make([]VariableOutputObj, 0, len(am.Workload.EnvVars))
	for _, v := range am.Workload.EnvVars {
		value := v.Value
		if v.IsSensitive {
			value = envvartypes.SensitiveValueMask
		}
		o.EnvVars = append(o.EnvVars, VariableOutputObj{
			Key:         v.Key,
			Value:       value,
			Description: v.Description,
			IsSensitive: v.IsSensitive,
		})
	}
	if am.Workload.TrpcConfig.FileName != "" {
		o.TrpcSpec = &TrpcSpecOutputObj{
			Language:    am.Workload.TrpcConfig.Language,
			FileName:    am.Workload.TrpcConfig.FileName,
			FilePath:    am.Workload.TrpcConfig.FilePath,
			FileContent: am.Workload.TrpcConfig.FileContent,
		}
	}
	if am.Workload.TafConfig.FileName != "" {
		o.TafSpec = &TafSpecOutputObj{
			FileName:    am.Workload.TafConfig.FileName,
			FilePath:    am.Workload.TafConfig.FilePath,
			FileContent: am.Workload.TafConfig.FileContent,
		}
	}
	return o
}

func normalizeComponentOutputs(components []ComponentOutputObj) []ComponentOutputObj {
	output := make([]ComponentOutputObj, 0, len(components))
	for _, comp := range components {
		// ScopeEnvNames 可能为 nil，导致 JSON 输出为 null，不符合预期，需转换为空数组。
		comp.ScopeEnvNames = emptySliceIfNil(comp.ScopeEnvNames)
		output = append(output, comp)
	}
	return output
}

// -----------------------------------------------------------------------------
// ListApps
// -----------------------------------------------------------------------------

// ListAppsQueryInput is the query input for listing apps in a workspace.
// 单个 Workspace 下的应用列表较少, 且 AppInfo 不会返回太多内容，因此不设置分页。
// [bkms-cli 使用] 避免破坏性修改。
type ListAppsQueryInput struct {
	// 应用名, 选填, 传入时根据 appName 进行过滤
	AppName string `form:"appName"`
}

// ListAppsOutput is the response for listing apps.
type ListAppsOutput struct {
	Data []*AppInfoOutputObj `json:"data"`
}

// AppInfoOutputObj is the app info output for list APIs.
type AppInfoOutputObj struct {
	// 应用 ID
	ID string `json:"id"`
	// 工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 应用名称
	Name string `json:"name"`
	// 应用类型
	Type string `json:"type"`
	// 应用显示名称
	DisplayName string `json:"displayName"`
	// 创建人
	Creator string `json:"creator"`
	// 应用使用的编程语言
	Language string `json:"language"`
	// 应用部署的环境列表
	DeployedEnvs []*AppDeployedEnvOutputObj `json:"deployedEnvs"`
	// 应用最近操作时间
	LastOperatedAt *time.Time `json:"lastOperatedAt,omitempty"`
}

// FromModel fills output fields from an Application model.
func (o *AppInfoOutputObj) FromModel(
	app *bkmsapp.Application,
	deployedEnvs []deploystatus.AppDeployStatus,
) *AppInfoOutputObj {
	language := ""
	if app.TrpcSpec != nil {
		language = app.TrpcSpec.Language
	}
	o.ID = app.ID
	o.WorkspaceID = app.WorkspaceID
	o.Name = app.Name
	o.Type = app.Type
	o.DisplayName = app.DisplayName
	o.Creator = app.Creator
	o.Language = language
	o.DeployedEnvs = make([]*AppDeployedEnvOutputObj, 0, len(deployedEnvs))
	for i := range deployedEnvs {
		o.DeployedEnvs = append(o.DeployedEnvs, new(AppDeployedEnvOutputObj).FromModel(deployedEnvs[i]))
	}
	return o
}

// -----------------------------------------------------------------------------
// UpdateAppDisplayName
// -----------------------------------------------------------------------------

// UpdateAppDisplayNameInput is the JSON body for updating app display name.
type UpdateAppDisplayNameInput struct {
	// 待更新的应用显示名
	DisplayName string `json:"displayName" binding:"required"`
}

// -----------------------------------------------------------------------------
// UpdateAppTrpcSpec / UpdateAppTafSpec
// -----------------------------------------------------------------------------

// UpdateAppModelSpecInput is the JSON body for updating app model spec (trpc/taf).
type UpdateAppModelSpecInput struct {
	// 应用模型规范
	AppModelSpec *AppModelSpecInput `json:"appModelSpec" binding:"required"`
}

// -----------------------------------------------------------------------------
// App deploy status API serializers
// -----------------------------------------------------------------------------

// AppDeployedEnvOutputObj is the JSON representation of an app's deployment status in one env or traffic lane.
// 结果可能来自标准环境，或来自当前应用拥有的特性环境。
type AppDeployedEnvOutputObj struct {
	// 环境 ID
	ID string `json:"id"`
	// 环境名称（英文标识）
	Name string `json:"name"`
	// 环境展示名称
	DisplayName string `json:"displayName"`
	// 环境类型（development / test / staging / production）
	Type string `json:"type"`
	// 环境类别（standard / feature）
	Kind string `json:"kind"`
	// 泳道名称，空字符串表示默认泳道
	TrafficLaneName string `json:"trafficLaneName"`
	// 部署状态
	DeployStatus string `json:"deployStatus"`
	// 部署的镜像 Tag
	ImageTag string `json:"imageTag"`
}

// FromModel fills output fields from one deploy status row.
func (o *AppDeployedEnvOutputObj) FromModel(row deploystatus.AppDeployStatus) *AppDeployedEnvOutputObj {
	*o = AppDeployedEnvOutputObj{
		ID:              row.EnvID,
		Name:            row.EnvName,
		DisplayName:     row.EnvDisplayName,
		Type:            row.EnvType,
		Kind:            row.EnvKind,
		TrafficLaneName: row.TrafficLaneName,
		DeployStatus:    row.DeployStatus,
		ImageTag:        row.ImageTag,
	}
	return o
}

// GetAppDeployStatusesOutput is the JSON response for querying app deploy statuses.
type GetAppDeployStatusesOutput struct {
	// 应用部署的环境列表，包含标准环境和当前应用拥有的特性环境
	Data []*AppDeployedEnvOutputObj `json:"data"`
}

func emptySliceIfNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
