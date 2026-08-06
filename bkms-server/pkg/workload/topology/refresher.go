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

package topology

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// 并发控制相关常量，用于 Refresher 和 Builder 中 errgroup.SetLimit
const (
	// MaxConcurrentK8sRequests K8s API 并发请求数上限，防止对集群 API Server 造成过大压力
	MaxConcurrentK8sRequests = 10
)

// Refresher 资源范围刷新器
type Refresher struct {
	store ResourceSnapshotStore
}

// NewRefresher 创建 Refresher 实例
func NewRefresher(store ResourceSnapshotStore) *Refresher {
	return &Refresher{store: store}
}

// TriggerRefresh 触发异步资源范围刷新
// 启动新的 goroutine 执行刷新，正确性由 Store 层乐观锁（DataVersion）保证
func (r *Refresher) TriggerRefresh(ctx context.Context, args RefreshArgs) {
	snapshotKey := args.SnapshotKey()

	go func() {
		trCtx := context.WithoutCancel(ctx)
		if err := r.Refresh(trCtx, args); err != nil {
			log.Errorf(trCtx, "topology refresher: refresh failed for snapshot %s: %v", snapshotKey, err)
		}
	}()
}

// Refresh 执行资源范围刷新的完整流程
// 1. 标记为 pending 状态
// 2. 根据刷新参数选择刷新策略（ResourceKeys → AppModel 部署 / ReleaseName → Helm 部署）
// 3. 刷新成功后，带乐观锁原子更新到 Store
func (r *Refresher) Refresh(ctx context.Context, args RefreshArgs) error {
	snapshotKey := args.SnapshotKey()
	log.Infof(ctx, "topology refresher: starting refresh for snapshot %s", snapshotKey)

	// 标记为 pending（仅更新状态，保留已有的 resources 和 extensionRelations）
	if err := r.store.UpdateStatus(
		ctx, args.AppID, args.EnvName, args.TrafficLaneName, RefreshStatusProgressing, "",
	); err != nil {
		return errors.Wrap(err, "set pending status")
	}

	// 根据刷新参数选择不同的刷新策略：
	// - 有 ResourceKeys：AppModel 部署，从部署记录中的资源引用刷新
	// - 有 ReleaseName：Helm 部署，从 Helm Release Manifest 刷新
	var snapshot *ResourceSnapshot
	var expectedVersion int64
	var err error
	if len(args.ResourceKeys) > 0 {
		snapshot, expectedVersion, err = r.doRefreshFromResourceKeys(ctx, args)
	} else {
		snapshot, expectedVersion, err = r.doRefreshFromHelmRelease(ctx, args)
	}
	if err != nil {
		uErr := r.store.UpdateStatus(
			ctx, args.AppID, args.EnvName, args.TrafficLaneName, RefreshStatusFailed, err.Error(),
		)
		if uErr != nil {
			return errors.Wrapf(uErr, "mark failed status for snapshot %s", args.SnapshotKey())
		}
		return errors.Wrap(err, "refresh failed")
	}

	// 刷新成功，带乐观锁原子更新
	// 若版本冲突说明已有更新的刷新完成，当前结果可安全丢弃
	snapshot.RefreshStatus = RefreshStatusSuccess
	snapshot.RefreshedAt = time.Now()
	if err = r.store.UpsertWithVersion(ctx, snapshot, expectedVersion); err != nil {
		if errors.Is(err, ErrVersionConflict) {
			log.Infof(
				ctx,
				"topology refresher: version conflict for snapshot %s, a newer refresh has completed, discarding",
				snapshotKey,
			)
			return nil
		}
		return errors.Wrap(err, "upsert refreshed resource snapshot")
	}

	log.Infof(
		ctx, "topology refresher: refresh completed for snapshot %s, resources=%d, relations=%d",
		snapshotKey, len(snapshot.Resources), len(snapshot.Relations),
	)
	return nil
}

