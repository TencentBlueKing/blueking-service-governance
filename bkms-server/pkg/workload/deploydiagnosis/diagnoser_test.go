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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil/dbfactory"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
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
		ctx               context.Context
		diApp             *fxtest.App
		appStore          bkmsapp.ApplicationStore
		appModelStore     appmodel.AppModelStore
		appCfgStore       appcfg.AppConfigFileStore
		appCfgVerStore    appcfg.AppConfigFileVersionStore
		buildConfigStore  build.ConfigStore
		envService        *bkmsenv.EnvService
		deployRecordStore appmodeldeploy.RecordStore
		oldConfig         *config.Config
		diagnoser         *Diagnoser
		app               *bkmsapp.Application
		appEnv            *envmodel.Environment
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
			bkmsapp.FxModule,
			appcfg.FxModule,
			appmodel.FxModule,
			bkmsenv.FxModule,
			build.FxModule,
			appmodeldeploy.FxModule,
			fx.Populate(
				&appStore,
				&appModelStore,
				&appCfgStore,
				&appCfgVerStore,
				&buildConfigStore,
				&envService,
				&deployRecordStore,
			),
		)
		diApp.RequireStart()

		app, _ = dbfactory.TrpcApplication(ctx, &dbfactory.TrpcApplicationStores{
			AppStore:                  appStore,
			AppModelStore:             appModelStore,
			AppConfigFileStore:        appCfgStore,
			AppConfigFileVersionStore: appCfgVerStore,
			BuildConfigStore:          buildConfigStore,
		}, &dbfactory.TrpcApplicationOpts{
			TrpcConfig: &appmodel.TrpcConfig{
				FileName: "trpc_go.yaml",
				FilePath: "/app/conf",
			},
		})
		appEnv = dbfactory.Env(ctx, envService, app.WorkspaceID)
		dbfactory.AppModelDeployRecord(ctx, deployRecordStore, app, appEnv, &dbfactory.AppModelDeployRecordOpts{
			Status:        appmodeldeploy.StatusDeployed,
			ClusterID:     "BCS-K8S-00000",
			Namespace:     "ns-diag",
			LabelSelector: map[string]string{"app": app.ID},
		})

		diagnoser = NewDiagnoser(&storereg.Registry{
			AppModelStore:             appModelStore,
			AppModelDeployRecordStore: deployRecordStore,
		})
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

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
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

			out, err := diagnoser.Diagnose(ctx, app, appEnv, "")
			Expect(err).NotTo(HaveOccurred())
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
})
