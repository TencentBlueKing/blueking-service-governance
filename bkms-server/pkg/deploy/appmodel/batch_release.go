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

	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
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

// workloadRollout 负责处理部署中的 Rollout CR
type workloadRollout struct {
	// manager 为 nil 时，主工作负载不涉及 Rollout
	manager   *rolloutManager
	namespace string
	name      string
	// manifest 为 nil 时，一次更新全部实例
	manifest map[string]any

	// verifyPartition 需要校验 webhook 已注入 partition，从而确保 workload 已处于 Rollout 控制中
	verifyPartition bool
	// applyAfterUpdate 遗留的 Rollout 有未结束的发布，需要在工作负载更新后下发
	applyAfterUpdate bool
}

// newWorkloadRollout ...
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

// SetDeployID 在待下发的 Rollout 上标记本次部署 ID，手动操作据此确认集群中的 Rollout 属于最新部署记录。
// 部署 ID 在创建部署记录后才有，所以不在 newWorkloadRollout 中生成；一次更新全部时不下发 Rollout，不做处理。
func (r *workloadRollout) SetDeployID(deployID string) error {
	if r.manifest == nil {
		return nil
	}
	return errors.Wrap(
		unstructured.SetNestedField(r.manifest, deployID, "metadata", "annotations", AnnotationKeyDeployID),
		"set deploy ID annotation of rollout",
	)
}

// BeforeWorkloadUpdate 在更新主工作负载之前调用，依据不同的发布方式确认如何下发 Rollout，并修改待下发的
// 主工作负载 main。
//
//   - 按策略分批：下发 Rollout；工作负载已存在时，等待控制器调谐完成。为 main 加上 rollout 标签。
//   - 按策略分批，且遗留的 Rollout 还有未结束的发布（发布中、回滚后）：删除 Rollout 并等待删除完成，
//     为 main 加上 rollout 标签，并把它的 partition 设为 replicas。Rollout 推迟到 AfterWorkloadUpdate 再下发，
//     原因见 shouldApplyAfterUpdate。
//   - 一次更新全部（不涉及分批发布）：删除遗留的 Rollout 并等待删除完成，移除工作负载上的 rollout 标签。
func (r *workloadRollout) BeforeWorkloadUpdate(ctx context.Context, main runtime.Object) error {
	if r.manager == nil {
		return nil
	}
	if r.manifest == nil {
		return errors.Wrap(r.manager.Cleanup(ctx, r.namespace, r.name), "clean up rollout")
	}
	gd, ok := main.(*tkex.GameDeployment)
	if !ok {
		return errors.Errorf("main workload %T is not a GameDeployment", main)
	}

	applyAfterUpdate, err := r.shouldApplyAfterUpdate(ctx)
	if err != nil {
		return err
	}
	if applyAfterUpdate {
		if gd.Spec.Replicas == nil {
			return errors.Errorf("replicas of workload %s/%s is not set", r.namespace, r.name)
		}
		if err = r.manager.delete(ctx, r.namespace, r.name); err != nil {
			return errors.Wrap(err, "delete rollout with unfinished release")
		}
		gd.Spec.UpdateStrategy.Partition = lo.ToPtr(intstr.FromInt32(*gd.Spec.Replicas))
		r.applyAfterUpdate = true
	} else {
		if r.verifyPartition, err = r.manager.Apply(ctx, r.namespace, r.manifest); err != nil {
			return errors.Wrap(err, "apply rollout")
		}
	}
	if gd.Labels == nil {
		gd.Labels = map[string]string{}
	}
	gd.Labels[batchrelease.WorkloadLabelKey] = batchrelease.WorkloadLabelValue
	return nil
}

