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

package overview

import (
	"context"
	"log/slog"
	"maps"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	podstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status/workload/pod"
	k8sworkload "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/workload"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// deployRecordForEnv 携带查 K8s 实例所需的最新 AppModel 部署记录。
type deployRecordForEnv struct {
	EnvName string
	Record  *appmodel.Record
}

// envClusterData 单环境从集群读到的实例数与主容器资源规格。
// Instances 为 nil 表示实例数不可用；Resources 为零值表示未读到规格。
type envClusterData struct {
	Instances *InstanceCounts
	Resources ResourceSpec
}

// envClusterDataByEnv envName -> 集群回查结果。
// 缺 key 表示该环境未能定位 workload 或查询失败。
type envClusterDataByEnv map[string]envClusterData

// deployRecordsByCluster clusterID -> 该集群上需要一并查询的环境部署记录。
// 约定：同一集群内各环境的 namespace 唯一。
type deployRecordsByCluster map[string][]deployRecordForEnv

// clusterQuerier 单集群内查询实例数所需的客户端与共享并发闸门。
// 客户端按集群创建一次，供该集群下各环境复用。
type clusterQuerier struct {
	clusterID string
	pods      *k8sclient.PodClient
	// workloads 主工作负载 Kind -> 客户端
	workloads map[string]*k8sclient.Client
	sem       *semaphore.Weighted
}

// newClusterQuerier 创建单集群查询器；集群配置只解析一次，Pod 与各类主工作负载客户端共用。
func newClusterQuerier(clusterID string, sem *semaphore.Weighted) *clusterQuerier {
	clusterCfg := cluster.NewConfig(clusterID)
	workloads := make(map[string]*k8sclient.Client)
	for _, driver := range k8sworkload.MainDrivers() {
		workloads[driver.Kind()] = k8sclient.NewWithGVR(clusterCfg, driver.GVR())
	}
	return &clusterQuerier{
		clusterID: clusterID,
		pods:      k8sclient.NewPodClient(clusterCfg),
		workloads: workloads,
		sem:       sem,
	}
}

// queryEnvClusterData 按集群并发查询各环境实例数与主容器资源规格。
//
// 集群之间并发；集群内各环境并发；单环境内 Pod List 与 GameDeployment Get 并发。
// 三层扇出均不设上限，真正的在途请求数由 sem 统一约束。
// 单环境失败只影响该环境，不中断其它环境/集群，也不使整次总览失败。
//
// Args:
//   - sem 本次请求内所有集群回查共享的在途请求闸门
//   - records 已过滤到表格行内、且含 AppModel 部署记录的环境
//
// Returns:
//   - envName -> 集群回查结果；失败或无法定位 workload 的环境不出现
func queryEnvClusterData(
	ctx context.Context,
	sem *semaphore.Weighted,
	records []deployRecordForEnv,
) envClusterDataByEnv {
	out := make(envClusterDataByEnv, len(records))
	if len(records) == 0 {
		return out
	}

	byCluster := groupDeployRecordsByCluster(records)
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for clusterID, items := range byCluster {
		wg.Go(func() {
			data := queryEnvClusterDataForCluster(ctx, sem, clusterID, items)
			mu.Lock()
			defer mu.Unlock()
			maps.Copy(out, data)
		})
	}
	wg.Wait()
	return out
}

// groupDeployRecordsByCluster 按 ClusterID 分组；无 ClusterID 的记录无法查 K8s，直接丢弃。
func groupDeployRecordsByCluster(records []deployRecordForEnv) deployRecordsByCluster {
	queryable := lo.Filter(records, func(item deployRecordForEnv, _ int) bool {
		return item.Record != nil && item.Record.ClusterID != ""
	})
	return lo.GroupBy(queryable, func(item deployRecordForEnv) string {
		return item.Record.ClusterID
	})
}

// queryEnvClusterDataForCluster 并发查询单集群上各环境的实例数与资源规格。
//
// 每个环境独立发起：
//   - Pod：命名空间内按 LabelSelector List（避免 AllNamespaces 宽拉）
//   - GameDeployment：按 ns/name Get（避免全量 List），同时读取 replicas 与主容器 resources
//
// Pod 与 GD 在同一环境内并发；环境之间也并发。
// 任一环境的查询失败只跳过该环境，不影响同集群其它环境。
//
// Args:
//   - sem 本次请求内所有集群回查共享的在途请求闸门
//   - clusterID BCS / 本地集群 ID
//   - items 同属于该集群的环境部署记录（约定 namespace 互不重复）
//
// Returns:
//   - envName -> 集群回查结果；完全失败的环境不写入
func queryEnvClusterDataForCluster(
	ctx context.Context,
	sem *semaphore.Weighted,
	clusterID string,
	items []deployRecordForEnv,
) envClusterDataByEnv {
	out := make(envClusterDataByEnv, len(items))
	if len(items) == 0 {
		return out
	}

	querier := newClusterQuerier(clusterID, sem)

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, item := range items {
		wg.Go(func() {
			data, err := queryEnvClusterDataForEnv(ctx, querier, item)
			if err != nil {
				log.ErrorAttrs(ctx, "query deploy overview cluster data failed",
					slog.String("cluster_id", clusterID),
					slog.String("env_name", item.EnvName),
					slog.String("namespace", item.Record.Namespace),
					slog.Any("error", err),
				)
				return
			}
			if data == nil {
				// 缺 GD 等不可用场景，按设计降级，不记错误日志
				return
			}
			mu.Lock()
			out[item.EnvName] = *data
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}

// queryEnvClusterDataForEnv 查询单个环境的实例数与主容器资源规格。
// Pod List 与主工作负载 Get 并发。工作负载 Get 失败才返回 error；
// Pod List 失败只打日志，实例数降级为 null，资源规格仍返回。
func queryEnvClusterDataForEnv(
	ctx context.Context,
	querier *clusterQuerier,
	item deployRecordForEnv,
) (*envClusterData, error) {
	kind, name := extractMainWorkload(item.Record)
	if name == "" {
		return nil, nil
	}

	var (
		pods    []unstructured.Unstructured
		wl      *k8sworkload.View
		podsErr error
		wlErr   error
	)

	var wg sync.WaitGroup
	wg.Go(func() {
		pods, podsErr = listPods(ctx, querier, item)
	})
	wg.Go(func() {
		wl, wlErr = getMainWorkload(ctx, querier, item, kind, name)
	})
	wg.Wait()

	if podsErr != nil {
		log.ErrorAttrs(ctx, "query deploy overview instances failed",
			slog.String("cluster_id", querier.clusterID),
			slog.String("env_name", item.EnvName),
			slog.String("namespace", item.Record.Namespace),
			slog.Any("error", podsErr),
		)
	}
	if wlErr != nil {
		return nil, wlErr
	}
	return &envClusterData{
		Resources: extractMainContainerResources(ctx, item.Record.Namespace, name, wl.Containers),
		Instances: instanceCounts(wl.Replicas, pods, podsErr == nil),
	}, nil
}

// getMainWorkload 按 ns/name Get 主工作负载，并解析出副本数与容器规格。
func getMainWorkload(
	ctx context.Context,
	querier *clusterQuerier,
	item deployRecordForEnv,
	kind, name string,
) (*k8sworkload.View, error) {
	driver, err := k8sworkload.Get(kind)
	if err != nil {
		return nil, errors.Wrap(err, "get workload driver")
	}
	client, ok := querier.workloads[kind]
	if !ok {
		return nil, errors.Errorf("no k8s client for workload kind %s", kind)
	}

	if err = querier.sem.Acquire(ctx, 1); err != nil {
		return nil, errors.Wrap(err, "acquire k8s request slot")
	}
	defer querier.sem.Release(1)

	ns := item.Record.Namespace
	res, err := client.Get(ctx, ns, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "get %s %s/%s", kind, ns, name)
	}
	return driver.View(res.Object)
}

// instanceCounts 仅在 Pod List 成功且工作负载带有 replicas 时返回实例数，否则为 nil。
// replicas 缺失时视为「期望数不可用」：K8s 缺省虽为 1，总览更稳妥地不去猜。
func instanceCounts(replicas *int32, pods []unstructured.Unstructured, podsOK bool) *InstanceCounts {
	if !podsOK || replicas == nil {
		return nil
	}
	running, abnormal := countPodStates(pods)
	return &InstanceCounts{
		Running:  running,
		Expected: *replicas,
		Abnormal: abnormal,
	}
}

// countPodStates 统计 Ready 与非 Ready 的 Pod 数。
func countPodStates(pods []unstructured.Unstructured) (running, abnormal int32) {
	for _, pod := range pods {
		if podstatus.IsReady(pod.Object) {
			running++
		} else {
			abnormal++
		}
	}
	return running, abnormal
}

// extractMainWorkload 从部署记录取主工作负载 Kind 与名称。
func extractMainWorkload(rec *appmodel.Record) (kind, name string) {
	return rec.MainWorkload()
}

// listPods 在环境命名空间内按 LabelSelector List Pod。
//
// selector 取自部署记录，值就是 GameDeployment 的 selector，所以只数得到这个 workload 的 Pod：
// 泳道的 workload 另有名字，不会被算进来；滚动更新期间新旧两代 Pod 并存，Running 可能暂时超过
// Expected，这是真实状态。
//
// 空 selector 会被 K8s 视为匹配全部，而标准环境的 namespace 由多个应用共用，
// 那样统计到的是整个 namespace 的实例，因此直接返回错误，由调用方把实例数降级为不可用。
//
// ResourceVersion="0" 表示读 apiserver 的 watch cache，不走 etcd。Pod 列表是本接口最大的一笔
// 查询，而总览只展示实例数，容忍数据略旧；需要强一致读的地方不要复用本函数。
func listPods(
	ctx context.Context,
	querier *clusterQuerier,
	item deployRecordForEnv,
) ([]unstructured.Unstructured, error) {
	if len(item.Record.LabelSelector) == 0 {
		return nil, errors.New("deploy record has no label selector")
	}
	if err := querier.sem.Acquire(ctx, 1); err != nil {
		return nil, errors.Wrap(err, "acquire k8s request slot")
	}
	defer querier.sem.Release(1)

	ns := item.Record.Namespace
	sel := labels.SelectorFromSet(item.Record.LabelSelector).String()
	list, err := querier.pods.List(ctx, ns, metav1.ListOptions{LabelSelector: sel, ResourceVersion: "0"})
	if err != nil {
		return nil, errors.Wrapf(err, "list pods in namespace %s", ns)
	}
	return list.Items, nil
}

// extractMainContainerResources 读取 workload 主容器的 CPU/内存 requests 与 limits。
// 找不到主容器或字段缺失时，对应字段为空字符串。
func extractMainContainerResources(
	ctx context.Context, ns, name string, containers []corev1.Container,
) ResourceSpec {
	c, found := lo.Find(containers, func(c corev1.Container) bool {
		return c.Name == defaults.WorkloadMainContainerName
	})
	if !found {
		if len(containers) > 0 {
			log.WarnAttrs(ctx, "deploy overview skips resources, workload has no main container",
				slog.String("namespace", ns),
				slog.String("name", name),
			)
		}
		return ResourceSpec{}
	}
	return ResourceSpec{
		CPULimits:      resourceQuantityString(c.Resources.Limits, corev1.ResourceCPU),
		CPURequests:    resourceQuantityString(c.Resources.Requests, corev1.ResourceCPU),
		MemoryLimits:   resourceQuantityString(c.Resources.Limits, corev1.ResourceMemory),
		MemoryRequests: resourceQuantityString(c.Resources.Requests, corev1.ResourceMemory),
	}
}

// resourceQuantityString 资源量不存在时返回空串，存在则透传 Quantity.String()。
func resourceQuantityString(list corev1.ResourceList, name corev1.ResourceName) string {
	q, ok := list[name]
	if !ok {
		return ""
	}
	return q.String()
}
