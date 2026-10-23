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
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// 等待 Rollout 被控制器调谐、等待 Rollout 删除完成的超时时间与轮询间隔
var (
	rolloutWaitTimeout  = 30 * time.Second
	rolloutWaitInterval = time.Second
)

// workloadRollout 一次部署中对主工作负载同名 Rollout CR 的处理，在更新主工作负载前后分别调用
// EnsureRollout 与 EnsurePartition：
//
//   - 按策略分批：更新前下发 Rollout 并为工作负载打上 rollout 标签（控制器缺少该标签时会直接整批更新），
//     更新后校验 webhook 已注入 partition；
//   - 一次更新全部：更新前删除遗留的 Rollout 并移除标签，否则仍会按旧策略分批；更新后放开可能残留的 partition。
//
// 联邦集群的主工作负载是 Deployment，不涉及 Rollout，两个阶段都不做任何处理。
type workloadRollout struct {
	// manager 为 nil 表示主工作负载不涉及 Rollout
	manager   *rolloutManager
	namespace string
	name      string
	// manifest 为 nil 表示一次更新全部实例
	manifest map[string]any

	// 由 EnsureRollout 根据集群现状设置：按策略分批且工作负载已存在时，EnsurePartition 需校验 webhook 已注入 partition
	verifyPartition bool
}

// newWorkloadRollout 创建主工作负载的 Rollout 处理，strategy 为 nil 时一次更新全部实例。
// 主工作负载不支持分批发布时返回 batchrelease.ErrWorkloadNotSupported。
func newWorkloadRollout(
	cfg *cluster.Config,
	namespace, workloadKind, workloadName string,
	strategy *releasestrategy.ReleaseStrategy,
) (*workloadRollout, error) {
	r := &workloadRollout{namespace: namespace, name: workloadName}
	if strategy != nil {
		manifest, err := batchrelease.BuildManifest(workloadKind, workloadName, strategy.Batches)
		if err != nil {
			return nil, errors.Wrap(err, "build rollout manifest")
		}
		r.manifest = manifest
	}
	if workloadKind == k8skind.GameDeploy {
		r.manager = newRolloutManager(cfg)
	}
	return r, nil
}

// EnsureRollout 确保集群中与主工作负载同名的 Rollout 符合本次的发布方式，必须在更新主工作负载之前调用：
// webhook 在工作负载更新时才查找 Rollout，此时 Rollout 的存在与否决定了本次是否分批。
//
//   - 按策略分批：下发 Rollout，工作负载已存在时等待控制器调谐完成，并为 meta（主工作负载的 ObjectMeta）
//     加上 rollout 标签；
//   - 一次更新全部：删除遗留的 Rollout 并等待删除完成，移除工作负载上的 rollout 标签。
func (r *workloadRollout) EnsureRollout(ctx context.Context, meta *metav1.ObjectMeta) error {
	if r.manager == nil {
		return nil
	}
	if r.manifest == nil {
		return errors.Wrap(r.manager.Cleanup(ctx, r.namespace, r.name), "clean up rollout")
	}

	workloadExisted, err := r.manager.Apply(ctx, r.namespace, r.manifest)
	if err != nil {
		return errors.Wrap(err, "apply rollout")
	}
	if meta.Labels == nil {
		meta.Labels = map[string]string{}
	}
	meta.Labels[batchrelease.WorkloadLabelKey] = batchrelease.WorkloadLabelValue
	r.verifyPartition = workloadExisted
	return nil
}

// EnsurePartition 在更新主工作负载后，确保其 partition 符合本次的发布方式。
// live 为更新请求返回的工作负载，已包含 mutating webhook 的修改。
//
// Pod 模板变化时，Kourse webhook 查找指向工作负载的 Rollout：找到则注入 partition = replicas，
// 由控制器逐批调低；找不到则直接放行、整批更新。webhook 读的是缓存，Rollout 刚创建或刚删除时
// 缓存可能尚未同步，因此分两种情况处理：
//
//   - 按策略分批且工作负载已存在：校验已注入 partition = replicas，否则返回错误。未注入时工作负载
//     已整批更新，而 Rollout 照样会进入 Healthy，不能当作分批发布继续轮询；
//   - 一次更新全部：webhook 可能仍按旧缓存注入了 partition，而 Rollout 删除后控制器不会再修改 partition，
//     实例会一直停留在旧版本，因此总是将 partition 恢复为 0（主工作负载 manifest 中的值）。不以本次是否删除了
//     Rollout 为条件：删除后的步骤失败时，重试已看不到遗留的 Rollout。
//
// 按策略分批但工作负载首次创建时没有旧版本实例，主工作负载不是 GameDeployment 时 webhook 不会介入，均不做处理。
func (r *workloadRollout) EnsurePartition(ctx context.Context, live *unstructured.Unstructured) error {
	switch {
	case r.manager == nil:
		return nil
	case r.manifest == nil:
		return r.manager.RestorePartition(ctx, live, 0)
	case r.verifyPartition:
		return errors.Wrap(verifyPartitionInjected(live), "verify batch release")
	default:
		return nil
	}
}