// doRefreshFromHelmRelease 基于 Helm Release Manifest 执行刷新（Helm 部署场景）
// 返回值中的 expectedVersion 为写入前读取到的 DataVersion，用于乐观锁校验
func (r *Refresher) doRefreshFromHelmRelease(ctx context.Context, args RefreshArgs) (
	*ResourceSnapshot, int64, error,
) {
	// 1. 获取 Helm Release Manifest
	debugLog := helm.NewHelmDebugLogger(ctx, args.ReleaseName, "topology-refresh")
	cfg, err := helm.NewActionConfiguration(args.ClusterID, args.Namespace, debugLog)
	if err != nil {
		return nil, 0, errors.Wrap(err, "init helm action configuration")
	}

	manifest, err := helm.GetReleaseManifest(cfg, args.ReleaseName)
	if err != nil {
		return nil, 0, errors.Wrap(err, "get release manifest")
	}

	// 2. 解析 Manifest 获取资源列表
	entries, err := ParseManifest(manifest, args.Namespace)
	if err != nil {
		return nil, 0, errors.Wrap(err, "parse manifest")
	}

	// 3. 从集群补充资源（ownerRef 链，如 Deployment → ReplicaSet / CronJob → Job）
	clusterCfg := cluster.NewConfig(args.ClusterID)
	supplementedEntries, clusterResources, err := r.supplementFromCluster(ctx, clusterCfg, args.Namespace, entries)
	if err != nil {
		return nil, 0, errors.Wrap(err, "supplement from cluster")
	}

	// 4. 收集扩展关系
	relations := NewRelationCollector(clusterResources).Collect()

	// 5. 构建最终的 ResourceSnapshot（含乐观锁版本号）
	return r.buildSnapshotWithVersion(ctx, args, supplementedEntries, relations)
}

// doRefreshFromResourceKeys 基于部署记录中的 ResourceKeys 执行刷新（AppModel 部署场景）
// 返回值中的 expectedVersion 为写入前读取到的 DataVersion，用于乐观锁校验
// 流程：
// 1. 将 ResourceKeys 转换为初始 ResourceEntry 列表
// 2. 从集群补充 ownerRef 链（如 Deployment → ReplicaSet）
// 3. 收集扩展关系
// 4. 构建最终的 ResourceSnapshot
func (r *Refresher) doRefreshFromResourceKeys(ctx context.Context, args RefreshArgs) (
	*ResourceSnapshot, int64, error,
) {
	// 1. 将 ResourceKeys 转换为 ResourceEntry 列表
	entries := make([]ResourceEntry, 0, len(args.ResourceKeys))
	for _, rk := range args.ResourceKeys {
		entries = append(entries, ResourceEntry{
			Kind:       rk.Kind,
			Namespace:  args.Namespace,
			Name:       rk.Name,
			IsManaged:  true,
			SourceType: SourceTypeAppModelDeploy,
		})
	}

	// 2. 从集群补充 ownerRef 链
	clusterCfg := cluster.NewConfig(args.ClusterID)
	supplementedEntries, clusterResources, err := r.supplementFromCluster(ctx, clusterCfg, args.Namespace, entries)
	if err != nil {
		return nil, 0, errors.Wrap(err, "supplement from cluster")
	}

	// 3. 收集扩展关系
	relations := NewRelationCollector(clusterResources).Collect()

	// 4. 构建最终的 ResourceSnapshot（含乐观锁版本号）
	return r.buildSnapshotWithVersion(ctx, args, supplementedEntries, relations)
}

