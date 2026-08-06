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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/discovery"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// 集成测试相关常量
const (
	// testAppID 测试应用 ID
	testAppID = "awesome-store"
	// testEnvName 测试环境名称
	testEnvName = "dev"
	// testReleaseName 测试 Helm Release 名称
	testReleaseName = "awesome-store"
	// testNamespace 固定测试命名空间
	testNamespace = "topo-it-test"
	// testDeploymentName 测试 Deployment 名称
	testDeploymentName = "awesome-store-web"
	// testDeploymentReplicas 测试 Deployment 的期望副本数
	testDeploymentReplicas = 2
	// testDataVersion 测试数据版本号
	testDataVersion = int64(1)
	// expectedStaticResourceCount manifest 中声明的静态资源数量（CM + Secret + SA + Deploy + SVC + Ing + HPA）
	expectedStaticResourceCount = 7
	// expectedNodeCount 精确节点数量（1 App + 7 静态 + 1 RS + 2 Pod）
	expectedNodeCount = 11
	// expectedEdgeCount 精确边数量（7 MANAGES + 3 CREATES + 1 ROUTES_TO + 4 SELECTS
	// + 8 MOUNTS + 1 SCALES + 5 REFERENCES）
	expectedEdgeCount = 29
	// testRSName 测试 ReplicaSet 名称
	testRSName = "awesome-store-web-7f4bcf96d6"
	// testPod1Name 测试 Pod 1 名称
	testPod1Name = "awesome-store-web-7f4bcf96d6-6ztxk"
	// testPod2Name 测试 Pod 2 名称
	testPod2Name = "awesome-store-web-7f4bcf96d6-vts8g"
	// testPod1IP 测试 Pod 1 的 IP
	testPod1IP = "127.0.0.1"
	// testPod2IP 测试 Pod 2 的 IP
	testPod2IP = "127.0.0.2"
	// testContainerImage 测试容器镜像
	testContainerImage = "nginx:1.25"
	// testServiceClusterIP 测试 Service 的 ClusterIP
	testServiceClusterIP = "127.0.0.3"
	// testIngressHost 测试 Ingress 的主机名
	testIngressHost = "awesome-store.example.com"
	// testGeneration 测试用的 metadata.generation 值，模拟 K8s API Server 返回的对象
	testGeneration = int64(1)
	// testEscapedNamespace 测试 Helm 资源显式声明的非环境命名空间
	testEscapedNamespace = "kk-system"
	// expectedEscapedNodeCount 跨命名空间场景节点数量（1 App + 1 Deploy + 1 RS + 2 Pod）
	expectedEscapedNodeCount = 5
	// expectedEscapedCreatesEdgeCount 跨命名空间场景 CREATES 边数量（Deploy -> RS + RS -> Pod x2）
	expectedEscapedCreatesEdgeCount = 3
)

var testManifestPath string

func init() {
	_, filename, _, _ := runtime.Caller(0)
	testManifestPath = filepath.Join(filepath.Dir(filename), "assets", "awesome_store_manifest.yaml")
}

