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

package status

import (
	"context"
	"time"

	helmrelease "helm.sh/helm/v3/pkg/release"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	buildimage "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
)

// DeploySource 最新一次部署来自哪类记录
type DeploySource string

const (
	// DeploySourceAppModel 直接发起的 AppModel 部署
	DeploySourceAppModel DeploySource = "appmodel"
	// DeploySourceHelm Helm 部署
	DeploySourceHelm DeploySource = "helm"
	// DeploySourceBuildAutoDeploy 一键构建部署，可能停在构建阶段，也可能已进入部署阶段
	DeploySourceBuildAutoDeploy DeploySource = "buildAutoDeploy"
)

// DeployPhase 各类记录的状态取值不统一（如 polling-timeout 与 pollingTimeout），这里归一成阶段
type DeployPhase string

const (
	// DeployPhaseInProgress 构建或部署仍在进行
	DeployPhaseInProgress DeployPhase = "inProgress"
	// DeployPhaseSucceeded 部署成功
	DeployPhaseSucceeded DeployPhase = "succeeded"
	// DeployPhaseFailed 构建或部署失败，含轮询超时、轮询中断
	DeployPhaseFailed DeployPhase = "failed"
	// DeployPhaseOther 已取消、已下线、已回滚、未知等，不归入以上三类
	DeployPhaseOther DeployPhase = "other"
)

// DeployTarget 运行中实例所在的集群位置，取自最近一次 AppModel / Helm 部署记录
type DeployTarget struct {
	// DeployID 部署记录 ID，可用于查看该次部署的资源清单快照
	DeployID string
	// ClusterID 部署所在集群
	ClusterID string
	// Namespace 部署所在命名空间
	Namespace string
	// LabelSelector 筛选该次部署 Pod 的标签；Helm 记录没有，为空
	LabelSelector map[string]string
	// ResourceKeys 该次部署创建的工作负载；Helm 记录没有，为空
	ResourceKeys []appmodel.ResourceKey
	// StartedAt 该次部署开始时间
	StartedAt time.Time
}

// LatestDeploy 应用在某环境、泳道下最新一次部署的状态，以及其运行实例的位置
type LatestDeploy struct {
	// Source 最新那条记录的类型
	Source DeploySource
	// Stage 一键构建部署所处阶段；其他来源为空
	Stage autodeploy.Stage
	// BuildID 一键构建部署的构建 ID；其他来源为空
	BuildID string
	// Status 最新那条记录上的原始状态
	Status string
	// Phase 归一后的阶段，判断成功、失败、进行中时用它而不是 Status
	Phase DeployPhase
	// Message 最新那条记录上的说明，失败时通常是失败原因
	Message string
	// StartedAt 最新那条记录的开始时间
	StartedAt time.Time
	// Target 运行实例所在位置。一键构建停在构建阶段时沿用上一次部署记录；从未部署过时为 nil
	Target *DeployTarget
}

// GetLatestDeploy 获取应用在指定环境、泳道下的最新一次部署，无任何记录时返回 ErrDeployRecordNotFound
//
// 与 GetLatestDeployStatus 同一口径：AppModel 应用会把一键构建部署记录也纳入比较，
// 因此构建阶段就失败的情况也能被看到
func (s *DeployStatusService) GetLatestDeploy(
	ctx context.Context,
	application *app.Application,
	envName, laneName string,
) (*LatestDeploy, error) {
	switch {
	case app.IsAppModelType(application.Type):
		return s.getLatestAppModelDeploy(ctx, application.ID, envName, laneName)
	case app.IsHelmBasedType(application.Type):
		return s.getLatestHelmDeploy(ctx, application.ID, envName, laneName)
	default:
		return nil, ErrUnsupportedAppType
	}
}