// supplementFromCluster 从集群中补充 ownerRef 链中的资源到 clusterResources（用于关系收集）
// 动态子资源（RS/Job/Pod）不写入 entries，由 Builder 实时发现
// 返回原始 entries（不含动态子资源）和集群中获取到的非结构化资源对象映射
func (r *Refresher) supplementFromCluster(
	ctx context.Context,
	clusterCfg *cluster.Config,
	namespace string,
	entries []ResourceEntry,
) ([]ResourceEntry, map[string]*unstructured.Unstructured, error) {
	clusterResources := make(map[string]*unstructured.Unstructured)
	var mu sync.Mutex

	// 并发获取所有 Manifest 中声明的资源的实时状态
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(MaxConcurrentK8sRequests)

	for _, entry := range entries {
		g.Go(func() error {
			resGVR, gvrErr := discovery.GetGroupVersionResource(clusterCfg, entry.Kind, "")
			if gvrErr != nil {
				log.Warnf(ctx, "topology refresher: cannot resolve GVR for kind %s, skipping: %v", entry.Kind, gvrErr)
				return nil
			}

			cli := k8sclient.NewWithGVR(clusterCfg, *resGVR)
			obj, err := cli.Get(gCtx, entry.Namespace, entry.Name, metav1.GetOptions{})
			if err != nil {
				if errors.Is(err, k8sclient.ErrResourceNotFound) {
					log.Warnf(
						ctx, "topology refresher: resource %s/%s/%s not found in cluster, skipping",
						entry.Kind, entry.Namespace, entry.Name,
					)
					return nil
				}
				return errors.Wrapf(err, "get %s/%s/%s from cluster", entry.Kind, entry.Namespace, entry.Name)
			}

			key := ResourceKey(entry.Kind, entry.Namespace, entry.Name)
			mu.Lock()
			clusterResources[key] = obj
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	// 补充 Deployment/StatefulSet → ReplicaSet 链（Pod 需要在 Builder 中实时获取）
	supplementG, supplementCtx := errgroup.WithContext(ctx)
	supplementG.SetLimit(MaxConcurrentK8sRequests)

	for _, entry := range entries {
		switch entry.Kind {
		case k8skind.Deploy, k8skind.CJ:
			// 仅对有中间层子资源的工作负载做 ownerRef 补链（补充 RS/Job）
		default:
			continue
		}

		supplementG.Go(func() error {
			return r.supplementOwnerRefChain(
				supplementCtx, clusterCfg, namespace, entry, clusterResources, &mu,
			)
		})
	}

	if err := supplementG.Wait(); err != nil {
		return nil, nil, errors.Wrap(err, "supplement owner ref chain")
	}

	return entries, clusterResources, nil
}

// ownerRefChainSpec 描述某种工作负载的 ownerRef 补链规则
// childKind: 中间层子资源类型（如 ReplicaSet、Job）
// filterActiveChild: 是否对中间层资源做活跃过滤（replicas > 0）
// 注意：Pod 由 Builder 实时发现，Refresher 仅补充中间层资源
type ownerRefChainSpec struct {
	childKind         string
	filterActiveChild bool
}

// workloadChainSpecs 有中间层子资源的工作负载的 ownerRef 补链规则注册表
// 无中间层的工作负载（STS/DS/Job/GameDeploy/GameSTS 等）直接 own Pod，由 Builder 实时发现，不需要在此注册
var workloadChainSpecs = map[string]ownerRefChainSpec{
	k8skind.Deploy:     {childKind: k8skind.RS, filterActiveChild: true},   // Deploy -> RS -> Pod
	k8skind.GameDeploy: {childKind: "", filterActiveChild: false},          // GameDeploy 直接 own Pod
	k8skind.GameSTS:    {childKind: "", filterActiveChild: false},          // GameSTS 直接 own Pod
	k8skind.STS:        {childKind: "", filterActiveChild: false},          // STS 直接 own Pod
	k8skind.DS:         {childKind: "", filterActiveChild: false},          // DS 直接 own Pod
	k8skind.CJ:         {childKind: k8skind.Job, filterActiveChild: false}, // CJ -> Job -> Pod
	k8skind.Job:        {childKind: "", filterActiveChild: false},          // Job 直接 own Pod
}

// supplementOwnerRefChain 补充单个工作负载的 ownerRef 链资源到 clusterResources（仅用于关系收集）
// 注意：动态子资源（RS/Job/Pod）不写入 snapshot entries，由 Builder 实时发现
func (r *Refresher) supplementOwnerRefChain(
	ctx context.Context,
	clusterCfg *cluster.Config,
	namespace string,
	workloadEntry ResourceEntry,
	clusterResources map[string]*unstructured.Unstructured,
	mu *sync.Mutex,
) error {
	spec, ok := workloadChainSpecs[workloadEntry.Kind]
	if !ok {
		return nil
	}

	// 无中间层的工作负载（STS/DS/Job）：Pod 由 Builder 实时发现，无需在 Refresher 阶段补充
	if spec.childKind == "" {
		return nil
	}

	// 有中间层的工作负载（Deploy -> RS，CJ -> Job）：补充中间层资源到 clusterResources
	// 从工作负载对象提取 labelSelector，用于缩小 List 查询范围
	workloadKey := ResourceKey(workloadEntry.Kind, namespace, workloadEntry.Name)
	mu.Lock()
	workloadObj := clusterResources[workloadKey]
	mu.Unlock()

	var labelSelector string
	if workloadObj != nil {
		labelSelector = extractLabelSelector(workloadObj)
	}

	childGVR, gvrErr := discovery.GetGroupVersionResource(clusterCfg, spec.childKind, "")
	if gvrErr != nil {
		return errors.Wrapf(gvrErr, "resolve %s GVR", spec.childKind)
	}
	childCli := k8sclient.NewWithGVR(clusterCfg, *childGVR)
	childList, err := childCli.List(ctx, namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return errors.Wrapf(err, "list %s in namespace %s", spec.childKind, namespace)
	}

	for i := range childList.Items {
		child := &childList.Items[i]
		if !hasOwnerRef(child, workloadEntry.Kind, workloadEntry.Name) {
			continue
		}

		// 活跃过滤（仅 Deployment -> ReplicaSet 场景需要）
		if spec.filterActiveChild {
			replicas, _, _ := unstructured.NestedInt64(child.Object, "spec", "replicas")
			if replicas == 0 {
				continue
			}
		}

		childKey := ResourceKey(spec.childKind, child.GetNamespace(), child.GetName())
		mu.Lock()
		if _, exists := clusterResources[childKey]; !exists {
			clusterResources[childKey] = child
		}
		mu.Unlock()

		// Pod 由 Builder 实时发现，无需在 Refresher 阶段补充
	}

	return nil
}

// buildSnapshotWithVersion 获取当前版本号并构建 ResourceSnapshot（乐观锁公共逻辑）
// 返回 snapshot 和 expectedVersion，供 Refresh 中 UpsertWithVersion 使用
func (r *Refresher) buildSnapshotWithVersion(
	ctx context.Context,
	args RefreshArgs,
	entries []ResourceEntry,
	relations []ResourceRelation,
) (*ResourceSnapshot, int64, error) {
	var expectedVersion int64
	dataVersion := int64(1)
	existing, err := r.store.Get(ctx, args.AppID, args.EnvName, args.TrafficLaneName)
	if err != nil {
		return nil, 0, errors.Wrap(err, "get existing resource snapshot for version")
	}
	if existing != nil {
		expectedVersion = existing.DataVersion
		dataVersion = existing.DataVersion + 1
	}

	snapshot := &ResourceSnapshot{
		AppID:           args.AppID,
		EnvName:         args.EnvName,
		TrafficLaneName: args.TrafficLaneName,
		ClusterID:       args.ClusterID,
		Namespace:       args.Namespace,
		ReleaseName:     args.ReleaseName,
		DataVersion:     dataVersion,
		Resources:       entries,
		Relations:       relations,
	}
	return snapshot, expectedVersion, nil
}

// hasOwnerRef 检查资源是否有指定的 ownerReference
func hasOwnerRef(obj *unstructured.Unstructured, ownerKind, ownerName string) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == ownerKind && ref.Name == ownerName {
			return true
		}
	}
	return false
}
