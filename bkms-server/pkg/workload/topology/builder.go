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
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// Builder 拓扑图构建器，从 ResourceSnapshot 和 K8s 集群实时状态构建拓扑图
type Builder struct{}

// NewBuilder 创建 Builder 实例
func NewBuilder() *Builder {
	return &Builder{}
}

// Build 从 ResourceSnapshot 构建拓扑图
// 流程：
//  1. 并发获取 K8s 集群中各静态资源的实时状态
//  2. 实时发现动态子资源（RS/Job/Pod 等由工作负载创建的资源）
//  3. 用实时 clusterResources 重新收集扩展关系，替换持久化中已过期的动态资源关系
//  4. 构建节点列表（含状态摘要）
//  5. 构建边列表（主边 + 辅助边）
func (b *Builder) Build(ctx context.Context, snapshot *ResourceSnapshot) (*Graph, error) {
	if snapshot == nil {
		return nil, errors.New("resource snapshot is nil")
	}

	// 1. 并发获取 K8s 集群实时状态（仅 snapshot 中记录的静态资源）
	clusterCfg := cluster.NewConfig(snapshot.ClusterID)
	clusterResources, fetchWarnings := b.fetchClusterResources(ctx, clusterCfg, snapshot.Resources)

	// 2. 实时发现动态子资源（RS/Job/Pod），这些资源不固化到 snapshot 中，每次查询时实时获取
	allEntries, dynamicWarnings := b.discoverDynamicResources(
		ctx,
		clusterCfg,
		snapshot.Namespace,
		snapshot.Resources,
		clusterResources,
	)
	fetchWarnings = append(fetchWarnings, dynamicWarnings...)

	// 3. 用实时 clusterResources 重新收集扩展关系，替换持久化中已过期的动态资源关系
	// 持久化的 Relations 中涉及 Pod/RS/Job 的精确关系（volume_mount、env_ref）
	// 可能因 Pod 滚动更新而过期，需用实时数据刷新
	realtimeRelations := NewRelationCollector(clusterResources).Collect()
	mergedRelations := mergeExtensionRelations(snapshot.Relations, realtimeRelations, snapshot.Resources)

	// 4. 构建节点列表
	nodes, nodeIDSet := b.buildNodes(allEntries, clusterResources, clusterCfg.IsFederation())

	// 5. 构建主边（ownerRef + helm_manifest）
	edges := b.buildPrimaryEdges(clusterResources, nodeIDSet, allEntries)

	// 6. 构建辅助边（来自合并后的 extensionRelations），传入主边用于去重
	auxiliaryEdges := b.buildAuxiliaryEdges(mergedRelations, nodeIDSet, clusterResources, edges)
	edges = append(edges, auxiliaryEdges...)

	// 7. 创建 APP 虚拟根节点，并为所有没有入边的顶层资源创建 MANAGES 边
	rootNode, rootEdges := b.buildAppRootNodeAndEdges(snapshot.AppID, nodes, edges)
	nodes = append([]Node{rootNode}, nodes...)
	edges = append(edges, rootEdges...)
	rootID := rootNode.ID

	// 8. 判断是否为 partial 拓扑
	isPartial := len(fetchWarnings) > 0
	for _, n := range nodes {
		if n.Status == k8sstatus.NotFound {
			isPartial = true
			break
		}
	}

	graph := &Graph{
		Metadata: Metadata{
			AppID:           snapshot.AppID,
			EnvName:         snapshot.EnvName,
			TrafficLaneName: snapshot.TrafficLaneName,
			ClusterID:       snapshot.ClusterID,
			Namespace:       snapshot.Namespace,
		},
		Nodes:       nodes,
		Edges:       edges,
		RootID:      rootID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		IsPartial:   isPartial,
		Warnings:    fetchWarnings,
		DataVersion: snapshot.DataVersion,
	}

	return graph, nil
}

// ======================== 阶段 1.5：实时发现动态子资源 ========================

