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

package workload

import (
	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

var _ = Describe("gameDeploymentToDeployment", func() {
	It("copies shared spec and metadata onto a native Deployment", func() {
		gd := tkex.GameDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "demo-app",
				Namespace: "ns-1",
				Labels: map[string]string{
					"app.kubernetes.io/name":            "demo-app",
					"io.tencent.bcs.dev/deletion-allow": "Always",
				},
				Annotations: map[string]string{
					"bkms.tencent.com/keep":                         "yes",
					"io.tencent.bcs.dev/update-strategy-type-allow": "true",
					"controller.kubernetes.io/pod-deletion-cost":    "10",
				},
			},
			Spec: tkex.GameDeploymentSpec{
				Replicas: lo.ToPtr(int32(3)),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app.kubernetes.io/name": "demo-app"},
				},
				UpdateStrategy: tkex.GameDeploymentUpdateStrategy{
					MaxUnavailable: lo.ToPtr(intstr.FromString("25%")),
					MaxSurge:       lo.ToPtr(intstr.FromString("25%")),
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.kubernetes.io/name": "demo-app"},
						Annotations: map[string]string{
							"controller.kubernetes.io/pod-deletion-cost": "10",
							"tke.cloud.tencent.com/networks":             "tke-route-eni",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "main", Image: "nginx:latest"}},
					},
				},
			},
		}

		deploy := gameDeploymentToDeployment(gd)

		Expect(deploy.Kind).To(Equal(kind.Deploy))
		Expect(deploy.APIVersion).To(Equal("apps/v1"))
		Expect(deploy.Name).To(Equal("demo-app"))
		Expect(deploy.Namespace).To(Equal("ns-1"))
		Expect(*deploy.Spec.Replicas).To(Equal(int32(3)))
		Expect(deploy.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app.kubernetes.io/name", "demo-app"))
		Expect(deploy.Spec.Template.Spec.Containers[0].Image).To(Equal("nginx:latest"))
		Expect(deploy.Spec.Strategy.Type).To(Equal(appsv1.RollingUpdateDeploymentStrategyType))
		Expect(deploy.Spec.Strategy.RollingUpdate.MaxUnavailable.String()).To(Equal("25%"))

		Expect(deploy.Labels).To(Equal(map[string]string{
			"app.kubernetes.io/name":            "demo-app",
			"io.tencent.bcs.dev/deletion-allow": "Always",
		}))
		Expect(deploy.Annotations).To(Equal(map[string]string{
			"bkms.tencent.com/keep":                         "yes",
			"io.tencent.bcs.dev/update-strategy-type-allow": "true",
			"controller.kubernetes.io/pod-deletion-cost":    "10",
		}))
		Expect(deploy.Spec.Template.Annotations).To(Equal(map[string]string{
			"controller.kubernetes.io/pod-deletion-cost": "10",
			"tke.cloud.tencent.com/networks":             "tke-route-eni",
		}))
		Expect(deploy.Spec.Template.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "demo-app"))
	})
})
