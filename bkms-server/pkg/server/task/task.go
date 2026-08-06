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

// Package task provides task implementation and collection.
package task

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/worker"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

// 任务名称
const (
	// PollingBuildStatus 轮询蓝盾流水线构建状态
	PollingBuildStatus = "PollingBuildStatus"
	// PollingTrpcDeployStatus 轮询 Trpc 应用部署状态
	PollingTrpcDeployStatus = "PollingTrpcDeployStatus"
	// PollingHelmDeployStatus 轮询 Helm 应用部署状态
	PollingHelmDeployStatus = "PollingHelmDeployStatus"
	// PollingWorkspaceInitStatus 轮询工作空间状态
	PollingWorkspaceInitStatus = "PollingWorkspaceInitStatus"
	// PollingHelmChartBuildStatus 轮询 Helm Chart 构建状态
	PollingHelmChartBuildStatus = "PollingHelmChartBuildStatus"
)

// EmptyResult 空返回，适用于任务函数无需返回数据的情况
type EmptyResult struct{}

var emptyResult = EmptyResult{}

func init() {
	// 任务注册
	worker.RegisterTask[PollingBuildStatusArgs, *EmptyResult](
		PollingBuildStatus, pollingBuildStatus,
	)
	worker.RegisterTask[PollingHelmDeployStatusArgs, *EmptyResult](
		PollingHelmDeployStatus, pollingHelmDeployStatus,
	)
	worker.RegisterTask[PollingTrpcDeployStatusArgs, *EmptyResult](
		PollingTrpcDeployStatus, pollingTrpcDeployStatus,
	)
	worker.RegisterTask[PollingWorkspaceInitStatusArgs, *EmptyResult](
		PollingWorkspaceInitStatus, pollingWorkspaceInitStatus,
	)
	worker.RegisterTask[snapshot.ImageDetailSyncArgs, *EmptyResult](
		snapshot.TaskImageDetailSync, imageDetailSync,
	)
	worker.RegisterTask[PollingHelmChartBuildStatusArgs, *EmptyResult](
		PollingHelmChartBuildStatus, pollingHelmChartBuildStatus,
	)
}
