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

// Package serializer defines Gin input and output serializers for workspace APIs.
package serializer

import (
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// -----------------------------------------------------------------------------
// 共用结构体
// -----------------------------------------------------------------------------

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}

// WorkspaceURIInput is the path input for APIs scoped by workspace.
type WorkspaceURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// WorkspaceRoleURIInput is the path input for workspace role APIs.
type WorkspaceRoleURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 角色 Code
	RoleCode string `uri:"roleCode" binding:"required,oneof=admin sre developer operator"`
}

// WorkspaceUserURIInput is the path input for workspace user APIs.
type WorkspaceUserURIInput struct {
	// 工作空间 ID
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	// 用户 ID
	UserID string `uri:"userID" binding:"required,uri_slug"`
}

// BkSystemsOutputObj is the JSON representation of BkSystems.
type BkSystemsOutputObj struct {
	// 蓝盾项目 ID; 创建时不传代表创建默认蓝盾项目
	BkCIProjectID string `json:"bkCIProjectID"`
	// 蓝盾项目 UID, 32 位字符串
	BkCIProjectUID string `json:"bkCIProjectUID"`
	// 蓝鲸容器服务（BCS）项目 ID
	BkBCSProjectID string `json:"bkBCSProjectID"`
	// 蓝鲸容器服务（BCS）项目 Code, 可读字符串, 如 bkce
	BkBCSProjectCode string `json:"bkBCSProjectCode"`
	// 蓝鲸日志平台项目 ID
	BkLogProjectID string `json:"bkLogProjectID"`
	// 蓝鲸监控平台项目 ID
	BkMonitorProjectID string `json:"bkMonitorProjectID"`
	// 蓝盾制品库项目 ID
	BkRepoProjectID string `json:"bkRepoProjectID"`
	// bkcc 业务 ID
	BkCCBizID string `json:"bkCCBizID"`
	// 二级业务 ID
	Level2BizID string `json:"level2BizID"`
	// 运营产品 ID
	ObsProductID string `json:"obsProductID"`
	// 运营产品名称
	ObsProductName string `json:"obsProductName"`
	// 是否绑定已有蓝盾项目
	IsBoundExistedBKCIProject bool `json:"isBoundExistedBKCIProject"`
}

// FromModel fills output fields from a BkSystems model.
func (o *BkSystemsOutputObj) FromModel(bk workspace.BkSystems) *BkSystemsOutputObj {
	*o = BkSystemsOutputObj{
		BkCIProjectID:             bk.BkCIProjectID,
		BkCIProjectUID:            bk.BkCIProjectUID,
		BkBCSProjectID:            bk.BkBCSProjectID,
		BkBCSProjectCode:          bk.BkBCSProjectCode,
		BkLogProjectID:            bk.BkLogProjectID,
		BkMonitorProjectID:        bk.BkMonitorProjectID,
		BkRepoProjectID:           bk.BkRepoProjectID,
		BkCCBizID:                 bk.BkCCBizID,
		Level2BizID:               bk.Level2BizID,
		ObsProductID:              bk.ObsProductID,
		ObsProductName:            bk.ObsProductName,
		IsBoundExistedBKCIProject: bk.IsBoundExistedBKCIProject,
	}
	return o
}

// ImageRegistryInput is the JSON body for image registry info.
type ImageRegistryInput struct {
	// 镜像仓库地址
	Registry string `json:"registry" binding:"required"`
	// 镜像仓库用户名
	Username string `json:"username" binding:"required"`
	// 镜像仓库密码
	Password string `json:"password" binding:"required"`
}

// ImageRegistryOutputObj is the JSON representation of image registry info.
type ImageRegistryOutputObj struct {
	// 镜像仓库地址
	Registry string `json:"registry"`
	// 镜像仓库用户名
	Username string `json:"username"`
	// 镜像仓库密码
	Password string `json:"password"`
}

