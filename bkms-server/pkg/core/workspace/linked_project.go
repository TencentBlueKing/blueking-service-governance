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

package workspace

import (
	"cmp"
	"context"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bcsapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	bkmonitorapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// ListLinkedProjects 列出工作空间的关联项目，按蓝盾、监控、容器排序，并以 ctx 中的当前用户身份查询平台侧状态。
//
// 查询平台失败不会报错，对应项目的状态为 LinkedProjectUnknown
func ListLinkedProjects(ctx context.Context, ws *Workspace) []LinkedProject {
	projects := linkedProjectsOf(ws)

	var wg sync.WaitGroup
	for i := range projects {
		wg.Go(func() {
			if err := fillLinkedProject(ctx, &projects[i]); err != nil {
				log.Warnf(ctx, "query %s project %s failed: %v", projects[i].System, projects[i].ProjectID, err)
			}
		})
	}
	wg.Wait()
	return projects
}

// SelectLinkedProjectsToDisable 从关联项目中选出要停用的项目，保持 projects 的顺序。
//
// systems 中的平台未关联或不能启停时返回 ErrLinkedProjectNotOperable
func SelectLinkedProjectsToDisable(projects []LinkedProject, systems []LinkedSystem) ([]LinkedProject, error) {
	for _, sys := range systems {
		p, ok := lo.Find(projects, func(p LinkedProject) bool { return p.System == sys })
		if !ok {
			return nil, errors.Wrapf(ErrLinkedProjectNotOperable, "workspace has no %s project", sys)
		}
		if !p.Operable() {
			return nil, errors.Wrapf(
				ErrLinkedProjectNotOperable, "%s project %s: %s", sys, p.ProjectID, p.BlockedReason,
			)
		}
	}
	return lo.Filter(projects, func(p LinkedProject, _ int) bool { return lo.Contains(systems, p.System) }), nil
}

// SelectLinkedProjectsToEnable 选出启用空间时需要一并启用的项目：平台支持启停且未确认处于启用中。
//
// 当前用户没有管理权限的已停用项目也会被选出，启用失败后空间保持停用
func SelectLinkedProjectsToEnable(projects []LinkedProject) []LinkedProject {
	return lo.Filter(projects, func(p LinkedProject, _ int) bool {
		return p.BlockedReason != LinkedProjectUnsupported && p.State != LinkedProjectReady
	})
}

// SetLinkedProjectsEnabled 以 ctx 中的当前用户身份按顺序启用或停用项目，单个项目失败不影响其他项目
func SetLinkedProjectsEnabled(ctx context.Context, projects []LinkedProject, enabled bool) []LinkedProjectResult {
	return lo.Map(projects, func(p LinkedProject, _ int) LinkedProjectResult {
		result := LinkedProjectResult{
			System:      p.System,
			ProjectID:   p.ProjectID,
			ProjectName: p.ProjectName,
			Success:     true,
		}
		if err := setLinkedProjectEnabled(ctx, p, enabled); err != nil {
			log.Warnf(ctx, "set %s project %s enabled=%t failed: %v", p.System, p.ProjectID, enabled, err)
			result.Success = false
			result.Reason, result.Detail, result.RequestID = classifyLinkedProjectError(err)
		}
		return result
	})
}

// linkedProjectsOf 根据空间记录的平台项目生成关联项目，名称暂为项目 ID，平台侧状态待查询
func linkedProjectsOf(ws *Workspace) []LinkedProject {
	sys := ws.BkSystems
	origin := lo.Ternary(sys.IsBoundExistedBKCIProject, LinkedProjectBound, LinkedProjectCreated)
	newProject := func(system LinkedSystem, projectID, code string) LinkedProject {
		return LinkedProject{
			System:      system,
			ProjectID:   projectID,
			ProjectName: projectID,
			Origin:      origin,
			State:       LinkedProjectUnknown,
			code:        code,
		}
	}

	var projects []LinkedProject
	if sys.BkCIProjectID != "" {
		projects = append(projects, newProject(LinkedSystemBKCI, sys.BkCIProjectID, sys.BkCIProjectID))
	}
	// 监控空间与容器项目一一对应，空间激活后才写入监控项目 ID
	if sys.BkBCSProjectCode != "" && sys.BkMonitorProjectID != "" {
		p := newProject(LinkedSystemBKMonitor, sys.BkMonitorProjectID, sys.BkBCSProjectCode)
		// 监控平台限制只能停用 bkms 创建的监控空间
		if !strings.HasPrefix(sys.BkBCSProjectCode, bkmonitorapi.BkmsSpaceIDPrefix) {
			p.BlockedReason = LinkedProjectUnsupported
		}
		projects = append(projects, p)
	}
	// 容器服务没有停用项目的能力，项目始终为启用中
	if sys.BkBCSProjectCode != "" {
		p := newProject(LinkedSystemBCS, sys.BkBCSProjectCode, sys.BkBCSProjectCode)
		p.State, p.BlockedReason = LinkedProjectReady, LinkedProjectUnsupported
		projects = append(projects, p)
	}
	return projects
}

// fillLinkedProject 从平台查询项目名称、启停状态与当前用户的管理权限
func fillLinkedProject(ctx context.Context, p *LinkedProject) error {
	user := auth.MustGetUser(ctx)
	switch p.System {
	case LinkedSystemBKCI:
		client, err := bkciapi.New(user)
		if err != nil {
			return errors.Wrap(err, "create bkci client")
		}
		proj, err := client.GetProject(ctx, p.code)
		if err != nil {
			return errors.Wrap(err, "get bkci project")
		}
		p.ProjectName = cmp.Or(proj.Name, p.ProjectName)
		p.State = lo.Ternary(proj.Enabled, LinkedProjectReady, LinkedProjectDisabled)
		if !proj.HasManagePerm {
			p.BlockedReason = LinkedProjectNoPermission
		}
	case LinkedSystemBKMonitor:
		client, err := bkmonitorapi.NewMonitorClient(user.ID)
		if err != nil {
			return errors.Wrap(err, "create bkmonitor client")
		}
		space, err := client.GetMetadataSpaceDetail(ctx, p.code)
		if err != nil {
			return errors.Wrap(err, "get bkmonitor space")
		}
		p.ProjectName = cmp.Or(space.SpaceName, p.ProjectName)
		disabled := space.Status == bkmonitorapi.SpaceStatusDisabled
		p.State = lo.Ternary(disabled, LinkedProjectDisabled, LinkedProjectReady)
	case LinkedSystemBCS:
		client, err := bcsapi.New(user)
		if err != nil {
			return errors.Wrap(err, "create bcs client")
		}
		proj, err := client.GetProject(ctx, p.code)
		if err != nil {
			return errors.Wrap(err, "get bcs project")
		}
		p.ProjectName = cmp.Or(proj.Name, p.ProjectName)
	}
	return nil
}

func setLinkedProjectEnabled(ctx context.Context, p LinkedProject, enabled bool) error {
	user := auth.MustGetUser(ctx)
	switch p.System {
	case LinkedSystemBKCI:
		client, err := bkciapi.New(user)
		if err != nil {
			return errors.Wrap(err, "create bkci client")
		}
		return client.SetProjectEnabled(ctx, p.code, enabled)
	case LinkedSystemBKMonitor:
		client, err := bkmonitorapi.NewMonitorClient(user.ID)
		if err != nil {
			return errors.Wrap(err, "create bkmonitor client")
		}
		status := bkmonitorapi.SpaceStatusDisabled
		if enabled {
			status = bkmonitorapi.SpaceStatusNormal
		}
		return client.SetBkmsSpaceStatus(ctx, p.code, status)
	default:
		return errors.Wrapf(ErrLinkedProjectNotOperable, "%s project %s", p.System, p.ProjectID)
	}
}