func (s *DeployStatusService) getLatestAppModelDeploy(
	ctx context.Context,
	appID, envName, laneName string,
) (*LatestDeploy, error) {
	buildRecord, err := s.getLatestBuildAutoDeployRecord(ctx, appID, envName, laneName)
	if err != nil {
		return nil, err
	}
	deployRecord, err := s.getLatestAppModelDeployRecord(ctx, appID, envName, laneName)
	if err != nil {
		return nil, err
	}
	if buildRecord == nil && deployRecord == nil {
		return nil, ErrDeployRecordNotFound
	}

	// 实例位置只能来自部署记录：构建阶段的记录还没有产出任何工作负载
	var target *DeployTarget
	if deployRecord != nil {
		target = &DeployTarget{
			DeployID:      deployRecord.ID.Hex(),
			ClusterID:     deployRecord.ClusterID,
			Namespace:     deployRecord.Namespace,
			LabelSelector: deployRecord.LabelSelector,
			ResourceKeys:  deployRecord.ResourceKeys,
			StartedAt:     deployRecord.StartedAt,
		}
	}

	if isBuildRecordLatest(buildRecord, deployRecord) {
		return &LatestDeploy{
			Source:    DeploySourceBuildAutoDeploy,
			Stage:     buildRecord.Stage,
			BuildID:   buildRecord.BuildID,
			Status:    buildRecord.Status,
			Phase:     resolveBuildAutoDeployPhase(buildRecord.Stage, buildRecord.Status),
			Message:   buildRecord.Message,
			StartedAt: buildRecord.StartedAt,
			Target:    target,
		}, nil
	}
	return &LatestDeploy{
		Source:    DeploySourceAppModel,
		Status:    string(deployRecord.Status),
		Phase:     resolveAppModelPhase(deployRecord.Status),
		Message:   deployRecord.Message,
		StartedAt: deployRecord.StartedAt,
		Target:    target,
	}, nil
}

func (s *DeployStatusService) getLatestHelmDeploy(
	ctx context.Context,
	appID, envName, laneName string,
) (*LatestDeploy, error) {
	record, err := s.getLatestHelmDeployRecord(ctx, appID, envName, laneName)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrDeployRecordNotFound
	}
	return &LatestDeploy{
		Source:    DeploySourceHelm,
		Status:    string(record.Status),
		Phase:     resolveHelmPhase(record.Status),
		Message:   record.Message,
		StartedAt: record.StartedAt,
		Target: &DeployTarget{
			DeployID:  record.ID.Hex(),
			ClusterID: record.ClusterID,
			Namespace: record.Namespace,
			StartedAt: record.StartedAt,
		},
	}, nil
}

// isBuildRecordLatest 判断一键构建部署记录是否比 AppModel 部署记录更新
// 两者关联同一次部署时以构建部署记录为准，与 selectLatestStatus 的取舍一致
func isBuildRecordLatest(buildRecord *autodeploy.Record, deployRecord *appmodel.Record) bool {
	if buildRecord == nil {
		return false
	}
	if deployRecord == nil {
		return true
	}
	if buildRecord.DeployID != "" && buildRecord.DeployID == deployRecord.ID.Hex() {
		return true
	}
	return !buildRecord.CreatedAt.Before(deployRecord.CreatedAt)
}

// resolveBuildAutoDeployPhase 构建阶段沿用构建记录的状态取值，进入部署阶段后与 AppModel 部署状态同步
func resolveBuildAutoDeployPhase(stage autodeploy.Stage, status string) DeployPhase {
	if stage == autodeploy.StageDeploy {
		return resolveAppModelPhase(appmodel.Status(status))
	}
	switch buildimage.Status(status) {
	// 构建成功后还要触发部署，在进入部署阶段前都算进行中
	case buildimage.StatusRunning, buildimage.StatusSuccess:
		return DeployPhaseInProgress
	case buildimage.StatusFailed, buildimage.StatusPollingTimeout, buildimage.StatusPollingBroken:
		return DeployPhaseFailed
	default:
		return DeployPhaseOther
	}
}

func resolveAppModelPhase(status appmodel.Status) DeployPhase {
	switch status {
	case appmodel.StatusDeploying:
		return DeployPhaseInProgress
	case appmodel.StatusDeployed:
		return DeployPhaseSucceeded
	case appmodel.StatusFailed, appmodel.StatusPollingTimeout, appmodel.StatusPollingBroken:
		return DeployPhaseFailed
	default:
		return DeployPhaseOther
	}
}

func resolveHelmPhase(status helmrelease.Status) DeployPhase {
	switch status {
	case helmrelease.StatusPendingInstall, helmrelease.StatusPendingUpgrade, helmrelease.StatusPendingRollback:
		return DeployPhaseInProgress
	case helmrelease.StatusDeployed:
		return DeployPhaseSucceeded
	case helmrelease.StatusFailed:
		return DeployPhaseFailed
	default:
		return DeployPhaseOther
	}
}
