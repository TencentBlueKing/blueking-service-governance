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

package gpa

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

// 真实集群集成测试：直接把 GPA CR 下发到测试 K8s 集群并验证完整生命周期。
//
// 运行前提（参考 pkg/core/env/clusteraddon/query_test.go）：
//   - 通过环境变量 FOR_TEST_KUBE_CONFIG_PATH 指向 kubeconfig，或
//     FOR_TEST_KUBE_APISERVER_URL / FOR_TEST_KUBE_CA_DATA / FOR_TEST_KUBE_TOKEN_VALUE 提供集群凭证
//   - 测试集群已注册 generalpodautoscalers.autoscaling.tkex.tencent.com CRD
//
// 未配置集群时整组用例自动 Skip。
var _ = Describe("GPAService against a real cluster", Label("k8s"), func() {
	const testNamespace = "default"

	var (
		svc        *GPAService
		ctx        context.Context
		env        *bkmsenv.Environment
		clusterCfg *cluster.Config
		mockers    []*mockey.Mocker
		crName     string
	)

	// addMock 注册一个 mock 并登记到清理列表，AfterEach 统一释放，避免跨用例 re-mock。
	addMock := func(m *mockey.Mocker) {
		mockers = append(mockers, m)
	}

	BeforeEach(func() {
		var err error
		clusterCfg, err = testutil.TestClusterConfig("")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())

		mockers = nil
		// 让被测代码的 newK8sClient 连到真实测试集群
		addMock(mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build())
		// 跳过工作负载名解析，聚焦真实 K8s 下发链路
		addMock(mockey.Mock((*GPAService).resolveScaleTargetName).Return("it-workload", nil).Build())

		svc = &GPAService{}
		ctx = context.Background()
		crName = "gpa-it-" + stringx.Random(6)
		env = &bkmsenv.Environment{Name: "dev", WorkspaceID: "ws-it"}
		env.Cluster.ClusterID = "test-cluster"
		env.Cluster.Namespace = testNamespace

		// 测试集群未注册 generalpodautoscalers CRD 时，GVR 解析失败，整组用例直接 Skip
		if _, err := svc.newK8sClient(env.Cluster.ClusterID); err != nil {
			Skip("gpa CRD not registered in test cluster: " + err.Error())
		}
	})

	AfterEach(func() {
		if svc != nil && clusterCfg != nil {
			// 兜底清理：Delete 对不存在资源幂等返回 nil
			_ = svc.Delete(ctx, env, crName)
		}
		for _, m := range mockers {
			m.Release()
		}
		mockers = nil
	})

	It("should apply, get, list and delete a gpa CR through its full lifecycle", func() {
		config := &GPAConfig{
			Name:        crName,
			AppID:       "app-it",
			MinReplicas: 2,
			MaxReplicas: 10,
			Metrics: []GPAMetric{
				{Resource: ResourceCPU, AverageUtilization: 60},
				{Resource: ResourceMemory, AverageUtilization: 70},
			},
		}

		// Apply：在集群中创建 GeneralPodAutoscaler CR
		Expect(svc.Apply(ctx, env, config)).To(Succeed())

		// Get：能回查到刚下发的 CR，labels 正确解析（status 可能尚未由控制器填充）
		status, err := svc.Get(ctx, env, crName)
		Expect(err).NotTo(HaveOccurred())
		Expect(status.Name).To(Equal(crName))
		Expect(status.AppID).To(Equal("app-it"))
		Expect(status.WorkspaceID).To(Equal("ws-it"))
		Expect(status.EnvName).To(Equal("dev"))

		// ListByEnv：按 workspaceID + envName label 过滤应能列出该 CR
		statuses, err := svc.ListByEnv(ctx, env)
		Expect(err).NotTo(HaveOccurred())
		names := make([]string, 0, len(statuses))
		for _, s := range statuses {
			names = append(names, s.Name)
		}
		Expect(names).To(ContainElement(crName))

		// Apply 再次下发（同名）应幂等成功（Server-Side Apply）
		Expect(svc.Apply(ctx, env, config)).To(Succeed())

		// Delete：删除 CR
		Expect(svc.Delete(ctx, env, crName)).To(Succeed())

		// 删除后再 Get 应返回 ErrCRNotFound
		_, err = svc.Get(ctx, env, crName)
		Expect(err).To(MatchError(ErrCRNotFound))
	})

	It("should return ErrCRNotFound when getting a non-existent CR", func() {
		_, err := svc.Get(ctx, env, "gpa-it-not-exist-"+stringx.Random(4))
		Expect(err).To(MatchError(ErrCRNotFound))
	})

	It("should delete a non-existent CR idempotently", func() {
		// k8sclient.Delete 对 NotFound 吞掉并返回 nil（允许重复删除），与 portpool 行为一致
		err := svc.Delete(ctx, env, "gpa-it-not-exist-"+stringx.Random(4))
		Expect(err).NotTo(HaveOccurred())
	})

	It("should write the expected spec fields into the cluster CR", func() {
		config := &GPAConfig{
			Name:        crName,
			AppID:       "app-it",
			MinReplicas: 3,
			MaxReplicas: 8,
			Metrics:     []GPAMetric{{Resource: ResourceCPU, AverageUtilization: 55}},
		}
		Expect(svc.Apply(ctx, env, config)).To(Succeed())

		// 用底层 client 原始读取，校验 spec 字段确实写入集群
		k8sClient, err := svc.newK8sClient(env.Cluster.ClusterID)
		Expect(err).NotTo(HaveOccurred())
		obj, err := k8sClient.Get(ctx, env.Cluster.Namespace, crName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		spec, found, err := unstructuredNestedMap(obj.Object, "spec")
		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(spec["minReplicas"]).To(BeEquivalentTo(int64(3)))
		Expect(spec["maxReplicas"]).To(BeEquivalentTo(int64(8)))

		scaleTargetRef := spec["scaleTargetRef"].(map[string]any)
		Expect(scaleTargetRef["kind"]).To(Equal("GameDeployment"))
		Expect(scaleTargetRef["name"]).To(Equal("it-workload"))
	})
})

// unstructuredNestedMap 从 unstructured Object 中取出嵌套的 map 字段。
func unstructuredNestedMap(obj map[string]any, key string) (map[string]any, bool, error) {
	v, ok := obj[key]
	if !ok {
		return nil, false, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false, errors.Errorf("field %q is not a map", key)
	}
	return m, true, nil
}