// discoverDynamicResources 实时发现工作负载创建的动态子资源（RS/Job/Pod）
// 这些资源不固化到 snapshot entries 中，每次拓扑查询时实时从集群获取，保证 Pod 列表永远最新
// 返回合并后的完整 entries 列表和发现过程中的警告信息
func (b *Builder) discoverDynamicResources(
	ctx context.Context,
	clusterCfg *cluster.Config,
	namespace string,
	staticEntries []ResourceEntry,
	clusterResources map[string]*unstructured.Unstructured,
) ([]ResourceEntry, []string) {
	allEntries := make([]ResourceEntry, len(staticEntries))
	copy(allEntries, staticEntries)
	var warnings []string
	var mu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(MaxConcurrentK8sRequests)

	for _, entry := range staticEntries {
		spec, ok := workloadChainSpecs[entry.Kind]
		if !ok {
			continue
		}

		g.Go(func() error {
			// 加锁从 clusterResources 中取出 workloadObj，传递给 discoverWorkloadChildren
			workloadNamespace := lo.Ternary(entry.Namespace == "", namespace, entry.Namespace)
			workloadKey := ResourceKey(entry.Kind, workloadNamespace, entry.Name)
			mu.Lock()
			workloadObj := clusterResources[workloadKey]
			mu.Unlock()

			newEntries, newResources, err := b.discoverWorkloadChildren(gCtx, clusterCfg, entry, spec, workloadObj)
			if err != nil {
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf(
					"failed to discover dynamic resources for %s/%s/%s: %v",
					entry.Kind, workloadNamespace, entry.Name, err,
				))
				mu.Unlock()
				return nil
			}

			// 统一去重：在 mu 保护下，对 newEntries 中的每一项进行去重检查
			mu.Lock()
			for _, e := range newEntries {
				key := ResourceKey(e.Kind, e.Namespace, e.Name)
				if _, exists := clusterResources[key]; exists {
					// 已存在于 clusterResources，跳过该 entry（不加入 allEntries）
					continue
				}
				allEntries = append(allEntries, e)
				clusterResources[key] = newResources[key]
			}
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()
	return allEntries, warnings
}

// discoverWorkloadChildren 发现单个工作负载的动态子资源
// 根据 workloadChainSpec 决定是否有中间层（RS/Job），以及最终的 Pod
// 该函数只负责"发现并返回资源"，不做任何去重判断，去重由上层 discoverDynamicResources 统一处理
func (b *Builder) discoverWorkloadChildren(
	ctx context.Context,
	clusterCfg *cluster.Config,
	workloadEntry ResourceEntry,
	spec ownerRefChainSpec,
	workloadObj *unstructured.Unstructured,
) ([]ResourceEntry, map[string]*unstructured.Unstructured, error) {
	// 如果 workloadObj 为空，说明该工作负载不存在，无需探测子资源
	if workloadObj == nil {
		return nil, nil, nil
	}

	// 从工作负载对象提取 spec.selector.matchLabels，用于缩小 List 范围
	labelSelector := extractLabelSelector(workloadObj)
	workloadNamespace := workloadObj.GetNamespace()

	// 无中间层的工作负载（STS/DS/Job）：直接发现 Pod
	if spec.childKind == "" {
		podEntries, podResources, err := b.discoverPods(
			ctx, clusterCfg, workloadNamespace, workloadObj, labelSelector,
		)
		if err != nil {
			return nil, nil, err
		}
		return podEntries, podResources, nil
	}

	// 有中间层的工作负载（Deploy -> RS，CJ -> Job）：先发现中间层，再发现 Pod
	childGVR, gvrErr := discovery.GetGroupVersionResource(clusterCfg, spec.childKind, "")
	if gvrErr != nil {
		return nil, nil, errors.Wrapf(gvrErr, "resolve %s GVR", spec.childKind)
	}
	childCli := k8sclient.NewWithGVR(clusterCfg, *childGVR)
	childList, err := childCli.List(ctx, workloadNamespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "list %s in namespace %s", spec.childKind, workloadNamespace)
	}

	var entries []ResourceEntry
	newResources := make(map[string]*unstructured.Unstructured)

	for i := range childList.Items {
		child := &childList.Items[i]
		if !hasOwnerRef(child, workloadEntry.Kind, workloadEntry.Name) {
			continue
		}

		// 活跃过滤（仅 Deployment -> ReplicaSet 场景需要，不展示那种历史的 RS，只要当前在用的那个）
		if spec.filterActiveChild {
			replicas, _, _ := unstructured.NestedInt64(child.Object, "spec", "replicas")
			if replicas == 0 {
				continue
			}
		}

		// 将所有匹配 ownerRef 且通过活跃过滤的子资源全部加入返回结果，不做去重
		childKey := ResourceKey(spec.childKind, child.GetNamespace(), child.GetName())
		newResources[childKey] = child
		entries = append(entries, ResourceEntry{
			Kind:         spec.childKind,
			Namespace:    child.GetNamespace(),
			Name:         child.GetName(),
			IsManaged:    false,
			SourceType:   SourceTypeOwnerReference,
			SourceReason: fmt.Sprintf("owned by %s/%s", workloadEntry.Kind, workloadEntry.Name),
		})

		// 从中间层对象提取 labelSelector（例如 RS 继承自 Deployment 的 selector）
		childLabelSelector := extractLabelSelector(child)

		// 发现中间层所管理的 Pod，直接将中间层对象作为 owner 传给 discoverPods
		podEntries, podResources, pErr := b.discoverPods(
			ctx,
			clusterCfg,
			child.GetNamespace(),
			child,
			childLabelSelector,
		)
		if pErr != nil {
			continue
		}
		entries = append(entries, podEntries...)
		for k, v := range podResources {
			newResources[k] = v
		}
	}

	return entries, newResources, nil
}

// discoverPods 实时发现指定父资源拥有的 Pod
// labelSelector 用于缩小 Pod 列表查询范围，为空时退化为全量查询
// 该函数只负责发现并返回资源，不做任何去重判断，去重由上层 discoverDynamicResources 统一处理
func (b *Builder) discoverPods(
	ctx context.Context,
	clusterCfg *cluster.Config,
	namespace string,
	owner *unstructured.Unstructured,
	labelSelector string,
) ([]ResourceEntry, map[string]*unstructured.Unstructured, error) {
	podCli := k8sclient.NewWithGVR(clusterCfg, gvr.Po)
	podList, err := podCli.List(ctx, namespace, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "list Pods in namespace %s", namespace)
	}

	var entries []ResourceEntry
	newResources := make(map[string]*unstructured.Unstructured)

	ownerKind := owner.GetKind()
	ownerName := owner.GetName()
	for i := range podList.Items {
		pod := &podList.Items[i]
		if !hasOwnerRef(pod, ownerKind, ownerName) {
			continue
		}

		podKey := ResourceKey(k8skind.Po, pod.GetNamespace(), pod.GetName())
		newResources[podKey] = pod
		entries = append(entries, ResourceEntry{
			Kind:         k8skind.Po,
			Namespace:    pod.GetNamespace(),
			Name:         pod.GetName(),
			IsManaged:    false,
			SourceType:   SourceTypeOwnerReference,
			SourceReason: fmt.Sprintf("owned by %s/%s", ownerKind, ownerName),
		})
	}

	return entries, newResources, nil
}

