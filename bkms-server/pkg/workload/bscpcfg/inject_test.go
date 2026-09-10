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

package bscpcfg_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

var _ = Describe("MergePodSpec", func() {
	const testInitImage = "bscp-init:test"
	const testSidecarImage = "bscp-sidecar:test"

	// buildFragment 构建测试用的 PodFragment。
	buildFragment := func() *bscpcfg.PodFragment {
		return bscpcfg.Build(bscpcfg.Params{
			BscpBizID:    "100",
			AppNames:     "order-svc",
			MountPath:    "/data/bscp",
			FeedAddr:     "feed.example.com:9510",
			Token:        "test-token",
			ProjectKey:   "BK-BSCP-00012",
			EnvName:      "dev",
			InitImage:    testInitImage,
			SidecarImage: testSidecarImage,
		})
	}

	// buildPodSpec 构建一个仅含主容器的 typed PodSpec。
	buildPodSpec := func() *corev1.PodSpec {
		return &corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "main", Image: "my-app:latest"},
			},
		}
	}

	Describe("when fragment is nil", func() {
		It("should not modify the podSpec and return nil", func() {
			podSpec := buildPodSpec()

			err := bscpcfg.MergePodSpec(podSpec, nil, "main")

			Expect(err).NotTo(HaveOccurred())
			Expect(podSpec.Containers).To(HaveLen(1))
			Expect(podSpec.InitContainers).To(BeEmpty())
			Expect(podSpec.Volumes).To(BeEmpty())
		})
	})

	Describe("when podSpec is nil", func() {
		It("should return ErrPodSpecNil", func() {
			err := bscpcfg.MergePodSpec(nil, buildFragment(), "main")

			Expect(err).To(MatchError(bscpcfg.ErrPodSpecNil))
		})
	})

	Describe("when main container is not found", func() {
		It("should return ErrMainContainerNotFound with container name", func() {
			err := bscpcfg.MergePodSpec(buildPodSpec(), buildFragment(), "non-existent")

			Expect(err).To(HaveOccurred())
			Expect(errors.Cause(err)).To(Equal(bscpcfg.ErrMainContainerNotFound))
			Expect(err.Error()).To(ContainSubstring("non-existent"))
		})

		It("should return ErrMainContainerNotFound when containers is empty", func() {
			podSpec := &corev1.PodSpec{Containers: []corev1.Container{}}

			err := bscpcfg.MergePodSpec(podSpec, buildFragment(), "main")

			Expect(err).To(HaveOccurred())
			Expect(errors.Cause(err)).To(Equal(bscpcfg.ErrMainContainerNotFound))
		})
	})

	Describe("normal injection", func() {
		It("should inject initContainers, sidecar, volumes and volumeMounts", func() {
			podSpec := buildPodSpec()

			err := bscpcfg.MergePodSpec(podSpec, buildFragment(), "main")

			Expect(err).NotTo(HaveOccurred())

			By("init container should be added")
			Expect(podSpec.InitContainers).To(HaveLen(1))
			Expect(podSpec.InitContainers[0].Name).To(Equal(bscpcfg.InitContainerName))

			By("sidecar container should be appended")
			Expect(podSpec.Containers).To(HaveLen(2))
			Expect(podSpec.Containers[0].Name).To(Equal("main"))
			Expect(podSpec.Containers[1].Name).To(Equal(bscpcfg.SidecarContainerName))

			By("volumes should be added")
			Expect(podSpec.Volumes).To(HaveLen(2))
			Expect(podSpec.Volumes[0].Name).To(Equal(bscpcfg.VolumeName))
			Expect(podSpec.Volumes[1].Name).To(Equal(bscpcfg.ShareVolumeName))

			By("main container should have the shared volume mount")
			mounts := podSpec.Containers[0].VolumeMounts
			Expect(mounts).To(HaveLen(1))
			Expect(mounts[0].Name).To(Equal(bscpcfg.ShareVolumeName))
			Expect(mounts[0].MountPath).To(Equal("/data/bscp"))
		})
	})

	Describe("idempotency - already injected", func() {
		It("should not inject again when already present", func() {
			podSpec := &corev1.PodSpec{
				InitContainers: []corev1.Container{
					{Name: bscpcfg.InitContainerName, Image: testInitImage},
				},
				Containers: []corev1.Container{
					{
						Name:  "main",
						Image: "my-app:latest",
						VolumeMounts: []corev1.VolumeMount{
							{Name: bscpcfg.ShareVolumeName, MountPath: "/data/bscp"},
						},
					},
					{Name: bscpcfg.SidecarContainerName, Image: testSidecarImage},
				},
				Volumes: []corev1.Volume{
					{Name: bscpcfg.VolumeName},
					{Name: bscpcfg.ShareVolumeName},
				},
			}

			err := bscpcfg.MergePodSpec(podSpec, buildFragment(), "main")

			Expect(err).NotTo(HaveOccurred())
			Expect(podSpec.InitContainers).To(HaveLen(1))
			Expect(podSpec.Containers).To(HaveLen(2))
			Expect(podSpec.Volumes).To(HaveLen(2))
			Expect(podSpec.Containers[0].VolumeMounts).To(HaveLen(1))
		})
	})

	Describe("multi-container - volumeMount only injected into main container", func() {
		It("should inject volumeMount only into the specified main container", func() {
			podSpec := &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "primary",
						Image: "primary:latest",
						VolumeMounts: []corev1.VolumeMount{
							{Name: "app-config", MountPath: "/etc/config"},
						},
					},
					{Name: "secondary", Image: "secondary:latest"},
				},
			}

			err := bscpcfg.MergePodSpec(podSpec, buildFragment(), "primary")

			Expect(err).NotTo(HaveOccurred())

			// primary 应保留原有 volumeMount 并追加注入的共享卷挂载
			primaryMounts := podSpec.Containers[0].VolumeMounts
			Expect(primaryMounts).To(HaveLen(2))
			Expect(primaryMounts[0].Name).To(Equal("app-config"))
			Expect(primaryMounts[1].Name).To(Equal(bscpcfg.ShareVolumeName))

			// secondary 不应有 volumeMount
			Expect(podSpec.Containers[1].VolumeMounts).To(BeEmpty())
		})
	})
})
