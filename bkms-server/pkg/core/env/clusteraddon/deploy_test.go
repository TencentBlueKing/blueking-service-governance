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

package clusteraddon_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

var _ = Describe("Deploy", func() {
	var (
		ctx context.Context
		// clusterID 为空时，使用本地 kubeconfig 指向的默认集群
		clusterID    string
		namespace    string
		addonDef     *clusteraddon.ClusterAddonDef
		repoIndex    *clusteraddon.RepoIndex
		chartVersion string
		store        clusteraddon.ClusterAddonDefStore
		mocker       *mockey.Mocker
		clusterCfg   *cluster.Config
	)

	BeforeEach(func() {
		ctx = context.Background()

		// 获取测试集群配置，mock cluster.NewConfig 使所有 k8s/helm 操作走测试集群
		var err error
		clusterCfg, err = testutil.TestClusterConfig("")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())
		mocker = mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()

		// 初始化 store 并通过 LoadBuiltinFromFolder 加载测试 addon 定义
		store, err = clusteraddon.NewClusterAddonDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		err = clusteraddon.LoadBuiltinFromFolder(ctx, store, "assets/testaddons/valid")
		Expect(err).NotTo(HaveOccurred())

		// 从 DB 中获取测试 addon 定义
		addonDef, err = store.Get(ctx, "bkms-test-chart")
		Expect(err).NotTo(HaveOccurred())

		// 检查 Helm 仓库是否可达并获取可用 chart 版本
		repoIndex, err = clusteraddon.FetchRepoIndex()
		if err != nil {
			Skip("Helm repo not reachable: " + err.Error())
		}

		versions := repoIndex.ListChartVersions(addonDef.ChartInfo.ChartName)
		if len(versions) == 0 {
			Skip("Chart " + addonDef.ChartInfo.ChartName + " not found in Helm repo")
		}
		chartVersion = versions[0]

		// 创建临时命名空间用于隔离测试
		namespace = "addon-test-" + stringx.Random(6)
	})

	AfterEach(func() {
		// 释放 mock，后续清理操作使用真实配置
		if mocker != nil {
			mocker.Release()
		}

		if namespace != "" {
			// 先尝试卸载 release（忽略错误，可能测试中已卸载）
			_ = clusteraddon.UninstallClusterAddon(ctx, addonDef, clusterID, namespace)
			// 回收命名空间
			clientSet, err := kubernetes.NewForConfig(clusterCfg.Rest)
			if err == nil {
				_ = clientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
			}
		}

		// 清理测试 addon 定义
		_, _ = store.Delete(ctx, addonDef.Name)
	})

	Describe("Full addon lifecycle: list → install → upgrade → uninstall", func() {
		// buildAndFindAddon 使用 DB 中的 addon 定义构建信息列表并返回匹配的 addon
		buildAndFindAddon := func() *clusteraddon.ClusterAddonInfo {
			addons := clusteraddon.BuildAddonInfoList(
				ctx, []*clusteraddon.ClusterAddonDef{addonDef}, namespace, clusterID, repoIndex,
			)
			Expect(addons).To(HaveLen(1))
			return addons[0]
		}

		It("should reflect correct status at each lifecycle stage", func() {
			By("1. 查询未安装状态的 addon 列表")
			addon := buildAndFindAddon()
			Expect(addon.InstallInfo.Status).To(Equal(helm.StatusUninstalled))
			Expect(addon.SupportedActions).To(Equal([]string{"install"}))
			Expect(addon.ChartInfo.AvailableVersions).NotTo(BeEmpty())
			Expect(addon.ChartInfo.DefaultChartVersion).To(Equal(chartVersion))
			Expect(addon.InstallInfo.CurrentChartVersion).To(BeEmpty())
			Expect(addon.InstallInfo.CurrentValues).To(BeEmpty())

			By("2. 安装 addon")
			err := clusteraddon.InstallOrUpgradeClusterAddon(
				ctx, addonDef, clusterID, namespace, chartVersion,
				map[string]any{"initialKey": "initialValue"},
			)
			Expect(err).NotTo(HaveOccurred())

			By("3. 查询已安装状态，验证 status、chartVersion、currentValues")
			addon = buildAndFindAddon()
			Expect(addon.InstallInfo.Status).To(Equal(helm.StatusDeployed))
			Expect(addon.SupportedActions).To(Equal([]string{"upgrade", "uninstall"}))
			Expect(addon.InstallInfo.CurrentChartVersion).To(Equal(chartVersion))
			Expect(addon.InstallInfo.CurrentValues).To(ContainSubstring("initialKey"))
			Expect(addon.InstallInfo.CurrentValues).To(ContainSubstring("initialValue"))

			By("4. 更新 addon（变更 values）")
			err = clusteraddon.InstallOrUpgradeClusterAddon(
				ctx, addonDef, clusterID, namespace, chartVersion,
				map[string]any{"updatedKey": "updatedValue"},
			)
			Expect(err).NotTo(HaveOccurred())

			By("5. 查询更新后状态，验证 values 已变更")
			addon = buildAndFindAddon()
			Expect(addon.InstallInfo.Status).To(Equal(helm.StatusDeployed))
			Expect(addon.InstallInfo.CurrentChartVersion).To(Equal(chartVersion))
			Expect(addon.InstallInfo.CurrentValues).To(ContainSubstring("updatedKey"))
			Expect(addon.InstallInfo.CurrentValues).To(ContainSubstring("updatedValue"))

			By("6. 卸载 addon")
			err = clusteraddon.UninstallClusterAddon(ctx, addonDef, clusterID, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("7. 查询卸载后状态，验证回到未安装")
			addon = buildAndFindAddon()
			Expect(addon.InstallInfo.Status).To(Equal(helm.StatusUninstalled))
			Expect(addon.SupportedActions).To(Equal([]string{"install"}))
			Expect(addon.InstallInfo.CurrentChartVersion).To(BeEmpty())
			Expect(addon.InstallInfo.CurrentValues).To(BeEmpty())
		})
	})
})