// extractLabelSelector 从工作负载对象的 spec.selector.matchLabels 提取 label selector 字符串
// 返回形如 "app=web,version=v1" 的选择器字符串，用于 K8s List 请求的 LabelSelector 过滤
// 如果对象没有 spec.selector.matchLabels（如 CronJob），返回空字符串（退化为全量查询）
func extractLabelSelector(obj *unstructured.Unstructured) string {
	matchLabels, found, _ := unstructured.NestedStringMap(obj.Object, "spec", "selector", "matchLabels")
	if !found || len(matchLabels) == 0 {
		return ""
	}

	return formatLabels(matchLabels)
}

// ======================== 阶段 2：并发获取集群资源 ========================

// fetchClusterResources 并发获取集群中各资源的实时状态
// 返回资源对象映射（key: "kind/namespace/name"）和获取时的警告列表
func (b *Builder) fetchClusterResources(
	ctx context.Context,
	clusterCfg *cluster.Config,
	entries []ResourceEntry,
) (map[string]*unstructured.Unstructured, []string) {
	resources := make(map[string]*unstructured.Unstructured)
	var warnings []string
	var mu sync.Mutex

	errGroup, gCtx := errgroup.WithContext(ctx)
	errGroup.SetLimit(MaxConcurrentK8sRequests)

	for _, entry := range entries {
		errGroup.Go(func() error {
			resGVR, gvrErr := discovery.GetGroupVersionResource(clusterCfg, entry.Kind, "")
			if gvrErr != nil {
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf(
					"cannot resolve GVR for kind %s (%s/%s), skipping: %v",
					entry.Kind, entry.Namespace, entry.Name, gvrErr,
				))
				mu.Unlock()
				return nil
			}

			cli := k8sclient.NewWithGVR(clusterCfg, *resGVR)
			obj, err := cli.Get(gCtx, entry.Namespace, entry.Name, metav1.GetOptions{})
			if err != nil {
				mu.Lock()
				warnings = append(warnings, fmt.Sprintf(
					"resource %s/%s/%s not found in cluster: %v",
					entry.Kind, entry.Namespace, entry.Name, err,
				))
				mu.Unlock()
				return nil
			}

			key := ResourceKey(entry.Kind, entry.Namespace, entry.Name)
			mu.Lock()
			resources[key] = obj
			mu.Unlock()
			return nil
		})
	}

	if err := errGroup.Wait(); err != nil {
		log.Errorf(ctx, "topology builder: error fetching cluster resources: %v", err)
		warnings = append(warnings, fmt.Sprintf("cluster resource fetch error: %v", err))
	}

	return resources, warnings
}

