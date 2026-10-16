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

package deploydiagnosis

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	helmrelease "helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/clusterresources"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/rules"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/deploydiagnosis/serializer"
)

func makePodManifest(name string, ready bool, restartCount int64) unstructured.Unstructured {
	readyStatus := "False"
	if ready {
		readyStatus = "True"
	}
	return unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": name},
		"spec": map[string]any{
			"containers": []any{
				map[string]any{"name": "main", "image": "nginx:latest"},
			},
		},
		"status": map[string]any{
			"phase": "Running",
			"containerStatuses": []any{
				map[string]any{"name": "main", "restartCount": restartCount},
			},
			"conditions": []any{
				map[string]any{"type": "Ready", "status": readyStatus},
			},
		},
	}}
}

var _ = Describe("Diagnoser.Diagnose", func() {
	var (
		ctx        context.Context
		diApp      *fxtest.App
		db         *mongo.Database
		appStore   *bkmsapp.ApplicationStore
		envService *bkmsenv.EnvService
		oldConfig  *config.Config
		diagnoser  *Diagnoser
		app        *bkmsapp.Application
		appEnv     *envmodel.Environment
	)

	BeforeEach(func() {
		ctx = context.Background()
		oldConfig = config.G
		config.G = &config.Config{
			BCS: config.BCSConfig{
				BaseUrl: "http://example.invalid",
				Token:   "token",
			},
			Development: config.DevConfig{UseKubeConfigCluster: false},
		}

		diApp = fxtest.New(
			GinkgoT(),
			appcfg.FxModule,
			bkmsenv.FxModule,
			fx.Populate(&envService),
		)
		diApp.RequireStart()
		db = testutil.NewDB(GinkgoT())
		appStore = testutil.NewStore(GinkgoT(), bkmsapp.AppMeta)

		app, _ = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore: appStore,
		}, &dbfactory.TrpcApplicationOpts{
			TrpcConfig: &appmodel.TrpcConfig{
				FileName: "trpc_go.yaml",
				FilePath: "/app/conf",
			},
		})
		appEnv = dbfactory.Env(ctx, envService, app.WorkspaceID)
		dbfactory.AppModelDeployRecord(ctx, db, app, appEnv, &dbfactory.AppModelDeployRecordOpts{
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-00000",
			Namespace:     "ns-diag",
			LabelSelector: map[string]string{"app": app.ID},
		})

		diagnoser = NewDiagnoser(&storereg.Registry{DB: db})
	})

	AfterEach(func() {
		config.G = oldConfig
		if diApp != nil {
			diApp.RequireStop()
		}
	})

	It("returns config-file-not-found after aggregating mocked pod logs", func() {
		logLine := "open /data/trpc_go.yaml: no such file or directory"
		mockey.PatchConvey("diagnose config file missing", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-a", false, 5)},
			}, nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).To(func(
				_ *k8sclient.PodClient,
				_ context.Context,
				_, _ string,
				opts *corev1.PodLogOptions,
			) ([]k8sclient.LogEntry, error) {
				if opts != nil && opts.Previous {
					return []k8sclient.LogEntry{{Content: logLine}}, nil
				}
				return nil, nil
			}).Build()
			mockEmptyDiagnosisEvents()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Partial).To(BeFalse())
			Expect(out.EmptyReason).To(BeEmpty())
			Expect(out.Summary.TotalCount).To(Equal(1))
			Expect(out.Summary.AbnormalCount).To(Equal(1))
			Expect(out.Groups).To(HaveLen(1))
			Expect(out.Groups[0].RuleID).To(Equal(rules.IDConfigFileNotFound))
			Expect(out.Groups[0].Instances[0].Evidence).To(ContainElement(logLine))
			Expect(out.Groups[0].Instances[0].LogSource).To(Equal(serializer.LogSourcePrevious))
		})
	})

	It("returns all_healthy when mocked pods are Ready", func() {
		mockey.PatchConvey("diagnose healthy pods", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-ok", true, 0)},
			}, nil).Build()
			mockEmptyDiagnosisEvents()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Partial).To(BeFalse())
			Expect(out.EmptyReason).To(Equal(serializer.EmptyReasonAllHealthy))
			Expect(out.Groups).To(BeEmpty())
		})
	})

	It("fails the whole request when listing pods fails", func() {
		mockey.PatchConvey("list pods error", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(nil, errors.New("kube down")).Build()

			_, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("list instances"))
		})
	})

	It("returns instances_unavailable for a deployed Helm app", func() {
		helmApp := dbfactory.HelmApplication(
			ctx,
			&dbfactory.HelmApplicationStores{AppStore: appStore},
			&dbfactory.HelmApplicationOpts{
				WorkspaceID: app.WorkspaceID,
			},
		)
		dbfactory.HelmDeployRecord(ctx, db, helmApp, appEnv, &dbfactory.HelmDeployRecordOpts{
			Status: helmrelease.StatusDeployed,
		})

		out, err := diagnoser.Diagnose(ctx, helmApp, appEnv, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(out.EmptyReason).To(Equal(serializer.EmptyReasonInstancesUnavailable))
		Expect(out.Partial).To(BeFalse())
		Expect(out.Deploy.DeployID).NotTo(BeEmpty())
		Expect(out.Groups).To(BeEmpty())
	})

	It("returns deploy-failed for a failed Helm deploy", func() {
		helmApp := dbfactory.HelmApplication(
			ctx,
			&dbfactory.HelmApplicationStores{AppStore: appStore},
			&dbfactory.HelmApplicationOpts{
				WorkspaceID: app.WorkspaceID,
			},
		)
		dbfactory.HelmDeployRecord(ctx, db, helmApp, appEnv, &dbfactory.HelmDeployRecordOpts{
			Status: helmrelease.StatusFailed,
		})

		out, err := diagnoser.Diagnose(ctx, helmApp, appEnv, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(out.EmptyReason).To(BeEmpty())
		Expect(out.Groups).To(HaveLen(1))
		Expect(out.Groups[0].RuleID).To(Equal(rules.IDDeployFailed))
	})

	It("does not mark logs partial when the log api returns no lines", func() {
		mockey.PatchConvey("empty logs", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-empty", false, 0)},
			}, nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).Return([]k8sclient.LogEntry{}, nil).Build()
			mockEmptyDiagnosisEvents()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Partial).To(BeFalse())
			Expect(out.Groups[0].Instances[0].LogSource).To(Equal(serializer.LogSourceNone))
		})
	})

	It("marks logs partial when listing logs fails", func() {
		mockey.PatchConvey("log list error", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-a", false, 0)},
			}, nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).Return(nil, errors.New("log backend down")).Build()
			mockEmptyDiagnosisEvents()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.Partial).To(BeTrue())
			Expect(out.PartialReasons).To(Equal([]string{serializer.PartialReasonLogs}))
		})
	})

	It("marks logs partial when previous logs fail even if current logs exist", func() {
		mockey.PatchConvey("previous log error", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-a", false, 2)},
			}, nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).To(func(
				_ *k8sclient.PodClient,
				_ context.Context,
				_, _ string,
				opts *corev1.PodLogOptions,
			) ([]k8sclient.LogEntry, error) {
				if opts != nil && opts.Previous {
					return nil, errors.New("previous log missing")
				}
				return []k8sclient.LogEntry{{Content: "still running"}}, nil
			}).Build()
			mockEmptyDiagnosisEvents()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.PartialReasons).To(Equal([]string{serializer.PartialReasonLogs}))
			Expect(out.Groups[0].Instances[0].LogSource).To(Equal(serializer.LogSourceCurrent))
		})
	})

	It("marks logs partial when abnormal instances exceed the fetch cap", func() {
		mockey.PatchConvey("log sample truncated", GinkgoT(), func() {
			items := make([]unstructured.Unstructured, 0, maxLogInstances+1)
			for i := 0; i < maxLogInstances+1; i++ {
				items = append(items, makePodManifest(fmt.Sprintf("pod-%d", i), false, 0))
			}
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{Items: items}, nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).Return([]k8sclient.LogEntry{}, nil).Build()
			mockEmptyDiagnosisEvents()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.PartialReasons).To(Equal([]string{serializer.PartialReasonLogs}))
			Expect(out.Summary.AbnormalCount).To(Equal(maxLogInstances + 1))
		})
	})

	It("marks events partial when listing events fails", func() {
		mockey.PatchConvey("event list error", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-ok", true, 0)},
			}, nil).Build()
			mockey.Mock(auth.GetUser).Return(auth.User{ID: "tester"}, nil).Build()
			mockey.Mock(clusterresources.New).Return(&clusterresources.ApiClient{}, nil).Build()
			mockey.Mock((*clusterresources.ApiClient).ListEvents).Return(nil, errors.New("event api down")).Build()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.PartialReasons).To(Equal([]string{serializer.PartialReasonEvents}))
			Expect(out.EmptyReason).To(Equal(serializer.EmptyReasonAllHealthy))
		})
	})

	It("marks events partial when the event page is truncated", func() {
		mockey.PatchConvey("event page truncated", GinkgoT(), func() {
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-ok", true, 0)},
			}, nil).Build()
			mockey.Mock(auth.GetUser).Return(auth.User{ID: "tester"}, nil).Build()
			mockey.Mock(clusterresources.New).Return(&clusterresources.ApiClient{}, nil).Build()
			mockey.Mock((*clusterresources.ApiClient).ListEvents).Return(&clusterresources.PaginatedEvents{
				Count: int64(maxEvents + 1),
				Data: []clusterresources.EventEntry{{
					ResourceKind:  "Pod",
					ResourcesName: "pod-ok",
					Level:         "Normal",
					Content:       "Started container",
				}},
			}, nil).Build()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(out.PartialReasons).To(Equal([]string{serializer.PartialReasonEvents}))
		})
	})

	It("attaches pod events to the instance and workload events to the group", func() {
		err := store.Of(db, appmodeldeploy.RecordMeta).Create(ctx, &appmodeldeploy.Record{
			WorkspaceID:   app.WorkspaceID,
			AppID:         app.ID,
			EnvName:       appEnv.Name,
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-00000",
			Namespace:     "ns-diag",
			LabelSelector: map[string]string{"app": app.ID},
			ImageTag:      "v2",
			ResourceKeys:  []appmodeldeploy.ResourceKey{{Kind: "Deployment", Name: "demo"}},
		})
		Expect(err).NotTo(HaveOccurred())

		mockey.PatchConvey("split pod and workload events", GinkgoT(), func() {
			var podQueries, workloadQueries int
			mockey.Mock((*k8sclient.Client).List).Return(&unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{makePodManifest("pod-a", false, 0)},
			}, nil).Build()
			mockey.Mock((*k8sclient.PodClient).ListLogs).Return([]k8sclient.LogEntry{}, nil).Build()
			mockey.Mock(auth.GetUser).Return(auth.User{ID: "tester"}, nil).Build()
			mockey.Mock(clusterresources.New).Return(&clusterresources.ApiClient{}, nil).Build()
			mockey.Mock((*clusterresources.ApiClient).ListEvents).To(func(
				_ *clusterresources.ApiClient,
				_ context.Context,
				_, _ string,
				params clusterresources.ListEventParams,
			) (*clusterresources.PaginatedEvents, error) {
				if len(params.ResourceKinds) == 1 && params.ResourceKinds[0] == "Pod" {
					podQueries++
					return &clusterresources.PaginatedEvents{
						Count: 1,
						Data: []clusterresources.EventEntry{{
							ResourceKind:  "Pod",
							ResourcesName: "pod-a",
							Level:         "Warning",
							Content:       "Back-off restarting failed container",
						}},
					}, nil
				}
				workloadQueries++
				return &clusterresources.PaginatedEvents{
					Count: 1,
					Data: []clusterresources.EventEntry{{
						ResourceKind:  "Deployment",
						ResourcesName: "demo",
						Level:         "Warning",
						Content:       "exceeded quota",
					}},
				}, nil
			}).Build()

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(podQueries).To(Equal(1))
			Expect(workloadQueries).To(Equal(1))
			Expect(out.Partial).To(BeFalse())
			Expect(out.Groups[0].Instances[0].RelatedEvents).To(Equal([]*serializer.DiagnosisEventObj{
				{Level: "Warning", Content: "Back-off restarting failed container"},
			}))
			Expect(out.Groups[0].RelatedEvents).To(Equal([]*serializer.DiagnosisEventObj{
				{Level: "Warning", Content: "exceeded quota"},
			}))
		})
	})
})

