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

package appmodel

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
	k8sworkload "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/workload"
)

// DeployState 部署状态
type DeployState struct {
	// Status 部署的总状态
	Status Status
	// Message 部署的信息（对状态的补充说明）
	Message string
	// BatchProgress 分批发布进度，仅分批发布记录在读取到 Rollout 状态时有值
	BatchProgress *BatchProgress
}

// DeployStateGetter 部署状态获取器，用于获取指定的部署记录的状态
type DeployStateGetter struct {
	// clusterID 集群 ID
	clusterID string
	// namespace 命名空间
	namespace string
	// resourceKeys 管理的资源信息
	resourceKeys ResourceKeys
	// workloadKind 主工作负载类型
	workloadKind string
	// releaseMode 发布方式
	releaseMode ReleaseMode
	// imageTag 本次部署的镜像标签
	imageTag string
	// oldImageTag 分批发布前的镜像标签，创建部署记录时写入初始进度
	oldImageTag string
}

// NewDeployStateGetter 创建部署状态获取器
func NewDeployStateGetter(record *Record) *DeployStateGetter {
	kind, _ := record.MainWorkload()
	getter := &DeployStateGetter{
		clusterID:    record.ClusterID,
		namespace:    record.Namespace,
		resourceKeys: record.ResourceKeys,
		workloadKind: kind,
		releaseMode:  record.GetReleaseMode(),
		imageTag:     record.ImageTag,
	}
	if record.BatchProgress != nil {
		getter.oldImageTag = record.BatchProgress.OldImageTag
	}
	return getter
}

// Get 获取部署状态
func (g *DeployStateGetter) Get(ctx context.Context) (*DeployState, error) {
	// 1. 检查依赖资源状态
	depStatus, depMsg, err := g.checkDependentResources(ctx)
	if err != nil {
		// 网络波动或临时错误，返回错误让上层重试
		return nil, errors.Wrapf(err, "check dependent resources")
	}
	// 如果依赖资源检查失败（资源被删除等真正的失败），快速返回
	if depStatus == StatusFailed {
		return &DeployState{Status: StatusFailed, Message: depMsg}, nil
	}

	// 分批发布由 Rollout 控制器推进，工作负载会长期停留在部分实例更新的状态，不能按健康状态判断
	if g.releaseMode == ReleaseModeCanary {
		return g.getBatchReleaseState(ctx)
	}

	// 2. 检查主要工作负载资源的状态
	healthResult, err := g.getWorkloadHealth(ctx)
	if err != nil {
		// 网络波动或临时错误，返回错误让上层重试
		return nil, errors.Wrapf(err, "get workload health")
	}

	// 3. 根据健康状态判断部署状态
	var status Status
	switch healthResult.Code {
	case k8sstatus.Healthy, k8sstatus.Available:
		// 资源健康，表示应用部署成功
		status = StatusDeployed
	case k8sstatus.Degraded, k8sstatus.Missing:
		// 资源降级或缺失，表示应用部署真正失败
		status = StatusFailed
	case k8sstatus.Progressing, k8sstatus.Unknown, k8sstatus.Suspended:
		// 进行中或未知状态，继续等待；挂起状态，视为部署中（等待恢复）
		status = StatusDeploying
	default:
		// 未知的健康状态码，视为部署中
		status = StatusDeploying
	}

	return &DeployState{Status: status, Message: healthResult.Message}, nil
}

// checkDependentResources 检查依赖资源状态
// 返回值：(状态, 消息, 错误)
// - 如果返回 error != nil，表示网络波动或临时错误，上层应重试
// - 如果返回 StatusFailed，表示资源真正缺失（如被删除），这是确定的失败状态
// - 如果返回 StatusDeploying，表示资源存在，继续检查工作负载状态
func (g *DeployStateGetter) checkDependentResources(ctx context.Context) (Status, string, error) {
	clusterCfg := cluster.NewConfig(g.clusterID)

	// 逐个检查依赖的资源是否存在
	for _, key := range g.resourceKeys {
		// 主工作负载后续单独检查
		if k8sworkload.IsMainKind(key.Kind) {
			continue
		}

		resGVR, err := discovery.GetGroupVersionResource(clusterCfg, key.Kind, "")
		if err != nil {
			// GVR 获取失败可能是临时错误，返回 error 让上层重试
			return "", "", errors.Wrapf(err, "get GVR for kind %s", key.Kind)
		}

		client := k8sclient.NewWithGVR(clusterCfg, *resGVR)
		if _, err = client.Get(ctx, g.namespace, key.Name, metav1.GetOptions{}); err != nil {
			// 资源不存在（如已被删除），这是确定的失败状态
			if errors.Is(err, k8sclient.ErrResourceNotFound) {
				return StatusFailed, fmt.Sprintf("dependent resource %s is missing", key), nil
			}
			// 其他错误（网络超时、权限问题等）可能是临时的，返回 error 让上层重试
			return "", "", errors.Wrapf(err, "get resource %s", key)
		}
	}

	// 所有依赖资源都存在，继续检查工作负载状态
	return StatusDeploying, "", nil
}

