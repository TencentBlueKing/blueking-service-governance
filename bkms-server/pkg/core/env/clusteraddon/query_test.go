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
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	helmrelease "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	clusteraddon "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/clusteraddon"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
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

			ctx = testutil.WithTenantOrDefault(context.Background())
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
			addons := clusteraddon.BuildAddonInfoList(
				ctx,
				addonDefs,
				addonEnvironment("fake-cluster", false),
				"",
				repoIndex,
			)

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

			addons := clusteraddon.BuildAddonInfoList(
				ctx,
				addonDefs,
				addonEnvironment("fake-cluster", false),
				"override-ns",
				repoIndex,
			)

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

			addons := clusteraddon.BuildAddonInfoList(
				ctx,
				addonDefs,
				addonEnvironment("fake-cluster", false),
				"",
				repoIndex,
			)

			Expect(addons).To(HaveLen(1))
			Expect(addons[0].InstallInfo.Namespace).To(Equal("custom-ns"))
		})
	})

	Describe("BuildAddonInfoList applicability", func() {
		DescribeTable("queries and returns only applicable addons",
			func(isFederation bool, expected []string) {
				mockey.PatchConvey("applicable addon queries", GinkgoT(), func() {
					var queried []string
					mockey.Mock(helm.NewActionConfiguration).To(
						func(_, namespace string, _ action.DebugLog) (*action.Configuration, error) {
							queried = append(queried, namespace)
							return &action.Configuration{}, nil
						},
					).Build()
					mockey.Mock(helm.ListReleases).Return(nil, nil).Build()
					defs := []*clusteraddon.ClusterAddonDef{
						{Name: "supported", ChartInfo: clusteraddon.HelmChartInfo{DefaultNamespace: "supported"}},
						{
							Name: "unsupported", UnsupportedOnFederation: true,
							ChartInfo: clusteraddon.HelmChartInfo{DefaultNamespace: "unsupported"},
						},
					}
					addons := clusteraddon.BuildAddonInfoList(
						context.Background(),
						defs,
						addonEnvironment("cluster", isFederation),
						"",
						clusteraddon.NewRepoIndex(&repo.IndexFile{}),
					)
					var names []string
					for _, addon := range addons {
						names = append(names, addon.Name)
					}
					Expect(names).To(ConsistOf(expected))
					Expect(queried).To(ConsistOf(expected))
				})
			},
			Entry("regular cluster", false, []string{"supported", "unsupported"}),
			Entry("federation cluster", true, []string{"supported"}),
		)
	})

	Describe("BuildAddonInfoList release status", func() {
		It("fills both addons from a single namespace query", func() {
			mockey.PatchConvey("shared release list", GinkgoT(), func() {
				defs := []*clusteraddon.ClusterAddonDef{gameDeployDef(), hookOperatorDef()}
				mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				query := mockey.Mock(helm.ListReleases).Return([]*helm.Release{
					{
						Name:         "custom-game",
						Chart:        helm.Chart{Name: defs[0].ChartInfo.ChartName, Version: "1.0.0"},
						DeployResult: helm.DeployResult{Status: helm.StatusDeployed},
						Values:       map[string]any{"replicas": 2},
					},
					{
						Name:         "custom-hook",
						Chart:        helm.Chart{Name: defs[1].ChartInfo.ChartName},
						DeployResult: helm.DeployResult{Status: helm.StatusFailed, Description: "install failed"},
					},
				}, nil).Build()
				addons := clusteraddon.BuildAddonInfoList(context.Background(), defs,
					addonEnvironment("cluster", false), "bcs-system", clusteraddon.NewRepoIndex(&repo.IndexFile{}))
				Expect(query.Times()).To(Equal(1))
				Expect(addons).To(HaveLen(2))
				Expect(addons[0].InstallInfo.Status).To(Equal(helm.StatusDeployed))
				Expect(addons[0].InstallInfo.CurrentChartVersion).To(Equal("1.0.0"))
				value, err := testutil.YAMLValueAt(addons[0].InstallInfo.CurrentValues, "replicas")
				Expect(err).NotTo(HaveOccurred())
				Expect(value).To(Equal(2))
				Expect(addons[1].InstallInfo.Status).To(Equal(helm.StatusFailed))
				Expect(addons[1].InstallInfo.Message).To(Equal("install failed"))
			})
		})

		It("marks the whole namespace unknown when listing fails", func() {
			mockey.PatchConvey("failed namespace query", GinkgoT(), func() {
				mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				query := mockey.Mock(helm.ListReleases).Return(nil, errors.New("access denied")).Build()
				addons := clusteraddon.BuildAddonInfoList(context.Background(),
					[]*clusteraddon.ClusterAddonDef{gameDeployDef(), hookOperatorDef()},
					addonEnvironment("cluster", false), "bcs-system", clusteraddon.NewRepoIndex(&repo.IndexFile{}))
				Expect(query.Times()).To(Equal(1))
				Expect(addons).To(HaveLen(2))
				for _, addon := range addons {
					Expect(addon.InstallInfo.Status).To(Equal(helm.StatusUnknown))
					Expect(addon.InstallInfo.Message).To(ContainSubstring("access denied"))
					Expect(addon.SupportedActions).To(BeEmpty())
				}
			})
		})
	})

	Describe("AddonAppTypeOf", func() {
		DescribeTable("should match AppModel apps by framework and Helm apps by type",
			func(app *bkmsapp.Application, expected string) {
				Expect(clusteraddon.AddonAppTypeOf(app)).To(Equal(expected))
			},
			Entry("legacy trpc", &bkmsapp.Application{Type: bkmsapp.AppTypeTRPC}, "trpc"),
			Entry("legacy taf", &bkmsapp.Application{Type: bkmsapp.AppTypeTAF}, "taf"),
			Entry("bkmsApp with trpc framework",
				&bkmsapp.Application{Type: bkmsapp.AppTypeBKMSApp, Framework: "trpc", Language: "go"}, "trpc"),
			Entry("bkmsApp with taf framework",
				&bkmsapp.Application{Type: bkmsapp.AppTypeBKMSApp, Framework: "taf"}, "taf"),
			// blank 尚未登记到组件定义，匹配不到任何组件，按管理方式表达见 M5
			Entry("bkmsApp with blank framework",
				&bkmsapp.Application{Type: bkmsapp.AppTypeBKMSApp, Framework: "blank", Language: "go"}, "blank"),
			Entry("helm", &bkmsapp.Application{Type: bkmsapp.AppTypeHelm}, "helm"),
			Entry("agones", &bkmsapp.Application{Type: bkmsapp.AppTypeAgones}, "agones"),
		)
	})

	Describe("InspectRequiredAddons", func() {
		It("queries only applicable required addons", func() {
			mockey.PatchConvey("applicable required addons", GinkgoT(), func() {
				defs := []*clusteraddon.ClusterAddonDef{gameDeployDef(), hookOperatorDef(), agonesDef()}
				defs[1].UnsupportedOnFederation = false
				mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				query := mockey.Mock(helm.ListReleases).Return(nil, nil).Build()
				missing, err := clusteraddon.InspectRequiredAddons(context.Background(), defs,
					bkmsapp.AppTypeTRPC, addonEnvironment("cluster", true), "")
				Expect(err).NotTo(HaveOccurred())
				Expect(
					missing,
				).To(Equal([]clusteraddon.AddonReference{{Name: defs[1].Name, DisplayName: defs[1].DisplayName}}))
				Expect(query.Times()).To(Equal(1))
			})
		})

		DescribeTable("queries each required namespace once",
			func(hookNamespace string, expectedNamespaces []string) {
				mockey.PatchConvey("required namespace queries", GinkgoT(), func() {
					defs := []*clusteraddon.ClusterAddonDef{gameDeployDef(), hookOperatorDef(), agonesDef()}
					defs[0].ChartInfo.DefaultNamespace = "game-ns"
					defs[1].ChartInfo.DefaultNamespace = hookNamespace
					var namespaces []string
					mockey.Mock(helm.NewActionConfiguration).To(
						func(clusterID, namespace string, _ action.DebugLog) (*action.Configuration, error) {
							Expect(clusterID).To(Equal("cluster"))
							namespaces = append(namespaces, namespace)
							return &action.Configuration{}, nil
						}).Build()
					query := mockey.Mock(helm.ListReleases).Return(nil, nil).Build()
					missing, err := clusteraddon.InspectRequiredAddons(context.Background(), defs,
						bkmsapp.AppTypeTRPC, addonEnvironment("cluster", false), "")
					Expect(err).NotTo(HaveOccurred())
					Expect(namespaces).To(Equal(expectedNamespaces))
					Expect(query.Times()).To(Equal(len(expectedNamespaces)))
					Expect(missing).To(Equal([]clusteraddon.AddonReference{
						{Name: defs[0].Name, DisplayName: defs[0].DisplayName},
						{Name: defs[1].Name, DisplayName: defs[1].DisplayName},
					}))
				})
			},
			Entry("shared namespace including an empty release list", "game-ns", []string{"game-ns"}),
			Entry("different namespaces", "hook-ns", []string{"game-ns", "hook-ns"}),
		)

		DescribeTable("classifies the actual release state",
			func(status clusteraddon.AddonStatus, isMissing bool) {
				mockey.PatchConvey("required addon states", GinkgoT(), func() {
					def := gameDeployDef()
					mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
					mockey.Mock(helm.ListReleases).Return([]*helm.Release{{
						Name: "custom-game", Chart: helm.Chart{Name: def.ChartInfo.ChartName},
						DeployResult: helm.DeployResult{Status: status},
					}}, nil).Build()
					missing, err := clusteraddon.InspectRequiredAddons(
						context.Background(),
						[]*clusteraddon.ClusterAddonDef{
							def,
						},
						bkmsapp.AppTypeTAF,
						addonEnvironment("cluster", false),
						"",
					)
					if status == helm.StatusUnknown {
						Expect(err).To(MatchError(ContainSubstring("release custom-game status is unknown")))
						return
					}
					Expect(err).NotTo(HaveOccurred())
					if isMissing {
						Expect(
							missing,
						).To(Equal([]clusteraddon.AddonReference{{Name: def.Name, DisplayName: def.DisplayName}}))
					} else {
						Expect(missing).To(BeEmpty())
					}
				})
			},
			Entry("deployed", helm.StatusDeployed, false),
			Entry("failed", helm.StatusFailed, true),
			Entry("pending upgrade", helm.StatusPendingUpgrade, true),
			Entry("unknown", helm.StatusUnknown, false),
		)

		DescribeTable("preserves query failures and cluster context",
			func(configFails bool) {
				mockey.PatchConvey("query failure", GinkgoT(), func() {
					cause := errors.New("access denied")
					config := mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
					query := mockey.Mock(helm.ListReleases).Return(nil, cause).Build()
					if configFails {
						config.Return(nil, cause)
					}
					_, err := clusteraddon.InspectRequiredAddons(context.Background(),
						[]*clusteraddon.ClusterAddonDef{gameDeployDef(), hookOperatorDef()},
						bkmsapp.AppTypeTRPC, addonEnvironment("target-cluster", false), "operator-ns")
					Expect(errors.Is(err, cause)).To(BeTrue())
					Expect(err.Error()).To(ContainSubstring("target-cluster"))
					Expect(err.Error()).To(ContainSubstring("operator-ns"))
					if configFails {
						Expect(query.Times()).To(BeZero())
					} else {
						Expect(query.Times()).To(Equal(1))
					}
				})
			},
			Entry("configuration", true),
			Entry("list releases", false),
		)

		It("does not query optional addons", func() {
			mockey.PatchConvey("optional addons", GinkgoT(), func() {
				def := gameDeployDef()
				def.OptionalForAppTypes = []string{bkmsapp.AppTypeHelm}
				config := mockey.Mock(helm.NewActionConfiguration).Return(&action.Configuration{}, nil).Build()
				missing, err := clusteraddon.InspectRequiredAddons(context.Background(),
					[]*clusteraddon.ClusterAddonDef{def}, bkmsapp.AppTypeHelm, addonEnvironment("cluster", false), "")
				Expect(err).NotTo(HaveOccurred())
				Expect(missing).To(BeEmpty())
				Expect(config.Times()).To(BeZero())
			})
		})
	})
})