// gvrMapping 资源 Kind 到 GVR 的静态映射（用于 mock discovery.GetGroupVersionResource）
var gvrMapping = map[string]schema.GroupVersionResource{
	k8skind.CM:     {Group: "", Version: "v1", Resource: "configmaps"},
	k8skind.Secret: {Group: "", Version: "v1", Resource: "secrets"},
	k8skind.SA:     {Group: "", Version: "v1", Resource: "serviceaccounts"},
	k8skind.Deploy: {Group: "apps", Version: "v1", Resource: "deployments"},
	k8skind.SVC:    {Group: "", Version: "v1", Resource: "services"},
	k8skind.Ing:    {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	k8skind.HPA:    {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
	k8skind.RS:     {Group: "apps", Version: "v1", Resource: "replicasets"},
	k8skind.Po:     {Group: "", Version: "v1", Resource: "pods"},
}

// gvrToKind GVR.Resource 到 Kind 的反向映射，用于 Get mock 中根据 lastGVR 确定请求的资源类型
var gvrToKind = func() map[string]string {
	m := make(map[string]string, len(gvrMapping))
	for kind, g := range gvrMapping {
		m[g.Resource] = kind
	}
	return m
}()

var _ = Describe("Integration - HelmRelease Topology", func() {
	var (
		ctx         context.Context
		diApp       *fxtest.App
		mockers     []*mockey.Mocker
		store       *ResourceSnapshotStoreMongo
		topologySvc *Service
		// allResources 所有预构造的 mock 资源（静态 + 动态），key 为 "Kind/Namespace/Name"
		allResources map[string]*unstructured.Unstructured
		// staticResources 仅静态资源（manifest 中声明的 7 种），用于 RelationCollector
		staticResources map[string]*unstructured.Unstructured
		// mockRS 和 mockPods 用于 List mock 返回
		mockRS   *unstructured.Unstructured
		mockPods []*unstructured.Unstructured
	)

	BeforeEach(func() {
		ctx = context.Background()

		// ========== 构造所有 mock 资源对象 ==========
		allResources = make(map[string]*unstructured.Unstructured)
		staticResources = make(map[string]*unstructured.Unstructured)

		// 静态资源：从 manifest 解析并注入运行时 status
		manifestBytes, readErr := os.ReadFile(testManifestPath)
		Expect(readErr).NotTo(HaveOccurred())
		staticResources = parseManifestToUnstructuredMap(string(manifestBytes), testNamespace)
		for key, obj := range staticResources {
			injectRuntimeStatus(obj)
			allResources[key] = obj
		}

		// 动态资源
		mockRS = buildMockReplicaSetInNamespace(testNamespace)
		mockPods = buildMockPodsInNamespace(testNamespace)

		rsKey := ResourceKey(mockRS.GetKind(), mockRS.GetNamespace(), mockRS.GetName())
		allResources[rsKey] = mockRS
		for _, pod := range mockPods {
			podKey := ResourceKey(pod.GetKind(), pod.GetNamespace(), pod.GetName())
			allResources[podKey] = pod
		}

		// ========== 设置所有 mock ==========
		mockers = nil

		// mock cluster.NewConfig：返回一个空的 Config（Rest 为 nil，不会被使用）
		m1 := mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build()
		mockers = append(mockers, m1)

		// mock discovery.GetGroupVersionResource：根据 kind 返回静态 GVR
		m2 := mockey.Mock(discovery.GetGroupVersionResource).To(func(
			_ *cluster.Config, kind, _ string,
		) (*schema.GroupVersionResource, error) {
			if gvr, ok := gvrMapping[kind]; ok {
				return &gvr, nil
			}
			return nil, errors.Errorf("unknown kind: %s", kind)
		}).Build()
		mockers = append(mockers, m2)

		// mock k8sclient.NewWithGVR：返回空 Client，并记录每个 Client 实例对应的 GVR
		// 使用 sync.Map 避免并发 goroutine 之间的竞态问题
		var clientGVRMap sync.Map // key: *k8sclient.Client, value: schema.GroupVersionResource
		m3 := mockey.Mock(k8sclient.NewWithGVR).To(func(
			_ *cluster.Config, gvr schema.GroupVersionResource,
		) *k8sclient.Client {
			cli := &k8sclient.Client{}
			clientGVRMap.Store(cli, gvr)
			return cli
		}).Build()
		mockers = append(mockers, m3)

		// mock (*k8sclient.Client).Get：根据 client 实例查找对应 GVR，再按 kind+namespace+name 精确查找
		m4 := mockey.Mock((*k8sclient.Client).Get).To(func(
			cli *k8sclient.Client, _ context.Context, namespace, name string, _ metav1.GetOptions,
		) (*unstructured.Unstructured, error) {
			gvrVal, ok := clientGVRMap.Load(cli)
			if !ok {
				return nil, k8sclient.ErrResourceNotFound
			}
			gvr := gvrVal.(schema.GroupVersionResource)
			kind := gvrToKind[gvr.Resource]
			key := ResourceKey(kind, namespace, name)
			if obj, found := allResources[key]; found {
				return obj, nil
			}
			return nil, k8sclient.ErrResourceNotFound
		}).Build()
		mockers = append(mockers, m4)

		// mock (*k8sclient.Client).List：根据 client 实例查找对应 GVR，区分 RS 和 Pod 列表
		m5 := mockey.Mock((*k8sclient.Client).List).To(func(
			cli *k8sclient.Client, _ context.Context, namespace string, _ metav1.ListOptions,
		) (*unstructured.UnstructuredList, error) {
			gvrVal, ok := clientGVRMap.Load(cli)
			if !ok {
				return &unstructured.UnstructuredList{}, nil
			}
			gvr := gvrVal.(schema.GroupVersionResource)
			switch gvr.Resource {
			case "replicasets":
				if mockRS.GetNamespace() != namespace {
					return &unstructured.UnstructuredList{}, nil
				}
				return &unstructured.UnstructuredList{Items: []unstructured.Unstructured{*mockRS}}, nil
			case "pods":
				items := make([]unstructured.Unstructured, 0, len(mockPods))
				for _, p := range mockPods {
					if p.GetNamespace() != namespace {
						continue
					}
					items = append(items, *p)
				}
				return &unstructured.UnstructuredList{Items: items}, nil
			default:
				return &unstructured.UnstructuredList{}, nil
			}
		}).Build()
		mockers = append(mockers, m5)

		// ========== 初始化 Store 和 Service ==========
		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()

		err := store.DeleteAll(ctx)
		Expect(err).NotTo(HaveOccurred())

		builder := NewBuilder()
		topologySvc = NewService(store, builder, nil, nil, nil, nil)
	})

	AfterEach(func() {
		// 释放所有 mock
		for _, m := range mockers {
			if m != nil {
				m.Release()
			}
		}

		// 清理 MongoDB 数据
		if store != nil {
			_ = store.DeleteAll(ctx)
		}
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("should build complete topology graph for HelmRelease scenario", func() {
		// ========== 步骤 1：构造 ResourceSnapshot 并入库 ==========

		// 读取 manifest 文件
		manifestBytes, err := os.ReadFile(testManifestPath)
		Expect(err).NotTo(HaveOccurred())
		manifestStr := string(manifestBytes)

		// 解析 manifest 得到 ResourceEntry 列表
		entries, err := ParseManifest(manifestStr, testNamespace)
		Expect(err).NotTo(HaveOccurred())
		Expect(entries).To(HaveLen(expectedStaticResourceCount))

		// 从预构造的静态资源中收集扩展关系
		relations := NewRelationCollector(staticResources).Collect()
		Expect(relations).NotTo(BeEmpty())

		// 构造 ResourceSnapshot
		snapshot := &ResourceSnapshot{
			AppID:           testAppID,
			EnvName:         testEnvName,
			TrafficLaneName: "",
			ClusterID:       "",
			Namespace:       testNamespace,
			ReleaseName:     testReleaseName,
			DataVersion:     testDataVersion,
			RefreshStatus:   RefreshStatusSuccess,
			RefreshedAt:     time.Now(),
			Resources:       entries,
			Relations:       relations,
		}

		// 写入 MongoDB
		err = store.UpsertWithVersion(ctx, snapshot, 0)
		Expect(err).NotTo(HaveOccurred())

		// ========== 步骤 2：调用 GetTopology ==========

		graph, err := topologySvc.GetTopology(ctx, testAppID, testEnvName, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(graph).NotTo(BeNil())

		// ========== 步骤 3：验证 Graph 顶层字段 ==========

		Expect(graph.IsPartial).To(BeFalse(), "graph should not be partial when all resources exist")
		Expect(graph.Warnings).To(BeEmpty(), "graph should have no warnings")
		Expect(graph.Metadata.AppID).To(Equal(testAppID))
		Expect(graph.Metadata.EnvName).To(Equal(testEnvName))
		Expect(graph.Metadata.TrafficLaneName).To(BeEmpty())
		Expect(graph.Metadata.ClusterID).To(BeEmpty())
		Expect(graph.Metadata.Namespace).To(Equal(testNamespace))
		Expect(graph.DataVersion).To(Equal(testDataVersion))
		Expect(graph.RootID).To(Equal(EncodeNodeID(NodeKindApp, "", testAppID)))
		Expect(graph.GeneratedAt).NotTo(BeEmpty())
		_, parseErr := time.Parse(time.RFC3339, graph.GeneratedAt)
		Expect(parseErr).NotTo(HaveOccurred(), "GeneratedAt should be valid RFC3339 format")

		// ========== 步骤 4：验证节点 ==========

		Expect(graph.Nodes).To(HaveLen(expectedNodeCount), fmt.Sprintf(
			"expected exactly %d nodes (1 App + 7 static + 1 RS + 2 Pod), got %d",
			expectedNodeCount, len(graph.Nodes),
		))

		// 构建节点查找辅助 map
		nodeByID := make(map[string]Node, len(graph.Nodes))
		for _, n := range graph.Nodes {
			nodeByID[n.ID] = n
		}

		// --- App 虚拟根节点 ---
		appNode := findNodeByKindAndName(graph.Nodes, NodeKindApp, testAppID)
		Expect(appNode).NotTo(BeNil(), "App root node should exist")
		Expect(appNode.Kind).To(Equal(NodeKindApp))
		Expect(appNode.Name).To(Equal(testAppID))
		Expect(appNode.Status).To(Equal(k8sstatus.Active))
		Expect(appNode.IsManaged).To(BeTrue())

		// --- Deployment 节点 ---
		deployNode := findNodeByKindAndName(graph.Nodes, k8skind.Deploy, testDeploymentName)
		Expect(deployNode).NotTo(BeNil(), "Deployment node should exist")
		Expect(deployNode.Status).To(Equal(k8sstatus.Available))
		Expect(deployNode.IsManaged).To(BeTrue())
		Expect(deployNode.Namespace).To(Equal(testNamespace))
		Expect(deployNode.Extras).To(HaveKeyWithValue(ExtrasKeyImage, testContainerImage))
		Expect(deployNode.Extras).To(HaveKeyWithValue(ExtrasKeyReplicas, "2"))
		Expect(deployNode.Extras).To(HaveKeyWithValue(ExtrasKeyReadyReplicas, "2"))

		// --- Service 节点 ---
		svcNode := findNodeByKindAndName(graph.Nodes, k8skind.SVC, "awesome-store-web")
		Expect(svcNode).NotTo(BeNil(), "Service node should exist")
		Expect(svcNode.Status).To(Equal(k8sstatus.Healthy))
		Expect(svcNode.IsManaged).To(BeTrue())
		Expect(svcNode.Namespace).To(Equal(testNamespace))
		Expect(svcNode.Extras).To(HaveKey(ExtrasKeyPorts))
		Expect(svcNode.Extras).To(HaveKey(ExtrasKeySelector))
		Expect(svcNode.Extras).To(HaveKeyWithValue(ExtrasKeyClusterIP, testServiceClusterIP))
		Expect(svcNode.Extras).To(HaveKeyWithValue(ExtrasKeyServiceType, "ClusterIP"))

		// --- Ingress 节点 ---
		ingNode := findNodeByKindAndName(graph.Nodes, k8skind.Ing, "awesome-store-web")
		Expect(ingNode).NotTo(BeNil(), "Ingress node should exist")
		Expect(ingNode.Status).NotTo(Equal(k8sstatus.NotFound))
		Expect(ingNode.IsManaged).To(BeTrue())
		Expect(ingNode.Namespace).To(Equal(testNamespace))
		Expect(ingNode.Extras).To(HaveKey(ExtrasKeyHost))
		Expect(ingNode.Extras[ExtrasKeyHost]).To(ContainSubstring(testIngressHost))

		// --- ConfigMap 节点 ---
		cmNode := findNodeByKindAndName(graph.Nodes, k8skind.CM, "awesome-store-config")
		Expect(cmNode).NotTo(BeNil(), "ConfigMap node should exist")
		Expect(cmNode.Status).To(Equal(k8sstatus.Healthy))
		Expect(cmNode.IsManaged).To(BeTrue())
		Expect(cmNode.Namespace).To(Equal(testNamespace))

		// --- Secret 节点 ---
		secretNode := findNodeByKindAndName(graph.Nodes, k8skind.Secret, "awesome-store-secret")
		Expect(secretNode).NotTo(BeNil(), "Secret node should exist")
		Expect(secretNode.Status).To(Equal(k8sstatus.Healthy))
		Expect(secretNode.IsManaged).To(BeTrue())
		Expect(secretNode.Namespace).To(Equal(testNamespace))

		// --- ServiceAccount 节点 ---
		saNode := findNodeByKindAndName(graph.Nodes, k8skind.SA, "awesome-store-web")
		Expect(saNode).NotTo(BeNil(), "ServiceAccount node should exist")
		Expect(saNode.Status).NotTo(Equal(k8sstatus.NotFound))
		Expect(saNode.IsManaged).To(BeTrue())
		Expect(saNode.Namespace).To(Equal(testNamespace))

		// --- HPA 节点 ---
		hpaNode := findNodeByKindAndName(graph.Nodes, k8skind.HPA, "awesome-store-web")
		Expect(hpaNode).NotTo(BeNil(), "HPA node should exist")
		Expect(hpaNode.Status).NotTo(Equal(k8sstatus.NotFound))
		Expect(hpaNode.IsManaged).To(BeTrue())
		Expect(hpaNode.Namespace).To(Equal(testNamespace))

		// --- ReplicaSet 节点（动态发现） ---
		rsNode := findNodeByKindAndName(graph.Nodes, k8skind.RS, testRSName)
		Expect(rsNode).NotTo(BeNil(), "ReplicaSet node should exist")
		Expect(rsNode.Kind).To(Equal(k8skind.RS))
		Expect(rsNode.IsManaged).To(BeFalse(), "ReplicaSet should not be managed")
		Expect(rsNode.Namespace).To(Equal(testNamespace))
		Expect(rsNode.Extras).To(HaveKeyWithValue(ExtrasKeyImage, testContainerImage))
		Expect(rsNode.Extras).To(HaveKeyWithValue(ExtrasKeyReplicas, "2"))
		Expect(rsNode.Extras).To(HaveKeyWithValue(ExtrasKeyReadyReplicas, "2"))

		// --- Pod 节点 1 ---
		pod1Node := findNodeByKindAndName(graph.Nodes, k8skind.Po, testPod1Name)
		Expect(pod1Node).NotTo(BeNil(), "Pod 1 node should exist")
		Expect(pod1Node.Kind).To(Equal(k8skind.Po))
		Expect(pod1Node.IsManaged).To(BeFalse(), "Pod should not be managed")
		Expect(pod1Node.Status).To(Equal(k8sstatus.Running))
		Expect(pod1Node.Namespace).To(Equal(testNamespace))
		Expect(pod1Node.Extras).To(HaveKeyWithValue(ExtrasKeyImage, testContainerImage))
		Expect(pod1Node.Extras).To(HaveKeyWithValue(ExtrasKeyPodIP, testPod1IP))

		// --- Pod 节点 2 ---
		pod2Node := findNodeByKindAndName(graph.Nodes, k8skind.Po, testPod2Name)
		Expect(pod2Node).NotTo(BeNil(), "Pod 2 node should exist")
		Expect(pod2Node.Kind).To(Equal(k8skind.Po))
		Expect(pod2Node.IsManaged).To(BeFalse(), "Pod should not be managed")
		Expect(pod2Node.Status).To(Equal(k8sstatus.Running))
		Expect(pod2Node.Namespace).To(Equal(testNamespace))
		Expect(pod2Node.Extras).To(HaveKeyWithValue(ExtrasKeyImage, testContainerImage))
		Expect(pod2Node.Extras).To(HaveKeyWithValue(ExtrasKeyPodIP, testPod2IP))

		// ========== 步骤 5：验证边 ==========

		// --- 边总数 ---
		Expect(graph.Edges).To(
			HaveLen(expectedEdgeCount),
			fmt.Sprintf("expected exactly %d edges, got %d", expectedEdgeCount, len(graph.Edges)),
		)

		// --- MANAGES 主边（App → 顶层资源）：7 条 ---
		managedKindsAndNames := []struct{ kind, name string }{
			{k8skind.Ing, "awesome-store-web"},
			{k8skind.SVC, "awesome-store-web"},
			{k8skind.Deploy, testDeploymentName},
			{k8skind.CM, "awesome-store-config"},
			{k8skind.Secret, "awesome-store-secret"},
			{k8skind.SA, "awesome-store-web"},
			{k8skind.HPA, "awesome-store-web"},
		}
		managesEdges := findEdgesByRelation(graph.Edges, EdgeRelationManages)
		Expect(managesEdges).To(HaveLen(len(managedKindsAndNames)), "should have exactly 7 MANAGES edges")
		for _, m := range managedKindsAndNames {
			edge := findEdge(graph.Edges, NodeKindApp, testAppID, m.kind, m.name, EdgeRelationManages)
			Expect(edge).NotTo(
				BeNil(), fmt.Sprintf("MANAGES edge from App to %s/%s should exist", m.kind, m.name),
			)
			Expect(edge.IsPrimary).To(BeTrue())
			Expect(edge.Reason.Type).To(Equal(RelationTypeAppRoot))
		}

		// --- CREATES 主边（Deployment → RS + RS → Pod x2）：3 条 ---
		createsEdges := findEdgesByRelation(graph.Edges, EdgeRelationCreates)
		Expect(createsEdges).To(HaveLen(3), fmt.Sprintf("should have exactly %d CREATES edges", 3))
		// Deployment → ReplicaSet
		createsDeployToRS := findEdge(
			graph.Edges, k8skind.Deploy, testDeploymentName,
			k8skind.RS, testRSName, EdgeRelationCreates,
		)
		Expect(createsDeployToRS).NotTo(BeNil(), "CREATES edge from Deployment to ReplicaSet should exist")
		Expect(createsDeployToRS.IsPrimary).To(BeTrue())
		Expect(createsDeployToRS.Reason.Type).To(Equal(RelationTypeOwnerReference))
		// ReplicaSet → Pod1
		createsRSToPod1 := findEdge(
			graph.Edges, k8skind.RS, testRSName,
			k8skind.Po, testPod1Name, EdgeRelationCreates,
		)
		Expect(createsRSToPod1).NotTo(BeNil(), "CREATES edge from RS to Pod 1 should exist")
		Expect(createsRSToPod1.IsPrimary).To(BeTrue())
		Expect(createsRSToPod1.Reason.Type).To(Equal(RelationTypeOwnerReference))
		// ReplicaSet → Pod2
		createsRSToPod2 := findEdge(
			graph.Edges, k8skind.RS, testRSName,
			k8skind.Po, testPod2Name, EdgeRelationCreates,
		)
		Expect(createsRSToPod2).NotTo(BeNil(), "CREATES edge from RS to Pod 2 should exist")
		Expect(createsRSToPod2.IsPrimary).To(BeTrue())
		Expect(createsRSToPod2.Reason.Type).To(Equal(RelationTypeOwnerReference))

		// --- ROUTES_TO 辅助边（Ingress → Service）：1 条 ---
		routesEdge := findEdge(
			graph.Edges, k8skind.Ing, "awesome-store-web",
			k8skind.SVC, "awesome-store-web", EdgeRelationRoutes,
		)
		Expect(routesEdge).NotTo(BeNil(), "ROUTES_TO edge from Ingress to Service should exist")
		Expect(routesEdge.IsPrimary).To(BeFalse())
		Expect(routesEdge.Reason.Type).To(Equal(RelationTypeBackendRef))

		// --- SELECTS 辅助边（Service/Deployment → Pod）：4 条 ---
		// Service → Pod x2, Deployment → Pod x2
		// 注意：RS → Pod 的 SELECTS 辅助边因已有同源同目标的 CREATES 主边而被去重跳过
		selectsEdges := findEdgesByRelation(graph.Edges, EdgeRelationSelects)
		Expect(selectsEdges).To(HaveLen(4), fmt.Sprintf("should have exactly %d SELECTS edges", 4))
		// 验证 Service → Pod 的 SELECTS 边
		for _, podName := range []string{testPod1Name, testPod2Name} {
			selEdge := findEdge(
				graph.Edges, k8skind.SVC, "awesome-store-web",
				k8skind.Po, podName, EdgeRelationSelects,
			)
			Expect(selEdge).NotTo(
				BeNil(), fmt.Sprintf("SELECTS edge from Service to %s should exist", podName),
			)
			Expect(selEdge.IsPrimary).To(BeFalse())
			Expect(selEdge.Reason.Type).To(Equal(RelationTypeLabelSelector))
			Expect(selEdge.Reason.MatchedLabels).To(
				HaveKeyWithValue("app.kubernetes.io/name", "awesome-store"),
			)
		}
		// 验证 Deployment → Pod 的 SELECTS 边
		for _, podName := range []string{testPod1Name, testPod2Name} {
			selEdge := findEdge(
				graph.Edges, k8skind.Deploy, testDeploymentName,
				k8skind.Po, podName, EdgeRelationSelects,
			)
			Expect(selEdge).NotTo(
				BeNil(), fmt.Sprintf("SELECTS edge from Deployment to %s should exist", podName),
			)
			Expect(selEdge.IsPrimary).To(BeFalse())
			Expect(selEdge.Reason.Type).To(Equal(RelationTypeLabelSelector))
		}
		// 验证 RS → Pod 的 SELECTS 辅助边不存在（因已有 CREATES 主边，被去重跳过）
		for _, podName := range []string{testPod1Name, testPod2Name} {
			selEdge := findEdge(
				graph.Edges, k8skind.RS, testRSName,
				k8skind.Po, podName, EdgeRelationSelects,
			)
			Expect(selEdge).To(BeNil(), fmt.Sprintf(
				"SELECTS edge from ReplicaSet to %s should NOT exist (deduplicated by primary edge)", podName,
			))
		}

		// --- MOUNTS 辅助边：8 条 ---
		// Deploy → CM
		// Deploy → Secret
		// RS → CM
		// RS → Secret
		// Pod1 → CM
		// Pod1 → Secret
		// Pod2 → CM
		// Pod2 → Secret
		mountsEdges := findEdgesByRelation(graph.Edges, EdgeRelationMounts)
		Expect(mountsEdges).To(HaveLen(8), "should have exactly 8 MOUNTS edges")
		// Deployment → ConfigMap
		mountsCMEdge := findEdge(
			graph.Edges, k8skind.Deploy, testDeploymentName,
			k8skind.CM, "awesome-store-config", EdgeRelationMounts,
		)
		Expect(mountsCMEdge).NotTo(BeNil(), "MOUNTS edge from Deployment to ConfigMap should exist")
		Expect(mountsCMEdge.IsPrimary).To(BeFalse())
		Expect(mountsCMEdge.Reason.Type).To(Equal(RelationTypeVolumeMount))
		// Deployment → Secret
		mountsSecretEdge := findEdge(
			graph.Edges, k8skind.Deploy, testDeploymentName,
			k8skind.Secret, "awesome-store-secret", EdgeRelationMounts,
		)
		Expect(mountsSecretEdge).NotTo(BeNil(), "MOUNTS edge from Deployment to Secret should exist")
		Expect(mountsSecretEdge.IsPrimary).To(BeFalse())
		Expect(mountsSecretEdge.Reason.Type).To(Equal(RelationTypeVolumeMount))
		// ReplicaSet → ConfigMap
		rsMountsCM := findEdge(
			graph.Edges, k8skind.RS, testRSName,
			k8skind.CM, "awesome-store-config", EdgeRelationMounts,
		)
		Expect(rsMountsCM).NotTo(BeNil(), "MOUNTS edge from ReplicaSet to ConfigMap should exist")
		Expect(rsMountsCM.IsPrimary).To(BeFalse())
		Expect(rsMountsCM.Reason.Type).To(Equal(RelationTypeVolumeMount))
		// ReplicaSet → Secret
		rsMountsSecret := findEdge(
			graph.Edges, k8skind.RS, testRSName,
			k8skind.Secret, "awesome-store-secret", EdgeRelationMounts,
		)
		Expect(rsMountsSecret).NotTo(BeNil(), "MOUNTS edge from ReplicaSet to Secret should exist")
		Expect(rsMountsSecret.IsPrimary).To(BeFalse())
		Expect(rsMountsSecret.Reason.Type).To(Equal(RelationTypeVolumeMount))
		// Pod → ConfigMap / Secret（2 个 Pod 各 2 条）
		for _, podName := range []string{testPod1Name, testPod2Name} {
			podMountsCM := findEdge(
				graph.Edges, k8skind.Po, podName,
				k8skind.CM, "awesome-store-config", EdgeRelationMounts,
			)
			Expect(podMountsCM).NotTo(
				BeNil(), fmt.Sprintf("MOUNTS edge from %s to ConfigMap should exist", podName),
			)
			Expect(podMountsCM.IsPrimary).To(BeFalse())
			podMountsSecret := findEdge(
				graph.Edges, k8skind.Po, podName,
				k8skind.Secret, "awesome-store-secret", EdgeRelationMounts,
			)
			Expect(podMountsSecret).NotTo(
				BeNil(), fmt.Sprintf("MOUNTS edge from %s to Secret should exist", podName),
			)
			Expect(podMountsSecret.IsPrimary).To(BeFalse())
		}

		// --- REFERENCES 辅助边（env_ref）：Deploy→Secret + RS→Secret + Pod1→Secret + Pod2→Secret ---
		// Deployment → Secret (env_ref)
		refSecretEdge := findEdge(
			graph.Edges, k8skind.Deploy, testDeploymentName,
			k8skind.Secret, "awesome-store-secret", EdgeRelationReferences,
		)
		Expect(refSecretEdge).NotTo(BeNil(), "REFERENCES edge from Deployment to Secret should exist")
		Expect(refSecretEdge.IsPrimary).To(BeFalse())
		Expect(refSecretEdge.Reason.Type).To(Equal(RelationTypeEnvRef))
		// ReplicaSet → Secret (env_ref)
		rsRefSecret := findEdge(
			graph.Edges, k8skind.RS, testRSName,
			k8skind.Secret, "awesome-store-secret", EdgeRelationReferences,
		)
		Expect(rsRefSecret).NotTo(BeNil(), "REFERENCES edge from ReplicaSet to Secret should exist")
		Expect(rsRefSecret.IsPrimary).To(BeFalse())
		Expect(rsRefSecret.Reason.Type).To(Equal(RelationTypeEnvRef))
		// Pod → Secret (env_ref)（2 个 Pod 各 1 条）
		for _, podName := range []string{testPod1Name, testPod2Name} {
			podRefSecret := findEdge(
				graph.Edges, k8skind.Po, podName,
				k8skind.Secret, "awesome-store-secret", EdgeRelationReferences,
			)
			Expect(podRefSecret).NotTo(
				BeNil(), fmt.Sprintf("REFERENCES edge from %s to Secret should exist", podName),
			)
			Expect(podRefSecret.IsPrimary).To(BeFalse())
			Expect(podRefSecret.Reason.Type).To(Equal(RelationTypeEnvRef))
		}

		// --- SCALES 辅助边（HPA → Deployment）：1 条 ---
		scalesEdge := findEdge(
			graph.Edges, k8skind.HPA, "awesome-store-web",
			k8skind.Deploy, testDeploymentName, EdgeRelationScales,
		)
		Expect(scalesEdge).NotTo(BeNil(), "SCALES edge from HPA to Deployment should exist")
		Expect(scalesEdge.IsPrimary).To(BeFalse())
		Expect(scalesEdge.Reason.Type).To(Equal(RelationTypeScaleTargetRef))
		Expect(scalesEdge.Reason.SourceFieldPath).To(Equal("spec.scaleTargetRef"))
		Expect(scalesEdge.Reason.TargetFieldPath).To(Equal("metadata.name"))

		// --- REFERENCES 辅助边（Deployment → ServiceAccount）：1 条 ---
		referencesEdge := findEdge(
			graph.Edges, k8skind.Deploy, testDeploymentName,
			k8skind.SA, "awesome-store-web", EdgeRelationReferences,
		)
		Expect(referencesEdge).NotTo(BeNil(), "REFERENCES edge from Deployment to ServiceAccount should exist")
		Expect(referencesEdge.IsPrimary).To(BeFalse())
		Expect(referencesEdge.Reason.Type).To(Equal(RelationTypeServiceAccountRef))
		Expect(referencesEdge.Reason.SourceFieldPath).To(Equal("spec.template.spec.serviceAccountName"))
		Expect(referencesEdge.Reason.TargetFieldPath).To(Equal("metadata.name"))

		// --- 边完整性校验：所有边的 SourceID / TargetID 都在节点中存在 ---
		for _, edge := range graph.Edges {
			_, srcExists := nodeByID[edge.SourceID]
			Expect(srcExists).To(
				BeTrue(), fmt.Sprintf("edge %s SourceID %s should exist in nodes", edge.Relation, edge.SourceID),
			)
			_, tgtExists := nodeByID[edge.TargetID]
			Expect(tgtExists).To(
				BeTrue(), fmt.Sprintf("edge %s TargetID %s should exist in nodes", edge.Relation, edge.TargetID),
			)
		}
	})

	It("should discover workload children from the workload namespace", func() {
		escapedDeploy := staticResources[ResourceKey(k8skind.Deploy, testNamespace, testDeploymentName)].DeepCopy()
		escapedDeploy.Object["metadata"].(map[string]any)["namespace"] = testEscapedNamespace
		allResources[ResourceKey(k8skind.Deploy, testEscapedNamespace, testDeploymentName)] = escapedDeploy

		mockRS = buildMockReplicaSetInNamespace(testEscapedNamespace)
		allResources[ResourceKey(k8skind.RS, testEscapedNamespace, testRSName)] = mockRS

		mockPods = buildMockPodsInNamespace(testEscapedNamespace)
		for _, pod := range mockPods {
			allResources[ResourceKey(k8skind.Po, testEscapedNamespace, pod.GetName())] = pod
		}

		snapshot := &ResourceSnapshot{
			AppID:           testAppID,
			EnvName:         testEnvName,
			TrafficLaneName: "",
			ClusterID:       "",
			Namespace:       testNamespace,
			ReleaseName:     testReleaseName,
			DataVersion:     testDataVersion,
			RefreshStatus:   RefreshStatusSuccess,
			RefreshedAt:     time.Now(),
			Resources: []ResourceEntry{
				{
					Kind:       k8skind.Deploy,
					APIVersion: "apps/v1",
					Namespace:  testEscapedNamespace,
					Name:       testDeploymentName,
					IsManaged:  true,
					SourceType: SourceTypeHelmManifest,
				},
			},
		}

		graph, err := NewBuilder().Build(ctx, snapshot)
		Expect(err).NotTo(HaveOccurred())
		Expect(graph).NotTo(BeNil())
		Expect(graph.Warnings).To(BeEmpty())
		Expect(graph.Nodes).To(HaveLen(expectedEscapedNodeCount))

		deployNode := findNodeByKindAndName(graph.Nodes, k8skind.Deploy, testDeploymentName)
		Expect(deployNode).NotTo(BeNil())
		Expect(deployNode.Namespace).To(Equal(testEscapedNamespace))

		rsNode := findNodeByKindAndName(graph.Nodes, k8skind.RS, testRSName)
		Expect(rsNode).NotTo(BeNil())
		Expect(rsNode.Namespace).To(Equal(testEscapedNamespace))

		for _, podName := range []string{testPod1Name, testPod2Name} {
			podNode := findNodeByKindAndName(graph.Nodes, k8skind.Po, podName)
			Expect(podNode).NotTo(BeNil())
			Expect(podNode.Namespace).To(Equal(testEscapedNamespace))
		}

		createsEdges := findEdgesByRelation(graph.Edges, EdgeRelationCreates)
		Expect(createsEdges).To(HaveLen(expectedEscapedCreatesEdgeCount))
		Expect(findEdge(
			graph.Edges, k8skind.Deploy, testDeploymentName,
			k8skind.RS, testRSName, EdgeRelationCreates,
		)).NotTo(BeNil())
		for _, podName := range []string{testPod1Name, testPod2Name} {
			Expect(findEdge(
				graph.Edges, k8skind.RS, testRSName,
				k8skind.Po, podName, EdgeRelationCreates,
			)).NotTo(BeNil())
		}
	})
})

// ======================== Manifest 解析 & 运行时 Status 注入 ========================

// parseManifestToUnstructuredMap 将多文档 YAML manifest 解析为 unstructured 对象 map，
// key 为 ResourceKey(kind, namespace, name)。
// 对于缺少 namespace 的非集群级别资源，使用 defaultNamespace 补全。
func parseManifestToUnstructuredMap(manifest, defaultNamespace string) map[string]*unstructured.Unstructured {
	result := make(map[string]*unstructured.Unstructured)

	docs := strings.Split(manifest, "---")
	for _, doc := range docs {
		trimmed := strings.TrimSpace(doc)
		if trimmed == "" {
			continue
		}

		var obj map[string]any
		if err := yaml.Unmarshal([]byte(trimmed), &obj); err != nil {
			continue
		}
		if obj == nil {
			continue
		}

		kindVal, _ := obj["kind"].(string)
		apiVersion, _ := obj["apiVersion"].(string)
		if kindVal == "" || apiVersion == "" {
			continue
		}

		// 注入 namespace
		metadata, _ := obj["metadata"].(map[string]any)
		if metadata == nil {
			continue
		}
		name, _ := metadata["name"].(string)
		if name == "" {
			continue
		}

		ns, _ := metadata["namespace"].(string)
		if ns == "" && !k8skind.IsClusterScoped(kindVal) {
			metadata["namespace"] = defaultNamespace
			ns = defaultNamespace
		}

		// yaml.v3 将数字解析为 int，但 unstructured deep copy 只支持 int64/float64，
		// 需要递归转换以避免 "cannot deep copy int" panic
		normalizeMapValues(obj)

		u := &unstructured.Unstructured{Object: obj}

		key := ResourceKey(kindVal, ns, name)
		result[key] = u
	}

	return result
}

// normalizeMapValues 递归遍历 map，将 yaml.v3 解析出的 int 转换为 int64，
// 以兼容 unstructured.NestedSlice 等方法的 deep copy 要求。
func normalizeMapValues(m map[string]any) {
	for k, v := range m {
		m[k] = normalizeValue(v)
	}
}

// normalizeValue 递归转换单个值中的 int → int64
func normalizeValue(v any) any {
	switch val := v.(type) {
	case int:
		return int64(val)
	case map[string]any:
		normalizeMapValues(val)
		return val
	case []any:
		for i, item := range val {
			val[i] = normalizeValue(item)
		}
		return val
	default:
		return v
	}
}

// injectRuntimeStatus 为从 manifest 解析出的静态资源注入运行时 status 信息，
// 模拟 K8s API Server 返回的真实对象（manifest 只包含声明态，不含 status）。
func injectRuntimeStatus(obj *unstructured.Unstructured) {
	switch obj.GetKind() {
	case k8skind.Deploy:
		// 注入 metadata.generation，模拟 K8s API Server 返回的对象
		if metadata, ok := obj.Object["metadata"].(map[string]any); ok {
			metadata["generation"] = testGeneration
		}
		obj.Object["status"] = map[string]any{
			"observedGeneration": testGeneration,
			"replicas":           int64(testDeploymentReplicas),
			"updatedReplicas":    int64(testDeploymentReplicas),
			"readyReplicas":      int64(testDeploymentReplicas),
			"availableReplicas":  int64(testDeploymentReplicas),
			"conditions": []any{
				map[string]any{"type": "Available", "status": "True"},
				map[string]any{"type": "Progressing", "status": "True"},
			},
		}
	case k8skind.SVC:
		// Service 的 clusterIP 是运行时分配的，manifest 中不包含
		if spec, ok := obj.Object["spec"].(map[string]any); ok {
			spec["clusterIP"] = testServiceClusterIP
		}
	case k8skind.Ing:
		obj.Object["status"] = map[string]any{
			"conditions": []any{
				map[string]any{"type": "Ready", "status": "True"},
			},
		}
	}
}

// buildMockPodSpec 构造与 Deployment manifest 一致的 Pod spec（serviceAccountName + containers + volumes），
// 供 ReplicaSet 的 spec.template.spec 和 Pod 的 spec 共用。
func buildMockPodSpec() map[string]any {
	return map[string]any{
		"serviceAccountName": "awesome-store-web",
		"containers": []any{
			map[string]any{
				"name":  "web",
				"image": testContainerImage,
				"env": []any{
					map[string]any{
						"name": "DB_PASSWORD",
						"valueFrom": map[string]any{
							"secretKeyRef": map[string]any{
								"name": "awesome-store-secret",
								"key":  "db-password",
							},
						},
					},
				},
				"volumeMounts": []any{
					map[string]any{
						"name":      "config-volume",
						"mountPath": "/etc/awesome-store",
					},
					map[string]any{
						"name":      "secret-volume",
						"mountPath": "/etc/awesome-store-secret",
						"readOnly":  true,
					},
				},
			},
		},
		"volumes": []any{
			map[string]any{
				"name": "config-volume",
				"configMap": map[string]any{
					"name": "awesome-store-config",
				},
			},
			map[string]any{
				"name": "secret-volume",
				"secret": map[string]any{
					"secretName": "awesome-store-secret",
				},
			},
		},
	}
}

// buildMockReplicaSetInNamespace 构造指定命名空间的 ReplicaSet 对象。
func buildMockReplicaSetInNamespace(namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       k8skind.RS,
			"metadata": map[string]any{
				"name":       testRSName,
				"namespace":  namespace,
				"generation": testGeneration,
				"labels": map[string]any{
					"app.kubernetes.io/name":      "awesome-store",
					"app.kubernetes.io/component": "web",
				},
				"ownerReferences": []any{
					map[string]any{
						"apiVersion": "apps/v1",
						"kind":       k8skind.Deploy,
						"name":       testDeploymentName,
						"uid":        "deploy-uid-001",
						"controller": true,
					},
				},
			},
			"spec": map[string]any{
				"replicas": int64(2),
				"selector": map[string]any{
					"matchLabels": map[string]any{
						"app.kubernetes.io/name":      "awesome-store",
						"app.kubernetes.io/component": "web",
					},
				},
				"template": map[string]any{
					"spec": buildMockPodSpec(),
				},
			},
			"status": map[string]any{
				"observedGeneration": testGeneration,
				"replicas":           int64(2),
				"readyReplicas":      int64(2),
			},
		},
	}
	return obj
}

// buildMockPodsInNamespace 构造指定命名空间的 Pod 对象。
func buildMockPodsInNamespace(namespace string) []*unstructured.Unstructured {
	podSpecs := []struct{ name, ip string }{
		{name: testPod1Name, ip: testPod1IP},
		{name: testPod2Name, ip: testPod2IP},
	}

	var pods []*unstructured.Unstructured
	for _, ps := range podSpecs {
		obj := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       k8skind.Po,
				"metadata": map[string]any{
					"name":      ps.name,
					"namespace": namespace,
					"labels": map[string]any{
						"app.kubernetes.io/name":      "awesome-store",
						"app.kubernetes.io/component": "web",
					},
					"ownerReferences": []any{
						map[string]any{
							"apiVersion": "apps/v1",
							"kind":       k8skind.RS,
							"name":       testRSName,
							"uid":        "rs-uid-001",
							"controller": true,
						},
					},
				},
				"spec": buildMockPodSpec(),
				"status": map[string]any{
					"phase": "Running",
					"podIP": ps.ip,
				},
			},
		}
		pods = append(pods, obj)
	}
	return pods
}

// ======================== 辅助函数 ========================

// findNodeByKindAndName 在节点列表中查找指定 Kind 和 Name 的节点
func findNodeByKindAndName(nodes []Node, kind, name string) *Node {
	for i := range nodes {
		if nodes[i].Kind == kind && nodes[i].Name == name {
			return &nodes[i]
		}
	}
	return nil
}

// findEdge 在边列表中查找指定源/目标 Kind+Name 和关系类型的边
func findEdge(edges []Edge, sourceKind, sourceName, targetKind, targetName string, relation EdgeRelation) *Edge {
	for i := range edges {
		e := &edges[i]
		if e.Relation != relation {
			continue
		}
		sKind, _, sName, sErr := DecodeNodeID(e.SourceID)
		tKind, _, tName, tErr := DecodeNodeID(e.TargetID)
		if sErr != nil || tErr != nil {
			continue
		}
		if sKind == sourceKind && sName == sourceName && tKind == targetKind && tName == targetName {
			return e
		}
	}
	return nil
}

// findEdgesByRelation 在边列表中查找所有指定关系类型的边
func findEdgesByRelation(edges []Edge, relation EdgeRelation) []Edge {
	var result []Edge
	for _, e := range edges {
		if e.Relation == relation {
			result = append(result, e)
		}
	}
	return result
}