// ======================== 阶段 3：构建节点 ========================

// buildNodes 从资源条目和集群资源构建拓扑节点列表
// 返回节点列表和节点 ID 集合（用于后续边构建时验证）
func (b *Builder) buildNodes(
	entries []ResourceEntry,
	clusterResources map[string]*unstructured.Unstructured,
	isFederation bool,
) ([]Node, map[string]bool) {
	nodes := make([]Node, 0, len(entries))
	nodeIDSet := make(map[string]bool, len(entries))

	for _, entry := range entries {
		nodeID := EncodeNodeID(entry.Kind, entry.Namespace, entry.Name)
		if nodeIDSet[nodeID] {
			// 去重
			continue
		}
		nodeIDSet[nodeID] = true

		key := ResourceKey(entry.Kind, entry.Namespace, entry.Name)
		obj := clusterResources[key]

		node := Node{
			ID:          nodeID,
			Kind:        entry.Kind,
			Namespace:   entry.Namespace,
			Name:        entry.Name,
			DisplayName: fmt.Sprintf("%s/%s", entry.Kind, entry.Name),
			IsManaged:   entry.IsManaged,
		}

		if obj != nil {
			// 从集群实时 manifest 中提取状态信息
			b.enrichNodeFromManifest(&node, obj, isFederation)
		} else {
			// 集群中未找到 — 标记为 NotFound
			node.Status = k8sstatus.NotFound
		}

		nodes = append(nodes, node)
	}

	return nodes, nodeIDSet
}

// enrichNodeFromManifest 从集群资源对象中提取状态和类型专属 extras 字段
func (b *Builder) enrichNodeFromManifest(node *Node, obj *unstructured.Unstructured, isFederation bool) {
	kind := node.Kind

	// 通过专属 parser 计算综合状态评估结果
	result := getResourceStatus(kind, obj, isFederation)
	node.Status = result.Code
	node.Reason = result.Message

	// 通过注册表提取类型专属 extras
	if provider, ok := kindExtrasProviders[kind]; ok {
		node.Extras = provider(obj)
	}
}

// ======================== 阶段 4：构建主边 ========================

// buildPrimaryEdges 基于 ownerReferences 和 Helm Manifest 从属关系构建主边
func (b *Builder) buildPrimaryEdges(
	clusterResources map[string]*unstructured.Unstructured,
	nodeIDSet map[string]bool,
	entries []ResourceEntry,
) []Edge {
	var edges []Edge
	edgeIDSet := make(map[string]bool)

	// 从 ownerReferences 构建主边
	for _, obj := range clusterResources {
		childKind := obj.GetKind()
		childNS := obj.GetNamespace()
		childName := obj.GetName()
		childNodeID := EncodeNodeID(childKind, childNS, childName)

		if !nodeIDSet[childNodeID] {
			continue
		}

		for _, ref := range obj.GetOwnerReferences() {
			parentNodeID := EncodeNodeID(ref.Kind, childNS, ref.Name)
			if !nodeIDSet[parentNodeID] {
				continue
			}

			relation := ownerRefToRelation(ref.Kind, childKind)
			edgeID := EncodeEdgeID(ref.Kind, childNS, ref.Name, childKind, childNS, childName, relation)
			if edgeIDSet[edgeID] {
				continue
			}
			edgeIDSet[edgeID] = true

			edges = append(edges, Edge{
				ID:        edgeID,
				SourceID:  parentNodeID,
				TargetID:  childNodeID,
				Relation:  relation,
				IsPrimary: true,
				Reason: EdgeReason{
					Type:    RelationTypeOwnerReference,
					Summary: fmt.Sprintf("%s/%s owns %s/%s", ref.Kind, ref.Name, childKind, childName),
				},
			})
		}
	}

	// 注意：对于 Helm Manifest 中直接管理但没有 ownerRef 的资源，
	// 其 MANAGES 边由 buildAppRootNodeAndEdges 函数统一构建（从 App 虚拟根节点出发）。
	return edges
}

