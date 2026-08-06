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

package tagdeletion

import (
	"context"

	"github.com/samber/lo"
	helmrelease "helm.sh/helm/v3/pkg/release"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	infrahelm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	trafficMgr "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
)

// -----------------------------------------------------------------------------
// 最新部署记录占用判断辅助函数
// -----------------------------------------------------------------------------

// collectLatestAppModelLaneUsages 从单条 appmodel 部署记录中判断目标 tag 是否仍在使用，
// 若仍在使用则返回该记录对应的 workload 名称列表与部署状态
// 仅当记录状态为 deployed 或正在进行中的部署状态时，才认为 tag 仍被占用
func collectLatestAppModelLaneUsages(
	record *appmodeldeploy.Record,
	tag string,
) []ImageUsage {
	if record == nil || record.ImageTag != tag {
		return nil
	}
	if record.Status == appmodeldeploy.StatusDeployed || isAppModelOngoingDeployStatus(record.Status) {
		workloadNames := collectAppModelWorkloadNames(record.ResourceKeys)
		usages := make([]ImageUsage, 0, len(workloadNames))
		for _, workloadName := range workloadNames {
			usages = append(usages, ImageUsage{
				WorkloadName: workloadName,
				Status:       string(record.Status),
			})
		}
		return usages
	}
	return nil
}

// collectLatestHelmLaneUsages 从单条 Helm 部署记录中判断目标 tag 是否仍在使用，
// 若仍在使用则返回该 release 对应的 workload 名称（即 ReleaseName）与部署状态
// 仅当记录状态为 deployed 或正在进行中的发布状态时，才认为 tag 仍被占用
func collectLatestHelmLaneUsages(
	record *helmdeploy.Record,
	tag string,
) []ImageUsage {
	if record == nil || record.ImageTag != tag || record.ReleaseName == "" {
		return nil
	}
	if record.Status == helmrelease.StatusDeployed || isHelmOngoingDeployStatus(record.Status) {
		return []ImageUsage{{
			WorkloadName: record.ReleaseName,
			Status:       string(record.Status),
		}}
	}
	return nil
}

// -----------------------------------------------------------------------------
// 部署状态判断辅助函数
// -----------------------------------------------------------------------------

// isAppModelOngoingDeployStatus 标记仍在处理中的状态
func isAppModelOngoingDeployStatus(status appmodeldeploy.Status) bool {
	switch status {
	case appmodeldeploy.StatusDeploying, appmodeldeploy.StatusPollingTimeout, appmodeldeploy.StatusPollingBroken:
		return true
	default:
		return false
	}
}

// shouldFallbackAppModel 判断是否需要回退检查最近一次成功部署记录
func shouldFallbackAppModel(record *appmodeldeploy.Record) bool {
	if record == nil {
		return false
	}
	if record.Status == appmodeldeploy.StatusDeployed || isAppModelOngoingDeployStatus(record.Status) {
		return false
	}
	return !record.Status.IsUninstall()
}

// isHelmOngoingDeployStatus 标记仍在处理中的状态
func isHelmOngoingDeployStatus(status helmrelease.Status) bool {
	switch status {
	case helmrelease.StatusPendingInstall,
		helmrelease.StatusPendingUpgrade,
		helmrelease.StatusPendingRollback,
		infrahelm.StatusPollingTimeout,
		infrahelm.StatusPollingBroken:
		return true
	default:
		return false
	}
}

// shouldFallbackHelm 活跃态和卸载态不回退；失败类非活跃态需要回退，
// 因为最近一次成功部署仍可能在继续使用
func shouldFallbackHelm(record *helmdeploy.Record) bool {
	if record == nil {
		return false
	}
	if record.Status == helmrelease.StatusDeployed || isHelmOngoingDeployStatus(record.Status) {
		return false
	}
	// 卸载态明确表示 release 已删除，不应被历史成功记录“复活”为仍在使用
	return record.Status != helmrelease.StatusUninstalling && record.Status != helmrelease.StatusUninstalled
}

// -----------------------------------------------------------------------------
// 流量泳道辅助函数
// -----------------------------------------------------------------------------

// listLaneNames 返回指定环境下的所有流量泳道名称，
// 并始终包含用空字符串表示的基线泳道。
func listLaneNames(
	ctx context.Context,
	tm trafficMgr.TrafficManager,
	workspaceID, envName string,
) ([]string, error) {
	lanes, err := tm.ListTrafficLanes(ctx, workspaceID, envName)
	if err != nil {
		return nil, err
	}
	laneNames := []string{""}
	for _, lane := range lanes {
		if lane == nil || lane.LaneName == "" {
			continue
		}
		laneNames = append(laneNames, lane.LaneName)
	}
	return laneNames, nil
}

// -----------------------------------------------------------------------------
// 工作负载辅助函数
// -----------------------------------------------------------------------------

// collectAppModelWorkloadNames 从 appmodel 的资源列表中提取 workload 名称并去重，
// 忽略 service、secret 等非 workload 资源。
func collectAppModelWorkloadNames(resourceKeys appmodeldeploy.ResourceKeys) []string {
	names := make([]string, 0, len(resourceKeys))
	for _, key := range resourceKeys {
		if !k8skind.IsWorkload(key.Kind) {
			continue
		}
		names = append(names, key.Name)
	}
	return lo.Uniq(names)
}
