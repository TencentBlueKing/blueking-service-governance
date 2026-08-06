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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

var _ = Describe("Query", func() {
	Describe("GetSupportedActions", func() {
		DescribeTable("should return correct actions for each status",
			func(status helmrelease.Status, expected []string) {
				Expect(clusteraddon.GetSupportedActions(status)).To(Equal(expected))
			},
			Entry("empty status (not installed)", helmrelease.Status(""), []string{"install"}),
			Entry("uninstalled", helm.StatusUninstalled, []string{"install"}),
			Entry("deployed", helm.StatusDeployed, []string{"upgrade", "uninstall"}),
			Entry("failed", helm.StatusFailed, []string{"install", "uninstall"}),
		)

		It("should return nil for pending status", func() {
			Expect(clusteraddon.GetSupportedActions(helm.StatusPendingInstall)).To(BeNil())
			Expect(clusteraddon.GetSupportedActions(helm.StatusPendingUpgrade)).To(BeNil())
			Expect(clusteraddon.GetSupportedActions(helm.StatusPendingRollback)).To(BeNil())
		})
	})

	Describe("BuildAddonInfoList", func() {
		var (
			repoIndex  *clusteraddon.RepoIndex
			mocker     *mockey.Mocker
			clusterCfg *cluster.Config
			ctx        context.Context
		)

		BeforeEach(func() {
			var err error
			clusterCfg, err = testutil.TestClusterConfig("")
			if errors.Is(err, testutil.ErrKubeConfigNotFound) {
				Skip(err.Error())
			}
			Expect(err).NotTo(HaveOccurred())
			mocker = mockey.Mock(cluster.NewConfig).Return(clusterCfg).Build()
			indexFile := &repo.IndexFile{
				Entries: map[string]repo.ChartVersions{
					"chart-a": {
						{Metadata: &chart.Metadata{Version: "2.0.0"}},
						{Metadata: &chart.Metadata{Version: "1.0.0"}},
					},
					"chart-b": {
						{Metadata: &chart.Metadata{Version: "3.0.0"}},
					},
				},
			}
			repoIndex = clusteraddon.NewRepoIndex(indexFile)

			ctx = context.Background()
		})

		AfterEach(func() {
			if mocker != nil {
				mocker.Release()
			}
		})

		It("should fill available versions from repo index", func() {
			addonDefs := []*clusteraddon.ClusterAddonDef{
				{
					Name: "addon-a",
					ChartInfo: clusteraddon.HelmChartInfo{
						ChartName:           "chart-a",
						DefaultChartVersion: "0.9.0",
						DefaultNamespace:    "ns-a",
					},
				},
			}

			// 注意：BuildAddonInfoList 内部会调用 FillAddonStatusFromCluster，
			// 该函数依赖真实集群连接，在无集群环境中会走错误分支（返回 install action）
			addons := clusteraddon.BuildAddonInfoList(ctx, addonDefs, "", "fake-cluster", repoIndex)

			Expect(addons).To(HaveLen(1))
			Expect(addons[0].ChartInfo.AvailableVersions).To(Equal([]string{"2.0.0", "1.0.0"}))
			// 仓库最新版本应覆盖定义中的默认版本
			Expect(addons[0].ChartInfo.DefaultChartVersion).To(Equal("2.0.0"))
			Expect(addons[0].InstallInfo.Namespace).To(Equal("ns-a"))
		})

		It("should keep default version when chart not found in repo", func() {
			addonDefs := []*clusteraddon.ClusterAddonDef{
				{
					Name: "addon-x",
					ChartInfo: clusteraddon.HelmChartInfo{
						ChartName:           "non-existent",
						DefaultChartVersion: "0.5.0",
					},
				},
			}

			addons := clusteraddon.BuildAddonInfoList(ctx, addonDefs, "override-ns", "fake-cluster", repoIndex)

			Expect(addons).To(HaveLen(1))
			Expect(addons[0].ChartInfo.AvailableVersions).To(BeNil())
			Expect(addons[0].ChartInfo.DefaultChartVersion).To(Equal("0.5.0"))
			Expect(addons[0].InstallInfo.Namespace).To(Equal("override-ns"))
		})

		It("should use addon default namespace when request namespace is empty", func() {
			addonDefs := []*clusteraddon.ClusterAddonDef{
				{
					Name: "addon-b",
					ChartInfo: clusteraddon.HelmChartInfo{
						ChartName:        "chart-b",
						DefaultNamespace: "custom-ns",
					},
				},
			}

			addons := clusteraddon.BuildAddonInfoList(ctx, addonDefs, "", "fake-cluster", repoIndex)

			Expect(addons).To(HaveLen(1))
			Expect(addons[0].InstallInfo.Namespace).To(Equal("custom-ns"))
		})
	})
})