// ownerRefToRelation 根据父/子资源类型确定关系类型
func ownerRefToRelation(parentKind, childKind string) EdgeRelation {
	switch {
	case parentKind == k8skind.Deploy && childKind == k8skind.RS:
		return EdgeRelationCreates
	case parentKind == k8skind.RS && childKind == k8skind.Po:
		return EdgeRelationCreates
	case parentKind == k8skind.STS && childKind == k8skind.Po:
		return EdgeRelationCreates
	case parentKind == k8skind.DS && childKind == k8skind.Po:
		return EdgeRelationCreates
	case parentKind == k8skind.Job && childKind == k8skind.Po:
		return EdgeRelationCreates
	case parentKind == k8skind.CJ && childKind == k8skind.Job:
		return EdgeRelationCreates
	default:
		return EdgeRelationManages
	}
}

// ======================== 阶段 5：构建辅助边 ========================

// buildAuxiliaryEdges 基于 Relations 构建辅助边（isPrimary=false）
// primaryEdges 参数用于构建主边索引，当辅助边的源目标对已有同方向主边时跳过该辅助边
func (b *Builder) buildAuxiliaryEdges(
	relations []ResourceRelation,
	nodeIDSet map[string]bool,
	clusterResources map[string]*unstructured.Unstructured,
	primaryEdges []Edge,
) []Edge {
	var edges []Edge
	edgeIDSet := make(map[string]bool)

	// 构建主边索引，key 格式为 "{sourceID}->{targetID}"，用于辅助边去重
	primaryEdgeIndex := make(map[string]bool, len(primaryEdges))
	for _, e := range primaryEdges {
		if e.IsPrimary {
			primaryEdgeIndex[e.SourceID+"->"+e.TargetID] = true
		}
	}

	for _, rel := range relations {
		// 跳过 owner_reference 类型（已在主边中处理）
		if rel.RelationType == RelationTypeOwnerReference {
			continue
		}

		sourceNodeID := EncodeNodeID(rel.SourceKind, rel.SourceNamespace, rel.SourceName)
		// label_selector 中 targetName 可能是通配符，需要特殊处理
		if rel.TargetName == TargetNameWildcard {
			// label_selector 指向通配 Pod，跳过（不构建精确边）
			// 需要匹配实际的 Pod 节点
			edges = append(
				edges, b.expandWildcardRelation(rel, nodeIDSet, edgeIDSet, clusterResources, primaryEdgeIndex)...,
			)
			continue
		}

		targetNodeID := EncodeNodeID(rel.TargetKind, rel.TargetNamespace, rel.TargetName)

		// 跳过：源或目标不在当前 snapshot 中
		if !nodeIDSet[sourceNodeID] || !nodeIDSet[targetNodeID] {
			continue
		}

		// 跳过：源和目标为同一节点
		if sourceNodeID == targetNodeID {
			continue
		}

		// 跳过：同源同目标已有主边
		if primaryEdgeIndex[sourceNodeID+"->"+targetNodeID] {
			continue
		}

		relation := relationTypeToEdgeRelation(rel.RelationType)
		edgeID := EncodeEdgeID(
			rel.SourceKind, rel.SourceNamespace, rel.SourceName,
			rel.TargetKind, rel.TargetNamespace, rel.TargetName, relation,
		)

		// 去重
		if edgeIDSet[edgeID] {
			continue
		}
		edgeIDSet[edgeID] = true

		edges = append(edges, Edge{
			ID:        edgeID,
			SourceID:  sourceNodeID,
			TargetID:  targetNodeID,
			Relation:  relation,
			IsPrimary: false,
			Reason: EdgeReason{
				Type:            rel.RelationType,
				Summary:         rel.Summary,
				MatchedLabels:   rel.MatchedLabels,
				SourceFieldPath: rel.SourceFieldPath,
				TargetFieldPath: rel.TargetFieldPath,
			},
		})
	}

	return edges
}

