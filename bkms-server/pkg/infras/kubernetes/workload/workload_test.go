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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	k8sstatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/status"
)

// podTemplate 构造带单个容器的 Pod 模板
func podTemplate(containerName, image string) map[string]any {
	return map[string]any{
		"spec": map[string]any{
			"containers": []any{
				map[string]any{"name": containerName, "image": image},
			},
		},
	}
}

var _ = Describe("Registry", func() {
	Context("when the kind is registered", func() {
		It("returns the driver", func() {
			driver, err := Get(k8skind.Deploy)
			Expect(err).NotTo(HaveOccurred())
			Expect(driver.Kind()).To(Equal(k8skind.Deploy))
			Expect(driver.GVR()).To(Equal(gvr.Deploy))
		})
	})

	Context("when the kind is not registered", func() {
		It("returns ErrUnsupportedKind", func() {
			_, err := Get(k8skind.STS)
			Expect(errors.Is(err, ErrUnsupportedKind)).To(BeTrue())

			_, err = GetMain(k8skind.STS)
			Expect(errors.Is(err, ErrUnsupportedKind)).To(BeTrue())
		})
	})

	Describe("MainDrivers", func() {
		It("lists main workload drivers, Deployment first", func() {
			kinds := make([]string, 0)
			for _, d := range MainDrivers() {
				kinds = append(kinds, d.Kind())
			}
			Expect(kinds).To(Equal([]string{k8skind.Deploy, k8skind.GameDeploy}))
		})
	})

	Describe("IsMainKind", func() {
		It("reports whether the kind can be a main workload", func() {
			Expect(IsMainKind(k8skind.Deploy)).To(BeTrue())
			Expect(IsMainKind(k8skind.GameDeploy)).To(BeTrue())
			Expect(IsMainKind(k8skind.SVC)).To(BeFalse())
			Expect(IsMainKind("")).To(BeFalse())
		})
	})
})

var _ = Describe("Deployment driver", func() {
	var driver Driver

	BeforeEach(func() {
		var err error
		driver, err = Get(k8skind.Deploy)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ParseStatus", func() {
		// 联邦网关不返回 conditions，只要 replicas 一致即可视为就绪
		manifest := map[string]any{
			"spec": map[string]any{"replicas": int64(1)},
			"status": map[string]any{
				"replicas":          int64(1),
				"readyReplicas":     int64(1),
				"updatedReplicas":   int64(1),
				"availableReplicas": int64(1),
			},
		}

		It("requires the Available condition on regular clusters", func() {
			result, err := driver.ParseStatus(manifest, ParseOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Progressing))
		})

		It("only checks replicas on federation clusters", func() {
			result, err := driver.ParseStatus(manifest, ParseOptions{Federation: true})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Available))
		})
	})

	Describe("View", func() {
		It("extracts replicas and containers", func() {
			view, err := driver.View(map[string]any{
				"apiVersion": "apps/v1",
				"kind":       k8skind.Deploy,
				"metadata":   map[string]any{"name": "app", "namespace": "ns"},
				"spec": map[string]any{
					"replicas": int64(3),
					"template": podTemplate("main", "busybox:1.0"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(*view.Replicas).To(Equal(int32(3)))
			Expect(view.Containers).To(HaveLen(1))
			Expect(view.Containers[0].Name).To(Equal("main"))
			Expect(view.Containers[0].Image).To(Equal("busybox:1.0"))
		})
	})

	Describe("Capabilities", func() {
		It("supports neither inplace update nor selected pod deletion", func() {
			main, err := GetMain(k8skind.Deploy)
			Expect(err).NotTo(HaveOccurred())
			Expect(main.Capabilities()).To(Equal(Capabilities{}))
		})
	})
})

var _ = Describe("GameDeployment driver", func() {
	var driver Driver

	BeforeEach(func() {
		var err error
		driver, err = Get(k8skind.GameDeploy)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("ParseStatus", func() {
		It("returns Healthy when all replicas are updated and ready", func() {
			result, err := driver.ParseStatus(map[string]any{
				"spec": map[string]any{"replicas": int64(2)},
				"status": map[string]any{
					"replicas":             int64(2),
					"readyReplicas":        int64(2),
					"updatedReplicas":      int64(2),
					"updatedReadyReplicas": int64(2),
					"observedGeneration":   int64(1),
				},
			}, ParseOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Healthy))
		})

		It("returns Suspended when the update strategy is paused", func() {
			result, err := driver.ParseStatus(map[string]any{
				"spec": map[string]any{
					"replicas":       int64(2),
					"updateStrategy": map[string]any{"paused": true},
				},
			}, ParseOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Code).To(Equal(k8sstatus.Suspended))
		})
	})

	Describe("View", func() {
		It("extracts replicas and containers", func() {
			view, err := driver.View(map[string]any{
				"apiVersion": "tkex.tencent.com/v1alpha1",
				"kind":       k8skind.GameDeploy,
				"metadata":   map[string]any{"name": "app", "namespace": "ns"},
				"spec": map[string]any{
					"replicas": int64(2),
					"template": podTemplate("main", "busybox:2.0"),
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(*view.Replicas).To(Equal(int32(2)))
			Expect(view.Containers).To(HaveLen(1))
			Expect(view.Containers[0].Image).To(Equal("busybox:2.0"))
		})
	})

	Describe("Capabilities", func() {
		It("supports inplace update and selected pod deletion", func() {
			main, err := GetMain(k8skind.GameDeploy)
			Expect(err).NotTo(HaveOccurred())
			Expect(main.Capabilities()).To(Equal(Capabilities{InplaceUpdate: true, SelectedPodDeletion: true}))
		})
	})
})
