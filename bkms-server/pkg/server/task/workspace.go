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

package task

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkiam"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/perm"
	bkmmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
)

// 等待 BKM Project 准备就绪，最长等待 workspace 创建后 10 分钟
const (
	pollingWorkspaceInitStatusMaxWaitDuration = 10 * time.Minute
	pollingWorkspaceInitStatusRetryInterval   = 1 * time.Minute
)

// PollingWorkspaceInitStatusArgs ...
type PollingWorkspaceInitStatusArgs struct {
	WorkspaceID string `json:"workspaceID"`
}

// pollingWorkspaceInitStatus 轮询工作空间状态
// 1. 检查工作空间对应的蓝鲸监控项目是否就绪，未就绪则轮询等待. (蓝鲸监控会根据容器项目异步创建对应的监控项目)
// 2. 如果就绪，给空间角色加上对应监控/日志项目的权限
// 3. 将蓝鲸监控项目 ID 写入 workspace, 并将空间状态调整为就绪
// 4. 给空间下的所有环境添加 APM 相关环境变量
func pollingWorkspaceInitStatus(
	ctx context.Context,
	args PollingWorkspaceInitStatusArgs,
) (result *EmptyResult, retErr error) {
	startedAt := time.Now()
	metricStatus := metrics.StatusOK
	defer func() {
		if retErr != nil && metricStatus == metrics.StatusOK {
			metricStatus = metrics.StatusErr
		}
		metrics.WorkspaceInitFinished(metricStatus, startedAt)
	}()

	log.Infof(ctx, "polling workspace status for workspace %s", args.WorkspaceID)

	mongoCli, dbName := database.Client(), database.Name()
	workspaceStore, err := workspace.NewWorkspaceStoreMongo(mongoCli, dbName)
	if err != nil {
		return nil, errors.Wrapf(err, "create workspace store")
	}
	ws, err := workspaceStore.Get(ctx, args.WorkspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s", args.WorkspaceID)
	}
	client, err := bkmonitor.New(ws.Creator)
	if err != nil {
		return nil, errors.Wrapf(err, "create bkmonitor client")
	}

	bkmProject, waitStatus, err := waitForBKMProjectReady(ctx, ws, client)
	if err != nil {
		metricStatus = waitStatus
		return nil, err
	}

	// 防止卡临界值的情况，防御性设计：
	// 比如：ctx 刚好卡在快要结束时执行完成了上面的代码(卡在只剩下0.001秒)，剩下的时间和代码，不能在超时之前，执行完成。
	storeCtx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	if err = finishWorkspaceInit(storeCtx, mongoCli, dbName, workspaceStore, ws, bkmProject); err != nil {
		return nil, err
	}

	return &emptyResult, nil
}

func waitForBKMProjectReady(
	ctx context.Context,
	ws *workspace.Workspace,
	client bkmonitor.Client,
) (*bkmonitor.Space, string, error) {
	for {
		bkmProject, err := client.GetMetadataSpaceDetail(ctx, ws.BkSystems.BkBCSProjectCode)
		if err == nil {
			return bkmProject, metrics.StatusOK, nil
		}
		// 如果超过时间限制后，bkm 项目仍未就绪， 则直接返回错误
		// TODO 对于这种后台异步错误，如何通知用户？
		//  1、 通过加 metrics 告警处理
		//  2、 新增一个接口用于手动调用重试
		if time.Since(ws.CreatedAt) > pollingWorkspaceInitStatusMaxWaitDuration {
			return nil, metrics.StatusTimeout, errors.Wrapf(
				err, "get BKM project %s timeout after %v", ws.ID, pollingWorkspaceInitStatusMaxWaitDuration,
			)
		}

		log.Infof(
			ctx, "get BKM project %s failed: %v, will retry after %v",
			ws.ID, err, pollingWorkspaceInitStatusRetryInterval,
		)
		// 等待一分钟后重试
		select {
		case <-ctx.Done():
			return nil, metrics.StatusCancelled, ctx.Err()
		case <-time.After(pollingWorkspaceInitStatusRetryInterval):
		}
	}
}

func finishWorkspaceInit(
	ctx context.Context,
	mongoCli *mongo.Client,
	dbName string,
	workspaceStore workspace.WorkspaceStore,
	ws *workspace.Workspace,
	bkmProject *bkmonitor.Space,
) error {
	permMgr := perm.NewManager()
	// 2. 给空间角色加上权限, 并将空间状态调整为就绪
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

	// 3. 更新工作空间的监控、日志信息，调整状态为就绪
	ws.State = workspace.StateReady
	ws.BkSystems.BkLogProjectID = cast.ToString(bkmProject.ID)
	ws.BkSystems.BkMonitorProjectID = cast.ToString(bkmProject.ID)
	if err := workspaceStore.Update(ctx, ws); err != nil {
		return errors.Wrapf(err, "update workspace %s", ws.ID)
	}

	// 4. 确保空间下的所有环境都有对应 ApmApp, 并添加 APM 相关环境变量
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

		// 监控侧 APM 创建是一个很消耗资源的操作，等待 500ms，避免频繁调用 和 对监控侧造成压力
		time.Sleep(500 * time.Millisecond)

		// APM 创建成功后，同步将 workspace 下 admin/sre 人员同步到新告警组。
		userGroupService.SyncMembersForEnvWithRetry(context.TODO(), ws, env.Name, ws.Creator)
	}
	return nil
}
