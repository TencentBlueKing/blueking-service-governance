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

package status_test

import (
	"context"
	"errors"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	helmrelease "helm.sh/helm/v3/pkg/release"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/autodeploy"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	. "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/trafficmanager"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

func reloadEnv(ctx context.Context, store envmodel.EnvironmentStore, id bson.ObjectID) *envmodel.Environment {
	env, err := store.Get(ctx, id)
	Expect(err).NotTo(HaveOccurred())
	return env
}

type listTrafficLanesMockFunc func(
	*trafficmanager.StubTrafficManager,
	context.Context,
	string,
	string,
) ([]*trafficmanager.TrafficLane, error)

func mockListTrafficLanesWithHook(hook listTrafficLanesMockFunc) {
	mocker := mockey.Mock((*trafficmanager.StubTrafficManager).ListTrafficLanes).To(hook).Build()
	DeferCleanup(func() {
		mocker.UnPatch()
	})
}

func mockListTrafficLanesReturn(lanes []*trafficmanager.TrafficLane, err error) {
	mockListTrafficLanesWithHook(func(
		_ *trafficmanager.StubTrafficManager,
		_ context.Context,
		_,
		_ string,
	) ([]*trafficmanager.TrafficLane, error) {
		return lanes, err
	})
}

var _ = Describe("DeployStatusService", func() {
	var (
		ctx                        context.Context
		diApp                      *fxtest.App
		appStore                   bkmsapp.ApplicationStore
		appModelStore              appmodel.AppModelStore
		appConfigFileStore         appcfg.AppConfigFileStore
		appConfigFileVersionStore  appcfg.AppConfigFileVersionStore
		buildConfigStore           build.ConfigStore
		buildAutoDeployRecordStore autodeploy.RecordStore
		envStore                   envmodel.EnvironmentStore
		envSvc                     *bkmsenv.EnvService
		appModelDeployRecordStore  appmodeldeploy.RecordStore
		helmDeployRecordStore      helmdeploy.RecordStore
		svc                        *DeployStatusService
	)

	BeforeEach(func() {
		ctx = context.Background()
		diApp = fxtest.New(
			GinkgoT(),
			bkmsapp.FxModule,
			appmodel.FxModule,
			appcfg.FxModule,
			bkmsenv.FxModule,
			appmodeldeploy.FxModule,
			helmdeploy.FxModule,
			build.FxModule,
			fx.Populate(
				&appStore,
				&appModelStore,
				&appConfigFileStore,
				&appConfigFileVersionStore,
				&buildConfigStore,
				&envStore,
				&envSvc,
				&appModelDeployRecordStore,
				&helmDeployRecordStore,
			),
		)
		diApp.RequireStart()
		var err error
		buildAutoDeployRecordStore, err = autodeploy.NewRecordStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		svc = NewDeployStatusService(
			appStore,
			envStore,
			buildAutoDeployRecordStore,
			appModelDeployRecordStore,
			helmDeployRecordStore,
		)
		svc.TrafficManager = &trafficmanager.StubTrafficManager{}
	})

	AfterEach(func() {
		Expect(envStore.DeleteAll(ctx)).To(Succeed())
		diApp.RequireStop()
	})

	Describe("ListForEnvironment", func() {
		Context("with env and tRPC app tracked in the env", func() {
			var (
				testEnv *envmodel.Environment
				trpcApp *bkmsapp.Application
			)

			BeforeEach(func() {
				trpcApp, _ = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
					AppStore:                  appStore,
					AppModelStore:             appModelStore,
					AppConfigFileStore:        appConfigFileStore,
					AppConfigFileVersionStore: appConfigFileVersionStore,
					BuildConfigStore:          buildConfigStore,
				}, nil)
				testEnv = dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
				Expect(envStore.AddApp(ctx, testEnv.ID, trpcApp.ID)).To(Succeed())
				testEnv = reloadEnv(ctx, envStore, testEnv.ID)
			})

			Context("when ListTrafficLanes fails", func() {
				It("should return an error", func() {
					mockListTrafficLanesReturn(nil, errors.New("rpc down"))

					_, err := svc.ListForEnvironment(ctx, testEnv)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("rpc down"))
				})
			})

			Context("when appmodel deploy record exists for a lane", func() {
				It("should return status for that lane", func() {
					mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: "lane-a"}}, nil)

					dbfactory.AppModelDeployRecord(
						ctx,
						appModelDeployRecordStore,
						trpcApp,
						testEnv,
						&dbfactory.AppModelDeployRecordOpts{
							TrafficLaneName: "lane-a",
							Status:          appmodeldeploy.StatusDeployed,
							ImageTag:        "v1",
						},
					)

					out, err := svc.ListForEnvironment(ctx, testEnv)
					Expect(err).NotTo(HaveOccurred())
					Expect(out).To(HaveLen(1))
					Expect(out[0]).To(MatchFields(IgnoreExtras, Fields{
						"TrafficLaneName": Equal("lane-a"),
						"DeployStatus":    Equal(string(appmodeldeploy.StatusDeployed)),
						"ImageTag":        Equal("v1"),
					}))
				})
			})

			Context("when first appmodel deploy attempt is still running", func() {
				It("should return the deploying status", func() {
					mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: ""}}, nil)

					dbfactory.AppModelDeployRecord(
						ctx,
						appModelDeployRecordStore,
						trpcApp,
						testEnv,
						&dbfactory.AppModelDeployRecordOpts{
							Status:   appmodeldeploy.StatusDeploying,
							ImageTag: "v1",
						},
					)
					out, err := svc.ListForEnvironment(ctx, testEnv)
					Expect(err).NotTo(HaveOccurred())
					Expect(out).To(HaveLen(1))
					Expect(out[0]).To(MatchFields(IgnoreExtras, Fields{
						"TrafficLaneName": Equal(""),
						"DeployStatus":    Equal(string(appmodeldeploy.StatusDeploying)),
						"ImageTag":        Equal("v1"),
					}))
				})
			})

			Context("when build auto deploy record is linked to latest deploy record", func() {
				It("should return the build auto deploy status", func() {
					mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: "lane-a"}}, nil)

					deployID := dbfactory.AppModelDeployRecord(
						ctx,
						appModelDeployRecordStore,
						trpcApp,
						testEnv,
						&dbfactory.AppModelDeployRecordOpts{
							TrafficLaneName: "lane-a",
							Status:          appmodeldeploy.StatusDeployed,
							ImageTag:        "v1",
						},
					)

					dbfactory.BuildAutoDeployRecord(
						ctx,
						buildAutoDeployRecordStore,
						trpcApp,
						testEnv,
						&dbfactory.BuildAutoDeployRecordOpts{
							TrafficLaneName: "lane-a",
							BuildID:         "build-1",
							DeployID:        deployID,
							Stage:           autodeploy.StageDeploy,
							Status:          string(appmodeldeploy.StatusDeploying),
						},
					)

					out, err := svc.ListForEnvironment(ctx, testEnv)
					Expect(err).NotTo(HaveOccurred())
					Expect(out).To(HaveLen(1))
					Expect(out[0]).To(MatchFields(IgnoreExtras, Fields{
						"TrafficLaneName": Equal("lane-a"),
						"DeployStatus":    Equal(string(appmodeldeploy.StatusDeploying)),
					}))
				})
			})

			Context("when direct deploy record is newer than unlinked build auto deploy record", func() {
				It("should return the direct deploy status", func() {
					mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: ""}}, nil)

					dbfactory.BuildAutoDeployRecord(
						ctx,
						buildAutoDeployRecordStore,
						trpcApp,
						testEnv,
						&dbfactory.BuildAutoDeployRecordOpts{
							BuildID:  "build-old",
							DeployID: bson.NewObjectID().Hex(),
							Stage:    autodeploy.StageDeploy,
							Status:   string(appmodeldeploy.StatusDeploying),
						},
					)
					time.Sleep(10 * time.Millisecond)

					dbfactory.AppModelDeployRecord(
						ctx,
						appModelDeployRecordStore,
						trpcApp,
						testEnv,
						&dbfactory.AppModelDeployRecordOpts{
							Status:   appmodeldeploy.StatusDeployed,
							ImageTag: "v2",
						},
					)

					out, err := svc.ListForEnvironment(ctx, testEnv)
					Expect(err).NotTo(HaveOccurred())
					Expect(out).To(HaveLen(1))
					Expect(out[0]).To(MatchFields(IgnoreExtras, Fields{
						"TrafficLaneName": Equal(""),
						"DeployStatus":    Equal(string(appmodeldeploy.StatusDeployed)),
						"ImageTag":        Equal("v2"),
					}))
				})
			})

			Context("when first build auto deploy attempt is still building", func() {
				It("should return the build auto deploy status", func() {
					mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: ""}}, nil)

					dbfactory.BuildAutoDeployRecord(
						ctx,
						buildAutoDeployRecordStore,
						trpcApp,
						testEnv,
						&dbfactory.BuildAutoDeployRecordOpts{
							BuildID: "build-first",
							Stage:   autodeploy.StageBuild,
							Status:  "running",
						},
					)

					out, err := svc.ListForEnvironment(ctx, testEnv)
					Expect(err).NotTo(HaveOccurred())
					Expect(out).To(HaveLen(1))
					Expect(out[0]).To(MatchFields(IgnoreExtras, Fields{
						"TrafficLaneName": Equal(""),
						"DeployStatus":    Equal("running"),
					}))
				})
			})
		})

		Context("when the application is helm-based", func() {
			var (
				testEnv *envmodel.Environment
				helmApp *bkmsapp.Application
			)

			BeforeEach(func() {
				helmApp = dbfactory.HelmApplication(ctx, &dbfactory.HelmApplicationStores{
					AppStore: appStore,
				}, nil)
				testEnv = dbfactory.Env(ctx, envSvc, helmApp.WorkspaceID)
				Expect(envStore.AddApp(ctx, testEnv.ID, helmApp.ID)).To(Succeed())
				testEnv = reloadEnv(ctx, envStore, testEnv.ID)
			})

			It("should read helm deploy records", func() {
				mockListTrafficLanesReturn(nil, nil)

				dbfactory.HelmDeployRecord(
					ctx,
					helmDeployRecordStore,
					helmApp,
					testEnv,
					&dbfactory.HelmDeployRecordOpts{
						Status:   helmrelease.StatusDeployed,
						ImageTag: "helm-v1",
					},
				)

				out, err := svc.ListForEnvironment(ctx, testEnv)
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(HaveLen(1))
				Expect(out[0]).To(MatchFields(IgnoreExtras, Fields{
					"TrafficLaneName": Equal(""),
					"DeployStatus":    Equal(string(helmrelease.StatusDeployed)),
					"ImageTag":        Equal("helm-v1"),
				}))
			})
		})

		Context("when GetAppsByIDs finds no application", func() {
			var (
				testEnv      *envmodel.Environment
				missingAppID string
			)

			BeforeEach(func() {
				wsID := "test-ws-" + stringx.Random(8)
				testEnv = dbfactory.Env(ctx, envSvc, wsID)
				missingAppID = "nonexistent-app-" + stringx.Random(8)
				Expect(envStore.AddApp(ctx, testEnv.ID, missingAppID)).To(Succeed())
				testEnv = reloadEnv(ctx, envStore, testEnv.ID)
			})

			It("should return an error", func() {
				mockListTrafficLanesReturn(nil, nil)

				fresh := reloadEnv(ctx, envStore, testEnv.ID)
				_, err := svc.ListForEnvironment(ctx, fresh)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, bkmsapp.ErrAppNotFound)).To(BeTrue())
			})
		})

		Context("when application type is unsupported", func() {
			var (
				testEnv  *envmodel.Environment
				weirdApp *bkmsapp.Application
			)

			BeforeEach(func() {
				wsID := "test-ws-" + stringx.Random(8)
				testEnv = dbfactory.Env(ctx, envSvc, wsID)
				name := "weird-app-" + stringx.Random(8)
				weirdApp = &bkmsapp.Application{
					ID:          name + "-" + stringx.Random(4),
					Name:        name,
					WorkspaceID: wsID,
					Type:        "weird",
					DisplayName: name,
					Creator:     "test",
				}
				Expect(appStore.CreateApp(ctx, weirdApp)).To(Succeed())
				Expect(envStore.AddApp(ctx, testEnv.ID, weirdApp.ID)).To(Succeed())
				testEnv = reloadEnv(ctx, envStore, testEnv.ID)
			})

			It("should return an error", func() {
				mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: ""}}, nil)

				_, err := svc.ListForEnvironment(ctx, testEnv)
				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrUnsupportedAppType)).To(BeTrue())
			})
		})
	})

	Describe("ListForAppsInWorkspace", func() {
		var (
			workspaceID string
			trpcApp     *bkmsapp.Application
			envStaging  *envmodel.Environment
			envDev      *envmodel.Environment
			featureEnv  *envmodel.Environment
			apps        []*bkmsapp.Application
		)

		BeforeEach(func() {
			trpcApp, _ = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, nil)
			workspaceID = trpcApp.WorkspaceID
			envStaging = dbfactory.Env(ctx, envSvc, workspaceID)
			envDev = dbfactory.Env(ctx, envSvc, workspaceID)
			apps = []*bkmsapp.Application{trpcApp}
		})

		Context("when environments include the requested apps", func() {
			BeforeEach(func() {
				Expect(envStore.AddApp(ctx, envStaging.ID, trpcApp.ID)).To(Succeed())
				Expect(envStore.AddApp(ctx, envDev.ID, trpcApp.ID)).To(Succeed())
				envStaging = reloadEnv(ctx, envStore, envStaging.ID)
				envDev = reloadEnv(ctx, envStore, envDev.ID)
			})

			It("should aggregate deploy statuses across envs and lanes", func() {
				mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: "lane-x"}}, nil)

				dbfactory.AppModelDeployRecord(
					ctx,
					appModelDeployRecordStore,
					trpcApp,
					envStaging,
					&dbfactory.AppModelDeployRecordOpts{
						Status:   appmodeldeploy.StatusDeployed,
						ImageTag: "v1",
					},
				)

				dbfactory.AppModelDeployRecord(
					ctx,
					appModelDeployRecordStore,
					trpcApp,
					envDev,
					&dbfactory.AppModelDeployRecordOpts{
						TrafficLaneName: "lane-x",
						Status:          appmodeldeploy.StatusDeploying,
						ImageTag:        "v2",
					},
				)

				out, err := svc.ListForAppsInWorkspace(
					ctx,
					workspaceID,
					apps,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(HaveKey(trpcApp.ID))
				Expect(out[trpcApp.ID]).To(ConsistOf(
					MatchFields(IgnoreExtras, Fields{
						"EnvName":         Equal(envStaging.Name),
						"TrafficLaneName": Equal(""),
						"DeployStatus":    Equal(string(appmodeldeploy.StatusDeployed)),
						"ImageTag":        Equal("v1"),
					}),
					MatchFields(IgnoreExtras, Fields{
						"EnvName":         Equal(envDev.Name),
						"TrafficLaneName": Equal("lane-x"),
						"DeployStatus":    Equal(string(appmodeldeploy.StatusDeploying)),
						"ImageTag":        Equal("v2"),
					}),
				))
			})
		})

		Context("when only one environment contains the filtered apps", func() {
			BeforeEach(func() {
				Expect(envStore.AddApp(ctx, envStaging.ID, trpcApp.ID)).To(Succeed())
				envStaging = reloadEnv(ctx, envStore, envStaging.ID)
				envDev = reloadEnv(ctx, envStore, envDev.ID)
			})

			It("should skip ListTrafficLanes for environments without the filtered apps", func() {
				var laneCalls int
				mockListTrafficLanesWithHook(func(
					_ *trafficmanager.StubTrafficManager,
					_ context.Context,
					_,
					_ string,
				) ([]*trafficmanager.TrafficLane, error) {
					laneCalls++
					return []*trafficmanager.TrafficLane{{LaneName: ""}}, nil
				})

				dbfactory.AppModelDeployRecord(
					ctx,
					appModelDeployRecordStore,
					trpcApp,
					envStaging,
					&dbfactory.AppModelDeployRecordOpts{
						Status:   appmodeldeploy.StatusDeployed,
						ImageTag: "v2",
					},
				)

				out, err := svc.ListForAppsInWorkspace(
					ctx,
					workspaceID,
					apps,
				)
				Expect(err).NotTo(HaveOccurred())
				// envStaging 含目标 app，envDev 不含 → 仅前者查询泳道
				Expect(laneCalls).To(Equal(1))
				Expect(out).To(HaveKey(trpcApp.ID))
				Expect(out[trpcApp.ID]).To(HaveLen(1))
				Expect(out[trpcApp.ID][0]).To(MatchFields(IgnoreExtras, Fields{
					"EnvName":         Equal(envStaging.Name),
					"TrafficLaneName": Equal(""),
					"DeployStatus":    Equal(string(appmodeldeploy.StatusDeployed)),
					"ImageTag":        Equal("v2"),
				}))
			})
		})

		Context("when the app owns a feature environment", func() {
			BeforeEach(func() {
				featureEnv = dbfactory.FeatEnv(ctx, envSvc, trpcApp, envStaging)
				Expect(envStore.AddApp(ctx, featureEnv.ID, trpcApp.ID)).To(Succeed())
				featureEnv = reloadEnv(ctx, envStore, featureEnv.ID)
			})

			It("should include feature envs once the owner app is tracked in the env", func() {
				mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: ""}}, nil)

				dbfactory.AppModelDeployRecord(
					ctx,
					appModelDeployRecordStore,
					trpcApp,
					featureEnv,
					&dbfactory.AppModelDeployRecordOpts{
						Status:   appmodeldeploy.StatusDeployed,
						ImageTag: "feat-v1",
					},
				)

				out, err := svc.ListForAppsInWorkspace(
					ctx,
					workspaceID,
					apps,
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(HaveKey(trpcApp.ID))
				Expect(out[trpcApp.ID]).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"EnvName":         Equal(featureEnv.Name),
					"EnvType":         Equal(featureEnv.Type),
					"TrafficLaneName": Equal(""),
					"DeployStatus":    Equal(string(appmodeldeploy.StatusDeployed)),
					"ImageTag":        Equal("feat-v1"),
				})))
			})
		})

		Context("when EnvStore.ListBatchAppEnvs fails", func() {
			It("should return an error", func() {
				mocker := mockey.Mock((*envmodel.EnvironmentStoreMongo).ListBatchAppEnvs).
					Return(nil, errors.New("list failed")).Build()
				DeferCleanup(func() {
					mocker.UnPatch()
				})

				_, err := svc.ListForAppsInWorkspace(ctx, workspaceID, apps)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("list failed"))
			})
		})
	})

	Describe("ListFeatureEnvsForApp", func() {
		var (
			trpcApp    *bkmsapp.Application
			otherApp   *bkmsapp.Application
			sourceEnv  *envmodel.Environment
			featureEnv *envmodel.Environment
		)

		BeforeEach(func() {
			trpcApp, _ = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, nil)
			otherApp, _ = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
				AppStore:                  appStore,
				AppModelStore:             appModelStore,
				AppConfigFileStore:        appConfigFileStore,
				AppConfigFileVersionStore: appConfigFileVersionStore,
				BuildConfigStore:          buildConfigStore,
			}, &dbfactory.TrpcApplicationOpts{WorkspaceID: trpcApp.WorkspaceID})
			sourceEnv = dbfactory.Env(ctx, envSvc, trpcApp.WorkspaceID)
			featureEnv = dbfactory.FeatEnv(ctx, envSvc, trpcApp, sourceEnv)
		})

		It("should return deploy statuses for all lanes", func() {
			Expect(envStore.AddApp(ctx, featureEnv.ID, trpcApp.ID)).To(Succeed())
			featureEnv = reloadEnv(ctx, envStore, featureEnv.ID)

			mockListTrafficLanesReturn([]*trafficmanager.TrafficLane{{LaneName: "lane-x"}}, nil)

			dbfactory.AppModelDeployRecord(
				ctx,
				appModelDeployRecordStore,
				trpcApp,
				featureEnv,
				&dbfactory.AppModelDeployRecordOpts{
					Status:   appmodeldeploy.StatusDeployed,
					ImageTag: "default-v1",
				},
			)
			dbfactory.AppModelDeployRecord(
				ctx,
				appModelDeployRecordStore,
				trpcApp,
				featureEnv,
				&dbfactory.AppModelDeployRecordOpts{
					TrafficLaneName: "lane-x",
					Status:          appmodeldeploy.StatusDeploying,
					ImageTag:        "lane-v2",
				},
			)

			out, err := svc.ListFeatureEnvsForApp(ctx, trpcApp, []envmodel.Environment{*featureEnv})
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(HaveKey(featureEnv.Name))
			Expect(out[featureEnv.Name]).To(ConsistOf(
				MatchFields(IgnoreExtras, Fields{
					"TrafficLaneName": Equal("lane-x"),
					"DeployStatus":    Equal(string(appmodeldeploy.StatusDeploying)),
					"ImageTag":        Equal("lane-v2"),
				}),
				MatchFields(IgnoreExtras, Fields{
					"TrafficLaneName": Equal(""),
					"DeployStatus":    Equal(string(appmodeldeploy.StatusDeployed)),
					"ImageTag":        Equal("default-v1"),
				}),
			))
		})

		It("should return an empty slice when the app has not been deployed in the feature env yet", func() {
			var laneCalls int
			mockListTrafficLanesWithHook(func(
				_ *trafficmanager.StubTrafficManager,
				_ context.Context,
				_,
				_ string,
			) ([]*trafficmanager.TrafficLane, error) {
				laneCalls++
				return []*trafficmanager.TrafficLane{{LaneName: ""}}, nil
			})

			out, err := svc.ListFeatureEnvsForApp(ctx, trpcApp, []envmodel.Environment{*featureEnv})
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(HaveKey(featureEnv.Name))
			Expect(out[featureEnv.Name]).To(BeEmpty())
			Expect(laneCalls).To(Equal(0))
		})

		It("should reject feature envs owned by another app", func() {
			foreignFeatureEnv := dbfactory.FeatEnv(ctx, envSvc, otherApp, sourceEnv)

			_, err := svc.ListFeatureEnvsForApp(ctx, trpcApp, []envmodel.Environment{*foreignFeatureEnv})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not belong to app"))
		})
	})
})