// getWorkloadHealth 获取主要工作负载健康状态
func (g *DeployStateGetter) getWorkloadHealth(ctx context.Context) (*k8sstatus.Result, error) {
	kind, resName := g.getMainWorkload()
	if resName == "" {
		return nil, errors.New("main workload is not managed by this deploy")
	}

	driver, err := k8sworkload.Get(kind)
	if err != nil {
		return nil, errors.Wrap(err, "get workload driver")
	}

	clusterCfg := cluster.NewConfig(g.clusterID)
	client := k8sclient.NewWithGVR(clusterCfg, driver.GVR())
	res, err := client.Get(ctx, g.namespace, resName, metav1.GetOptions{})
	if err != nil {
		if errors.Is(err, k8sclient.ErrResourceNotFound) {
			return &k8sstatus.Result{
				Code:    k8sstatus.Missing,
				Message: fmt.Sprintf("%s %s is missing", kind, resName),
			}, nil
		}
		return nil, errors.Wrapf(err, "get %s %s", kind, resName)
	}

	return driver.ParseStatus(res.Object, k8sworkload.ParseOptions{Federation: clusterCfg.IsFederation()})
}

// getBatchReleaseState 读取与主工作负载（GameDeployment）同名的 Rollout，得到分批发布的状态与进度：
// 回滚完成为已回滚，批次状态为已完成时为已部署，其余仍为部署中。
// 主工作负载不存在视为部署失败；读取 Rollout 失败（含被删除）返回错误，由上层保留上一次进度并重试
func (g *DeployStateGetter) getBatchReleaseState(ctx context.Context) (*DeployState, error) {
	kind, name := g.getMainWorkload()
	if kind != k8skind.GameDeploy || name == "" {
		return nil, errors.Errorf("main workload %s/%s does not support batch release", kind, name)
	}

	clusterCfg := cluster.NewConfig(g.clusterID)
	workload, err := k8sclient.NewWithGVR(clusterCfg, gvr.GameDeploy).Get(ctx, g.namespace, name, metav1.GetOptions{})
	if errors.Is(err, k8sclient.ErrResourceNotFound) {
		return &DeployState{Status: StatusFailed, Message: fmt.Sprintf("%s %s is missing", kind, name)}, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get %s %s", kind, name)
	}
	snapshot, err := batchrelease.ParseWorkloadSnapshot(workload.Object)
	if err != nil {
		return nil, errors.Wrapf(err, "%s %s", kind, name)
	}
	status, err := batchrelease.NewRolloutClient(clusterCfg).GetStatus(ctx, g.namespace, name)
	if err != nil {
		return nil, errors.Wrap(err, "get rollout status")
	}

	progress := &BatchProgress{
		CurrentBatchIndex: status.CurrentStepIndex,
		BatchPhase:        status.BatchPhase(snapshot),
		RolloutPhase:      status.Phase,
		Message:           status.ProgressMessage(),
		PauseStartTime:    status.PauseStartTime,
		UpdatedReplicas:   int32(snapshot.UpdatedReplicas), //nolint:gosec // G115: 副本数不会超过 int32
		Replicas:          int32(snapshot.DesiredReplicas), //nolint:gosec // G115: 副本数不会超过 int32
		OldImageTag:       g.oldImageTag,
		NewImageTag:       g.imageTag,
	}
	state := &DeployState{Status: StatusDeploying, Message: progress.Message, BatchProgress: progress}
	switch {
	case status.IsRolledBack(snapshot):
		state.Status = StatusRolledBack
	case progress.BatchPhase == batchrelease.BatchPhaseCompleted:
		state.Status = StatusDeployed
	}
	return state, nil
}

// getMainWorkload 获取主工作负载 Kind 与名称
func (g *DeployStateGetter) getMainWorkload() (kind, name string) {
	if g.workloadKind == "" {
		return "", ""
	}
	return g.workloadKind, g.resourceKeys.NameByKind(g.workloadKind)
}