// WorkspaceInfoOutputObj is the JSON representation of workspace brief info.
type WorkspaceInfoOutputObj struct {
	// 工作空间唯一标识
	ID string `json:"id"`
	// 展示用名称，一般为中文名
	DisplayName string `json:"displayName"`
	// 描述信息
	Description string `json:"description"`
	// 关联的蓝鲸体系系统的信息
	BkSystems *BkSystemsOutputObj `json:"bkSystems"`
	// 工作空间状态
	State string `json:"state"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新人
	Updater string `json:"updater"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from a Workspace model.
func (o *WorkspaceInfoOutputObj) FromModel(ws workspace.Workspace) *WorkspaceInfoOutputObj {
	*o = WorkspaceInfoOutputObj{
		ID:          ws.ID,
		DisplayName: ws.DisplayName,
		Description: ws.Description,
		BkSystems:   new(BkSystemsOutputObj).FromModel(ws.BkSystems),
		State:       string(ws.State),
		Creator:     ws.Creator,
		CreatedAt:   ws.CreatedAt,
		Updater:     ws.Updater,
		UpdatedAt:   ws.UpdatedAt,
	}
	return o
}

// WorkspaceDetailOutputObj is the JSON representation of workspace detail.
type WorkspaceDetailOutputObj struct {
	// 工作空间唯一标识
	ID string `json:"id"`
	// 展示用名称，一般为中文名
	DisplayName string `json:"displayName"`
	// 描述信息
	Description string `json:"description"`
	// 使用的镜像仓库类型
	ImageRegistryType string `json:"imageRegistryType"`
	// 关联的蓝鲸体系系统的信息
	BkSystems *BkSystemsOutputObj `json:"bkSystems"`
	// 镜像仓库信息
	ImageRegistry *ImageRegistryOutputObj `json:"imageRegistry"`
	// 工作空间状态
	State string `json:"state"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新人
	Updater string `json:"updater"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from a Workspace model and ImageRegistry.
func (o *WorkspaceDetailOutputObj) FromModel(
	ws workspace.Workspace,
	imageRegistry *bkmsreg.ImageRegistry,
) *WorkspaceDetailOutputObj {
	irOutput := &ImageRegistryOutputObj{}
	if imageRegistry != nil {
		irOutput.Registry = imageRegistry.Registry
		irOutput.Username = imageRegistry.Username
		irOutput.Password = imageRegistry.Password
	}
	*o = WorkspaceDetailOutputObj{
		ID:                ws.ID,
		DisplayName:       ws.DisplayName,
		Description:       ws.Description,
		ImageRegistryType: string(ws.ImageRegistryType),
		BkSystems:         new(BkSystemsOutputObj).FromModel(ws.BkSystems),
		ImageRegistry:     irOutput,
		State:             string(ws.State),
		Creator:           ws.Creator,
		CreatedAt:         ws.CreatedAt,
		Updater:           ws.Updater,
		UpdatedAt:         ws.UpdatedAt,
	}
	return o
}

// -----------------------------------------------------------------------------
// ListWorkspaces
// -----------------------------------------------------------------------------

// ListWorkspacesQueryInput is the query input for listing workspaces.
type ListWorkspacesQueryInput struct {
	// 搜索关键词, 匹配 ID & displayName, 不区分大小写 可选
	Keyword *string `form:"keyword" binding:"omitempty,max=64"`
}

// ListWorkspacesOutput is the JSON response for ListWorkspaces.
type ListWorkspacesOutput struct {
	Data []*WorkspaceInfoOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// ListWorkspacesOverview
// -----------------------------------------------------------------------------

// ListWorkspacesOverviewQueryInput is the query input for listing workspaces overview.
type ListWorkspacesOverviewQueryInput struct {
	// 返回的工作空间数量上限（按最近操作时间排序后取前 N 个）
	Limit int32 `form:"limit" binding:"required,min=1"`
}

// ListWorkspacesOverviewOutput is the JSON response for ListWorkspacesOverview.
type ListWorkspacesOverviewOutput struct {
	Data []*WorkspaceWithAppsOutputObj `json:"data"`
}

// WorkspaceWithAppsOutputObj is the JSON representation of workspace with apps.
type WorkspaceWithAppsOutputObj struct {
	// 工作空间唯一标识
	ID string `json:"id"`
	// 展示用名称，一般为中文名
	DisplayName string `json:"displayName"`
	// 描述信息
	Description string `json:"description"`
	// 工作空间状态
	State string `json:"state"`
	// 创建人
	Creator string `json:"creator"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新人
	Updater string `json:"updater"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// 工作空间最近操作时间（来自当前用户的审计日志）
	LastOperatedAt *time.Time `json:"lastOperatedAt,omitempty"`
	// 该空间下的应用列表（按当前用户操作时间倒序排序）
	Apps []*AppInfoOutputObj `json:"apps"`
}

// AppInfoOutputObj is the JSON representation of app brief info for overview.
type AppInfoOutputObj struct {
	// 应用 ID
	ID string `json:"id"`
	// 工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 应用名称
	Name string `json:"name"`
	// 应用类型
	Type string `json:"type"`
	// 展示用名称
	DisplayName string `json:"displayName"`
	// 创建人
	Creator string `json:"creator"`
	// 应用使用的编程语言（如 go、cpp），仅 trpc 类型应用有值
	Language string `json:"language"`
	// 应用部署的环境列表
	DeployedEnvs []*AppDeployedEnvOutputObj `json:"deployedEnvs"`
	// 应用最近操作时间（来自审计日志）
	LastOperatedAt *time.Time `json:"lastOperatedAt,omitempty"`
}

// AppDeployedEnvOutputObj is the JSON representation of app deployed env brief info.
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

// -----------------------------------------------------------------------------
// GetWorkspace
// -----------------------------------------------------------------------------

// GetWorkspaceOutput is the JSON response for GetWorkspace.
type GetWorkspaceOutput struct {
	Data *WorkspaceDetailOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// GetUserStatistics
// -----------------------------------------------------------------------------

// GetUserStatisticsOutput is the JSON response for GetUserStatistics.
type GetUserStatisticsOutput struct {
	Data *UserStatisticsOutputObj `json:"data"`
}

// UserStatisticsOutputObj is the JSON representation of user statistics.
type UserStatisticsOutputObj struct {
	// 用户工作空间数量
	WorkspaceCount int64 `json:"workspaceCount,string"`
	// 用户总应用数量
	AppCount int64 `json:"appCount,string"`
	// 用户总环境数量
	EnvCount int64 `json:"envCount,string"`
	// 用户工作空间统计
	WorkspaceStatistics []*UserWorkspaceStatisticsOutputObj `json:"workspaceStatistics"`
}

// UserWorkspaceStatisticsOutputObj is the JSON representation of per-workspace statistics.
type UserWorkspaceStatisticsOutputObj struct {
	// 工作空间 ID
	WorkspaceID string `json:"workspaceID"`
	// 用户某工作空间下的应用数量
	AppCount int64 `json:"appCount,string"`
	// 用户某工作空间下的环境数量
	EnvCount int64 `json:"envCount,string"`
}

// -----------------------------------------------------------------------------
// ListWorkspaceRoleMemberGroups
// -----------------------------------------------------------------------------

// ListWorkspaceRoleMemberGroupsOutput is the JSON response for ListWorkspaceRoleMemberGroups.
type ListWorkspaceRoleMemberGroupsOutput struct {
	Data []*RoleMemberGroupOutputObj `json:"data"`
}

// RoleMemberGroupOutputObj is the JSON representation of a role member group.
type RoleMemberGroupOutputObj struct {
	// 角色 ID
	RoleID string `json:"roleID"`
	// 角色 Code
	RoleCode string `json:"roleCode"`
	// 角色名称
	RoleName string `json:"roleName"`
	// 用户组 ID
	UserGroupID int64 `json:"userGroupID,string"`
	// 用户组成员
	Members []string `json:"members"`
}

// -----------------------------------------------------------------------------
// CreateWorkspace
// -----------------------------------------------------------------------------

// CreateWorkspaceInput is the JSON body for creating a workspace.
type CreateWorkspaceInput struct {
	// 工作空间 ID, 1-27 字符的空间 ID，由小写字母、数字、中划线组成，以小写字母开头，不能以中划线结尾
	ID string `json:"id" binding:"required,workspace_id"`
	// 展示用名称，一般为中文名（1-32 字符）
	DisplayName string `json:"displayName" binding:"required,min=1,max=32"`
	// 描述信息（0-512 字符）
	Description string `json:"description" binding:"max=512"`
	// 蓝盾项目 ID;
	// 创建时，新建容器项目，无需填写
	// 创建时，绑定已有容器项目，必填.
	BkCIProjectID string `json:"bkCIProjectID"`
	// 镜像仓库信息, 传入时代表绑定已有镜像仓库, 不传入则在容器项目的制品库中创建默认镜像仓库
	ImageRegistry *ImageRegistryInput `json:"imageRegistry"`
	// 空间管理员用户, 创建者即使不在列表中也会默认加入管理员
	Managers []string `json:"managers"`
	// bkccID 业务 ID
	// 创建时，新建容器项目，必填。
	// 创建时，绑定已有容器项目，无需填写。
	BkCCBizID int64 `json:"bkCCBizID"`
}

// CreateWorkspaceOutput is the JSON response for CreateWorkspace.
type CreateWorkspaceOutput struct {
	Data *WorkspaceDetailOutputObj `json:"data"`
}

// -----------------------------------------------------------------------------
// UpdateWorkspaceInfo
// -----------------------------------------------------------------------------

// UpdateWorkspaceInfoInput is the JSON body for updating workspace info.
type UpdateWorkspaceInfoInput struct {
	// 展示用名称，一般为中文名（1-32 字符）
	DisplayName string `json:"displayName" binding:"required,min=1,max=32"`
	// 描述信息（0-512 字符）
	Description string `json:"description" binding:"max=512"`
}

// -----------------------------------------------------------------------------
// AddWorkspaceUser
// -----------------------------------------------------------------------------

// AddWorkspaceUserInput is the JSON body for adding workspace users.
type AddWorkspaceUserInput struct {
	// 用户列表
	UserIDs []string `json:"userIDs" binding:"required,min=1"`
}

// -----------------------------------------------------------------------------
// SetWorkspaceState
// -----------------------------------------------------------------------------

// SetWorkspaceStateInput is the JSON body for setting workspace state.
type SetWorkspaceStateInput struct {
	// 工作空间状态
	State string `json:"state" binding:"required,oneof=Ready Disabled"`
}