// shouldApplyAfterUpdate 判断本次是否要等工作负载更新之后再下发 Rollout。
// 遗留的 Rollout 还有未结束的发布，并且工作负载已经存在时，返回 true。
//
// 发布还没结束时，工作负载的目标版本仍是上一次发布的版本。如果这时先修改 Rollout spec，控制器会抢在平台
// 更新工作负载之前，按新的批次从第 0 批重新发布上一次的版本。例如：
//
//	旧：A -> B，第 0 批 30%，暂停中（30% 实例为 B）
//	新：发布 C，第 0 批 80%
//	先下发 Rollout：控制器按新批次把 B 扩大到 80%，平台更新工作负载后这些实例再被替换为 C，
//	               用户放弃的 B 反而先被扩大
//	推迟下发：写入 partition = replicas 后 B 的实例先退回 A，下发 Rollout 后从 A 开始将 80% 实例更新为 C
//
// 回滚后同理：B 的实例已全部退回 A，先修改 Rollout spec 会重新发布被回滚的 B。
//
// 工作负载不存在时没有旧版本实例，按首次创建处理：先下发 Rollout，不写入 partition。如果也推迟下发，
// Rollout 首次调谐会把当前版本当作基线、不开始发布，平台写入的 partition = replicas 就没人调低。
func (r *workloadRollout) shouldApplyAfterUpdate(ctx context.Context) (bool, error) {
	unfinished, err := r.manager.hasUnfinishedRelease(ctx, r.namespace, r.name)
	if err != nil {
		return false, errors.Wrap(err, "check unfinished release of rollout")
	}
	if !unfinished {
		return false, nil
	}
	exists, err := r.manager.workloadExists(ctx, r.namespace, r.name)
	if err != nil {
		return false, errors.Wrap(err, "check workload exists")
	}
	return exists, nil
}

// AfterWorkloadUpdate 在更新主工作负载之后调用，确保它的 partition 符合本次的发布方式；遗留的 Rollout 还有
// 未结束的发布时，也在这里下发 Rollout。live 是更新请求返回的工作负载，已包含 mutating webhook 的修改。
//
// Pod 模板变化时，Kourse webhook 会查找指向该工作负载的 Rollout：找到就注入 partition = replicas，
// 再由控制器逐批调低；找不到就直接放行，工作负载整批更新。webhook 读的是缓存，Rollout 刚创建或刚删除时
// 缓存可能还没同步，所以不能假定 partition 就是预期的值。
func (r *workloadRollout) AfterWorkloadUpdate(ctx context.Context, live *unstructured.Unstructured) error {
	switch {
	case r.manager == nil:
		// 主工作负载不是 GameDeployment，webhook 不会介入
		return nil
	case r.manifest == nil:
		// 一次更新全部：webhook 可能仍按旧缓存注入了 partition，而 Rollout 删除后控制器不会再修改 partition，
		// 实例会一直停在旧版本，所以要恢复为 0（主工作负载 manifest 中的值）。不管本次有没有删除 Rollout 都要恢复：
		// 如果删除之后的步骤失败了，重试时已经看不到遗留的 Rollout
		return r.manager.RestorePartition(ctx, live, 0)
	case r.applyAfterUpdate:
		// 遗留的 Rollout 还有未结束的发布：partition 由平台写入，确认它没被覆盖后再下发 Rollout
		if err := verifyPartitionHeld(live); err != nil {
			return errors.Wrap(err, "verify batch release")
		}
		_, err := r.manager.Apply(ctx, r.namespace, r.manifest)
		return errors.Wrap(err, "apply rollout")
	case r.verifyPartition:
		// 按策略分批且工作负载已存在：确认 Rollout webhook 注入了 partition = replicas，否则返回错误，部署记录置为失败。
		// 避免刚下发的 Rollout 还没控制 workload 就已经触发了更新。
		return errors.Wrap(verifyPartitionInjected(live), "verify batch release")
	default:
		// 按策略分批，但工作负载是首次创建，没有旧版本实例
		return nil
	}
}