// expandWildcardRelation 展开通配的 label_selector 关系为具体的 Pod 节点边
// primaryEdgeIndex 用于跳过同源同目标已有主边的辅助边
func (b *Builder) expandWildcardRelation(
	rel ResourceRelation,
	nodeIDSet map[string]bool,
	edgeIDSet map[string]bool,
	clusterResources map[string]*unstructured.Unstructured,
	primaryEdgeIndex map[string]bool,
) []Edge {
	var edges []Edge

	sourceNodeID := EncodeNodeID(rel.SourceKind, rel.SourceNamespace, rel.SourceName)
	if !nodeIDSet[sourceNodeID] {
		return nil
	}

	// 遍历 nodeIDSet 中的所有节点，找出匹配的 Pod
	for nodeID := range nodeIDSet {
		kind, ns, _, err := DecodeNodeID(nodeID)
		if err != nil {
			continue
		}
		// 只匹配同命名空间的 Pod（label_selector 的目标类型）
		if kind != rel.TargetKind || ns != rel.SourceNamespace {
			continue
		}

		// 跳过自指
		if nodeID == sourceNodeID {
			continue
		}

		_, targetNS, targetName, _ := DecodeNodeID(nodeID)

		// 校验 Pod 的实际 labels 是否匹配 rel.MatchedLabels
		if len(rel.MatchedLabels) > 0 {
			resKey := ResourceKey(kind, targetNS, targetName)
			obj := clusterResources[resKey]
			if obj == nil || !isLabelsMatch(rel.MatchedLabels, obj.GetLabels()) {
				continue
			}
		}

		// 跳过：同源同目标已有主边
		if primaryEdgeIndex[sourceNodeID+"->"+nodeID] {
			continue
		}

		relation := relationTypeToEdgeRelation(rel.RelationType)

		// 为每个目标 Pod 创建辅助边（使用完整 nodeID 来生成唯一 edge ID）
		edgeID := EncodeEdgeID(
			rel.SourceKind, rel.SourceNamespace, rel.SourceName, kind, targetNS, targetName, relation,
		)

		if edgeIDSet[edgeID] {
			continue
		}
		edgeIDSet[edgeID] = true

		edges = append(edges, Edge{
			ID:        edgeID,
			SourceID:  sourceNodeID,
			TargetID:  nodeID,
			Relation:  relation,
			IsPrimary: false,
			Reason: EdgeReason{
				Type:          rel.RelationType,
				Summary:       rel.Summary,
				MatchedLabels: rel.MatchedLabels,
			},
		})
	}

	return edges
}

// relationTypeToEdgeRelation 将扩展关系类型映射为边关系类型
func relationTypeToEdgeRelation(relationType RelationType) EdgeRelation {
	switch relationType {
	case RelationTypeLabelSelector:
		return EdgeRelationSelects
	case RelationTypeVolumeMount:
		return EdgeRelationMounts
	case RelationTypeBackendRef:
		return EdgeRelationRoutes
	case RelationTypeEnvRef:
		return EdgeRelationReferences
	case RelationTypeScaleTargetRef:
		return EdgeRelationScales
	case RelationTypeServiceAccountRef:
		return EdgeRelationReferences
	default:
		return EdgeRelationManages
	}
}

// ======================== 扩展关系合并 ========================

// potentiallyDynamicKinds 可能由工作负载动态创建的资源类型（Pod/ReplicaSet/Job）
// 当这些类型的资源出现在 snapshot.Resources（静态 entries）中时，视为静态资源，其持久化关系应保留
// 只有不在 snapshot.Resources 中的才视为动态资源（如 CronJob 创建的 Job、Deployment 创建的 RS）
var potentiallyDynamicKinds = map[string]bool{
	k8skind.Po:  true,
	k8skind.RS:  true,
	k8skind.Job: true,
}

// buildStaticResourceSet 从静态 entries 构建资源标识集合（"Kind/Namespace/Name"）
func buildStaticResourceSet(staticEntries []ResourceEntry) map[string]bool {
	set := make(map[string]bool, len(staticEntries))
	for _, e := range staticEntries {
		set[ResourceKey(e.Kind, e.Namespace, e.Name)] = true
	}
	return set
}

