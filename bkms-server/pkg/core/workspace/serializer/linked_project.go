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

package serializer

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
)

// -----------------------------------------------------------------------------
// ListWorkspaceLinkedProjects
// -----------------------------------------------------------------------------

// ListWorkspaceLinkedProjectsOutput is the JSON response for listing workspace linked projects.
type ListWorkspaceLinkedProjectsOutput struct {
	Data []LinkedProjectOutputObj `json:"data"`
}

// LinkedProjectOutputObj is the JSON representation of a workspace linked project.
type LinkedProjectOutputObj struct {
	// 所属平台，可选值：bkci（蓝盾）、bkmonitor（监控）、bcs（容器服务）
	System string `json:"system"`
	// 平台项目 ID：蓝盾、容器为项目 Code，监控为空间 ID
	ProjectID string `json:"projectID"`
	// 平台项目名称，查询平台失败时为项目 ID
	ProjectName string `json:"projectName"`
	// 来源，可选值：created（随空间创建）、bound（绑定的已有项目）
	Origin string `json:"origin"`
	// 平台侧状态，可选值：Ready（启用中）、Disabled（已停用）、Unknown（查询平台失败）
	State string `json:"state"`
	// 不能启停的原因，为空表示可以启停。可选值：unsupported（平台不支持）、noPermission（没有项目管理权限）
	BlockedReason string `json:"blockedReason"`
}

// NewLinkedProjectOutputObjs converts linked projects to output objects.
func NewLinkedProjectOutputObjs(projects []workspace.LinkedProject) []LinkedProjectOutputObj {
	return lo.Map(projects, func(p workspace.LinkedProject, _ int) LinkedProjectOutputObj {
		return LinkedProjectOutputObj{
			System:        string(p.System),
			ProjectID:     p.ProjectID,
			ProjectName:   p.ProjectName,
			Origin:        string(p.Origin),
			State:         string(p.State),
			BlockedReason: string(p.BlockedReason),
		}
	})
}

// -----------------------------------------------------------------------------
// DisableWorkspaceLinkedProjects
// -----------------------------------------------------------------------------

// DisableWorkspaceLinkedProjectsInput is the JSON body for disabling linked projects of a disabled workspace.
type DisableWorkspaceLinkedProjectsInput struct {
	// 要停用的关联项目所属平台，可选值：bkci、bkmonitor
	LinkedProjects []string `json:"linkedProjects" binding:"required,min=1,unique,dive,oneof=bkci bkmonitor"`
}

// DisableWorkspaceLinkedProjectsOutput is the JSON response for disabling linked projects.
type DisableWorkspaceLinkedProjectsOutput struct {
	Data *DisableWorkspaceLinkedProjectsOutputObj `json:"data"`
}

// DisableWorkspaceLinkedProjectsOutputObj is the result of disabling linked projects.
type DisableWorkspaceLinkedProjectsOutputObj struct {
	// 关联项目停用结果
	LinkedProjects []LinkedProjectResultOutputObj `json:"linkedProjects"`
}

// -----------------------------------------------------------------------------
// 关联项目启停结果
// -----------------------------------------------------------------------------

// LinkedProjectResultOutputObj is the result of enabling or disabling a linked project.
type LinkedProjectResultOutputObj struct {
	// 所属平台
	System string `json:"system"`
	// 平台项目 ID
	ProjectID string `json:"projectID"`
	// 平台项目名称
	ProjectName string `json:"projectName"`
	// 是否成功
	Success bool `json:"success"`
	// 失败原因，可选值：archived、notFound、permission、unavailable、unknown；成功时为空
	Reason string `json:"reason"`
	// 平台返回的原始错误信息，仅用于排查
	Detail string `json:"detail"`
	// 平台请求 ID
	RequestID string `json:"requestID"`
}

// NewLinkedProjectResultOutputObjs converts linked project results to output objects.
func NewLinkedProjectResultOutputObjs(results []workspace.LinkedProjectResult) []LinkedProjectResultOutputObj {
	return lo.Map(results, func(r workspace.LinkedProjectResult, _ int) LinkedProjectResultOutputObj {
		return LinkedProjectResultOutputObj{
			System:      string(r.System),
			ProjectID:   r.ProjectID,
			ProjectName: r.ProjectName,
			Success:     r.Success,
			Reason:      string(r.Reason),
			Detail:      r.Detail,
			RequestID:   r.RequestID,
		}
	})
}

// ToLinkedProjectSystems converts platform names in request bodies to linked project systems.
func ToLinkedProjectSystems(systems []string) []workspace.LinkedSystem {
	return lo.Map(systems, func(s string, _ int) workspace.LinkedSystem { return workspace.LinkedSystem(s) })
}