// SnapshotObjects 返回需要保存资源快照的额外资源：分批发布时追加 Rollout CR，不修改 extraObjs
func (r *workloadRollout) SnapshotObjects(extraObjs []unstructured.Unstructured) []unstructured.Unstructured {
	if r.manifest == nil {
		return extraObjs
	}
	// 将容量收束到长度，避免 append 写入 extraObjs 的底层数组
	clipped := extraObjs[:len(extraObjs):len(extraObjs)]
	return append(clipped, unstructured.Unstructured{Object: r.manifest})
}

// deleteRollout 删除部署记录中主工作负载同名的 Rollout CR；主工作负载不是 GameDeployment 时不会有 Rollout
func deleteRollout(ctx context.Context, cfg *cluster.Config, record *Record) error {
	kind, name := record.MainWorkload()
	if kind != k8skind.GameDeploy || name == "" {
		return nil
	}
	return batchrelease.NewRolloutClient(cfg).Delete(ctx, record.Namespace, name)
}

// rolloutManager 在更新主工作负载（GameDeployment，仅非联邦集群）前后处理与其同名的 Rollout CR
type rolloutManager struct {
	clusterID string
	rollouts  *batchrelease.RolloutClient
	workloads *k8sclient.Client
}

func newRolloutManager(cfg *cluster.Config) *rolloutManager {
	return &rolloutManager{
		clusterID: cfg.ClusterID,
		rollouts:  batchrelease.NewRolloutClient(cfg),
		workloads: k8sclient.NewWithGVR(cfg, gvr.GameDeploy),
	}
}

// Apply 下发 Rollout CR，返回下发前主工作负载是否已存在。
//
// 工作负载已存在时，等待控制器调谐过该 Rollout 的当前 spec 后才返回：
//   - status.workloadRevision 已写入：webhook 与控制器共用缓存，缓存中没有 Rollout 时更新工作负载
//     不会注入 partition，会直接整批更新；
//   - 本次修改了已有 Rollout 的 spec 时，status.rolloutRevision 已离开修改前的值：遗留的 Rollout 早已写入
//     workloadRevision，不能据此判断。控制器若先看到工作负载版本变化，会按旧批次开始一轮，
//     随后发现 spec 变化再从第 0 批重来，旧批次已更新的实例会被白白更新一次再退回。
//
// 工作负载首次创建时不存在旧版本实例，无需分批，也不等待（Rollout 指向不存在的工作负载时控制器不写入版本）。
func (m *rolloutManager) Apply(ctx context.Context, namespace string, manifest map[string]any) (bool, error) {
	name, _, _ := unstructured.NestedString(manifest, "metadata", "name")
	workloadExists, err := m.workloadExists(ctx, namespace, name)
	if err != nil {
		return false, err
	}
	if !workloadExists {
		if _, err = m.rollouts.Apply(ctx, namespace, manifest); err != nil {
			return false, err
		}
		return false, nil
	}

	prev, err := m.rollouts.Get(ctx, namespace, name)
	if err != nil && !errors.Is(err, batchrelease.ErrRolloutNotFound) {
		return false, err
	}
	var prevStatus batchrelease.Status
	if prev != nil {
		if prevStatus, err = batchrelease.ParseStatus(prev.Object); err != nil {
			return false, errors.Wrapf(err, "rollout %s/%s in cluster %s", namespace, name, m.clusterID)
		}
	}

	applied, err := m.rollouts.Apply(ctx, namespace, manifest)
	if err != nil {
		return false, err
	}
	specChanged := prev != nil && applied.GetGeneration() != prev.GetGeneration()

	err = wait.PollUntilContextTimeout(
		ctx, rolloutWaitInterval, rolloutWaitTimeout, true,
		func(ctx context.Context) (bool, error) {
			status, getErr := m.rollouts.GetStatus(ctx, namespace, name)
			if getErr != nil {
				return false, getErr
			}
			if status.WorkloadRevision == "" {
				return false, nil
			}
			return !specChanged || status.RolloutRevision != prevStatus.RolloutRevision, nil
		},
	)
	if err != nil {
		return false, errors.Wrapf(
			err, "wait rollout %s/%s to be reconciled in cluster %s", namespace, name, m.clusterID,
		)
	}
	return true, nil
}