func mockEmptyDiagnosisEvents() {
	mockey.Mock(auth.GetUser).Return(auth.User{ID: "tester"}, nil).Build()
	mockey.Mock(clusterresources.New).Return(&clusterresources.ApiClient{}, nil).Build()
	mockey.Mock((*clusterresources.ApiClient).ListEvents).Return(&clusterresources.PaginatedEvents{}, nil).Build()
}

var _ = Describe("frameworkFromAppModel", func() {
	It("reads the trpc file and joins the start command", func() {
		fw := frameworkFromAppModel(bkmsapp.AppTypeTRPC, &appmodel.AppModel{
			Workload: appmodel.Workload{
				Command:    []string{"/app/server"},
				Args:       []string{"-conf", "trpc_go.yaml"},
				TrpcConfig: appmodel.TrpcConfig{FileName: "trpc_go.yaml", FilePath: "/app/conf"},
				TafConfig:  appmodel.TafConfig{FileName: "taf.yaml", FilePath: "/taf"},
			},
		})

		Expect(fw.FileName).To(Equal("trpc_go.yaml"))
		Expect(fw.FilePath).To(Equal("/app/conf"))
		Expect(fw.Command).To(Equal("/app/server -conf trpc_go.yaml"))
	})

	It("reads the taf file", func() {
		fw := frameworkFromAppModel(bkmsapp.AppTypeTAF, &appmodel.AppModel{
			Workload: appmodel.Workload{
				Command:    []string{"/app/taf"},
				TafConfig:  appmodel.TafConfig{FileName: "taf.conf", FilePath: "/usr/local/app"},
				TrpcConfig: appmodel.TrpcConfig{FileName: "trpc_go.yaml", FilePath: "/app/conf"},
			},
		})

		Expect(fw.FileName).To(Equal("taf.conf"))
		Expect(fw.FilePath).To(Equal("/usr/local/app"))
		Expect(fw.Command).To(Equal("/app/taf"))
	})
})

var _ = Describe("instanceIDForEvent", func() {
	pods := map[string]struct{}{"pod-a": {}}

	DescribeTable("classifies pod and workload events",
		func(kind, name, want string) {
			Expect(instanceIDForEvent(kind, name, pods)).To(Equal(want))
		},
		Entry("pod kind", "Pod", "pod-a", "pod-a"),
		Entry("deployment kind", "Deployment", "demo", ""),
		Entry("empty kind matching a pod name", "", "pod-a", "pod-a"),
		Entry("empty kind for a workload name", "", "demo", ""),
	)
})
