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

package chartbuildpoll

import (
	"context"
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelinevar"
	helmchartbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

// fetchAndUpdateChartBuildRecord 查一次蓝盾并把状态写回 record。
// 仅状态变化时更新 Extras；结束时间用 now，与现网 Chart 轮询一致。
// 未识别的蓝盾状态保持原 status，不引入 StatusUnknown。
func fetchAndUpdateChartBuildRecord(
	ctx context.Context,
	client bkciapi.Client,
	pipeline *bkci.Pipeline,
	record *helmchartbuild.Record,
	buildID string,
) error {
	buildState, err := client.GetPipelineBuildState(ctx, pipeline.ProjectCode, pipeline.ID, buildID)
	if err != nil {
		log.Errorf(
			ctx, "failed to get project %s pipeline %s chart build %s: %v",
			pipeline.ProjectCode, pipeline.ID, buildID, err,
		)
		return err
	}

	nextStatus := record.Status
	buildStatus := bkciapi.PipelineBuildStatus(buildState.Status)
	switch {
	case buildStatus.IsSuccess():
		nextStatus = helmchartbuild.StatusSuccess
	case buildStatus.IsFailure():
		nextStatus = helmchartbuild.StatusFailed
	case buildStatus.IsCancel():
		nextStatus = helmchartbuild.StatusCanceled
	case buildStatus.IsRunning():
		nextStatus = helmchartbuild.StatusRunning
	}

	if nextStatus != record.Status {
		record.Status = nextStatus
		record.Extras = collectHelmBuildExtras(buildState.Variables)
		if buildStatus.IsFinished() {
			endedAt := time.Now()
			record.EndedAt = &endedAt
		}
	}
	return nil
}

// collectHelmBuildExtras 从蓝盾 Variables 中提取 Chart 构建关心的额外信息
// 仅取常用字段，不存在时填空字符串以保证字段稳定
func collectHelmBuildExtras(variables map[string]string) map[string]string {
	extras := make(map[string]string, len(pipelinevar.RequiredVariables))
	for _, n := range pipelinevar.RequiredVariables {
		extras[n] = variables[n]
	}
	return extras
}