func gameDeployDef() *clusteraddon.ClusterAddonDef {
	return &clusteraddon.ClusterAddonDef{
		Name:                    "bcs-gamedeployment-operator",
		DisplayName:             "Gamedeploy",
		ChartInfo:               clusteraddon.HelmChartInfo{ChartName: "bcs-gamedeployment-operator"},
		UnsupportedOnFederation: true,
		RequiredForAppTypes:     []string{bkmsapp.AppTypeTRPC, bkmsapp.AppTypeTAF},
	}
}

func hookOperatorDef() *clusteraddon.ClusterAddonDef {
	return &clusteraddon.ClusterAddonDef{
		Name:                    "bcs-hook-operator",
		DisplayName:             "Hook-operator",
		ChartInfo:               clusteraddon.HelmChartInfo{ChartName: "bcs-hook-operator"},
		UnsupportedOnFederation: true,
		RequiredForAppTypes:     []string{bkmsapp.AppTypeTRPC, bkmsapp.AppTypeTAF},
	}
}

func agonesDef() *clusteraddon.ClusterAddonDef {
	return &clusteraddon.ClusterAddonDef{
		Name:                "agones",
		DisplayName:         "agones",
		RequiredForAppTypes: []string{bkmsapp.AppTypeAgones},
	}
}

// addonEnvironment 构造组件测试所需的内存环境对象，仅设置集群信息，不写入数据库。
func addonEnvironment(clusterID string, isFederation bool) *envmodel.Environment {
	return &envmodel.Environment{Cluster: envmodel.BizCluster{ClusterID: clusterID, IsFederation: isFederation}}
}