// Cleanup 删除主工作负载遗留的同名 Rollout CR 并移除其 rollout 标签。
// 集群未安装 Rollout CRD 时视为没有遗留。
func (m *rolloutManager) Cleanup(ctx context.Context, namespace, name string) error {
	_, err := m.rollouts.Get(ctx, namespace, name)
	switch {
	case err == nil:
		if err = m.rollouts.Delete(ctx, namespace, name); err != nil {
			return err
		}
		if err = m.waitRolloutDeleted(ctx, namespace, name); err != nil {
			return err
		}
	case errors.Is(err, batchrelease.ErrRolloutNotFound), errors.Is(err, batchrelease.ErrRolloutNotInstalled):
	default:
		return err
	}

	workload, err := m.workloads.Get(ctx, namespace, name, metav1.GetOptions{})
	if errors.Is(err, k8sclient.ErrResourceNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if workload.GetLabels()[batchrelease.WorkloadLabelKey] != batchrelease.WorkloadLabelValue {
		return nil
	}
	patch := []map[string]any{{"op": "remove", "path": "/metadata/labels/" + batchrelease.WorkloadLabelKey}}
	return errors.Wrap(m.patchWorkload(ctx, namespace, name, patch), "remove rollout label")
}

// RestorePartition 将工作负载的 partition 恢复为本次更新期望的值 want，workload 为更新请求返回的工作负载，
// 与期望值相同时不访问集群。
//
// 用于清理 Rollout 后修改 Pod 模板：webhook 在更新请求中、patch 应用之后执行，其缓存可能尚未同步
// Rollout 的删除，仍会注入 partition = replicas，覆盖请求中写入的值；而 Rollout 删除后控制器不会再修改
// partition（Rollout 没有 finalizer），实例会一直停留在旧版本。
//
// 纠正一次即可：纠正的 patch 不修改 Pod 模板，webhook 不会介入。
func (m *rolloutManager) RestorePartition(
	ctx context.Context, workload *unstructured.Unstructured, want int64,
) error {
	_, partition, err := replicasAndPartition(workload.Object)
	if err != nil {
		return err
	}
	if partition == want {
		return nil
	}
	log.Warnf(
		ctx, "workload %s/%s in cluster %s has partition %d after update, restore to %d",
		workload.GetNamespace(), workload.GetName(), m.clusterID, partition, want,
	)
	patch := []map[string]any{{"op": "replace", "path": "/spec/updateStrategy/partition", "value": want}}
	if err = m.patchWorkload(ctx, workload.GetNamespace(), workload.GetName(), patch); err != nil {
		return errors.Wrap(err, "restore partition")
	}
	return nil
}

func (m *rolloutManager) workloadExists(ctx context.Context, namespace, name string) (bool, error) {
	_, err := m.workloads.Get(ctx, namespace, name, metav1.GetOptions{})
	if errors.Is(err, k8sclient.ErrResourceNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *rolloutManager) waitRolloutDeleted(ctx context.Context, namespace, name string) error {
	err := wait.PollUntilContextTimeout(
		ctx, rolloutWaitInterval, rolloutWaitTimeout, true,
		func(ctx context.Context) (bool, error) {
			_, getErr := m.rollouts.Get(ctx, namespace, name)
			if errors.Is(getErr, batchrelease.ErrRolloutNotFound) ||
				errors.Is(getErr, batchrelease.ErrRolloutNotInstalled) {
				return true, nil
			}
			return false, getErr
		},
	)
	if err != nil {
		return errors.Wrapf(err, "wait rollout %s/%s to be deleted in cluster %s", namespace, name, m.clusterID)
	}
	return nil
}

func (m *rolloutManager) patchWorkload(
	ctx context.Context, namespace, name string, patch []map[string]any,
) error {
	data, err := json.Marshal(patch)
	if err != nil {
		return errors.Wrap(err, "marshal workload patch")
	}
	_, err = m.workloads.Patch(ctx, namespace, name, types.JSONPatchType, data, metav1.PatchOptions{})
	return err
}

// verifyPartitionInjected 校验分批部署更新后的工作负载已被 webhook 注入 partition = replicas。
// 未注入说明 webhook 没有找到 Rollout，工作负载会整批更新，不能当作分批发布继续。
func verifyPartitionInjected(workload *unstructured.Unstructured) error {
	replicas, partition, err := replicasAndPartition(workload.Object)
	if err != nil {
		return err
	}
	if partition != replicas {
		return errors.Errorf(
			"partition of workload %s/%s is %d rather than replicas %d, rollout webhook did not take effect",
			workload.GetNamespace(), workload.GetName(), partition, replicas,
		)
	}
	return nil
}

// replicasAndPartition 读取工作负载的 spec.replicas 与 spec.updateStrategy.partition，缺失时为 0
func replicasAndPartition(workload map[string]any) (replicas, partition int64, err error) {
	if replicas, _, err = unstructured.NestedInt64(workload, "spec", "replicas"); err != nil {
		return 0, 0, errors.Wrap(err, "parse workload replicas")
	}
	if partition, _, err = unstructured.NestedInt64(workload, "spec", "updateStrategy", "partition"); err != nil {
		return 0, 0, errors.Wrap(err, "parse workload partition")
	}
	return replicas, partition, nil
}