// isDynamicResource 判断某个资源在当前 snapshot 中是否为动态资源
// 只有属于 potentiallyDynamicKinds 且不在静态 entries 中的资源才被视为动态
func isDynamicResource(staticSet map[string]bool, kind, namespace, name string) bool {
	if !potentiallyDynamicKinds[kind] {
		return false
	}
	return !staticSet[ResourceKey(kind, namespace, name)]
}

// involveDynamicResource 判断扩展关系的源或目标是否涉及动态资源
func involveDynamicResource(staticSet map[string]bool, rel ResourceRelation) bool {
	return isDynamicResource(staticSet, rel.SourceKind, rel.SourceNamespace, rel.SourceName) ||
		isDynamicResource(staticSet, rel.TargetKind, rel.TargetNamespace, rel.TargetName)
}

// mergeExtensionRelations 合并持久化的扩展关系与实时收集的扩展关系
// staticEntries 为 snapshot.Resources（Helm/AppModel 声明的静态资源），用于区分静态/动态资源
// 策略：
//   - 持久化关系中涉及动态资源（非静态的 Pod/RS/Job）的关系 → 丢弃（已过期）
//   - 持久化关系中仅涉及静态资源的关系 → 保留（包括直接声明的 Job 等）
//   - 实时关系中涉及动态资源的非 owner_reference 关系 → 采纳（最新状态）
//   - 实时关系中 owner_reference 类型 → 丢弃（由 buildPrimaryEdges 处理）
func mergeExtensionRelations(
	persisted, realtime []ResourceRelation,
	staticEntries []ResourceEntry,
) []ResourceRelation {
	staticSet := buildStaticResourceSet(staticEntries)
	var merged []ResourceRelation

	// 保留持久化关系中不涉及动态资源的部分
	// 例如：静态 Job → ConfigMap 的 volume_mount 会被保留，因为 Job 在 staticEntries 中
	for _, rel := range persisted {
		if !involveDynamicResource(staticSet, rel) {
			merged = append(merged, rel)
		}
	}

	// 追加实时关系中涉及动态资源且非 owner_reference 的部分
	for _, rel := range realtime {
		if rel.RelationType == RelationTypeOwnerReference {
			continue
		}
		if involveDynamicResource(staticSet, rel) {
			merged = append(merged, rel)
		}
	}

	return merged
}

// ======================== APP 虚拟根节点 ========================

// buildAppRootNodeAndEdges 创建 APP 虚拟根节点，并为所有没有主边入边的顶层资源节点创建 MANAGES 边
// APP 根节点是整个拓扑图的唯一根，不对应任何 K8s 资源
func (b *Builder) buildAppRootNodeAndEdges(appID string, nodes []Node, edges []Edge) (Node, []Edge) {
	// 创建虚拟根节点（Kind=App, Namespace 为空, Name 为 appID）
	rootNode := Node{
		ID:          EncodeNodeID(NodeKindApp, "", appID),
		Kind:        NodeKindApp,
		Name:        appID,
		DisplayName: appID,
		Status:      k8sstatus.Active,
		IsManaged:   true,
	}

	// 收集所有作为 target 的节点（有主边入边的节点）
	hasIncoming := make(map[string]bool)
	for _, e := range edges {
		if e.IsPrimary {
			hasIncoming[e.TargetID] = true
		}
	}

	// 为所有没有主边入边的节点创建从根节点出发的 MANAGES 边
	var rootEdges []Edge
	for _, n := range nodes {
		if hasIncoming[n.ID] {
			continue
		}
		edgeID := EncodeEdgeID(NodeKindApp, "", appID, n.Kind, n.Namespace, n.Name, EdgeRelationManages)
		rootEdges = append(rootEdges, Edge{
			ID:        edgeID,
			SourceID:  rootNode.ID,
			TargetID:  n.ID,
			Relation:  EdgeRelationManages,
			IsPrimary: true,
			Reason: EdgeReason{
				Type:    RelationTypeAppRoot,
				Summary: fmt.Sprintf("app %s manages %s/%s", appID, n.Kind, n.Name),
			},
		})
	}

	return rootNode, rootEdges
}

// isLabelsMatch 检查 selector 是否为 labels 的子集，
// 即 selector 中的每个 key-value 都存在于 labels 中
func isLabelsMatch(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}
