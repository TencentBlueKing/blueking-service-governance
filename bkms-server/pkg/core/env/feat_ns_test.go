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

package env_test

import (
	"context"
	"errors"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

// 真实集群集成测试：验证特性环境 namespace 创建、托管 labels，以及对 AlreadyExists 的幂等处理。
//
// 运行前提（参考 pkg/infras/kubernetes/client/client_test.go）：
//   - 通过环境变量 FOR_TEST_KUBE_CONFIG_PATH 指向 kubeconfig，或
//     FOR_TEST_KUBE_APISERVER_URL / FOR_TEST_KUBE_CA_DATA / FOR_TEST_KUBE_TOKEN_VALUE 提供集群凭证
//
// 未配置集群时整组用例自动 Skip。
var _ = Describe("FeatureEnvNamespaceInitializer against a real cluster", Label("k8s"), func() {
	var (
		ctx         context.Context
		initializer bkmsenv.FeatureEnvNamespaceInitializer
		clientSet   *kubernetes.Clientset
		clusterCfg  *cluster.Config
		namespace   string
		workspaceID string
		appID       string
		envName     string
		mocker      *mockey.Mocker
	)

	BeforeEach(func() {
		var err error
		clusterCfg, err = testutil.TestClusterConfig("test-cluster")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())

		clientSet, err = kubernetes.NewForConfig(clusterCfg.Rest)
		Expect(err).NotTo(HaveOccurred())

		mocker = mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()
		initializer = bkmsenv.NewFeatureEnvNamespaceInitializer()
		ctx = context.Background()

		suffix := stringx.Random(6)
		namespace = "feat-it-" + suffix
		workspaceID = "ws-it-" + suffix
		appID = "app-it-" + suffix
		envName = namespace
	})

	AfterEach(func() {
		if clientSet != nil && namespace != "" {
			_ = clientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
		}
		if mocker != nil {
			mocker.Release()
		}
	})

	It("creates a namespace with managed labels and treats AlreadyExists as success", func() {
		ownerLabels := map[string]string{
			bkmsenv.FeatureEnvNSLabelWorkspaceID: workspaceID,
			bkmsenv.FeatureEnvNSLabelEnvName:     envName,
			bkmsenv.FeatureEnvNSLabelAppID:       appID,
			bkmsenv.FeatureEnvNSLabelController:  bkmsenv.FeatureEnvNSControllerValue,
		}

		Expect(initializer.Initialize(ctx, clusterCfg.ClusterID, namespace, ownerLabels)).To(Succeed())

		ns, err := clientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(ns.Labels).To(HaveKeyWithValue(bkmsenv.FeatureEnvNSLabelWorkspaceID, workspaceID))
		Expect(ns.Labels).To(HaveKeyWithValue(bkmsenv.FeatureEnvNSLabelEnvName, envName))
		Expect(ns.Labels).To(HaveKeyWithValue(bkmsenv.FeatureEnvNSLabelAppID, appID))
		Expect(ns.Labels).To(HaveKeyWithValue(
			bkmsenv.FeatureEnvNSLabelController, bkmsenv.FeatureEnvNSControllerValue,
		))

		// 再次创建同名 namespace：AlreadyExists 应被视为成功。
		Expect(initializer.Initialize(ctx, clusterCfg.ClusterID, namespace, ownerLabels)).To(Succeed())

		_, err = clientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
	})
})
