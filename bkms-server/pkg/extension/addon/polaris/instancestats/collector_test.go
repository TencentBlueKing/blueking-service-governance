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

package instancestats_test

import (
	"context"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/instancestats"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

var _ = Describe("Collector", func() {
	var (
		ctx     context.Context
		appID   string
		store   appmodeldeploy.RecordStore
		config  *polaris.PolarisConfig
		diApp   *fxtest.App
		mockers []*mockey.Mocker
	)

	BeforeEach(func() {
		ctx = context.Background()
		appID = "test-app-" + stringx.Random(6)

		diApp = fxtest.New(
			GinkgoT(),
			appmodeldeploy.FxModule,
			fx.Populate(&store),
		)
		diApp.RequireStart()

		config = &polaris.PolarisConfig{
			AppID: appID,
			Properties: polaris.Properties{
				PolarisName:      "service-a",
				PolarisNamespace: "Production",
				ServicePort:      8080,
			},
			ScopeEnvNames: []string{"stable", "test"},
			EnvStates: map[string]polaris.PolarisEnvState{
				"stable": {
					AppliedFields: &polaris.RedeployRequiredFields{
						InstanceKey:  "demo",
						PolarisToken: "token",
						ServicePort:  8080,
					},
				},
			},
		}
	})

	AfterEach(func() {
		for _, mocker := range mockers {
			mocker.Release()
		}
		diApp.RequireStop()
	})

	It("counts matched healthy, isolated and total instances", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:           appID,
			EnvName:         "stable",
			TrafficLaneName: "",
			ClusterID:       "BCS-K8S-1",
			Namespace:       "default",
			LabelSelector:   map[string]string{"app": "demo"},
		})
		Expect(err).NotTo(HaveOccurred())

		mockers = append(mockers, mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build())
		mockers = append(mockers, mockey.Mock(k8sclient.NewWithGVR).
			To(func(*cluster.Config, schema.GroupVersionResource) *k8sclient.Client {
				return &k8sclient.Client{}
			}).
			Build())
		mockers = append(mockers, mockey.Mock((*k8sclient.Client).List).
			To(func(
				*k8sclient.Client,
				context.Context,
				string,
				metav1.ListOptions,
			) (*unstructured.UnstructuredList, error) {
				return &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
					{Object: map[string]any{"status": map[string]any{"podIP": "127.0.0.1"}}},
					{Object: map[string]any{"status": map[string]any{"podIP": "127.0.0.2"}}},
				}}, nil
			}).
			Build())
		mockers = append(mockers, mockey.Mock(polarisInfra.GetInstances).
			Return([]*polarisInfra.Instance{
				{IP: "127.0.0.1", Port: 8080, Weight: 100, IsHealthy: true},
				{IP: "127.0.0.2", Port: 8080, Weight: 50, IsHealthy: true, IsIsolated: true},
				{IP: "127.0.0.2", Port: 9090, Weight: 80, IsHealthy: true},
				{IP: "127.0.0.9", Port: 8080, Weight: 100, IsHealthy: true},
				{IP: "127.0.0.8", Port: 8080, Weight: 200, IsHealthy: false},
				{IP: "127.0.0.7", Port: 8080, Weight: 0, IsHealthy: true},
			}, nil).
			Build())

		result, err := instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.EnvStats["stable"]).To(Equal(instancestats.Stats{
			HealthyInstanceCount:  1,
			HealthyInstanceWeight: 100,
			IsolatedInstanceCount: 1,
			TotalInstanceCount:    2,
		}))
		Expect(result.EnvStats["test"]).To(Equal(instancestats.Stats{}))
		// 全量健康实例含非平台匹配（127.0.0.9 / 异端口），不含隔离、不健康与零权重实例
		Expect(result.TotalHealthyInstanceCount).To(Equal(3))
		Expect(result.TotalHealthyInstanceWeight).To(Equal(280))
	})

	It("counts pods carrying the polaris weight annotation", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:           appID,
			EnvName:         "stable",
			TrafficLaneName: "",
			ClusterID:       "BCS-K8S-1",
			Namespace:       "default",
			LabelSelector:   map[string]string{"app": "demo"},
		})
		Expect(err).NotTo(HaveOccurred())

		mockers = append(mockers, mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build())
		mockers = append(mockers, mockey.Mock(k8sclient.NewWithGVR).
			To(func(*cluster.Config, schema.GroupVersionResource) *k8sclient.Client {
				return &k8sclient.Client{}
			}).
			Build())
		mockers = append(mockers, mockey.Mock((*k8sclient.Client).List).
			To(func(
				*k8sclient.Client,
				context.Context,
				string,
				metav1.ListOptions,
			) (*unstructured.UnstructuredList, error) {
				return &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
					{Object: map[string]any{"status": map[string]any{"podIP": "127.0.0.1"}}},
					{Object: map[string]any{
						"metadata": map[string]any{
							"annotations": map[string]any{polaris.AnnotationKeyWeight: "20"},
						},
						"status": map[string]any{"podIP": "127.0.0.2"},
					}},
				}}, nil
			}).
			Build())
		mockers = append(mockers, mockey.Mock(polarisInfra.GetInstances).
			Return([]*polarisInfra.Instance{
				{IP: "127.0.0.1", Port: 8080, Weight: 100, IsHealthy: true},
			}, nil).
			Build())

		result, err := instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.EnvStats["stable"].WeightOverriddenInstanceCount).To(Equal(1))
		Expect(result.EnvStats["test"].WeightOverriddenInstanceCount).To(BeZero())
	})

	It("uses the latest deploy record even when its status is not deployed", func() {
		// 较早的成功记录：若误用 GetLatestByStatuses(Deployed) 会落到这里
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:           appID,
			EnvName:         "stable",
			TrafficLaneName: "",
			Status:          appmodeldeploy.StatusDeployed,
			ClusterID:       "BCS-K8S-OLD",
			Namespace:       "old-ns",
			LabelSelector:   map[string]string{"app": "old"},
		})
		Expect(err).NotTo(HaveOccurred())

		// MongoDB DateTime 精度为毫秒，保证 failed 记录成为 GetLatest 结果
		time.Sleep(5 * time.Millisecond)

		_, err = store.Create(ctx, &appmodeldeploy.Record{
			AppID:           appID,
			EnvName:         "stable",
			TrafficLaneName: "",
			Status:          appmodeldeploy.StatusFailed,
			ClusterID:       "BCS-K8S-NEW",
			Namespace:       "new-ns",
			LabelSelector:   map[string]string{"app": "new"},
		})
		Expect(err).NotTo(HaveOccurred())

		var listedNamespace string
		mockers = append(mockers, mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build())
		mockers = append(mockers, mockey.Mock(k8sclient.NewWithGVR).
			To(func(*cluster.Config, schema.GroupVersionResource) *k8sclient.Client {
				return &k8sclient.Client{}
			}).
			Build())
		mockers = append(mockers, mockey.Mock((*k8sclient.Client).List).
			To(func(
				_ *k8sclient.Client,
				_ context.Context,
				namespace string,
				_ metav1.ListOptions,
			) (*unstructured.UnstructuredList, error) {
				listedNamespace = namespace
				return &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
					{Object: map[string]any{"status": map[string]any{"podIP": "127.0.0.3"}}},
				}}, nil
			}).
			Build())
		mockers = append(mockers, mockey.Mock(polarisInfra.GetInstances).
			Return([]*polarisInfra.Instance{
				{IP: "127.0.0.3", Port: 8080, Weight: 100, IsHealthy: true},
			}, nil).
			Build())

		result, err := instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).NotTo(HaveOccurred())
		Expect(listedNamespace).To(Equal("new-ns"))
		Expect(result.EnvStats["stable"]).To(Equal(instancestats.Stats{
			HealthyInstanceCount:  1,
			HealthyInstanceWeight: 100,
			TotalInstanceCount:    1,
		}))
		Expect(result.TotalHealthyInstanceCount).To(Equal(1))
		Expect(result.TotalHealthyInstanceWeight).To(Equal(100))
	})

	It("returns zeros without querying dependencies for undeployed environments", func() {
		config.EnvStates = nil

		result, err := instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(&instancestats.Result{
			EnvStats: map[string]instancestats.Stats{
				"stable": {},
				"test":   {},
			},
			TotalHealthyInstanceCount:  0,
			TotalHealthyInstanceWeight: 0,
		}))
	})

	It("returns zeros when the deployed environment has no deploy record", func() {
		result, err := instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.EnvStats["stable"]).To(Equal(instancestats.Stats{}))
		Expect(result.EnvStats["test"]).To(Equal(instancestats.Stats{}))
		Expect(result.TotalHealthyInstanceCount).To(BeZero())
		Expect(result.TotalHealthyInstanceWeight).To(BeZero())
	})

	It("keeps stats for other environments when one deployed environment has no deploy record", func() {
		config.EnvStates["missing"] = polaris.PolarisEnvState{
			AppliedFields: &polaris.RedeployRequiredFields{
				InstanceKey:  "demo",
				PolarisToken: "token",
				ServicePort:  8080,
			},
		}

		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:           appID,
			EnvName:         "stable",
			TrafficLaneName: "",
			ClusterID:       "BCS-K8S-1",
			Namespace:       "default",
			LabelSelector:   map[string]string{"app": "demo"},
		})
		Expect(err).NotTo(HaveOccurred())

		mockers = append(mockers, mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build())
		mockers = append(mockers, mockey.Mock(k8sclient.NewWithGVR).
			To(func(*cluster.Config, schema.GroupVersionResource) *k8sclient.Client {
				return &k8sclient.Client{}
			}).
			Build())
		mockers = append(mockers, mockey.Mock((*k8sclient.Client).List).
			To(func(
				*k8sclient.Client,
				context.Context,
				string,
				metav1.ListOptions,
			) (*unstructured.UnstructuredList, error) {
				return &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
					{Object: map[string]any{"status": map[string]any{"podIP": "127.0.0.1"}}},
				}}, nil
			}).
			Build())
		mockers = append(mockers, mockey.Mock(polarisInfra.GetInstances).
			Return([]*polarisInfra.Instance{
				{IP: "127.0.0.1", Port: 8080, Weight: 100, IsHealthy: true},
			}, nil).
			Build())

		result, err := instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.EnvStats["stable"]).To(Equal(instancestats.Stats{
			HealthyInstanceCount:  1,
			HealthyInstanceWeight: 100,
			TotalInstanceCount:    1,
		}))
		Expect(result.EnvStats["missing"]).To(Equal(instancestats.Stats{}))
		Expect(result.TotalHealthyInstanceCount).To(Equal(1))
		Expect(result.TotalHealthyInstanceWeight).To(Equal(100))
	})

	It("returns an error when listing pods fails", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:           appID,
			EnvName:         "stable",
			TrafficLaneName: "",
			ClusterID:       "BCS-K8S-1",
			Namespace:       "default",
		})
		Expect(err).NotTo(HaveOccurred())

		mockers = append(mockers, mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build())
		mockers = append(mockers, mockey.Mock(k8sclient.NewWithGVR).
			To(func(*cluster.Config, schema.GroupVersionResource) *k8sclient.Client {
				return &k8sclient.Client{}
			}).
			Build())
		mockers = append(mockers, mockey.Mock((*k8sclient.Client).List).
			Return(nil, errors.New("list pods failed")).
			Build())

		_, err = instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("list pod IPs for env stable"))
	})

	It("returns an error when listing Polaris instances fails", func() {
		_, err := store.Create(ctx, &appmodeldeploy.Record{
			AppID:           appID,
			EnvName:         "stable",
			TrafficLaneName: "",
			ClusterID:       "BCS-K8S-1",
			Namespace:       "default",
		})
		Expect(err).NotTo(HaveOccurred())

		mockers = append(mockers, mockey.Mock(cluster.NewConfig).Return(&cluster.Config{}).Build())
		mockers = append(mockers, mockey.Mock(k8sclient.NewWithGVR).
			To(func(*cluster.Config, schema.GroupVersionResource) *k8sclient.Client {
				return &k8sclient.Client{}
			}).
			Build())
		mockers = append(mockers, mockey.Mock((*k8sclient.Client).List).
			Return(&unstructured.UnstructuredList{}, nil).
			Build())
		mockers = append(mockers, mockey.Mock(polarisInfra.GetInstances).
			Return(nil, errors.New("list Polaris instances failed")).
			Build())

		_, err = instancestats.NewCollector(store).Collect(ctx, appID, config)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("get polaris instances for service Production/service-a"))
	})
})

var _ = Describe("CountMatched", func() {
	It("ignores nil and unmatched instances", func() {
		stats := instancestats.CountMatched(
			map[string]struct{}{"127.0.0.1": {}},
			8080,
			[]*polarisInfra.Instance{
				nil,
				{IP: "127.0.0.2", Port: 8080, IsHealthy: true},
				{IP: "127.0.0.1", Port: 9090, IsHealthy: true},
			},
		)

		Expect(stats).To(Equal(instancestats.Stats{}))
	})
})