// SnapshotObjects 返回需要保存资源快照的额外资源。分批发布时在 extraObjs 之后追加 Rollout CR，
// 不修改 extraObjs 本身。
func (r *workloadRollout) SnapshotObjects(extraObjs []unstructured.Unstructured) []unstructured.Unstructured {
	if r.manifest == nil {
		return extraObjs
	}
	// 把容量截到长度，避免 append 写进 extraObjs 的底层数组
	clipped := extraObjs[:len(extraObjs):len(extraObjs)]
	return append(clipped, unstructured.Unstructured{Object: r.manifest})
}

// OperateBatchRelease 对环境 / 泳道进行中的分批发布执行手动操作，返回被操作的部署记录。
// 集群中的 Rollout 不属于最新部署记录时返回 ErrReleaseSwitching，读取之后 Rollout 又被修改时返回 ErrRolloutChanged。
func OperateBatchRelease(
	ctx context.Context, db *mongo.Database, appID, envName, trafficLaneName string, action batchrelease.Action,
) (*Record, error) {
	record, err := GetLatest(ctx, db, appID, envName, trafficLaneName)
	if err != nil && !errors.Is(err, ErrDeployRecordNotFound) {
		return nil, err
	}
	if !record.IsBatchReleasing() {
		return nil, errors.WithStack(batchrelease.ErrNoBatchReleaseInProgress)
	}
	kind, name := record.MainWorkload()
	if kind != k8skind.GameDeploy || name == "" {
		return nil, errors.Errorf(
			"main workload %s %q of deploy record %s does not support batch release", kind, name, record.ID.Hex(),
		)
	}

	rollouts := batchrelease.NewRolloutClient(cluster.NewConfig(record.ClusterID))
	rollout, err := rollouts.Get(ctx, record.Namespace, name)
	if err != nil {
		return nil, errors.Wrap(err, "get rollout")
	}
	// 再次部署时部署记录先于 Rollout 切换，这期间集群中仍是上一次发布的 Rollout，批次数等都与最新记录对不上
	if deployID := rollout.GetAnnotations()[AnnotationKeyDeployID]; deployID != record.ID.Hex() {
		return nil, errors.Wrapf(
			batchrelease.ErrReleaseSwitching, "rollout %s/%s belongs to deploy %q rather than deploy record %s",
			record.Namespace, name, deployID, record.ID.Hex(),
		)
	}
	status, err := batchrelease.ParseStatus(rollout.Object)
	if err != nil {
		return nil, errors.Wrapf(err, "rollout %s/%s", record.Namespace, name)
	}
	var batchCount int
	if record.ReleaseStrategy != nil {
		batchCount = len(record.ReleaseStrategy.Batches)
	}
	patch, err := status.ActionPatch(action, batchCount)
	if err != nil {
		return nil, err
	}
	// patch 基于这次读到的状态生成，Rollout 此后有任何修改都应让用户刷新后重新决定，而不是重试这个 patch
	if err = rollouts.UpdateStatus(ctx, record.Namespace, name, rollout.GetResourceVersion(), patch); err != nil {
		return nil, errors.Wrapf(err, "%s batch release of deploy record %s", action, record.ID.Hex())
	}
	return record, nil
}

// deleteRollout 删除与部署记录中主工作负载同名的 Rollout CR。主工作负载不是 GameDeployment 时不会有 Rollout，
// 直接返回。
func deleteRollout(ctx context.Context, cfg *cluster.Config, record *Record) error {
	kind, name := record.MainWorkload()
	if kind != k8skind.GameDeploy || name == "" {
		return nil
	}
	return batchrelease.NewRolloutClient(cfg).Delete(ctx, record.Namespace, name)
}

// rolloutManager 在更新主工作负载前后，处理与它同名的 Rollout CR。主工作负载只能是 GameDeployment（仅非联邦集群）。
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

