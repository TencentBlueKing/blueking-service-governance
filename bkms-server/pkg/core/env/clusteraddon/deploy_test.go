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
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/postrender"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

var _ = Describe("InstallOrUpgradeClusterAddon", func() {
	DescribeTable("checks applicability before pulling the chart",
		func(unsupported, isFederation, rejected bool) {
			pullErr := errors.New("chart pull stopped for test")
			pullMock := mockey.Mock(helmdeploy.PullChart).Return(nil, nil, pullErr).Build()
			defer pullMock.UnPatch()
			def := &clusteraddon.ClusterAddonDef{Name: "test-addon", UnsupportedOnFederation: unsupported}
			err := clusteraddon.InstallOrUpgradeClusterAddon(
				context.Background(), def, addonEnvironment("cluster", isFederation), "namespace", "1.0.0", nil,
			)
			if rejected {
				Expect(errors.Is(err, clusteraddon.ErrAddonNotApplicable)).To(BeTrue())
				Expect(pullMock.Times()).To(BeZero())
			} else {
				Expect(errors.Is(err, pullErr)).To(BeTrue())
				Expect(pullMock.Times()).To(Equal(1))
			}
		},
		Entry("rejects unsupported addons on federation clusters", true, true, true),
		Entry("allows those addons on regular clusters", true, false, false),
		Entry("allows supported addons on federation clusters", false, true, false),
	)

	DescribeTable("uses the installed release name or the configured name for a new installation",
		func(installed *helm.Release, lookupErr error, expectedName string) {
			mockey.PatchConvey("install or upgrade release name", GinkgoT(), func() {
				def := &clusteraddon.ClusterAddonDef{Name: "bcs-hook-operator", ChartInfo: clusteraddon.HelmChartInfo{
					ChartName: "bcs-hook-operator", ReleaseName: "bcs-hook-operator",
				}}
				mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				mockey.Mock(helm.GetReleaseByChart).Return(installed, lookupErr).Build()
				mockey.Mock(helm.GetReleaseStatus).Return(nil, driver.ErrReleaseNotFound).Build()
				mockey.Mock(helmdeploy.PullChart).Return(&chart.Chart{}, &helmdeploy.LintResult{}, nil).Build()
				mockey.Mock(helmdeploy.RunHelmRelease).
					To(func(_ *action.Configuration, name, namespace string, _ *chart.Chart, _ map[string]any, _ bool, _ postrender.PostRenderer) (*helmrelease.Release, error) {
						Expect(name).To(Equal(expectedName))
						Expect(namespace).To(Equal("bcs-system"))
						return &helmrelease.Release{}, nil
					}).
					Build()
				Expect(clusteraddon.InstallOrUpgradeClusterAddon(context.Background(), def,
					addonEnvironment("cluster", false), "bcs-system", "1.0.0", nil)).To(Succeed())
			})
		},
		Entry("upgrade the discovered release", &helm.Release{Name: "hook-operator"}, nil, "hook-operator"),
		Entry("install with the configured name", nil, driver.ErrReleaseNotFound, "bcs-hook-operator"),
	)

	It("rejects a configured release name occupied by another chart", func() {
		mockey.PatchConvey("release name conflict", GinkgoT(), func() {
			def := hookOperatorDef()
			mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
			mockey.Mock(helmdeploy.PullChart).Return(&chart.Chart{}, &helmdeploy.LintResult{}, nil).Build()
			mockey.Mock(helm.GetReleaseByChart).Return(nil, driver.ErrReleaseNotFound).Build()
			mockey.Mock(helm.GetReleaseStatus).To(func(_ *action.Configuration, name string) (*helm.Release, error) {
				Expect(name).To(Equal(def.ChartInfo.ChartName))
				return &helm.Release{Name: name, Chart: helm.Chart{Name: "another-chart"}}, nil
			}).Build()
			deploy := mockey.Mock(helmdeploy.RunHelmRelease).Return(&helmrelease.Release{}, nil).Build()
			err := clusteraddon.InstallOrUpgradeClusterAddon(context.Background(), def,
				addonEnvironment("cluster", false), "bcs-system", "1.0.0", nil)
			Expect(err).To(MatchError(ContainSubstring("is already used by chart another-chart")))
			Expect(deploy.Times()).To(BeZero())
		})
	})
})

var _ = Describe("UninstallClusterAddon", func() {
	It("uninstalls the discovered release", func() {
		mockey.PatchConvey("uninstall release name", GinkgoT(), func() {
			def := &clusteraddon.ClusterAddonDef{Name: "bcs-hook-operator", ChartInfo: clusteraddon.HelmChartInfo{
				ChartName: "bcs-hook-operator", ReleaseName: "bcs-hook-operator",
			}}
			mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
			mockey.Mock(helm.GetReleaseByChart).Return(&helm.Release{Name: "hook-operator"}, nil).Build()
			mockey.Mock((*action.Uninstall).Run).
				To(func(_ *action.Uninstall, name string) (*helmrelease.UninstallReleaseResponse, error) {
					Expect(name).To(Equal("hook-operator"))
					return &helmrelease.UninstallReleaseResponse{}, nil
				}).
				Build()
			Expect(
				clusteraddon.UninstallClusterAddon(context.Background(), def, "cluster", "bcs-system"),
			).To(Succeed())
		})
	})
})

var _ = Describe("Deploy", func() {
	var (
		ctx context.Context
		// clusterID 为空时，使用本地 kubeconfig 指向的默认集群
		clusterID    string
		namespace    string
		addonDef     *clusteraddon.ClusterAddonDef
		repoIndex    *clusteraddon.RepoIndex
		chartVersion string
		store        *clusteraddon.ClusterAddonDefStore
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
		store = testutil.NewStore(GinkgoT(), clusteraddon.ClusterAddonDefMeta)

		err = clusteraddon.LoadBuiltinFromFolder(ctx, store, "assets/testaddons/valid")
		Expect(err).NotTo(HaveOccurred())

		// 从 DB 中获取测试 addon 定义
		addonDef, err = store.Get(ctx, clusteraddon.ByName("bkms-test-chart"))
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
		_ = store.DeleteIfExists(ctx, clusteraddon.ByName(addonDef.Name))
	})

	Describe("Full addon lifecycle: list → install → upgrade → uninstall", func() {
		// buildAndFindAddon 使用 DB 中的 addon 定义构建信息列表并返回匹配的 addon
		buildAndFindAddon := func() *clusteraddon.ClusterAddonInfo {
			addons := clusteraddon.BuildAddonInfoList(
				ctx,
				[]*clusteraddon.ClusterAddonDef{addonDef},
				addonEnvironment(clusterID, false),
				namespace,
				repoIndex,
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
				ctx, addonDef, addonEnvironment(clusterID, false), namespace, chartVersion,
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
				ctx, addonDef, addonEnvironment(clusterID, false), namespace, chartVersion,
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
