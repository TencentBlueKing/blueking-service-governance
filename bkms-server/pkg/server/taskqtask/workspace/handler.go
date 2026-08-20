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
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// maxWaitDuration 等待 BKM Project 就绪的最长时间, 自 workspace 创建起算
const maxWaitDuration = 10 * time.Minute

// initHandler 工作空间初始化任务处理函数
//
// 1. 检查工作空间对应的蓝鲸监控项目是否就绪，未就绪则通过 ErrFixedRetry 等待下次重试
// 2. 如果就绪，给空间角色加上对应监控/日志项目的权限
// 3. 将蓝鲸监控项目 ID 写入 workspace, 并将空间状态调整为就绪
// 4. 给空间下的所有环境添加 APM 相关环境变量
func initHandler(ctx context.Context, args InitializationArgs) error {
	log.Infof(ctx, "polling workspace status for workspace %s", args.WorkspaceID)

	mongoCli, dbName := database.Client(), database.Name()
	workspaceStore, err := workspace.NewWorkspaceStoreMongo(mongoCli, dbName)
	if err != nil {
		return errors.Wrapf(err, "create workspace store")
	}
	ws, err := workspaceStore.Get(ctx, args.WorkspaceID)
	if err != nil {
		return errors.Wrapf(err, "get workspace %s", args.WorkspaceID)
	}
	client, err := bkmonitor.New(ws.Creator)
	if err != nil {
		return errors.Wrapf(err, "create bkmonitor client")
	}

	// 检查 BKM 项目是否就绪
	bkmProject, err := client.GetMetadataSpaceDetail(ctx, ws.BkSystems.BkBCSProjectCode)
	if err != nil {
		// 超时则停止重试。StopRetry 不触发 exhausted 回调, 故此处自行上报终态
		if time.Since(ws.CreatedAt) > maxWaitDuration {
			reportInitFinished(ws, metrics.StatusTimeout)
			log.Errorf(ctx, "workspace %s init timeout after %v, give up: %v", ws.ID, maxWaitDuration, err)
			return errors.Wrapf(taskq.ErrStopRetry,
				"get BKM project %s timeout after %v: %v", ws.ID, maxWaitDuration, err)
		}
		// 未就绪，等待下次重试
		log.Infof(ctx, "get BKM project %s failed: %v, will retry", ws.ID, err)
		return errors.Wrap(taskq.ErrFixedRetry, "bkm project not ready")
	}

	// BKM 项目就绪，执行激活工作空间
	if err = activateWorkspace(ctx, workspaceStore, ws, bkmProject); err != nil {
		return errors.Wrap(taskq.ErrFixedRetry, err.Error())
	}

	reportInitFinished(ws, metrics.StatusOK)
	return nil
}

// activateWorkspace 激活工作空间（iam 授权、ws 改为就绪、创建并绑定 APM）
func activateWorkspace(
	ctx context.Context,
	workspaceStore workspace.WorkspaceStore,
	ws *workspace.Workspace,
	bkmProject *bkmonitor.Space,
) error {
	mongoCli, dbName := database.Client(), database.Name()
	permMgr := perm.NewManager()

	// 给空间角色加上权限
	workspaceAuthData := bkiam.WorkspaceData{
		WorkspaceID:   ws.ID,
		WorkspaceName: ws.DisplayName,
		BKMonitor: &bkiam.BKMonitorOptions{
			SpaceID:   fmt.Sprintf("%d", bkmProject.ID),
			SpaceName: bkmProject.SpaceName,
		},
		BKLog: &bkiam.BKLogOptions{
			SpaceID:   fmt.Sprintf("%d", bkmProject.ID),
			SpaceName: bkmProject.SpaceName,
		},
	}
	if err := permMgr.UpdateWorkspaceAdmin(ctx, workspaceAuthData); err != nil {
		return errors.Wrapf(err, "update workspace admin %s", ws.ID)
	}
	if err := permMgr.UpdateWorkspaceScopeBuiltinRoles(ctx, workspaceAuthData); err != nil {
		return errors.Wrapf(err, "update workspace scope builtin roles %s", ws.ID)
	}

	// 更新工作空间的监控、日志信息，调整状态为就绪
	ws.State = workspace.StateReady
	ws.BkSystems.BkLogProjectID = cast.ToString(bkmProject.ID)
	ws.BkSystems.BkMonitorProjectID = cast.ToString(bkmProject.ID)
	if err := workspaceStore.Update(ctx, ws); err != nil {
		return errors.Wrapf(err, "update workspace %s", ws.ID)
	}

	// 确保空间下的所有环境都有对应 ApmApp, 并添加 APM 相关环境变量
	envStore, err := envmodel.NewEnvironmentStoreMongo(mongoCli, dbName)
	if err != nil {
		return errors.Wrapf(err, "create env store")
	}
	scopedEnvVarStore, err := envvars.NewScopedEnvVarStoreMongo(mongoCli, dbName)
	if err != nil {
		return errors.Wrapf(err, "create scoped env var store")
	}
	apmStore, err := bkmmodel.NewApmInstConfigStoreMongo(mongoCli, dbName)
	if err != nil {
		return errors.Wrapf(err, "create apm store")
	}
	userGroupService := bkmmodel.NewUserGroupService(permMgr, envStore)
	apmService := bkmmodel.NewApmService(apmStore, scopedEnvVarStore)

	envs, err := envStore.ListStdEnvs(ctx, ws.ID)
	if err != nil {
		return errors.Wrapf(err, "list envs for workspace %s", ws.ID)
	}
	for _, env := range envs {
		if _, err = apmService.CreateAndBindToEnv(
			ctx,
			env.ID,
			env.Name,
			ws.BkSystems.BkBCSProjectCode,
			bkmmodel.CreateApmInstParams{
				WorkspaceID:  ws.ID,
				Username:     ws.Creator,
				BkmProjectID: bkmProject.ID,
			},
		); err != nil {
			return errors.Wrapf(err, "create and bind apm to env %s/%s", ws.ID, env.Name)
		}

		// 监控侧 APM 创建是一个很消耗资源的操作，等待 500ms，避免频繁调用
		time.Sleep(500 * time.Millisecond)

		// APM 创建成功后，异步将 workspace 下 admin/sre 人员同步到新告警组
		// 告警组同步自带最长 10s 的 NotFound 退避重试，同步执行会让本任务按环境数
		// 累计占住 worker 并发槽位，批量建空间时足以把部署状态轮询等短任务饿死
		go userGroupService.SyncMembersForEnvWithRetry(
			context.WithoutCancel(ctx), ws, env.Name, ws.Creator,
		)
	}
	return nil
}

// onExhausted 在重试耗尽时上报失败终态。
func onExhausted(ctx context.Context, workspaceID string, lastErr error) {
	log.Errorf(ctx, "workspace init task exhausted for %s: %v", workspaceID, lastErr)

	workspaceStore, err := workspace.NewWorkspaceStoreMongo(database.Client(), database.Name())
	if err != nil {
		log.Errorf(ctx, "workspace init exhausted: create store for %s: %v", workspaceID, err)
		return
	}
	ws, err := workspaceStore.Get(ctx, workspaceID)
	if err != nil {
		log.Errorf(ctx, "workspace init exhausted: get workspace %s: %v", workspaceID, err)
		return
	}
	reportInitFinished(ws, metrics.StatusErr)
}

// reportInitFinished 上报初始化终态结果与耗时, 耗时自 workspace 创建起算
func reportInitFinished(ws *workspace.Workspace, status string) {
	metrics.WorkspaceInitFinished(status, ws.CreatedAt)
}