// Apply 下发 Rollout CR，返回下发之前主工作负载是否已存在。
//
// 工作负载已存在时，要等控制器调谐过该 Rollout 的当前 spec 才返回，判断条件有两个：
//   - status.workloadRevision 已写入。webhook 与控制器共用缓存，缓存里还没有 Rollout 时更新工作负载，
//     不会注入 partition，工作负载会直接整批更新。
//   - 如果本次修改了已有 Rollout 的 spec，还要等 status.rolloutRevision 离开修改前的值。遗留的 Rollout
//     早就写入了 workloadRevision，单看它判断不了。控制器如果先看到工作负载版本变化，会按旧批次开始一轮，
//     随后发现 spec 变了再从第 0 批重来，按旧批次已更新的实例会被白白更新一次再退回。
//
// 工作负载首次创建时没有旧版本实例，不需要分批，也不等待（Rollout 指向不存在的工作负载时，控制器不会写入版本）。
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

// Cleanup 删除与主工作负载同名的遗留 Rollout CR，并移除工作负载上的 rollout 标签。
// 集群未安装 Rollout CRD 时，视为没有遗留。
func (m *rolloutManager) Cleanup(ctx context.Context, namespace, name string) error {
	_, err := m.rollouts.Get(ctx, namespace, name)
	switch {
	case err == nil:
		if err = m.delete(ctx, namespace, name); err != nil {
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

// RestorePartition 把工作负载的 partition 恢复为本次更新期望的值 want。workload 是更新请求返回的工作负载，
// partition 已经是期望值时不访问集群。
//
// 用于清理 Rollout 之后修改 Pod 模板的场景：webhook 在更新请求中、patch 应用之后执行，它的缓存可能还没同步
// Rollout 的删除，仍会注入 partition = replicas，覆盖请求中写入的值；而 Rollout 删除后控制器不会再修改
// partition（Rollout 没有 finalizer），实例会一直停在旧版本。
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

// hasUnfinishedRelease 判断前序的分批发布是否还未完成。
func (m *rolloutManager) hasUnfinishedRelease(ctx context.Context, namespace, name string) (bool, error) {
	status, err := m.rollouts.GetStatus(ctx, namespace, name)
	if errors.Is(err, batchrelease.ErrRolloutNotFound) || errors.Is(err, batchrelease.ErrRolloutNotInstalled) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return status.Phase != "" && status.Phase != batchrelease.RolloutPhaseHealthy, nil
}

// delete 删除同名 Rollout，并等待删除完成
func (m *rolloutManager) delete(ctx context.Context, namespace, name string) error {
	if err := m.rollouts.Delete(ctx, namespace, name); err != nil {
		return err
	}
	return m.waitRolloutDeleted(ctx, namespace, name)
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

// verifyPartitionInjected 校验分批部署更新后，webhook 已在工作负载中注入 partition = replicas。
// 没有注入说明 webhook 没找到 Rollout，工作负载会整批更新，不能当作分批发布继续。
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

// verifyPartitionHeld 校验平台写入的 partition 没有被覆盖。partition 小于 replicas，说明下发 Rollout 之前
// 已经有实例更新为本次的版本，不能当作分批发布继续。
func verifyPartitionHeld(workload *unstructured.Unstructured) error {
	replicas, partition, err := replicasAndPartition(workload.Object)
	if err != nil {
		return err
	}
	if partition < replicas {
		return errors.Errorf(
			"partition of workload %s/%s is %d rather than replicas %d, instances are updated before batch release",
			workload.GetNamespace(), workload.GetName(), partition, replicas,
		)
	}
	return nil
}

// replicasAndPartition 读取工作负载的 spec.replicas 和 spec.updateStrategy.partition，字段缺失时返回 0
func replicasAndPartition(workload map[string]any) (replicas, partition int64, err error) {
	if replicas, _, err = unstructured.NestedInt64(workload, "spec", "replicas"); err != nil {
		return 0, 0, errors.Wrap(err, "parse workload replicas")
	}
	if partition, _, err = unstructured.NestedInt64(workload, "spec", "updateStrategy", "partition"); err != nil {
		return 0, 0, errors.Wrap(err, "parse workload partition")
	}
	return replicas, partition, nil
}
