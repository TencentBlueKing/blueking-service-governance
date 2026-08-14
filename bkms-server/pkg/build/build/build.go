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

// Package build 组合镜像构建与后续自动部署流程，作为构建入口的编排层
package build

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	deploypkg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/taskqtask/buildpoll"
)

// StartOptions describes optional build follow-up workflow.
type StartOptions struct {
	AutoDeploy            *buildpoll.AutoDeployArgs
	EnvStore              envmodel.EnvironmentStore
	AutoDeployRecordStore autodeploy.RecordStore
}

// StartAndScheduleBuild 同步触发蓝盾构建后，投递镜像构建状态轮询任务
func StartAndScheduleBuild(
	ctx context.Context,
	buildService *Service,
	app *bkmsapp.Application,
	branch, imageTag string,
	opts StartOptions,
) (*imagebuild.Record, error) {
	result, err := buildService.Build(ctx, app, branch, imageTag)
	if err != nil {
		return nil, errors.Wrapf(err, "start build for %s", app.ID)
	}

	taskArgs := buildpoll.Args{
		WorkspaceID:  app.WorkspaceID,
		PipelineType: result.PipelineType,
		AppID:        app.ID,
		BuildID:      result.Record.BuildID,
	}
	if opts.AutoDeploy != nil {
		if opts.AutoDeployRecordStore == nil {
			return nil, errors.New("build auto deploy record store is nil")
		}
		if opts.EnvStore == nil {
			return nil, errors.New("environment store is nil")
		}
		taskArgs.AutoDeploy = opts.AutoDeploy
		if err = opts.AutoDeployRecordStore.Create(ctx, &autodeploy.Record{
			WorkspaceID:     app.WorkspaceID,
			AppID:           app.ID,
			AppType:         app.Type,
			EnvName:         opts.AutoDeploy.EnvName,
			TrafficLaneName: opts.AutoDeploy.TrafficLaneName,
			BuildID:         result.Record.BuildID,
			Branch:          branch,
			ImageTag:        imageTag,
			PipelineID:      result.Record.PipelineID,
			Stage:           autodeploy.StageBuild,
			Status:          string(result.Record.Status),
			Operator:        result.Record.Operator,
			StartedAt:       result.Record.StartedAt,
		}); err != nil {
			return nil, errors.Wrap(err, "create build auto deploy record")
		}
		deploypkg.TrackEnvAddApp(ctx, opts.EnvStore, app.WorkspaceID, opts.AutoDeploy.EnvName, app.ID)
	}

	err = taskq.Enqueue(
		ctx,
		buildpoll.Task.NewTask(taskArgs),
		asynq.ProcessIn(buildpoll.PollingInterval(time.Since(result.Record.StartedAt))),
	)
	if err != nil {
		if opts.AutoDeploy != nil {
			record, getErr := opts.AutoDeployRecordStore.GetByBuildID(ctx, app.ID, result.Record.BuildID)
			if getErr == nil {
				record.Status = string(imagebuild.StatusFailed)
				record.Message = "apply polling build status task failed"
				record.EndedAt = result.Record.CreatedAt
				if updateErr := opts.AutoDeployRecordStore.Update(ctx, record); updateErr != nil {
					err = errors.Wrap(updateErr, "update build auto deploy record")
				}
			}
		}
		return nil, errors.Wrap(err, "enqueue polling build status task")
	}
	return result.Record, nil
}
