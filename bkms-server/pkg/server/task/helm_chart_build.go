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
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelinevar"
	helmchartbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
)

// collectHelmBuildExtras 从蓝盾 Variables 中提取 Helm Chart 构建关心的额外信息（commit / repo 等）
// 仅取常用字段，不存在时填空字符串以保证字段稳定
func collectHelmBuildExtras(variables map[string]string) map[string]string {
	extras := make(map[string]string, len(pipelinevar.RequiredVariables))
	for _, name := range pipelinevar.RequiredVariables {
		extras[name] = variables[name]
	}
	return extras
}

// PollingHelmChartBuildStatusArgs 轮询 Helm Chart 构建状态的参数
type PollingHelmChartBuildStatusArgs struct {
	WorkspaceID string `json:"workspaceID"`
	AppID       string `json:"appID"`
	BuildID     string `json:"buildID"`
}

// String 参数内容字符串化
func (args PollingHelmChartBuildStatusArgs) String() string {
	return fmt.Sprintf(
		"<workspace: %s, appID: %s, buildID: %s>",
		args.WorkspaceID, args.AppID, args.BuildID,
	)
}

// pollingHelmChartBuildStatus 轮询蓝盾 Helm Chart 构建状态
func pollingHelmChartBuildStatus(ctx context.Context, args PollingHelmChartBuildStatusArgs) (*EmptyResult, error) {
	log.Infof(ctx, "start polling helm chart build %s status, timeout: %ds", args, helmChartBuildPollingTimeout)

	// HelmChart 构建相对轻量，使用固定常量设置轮询间隔和超时
	ctx, cancel, ticker := setPollingContext(ctx, config.PollConfig{
		Interval: helmChartBuildPollingInterval,
		Timeout:  helmChartBuildPollingTimeout,
	})
	defer cancel()
	defer ticker.Stop()

	// 获取构建记录
	store, err := helmchartbuild.NewRecordStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create helm chart build record store")
	}
	record, err := store.Get(ctx, args.AppID, args.BuildID)
	if err != nil {
		return nil, errors.Wrap(err, "get helm chart build record")
	}

	// 获取流水线信息
	pipelineStore, err := bkci.NewPipelineStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "create pipeline store")
	}
	pipeline, err := pipelineStore.GetByWorkspaceAndType(
		ctx, args.WorkspaceID, string(bkci.PipelineTypeHelmGitBuild),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s helm-git-build pipeline", args.WorkspaceID)
	}

	user, err := auth.GetUser(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get authed user")
	}

	apiClient, err := bkciapi.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "create bkci api client")
	}

	// 获取状态失败重试次数（防止网络波动等原因）
	failureRetryCount := TotalFailureRetryCount

	for {
		curStatus := record.Status

		select {
		case <-ctx.Done():
			log.Warnf(ctx, "context timeout, stop update helm chart build %s status", args)
			record.Status = helmchartbuild.StatusPollingTimeout
		case <-ticker.C:
			// 获取蓝盾流水线构建状态
			buildState, fetchErr := apiClient.GetPipelineBuildState(
				ctx, pipeline.ProjectCode, pipeline.ID, args.BuildID,
			)
			if fetchErr != nil {
				log.Errorf(ctx, "failed to get helm chart build %s state: %v", args, fetchErr)
				failureRetryCount--
				if failureRetryCount <= 0 {
					log.Errorf(ctx, "stop polling helm chart build %s after %d retries", args, TotalFailureRetryCount)
					record.Status = helmchartbuild.StatusPollingBroken
				}
			} else {
				// 根据蓝盾状态更新构建记录状态
				buildStatus := bkciapi.PipelineBuildStatus(buildState.Status)
				switch {
				case buildStatus.IsSuccess():
					record.Status = helmchartbuild.StatusSuccess
				case buildStatus.IsFailure():
					record.Status = helmchartbuild.StatusFailed
				case buildStatus.IsCancel():
					record.Status = helmchartbuild.StatusCanceled
				case buildStatus.IsRunning():
					record.Status = helmchartbuild.StatusRunning
				}
				if buildStatus.IsFinished() {
					endedAt := time.Now()
					record.EndedAt = &endedAt
				}
				// 从蓝盾 Variables 中回填 commit / repo 等额外信息，供制品 / 构建详情展示
				record.Extras = collectHelmBuildExtras(buildState.Variables)
			}
		}

		// 若 record 状态变更，则需要保存入库
		if record.Status != curStatus {
			log.Infof(ctx, "helm chart build %s status changed from %s to %s", args, curStatus, record.Status)
			func() {
				saveCtx, saveCancel := context.WithTimeout(context.Background(), saveStatusTimeout)
				defer saveCancel()
				if err = store.Update(saveCtx, record); err != nil {
					log.Errorf(saveCtx, "failed to update helm chart build record: %v", err)
				}
			}()
		}

		// 如果最新状态已经是结束态，退出轮询
		if record.IsTerminated() {
			log.Infof(ctx, "helm chart build %s status is %s (Terminated), stop polling", args, record.Status)

			// 转换为操作结果 & 记录操作审计
			opResult := lo.Ternary(
				record.Status == helmchartbuild.StatusSuccess,
				audit.ResultSuccess,
				audit.ResultFailed,
			)
			go audit.AddOperationRecordAsync(
				context.WithoutCancel(ctx), audit.OperationTypeBuild, audit.ResourceTypeApp, args.AppID,
				audit.WithResult(opResult), audit.WithAttribute(audit.AttributeHelmChart),
				audit.WithWorkspaceID(args.WorkspaceID), audit.WithAppID(args.AppID),
			)

			return &emptyResult, nil
		}
	}
}
