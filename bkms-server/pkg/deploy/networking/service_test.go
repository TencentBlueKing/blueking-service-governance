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

package networking

import (
	"context"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/networking"
)

var _ = Describe("Test ServiceSyncer", func() {
	var syncer *serviceSyncer
	var clientSet *kubernetes.Clientset

	var ctx context.Context
	var appID1 string

	var mocker *mockey.Mocker

	Describe("test sync", func() {
		var svc1, svc3 networking.Service

		BeforeEach(func() {
			cfg, err := testutil.TestClusterConfig("")
			if errors.Is(err, testutil.ErrKubeConfigNotFound) {
				Skip(err.Error())
			}
			Expect(err).NotTo(HaveOccurred())

			// Set up the mocker to make the syncer always use the test config.
			mocker = mockey.Mock(cluster.NewConfig).Return(cfg).Build()

			syncer = NewServiceSyncer(&envmodel.Environment{Cluster: envmodel.BizCluster{
				ClusterID: cfg.ClusterID,
				Namespace: "default",
			}})

			clientSet, err = kubernetes.NewForConfig(cfg.Rest)
			Expect(err).NotTo(HaveOccurred())

			ctx = context.Background()

			appID1 = stringx.Random(6) + "-foo"

			svc1 = networking.Service{
				AppID:    appID1,
				Name:     "test-service1-" + stringx.Random(6),
				Selector: map[string]string{"foo": stringx.Random(6)},
				Ports: []networking.ServicePort{
					{Name: "http", Protocol: networking.ProtocolTCP, Port: 80, TargetPort: "8080"},
				},
				TrafficLaneEnabled: true,
			}
			svc2 := networking.Service{
				AppID:    appID1,
				Name:     "test-service2-" + stringx.Random(6),
				Selector: map[string]string{"foo": stringx.Random(6)},
				Ports: []networking.ServicePort{
					{Name: "http", Protocol: networking.ProtocolTCP, Port: 80, TargetPort: "8080"},
				},
				TrafficLaneEnabled: true,
			}

			svc3 = networking.Service{
				AppID:    stringx.Random(6) + "-bar",
				Name:     "test-service3-" + stringx.Random(6),
				Selector: map[string]string{"bar": stringx.Random(6)},
				Ports: []networking.ServicePort{
					{Name: "http", Protocol: networking.ProtocolTCP, Port: 80, TargetPort: "8000"},
				},
				TrafficLaneEnabled: false,
			}

			err = syncer.Sync(ctx, appID1, []networking.Service{svc1, svc2})
			Expect(err).NotTo(HaveOccurred())

			err = syncer.Sync(ctx, svc3.AppID, []networking.Service{svc3})
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			selector := labels.SelectorFromSet(labels.Set{
				ServiceControllerLabelKey: ServiceControllerLabelValue,
			})

			listOptions := metav1.ListOptions{
				LabelSelector: selector.String(),
			}
			servicesToDelete, err := clientSet.CoreV1().Services("default").List(ctx, listOptions)
			Expect(err).NotTo(HaveOccurred())

			for _, svc := range servicesToDelete.Items {
				err := clientSet.CoreV1().Services("default").Delete(ctx, svc.Name, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			}

			mocker.Release()
		})

		It("sync services successfully when existing services", func() {
			// 1. 校验存在的 services 是否符合预期
			selector := labels.SelectorFromSet(labels.Set{
				ServiceControllerLabelKey: ServiceControllerLabelValue,
			})

			listOptions := metav1.ListOptions{
				LabelSelector: selector.String(),
			}
			services, err := clientSet.CoreV1().Services("default").List(ctx, listOptions)
			Expect(err).NotTo(HaveOccurred())
			Expect(services.Items).To(HaveLen(3))

			// 检查 svc1 是否符合预期
			svcInCluster, err := clientSet.CoreV1().Services("default").Get(ctx, svc1.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svcInCluster.GetLabels()[ServiceAppLabelKey]).To(Equal(svc1.AppID))
			Expect(svcInCluster.GetLabels()[ServiceControllerLabelKey]).To(Equal("bkms-syncer"))
			Expect(svcInCluster.GetAnnotations()[ServiceTrafficlaneEnabledAnnoKey]).To(Equal("true"))
			Expect(svcInCluster.Spec.Selector).To(Equal(svc1.Selector))
			Expect(svcInCluster.Spec.Ports[0].Name).To(Equal("http"))
			Expect(svcInCluster.Spec.Ports[0].TargetPort.IntVal).To(Equal(int32(8080)))

			// 2. 新增 svc4, 更新 svc1, 并测试结果是否符合预期
			svc4 := networking.Service{
				AppID:    appID1,
				Name:     "test-service4-" + stringx.Random(6),
				Selector: map[string]string{"bar": stringx.Random(6)},
				Ports: []networking.ServicePort{
					{Name: "metrics", Protocol: networking.ProtocolTCP, Port: 80, TargetPort: "5000"},
				},
				TrafficLaneEnabled: false,
			}
			updateSvc1 := svc1
			updateSvc1.Selector = map[string]string{"bar": stringx.Random(6)}
			updateSvc1.Ports = []networking.ServicePort{
				{Name: "dns", Protocol: networking.ProtocolUDP, Port: 53, TargetPort: "53"},
			}

			err = syncer.Sync(ctx, appID1, []networking.Service{updateSvc1, svc4})
			Expect(err).NotTo(HaveOccurred())

			services, err = clientSet.CoreV1().Services("default").List(ctx, listOptions)
			Expect(err).NotTo(HaveOccurred())
			// 集群内应该有"新svc1", "svc4" 和 "svc3". "svc2" 被删除了
			Expect(services.Items).To(HaveLen(3))
			// 检查 svc1 被更新
			svcInCluster, err = clientSet.CoreV1().Services("default").Get(ctx, svc1.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svcInCluster.Spec.Selector).To(Equal(updateSvc1.Selector))
			Expect(svcInCluster.Spec.Ports[0].Name).To(Equal("dns"))
			Expect(svcInCluster.Spec.Ports[0].TargetPort.IntVal).To(Equal(int32(53)))
			// 检查 svc4 被新增
			svcInCluster, err = clientSet.CoreV1().Services("default").Get(ctx, svc4.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svcInCluster.GetLabels()[ServiceAppLabelKey]).To(Equal(svc4.AppID))
			Expect(svcInCluster.GetLabels()[ServiceControllerLabelKey]).To(Equal("bkms-syncer"))
			Expect(svcInCluster.GetAnnotations()[ServiceTrafficlaneEnabledAnnoKey]).To(Equal("false"))
			// 检查 svc3 被保留
			svcInCluster, err = clientSet.CoreV1().Services("default").Get(ctx, svc3.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(svcInCluster.Name).To(Equal(svc3.Name))
		})
	})

	Describe("test some methods without cluster", func() {
		BeforeEach(func() {
			syncer = &serviceSyncer{namespace: "default"}
		})

		Context("validate services", func() {
			It("validate successfully", func() {
				appID := stringx.Random(6)
				err := syncer.validate(appID, []networking.Service{
					{AppID: appID, Name: "test-service-" + stringx.Random(6)},
					{AppID: appID, Name: "test-service-" + stringx.Random(6)},
				})
				Expect(err).NotTo(HaveOccurred())
			})

			It("validate failed", func() {
				appID := stringx.Random(6)
				err := syncer.validate(appID, []networking.Service{
					{AppID: appID, Name: "test-service-" + stringx.Random(6)},
					{AppID: stringx.Random(6), Name: "test-service-" + stringx.Random(6)},
				})
				Expect(err.Error()).To(ContainSubstring("multiple appIDs found"))
			})
		})

		It("generate service manifest", func() {
			svc := networking.Service{
				AppID:    stringx.Random(6),
				Name:     stringx.Random(6),
				Selector: map[string]string{"foo": stringx.Random(6), "bar": stringx.Random(6)},
				Ports: []networking.ServicePort{
					{Name: "http", Protocol: networking.ProtocolTCP, Port: 80, TargetPort: "8080"},
				},
				TrafficLaneEnabled: true,
			}
			manifest, err := syncer.genManifest(svc)
			Expect(err).NotTo(HaveOccurred())

			Expect(mapx.GetStr(manifest, "metadata.name")).To(Equal(svc.Name))
			labels := mapx.GetMap(manifest, "metadata.labels")
			Expect(labels[ServiceAppLabelKey].(string)).To(Equal(svc.AppID))
			Expect(labels[ServiceControllerLabelKey].(string)).To(Equal("bkms-syncer"))
			Expect(mapx.GetMap(manifest, "metadata.annotations")[ServiceTrafficlaneEnabledAnnoKey]).To(Equal("true"))

			spec := mapx.GetMap(manifest, "spec")
			Expect(spec["selector"].(map[string]any)["foo"]).To(Equal(svc.Selector["foo"]))
			Expect(
				spec["ports"].([]any)[0].(map[string]any)["targetPort"],
			).To(BeNumerically("==", 8080))
		})
	})
})
