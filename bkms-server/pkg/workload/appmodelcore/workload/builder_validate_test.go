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
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/cfgrender"
)

var _ = Describe("validateVolumeMountPaths", func() {
	It("should pass when no mount paths conflict", func() {
		mounts := []corev1.VolumeMount{
			{Name: "vol-a", MountPath: "/etc/app/config.yaml"},
			{Name: "vol-b", MountPath: "/etc/nginx/nginx.conf"},
			{Name: "vol-c", MountPath: "/data/logs"},
		}
		Expect(validateVolumeMountPaths("main", mounts)).To(Succeed())
	})

	It("should return error when duplicate MountPath exists", func() {
		mounts := []corev1.VolumeMount{
			{Name: "framework-cfg", MountPath: "/etc/app/config.yaml"},
			{Name: "plain-cfg", MountPath: "/etc/app/config.yaml"},
		}
		err := validateVolumeMountPaths("main", mounts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`duplicate MountPath "/etc/app/config.yaml"`))
		Expect(err.Error()).To(ContainSubstring(`"framework-cfg"`))
		Expect(err.Error()).To(ContainSubstring(`"plain-cfg"`))
	})

	It("should pass when mounts is empty or nil", func() {
		Expect(validateVolumeMountPaths("init", nil)).To(Succeed())
	})

	It("should pass when there is only one mount", func() {
		mounts := []corev1.VolumeMount{
			{Name: "only-vol", MountPath: "/etc/app/config.yaml"},
		}
		Expect(validateVolumeMountPaths("main", mounts)).To(Succeed())
	})

	It("should report error on the first conflict when multiple duplicates exist", func() {
		mounts := []corev1.VolumeMount{
			{Name: "vol-a", MountPath: "/etc/app/a.yaml"},
			{Name: "vol-b", MountPath: "/etc/app/a.yaml"},
			{Name: "vol-c", MountPath: "/data/b.conf"},
			{Name: "vol-d", MountPath: "/data/b.conf"},
		}
		err := validateVolumeMountPaths("main", mounts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("duplicate MountPath"))
	})

	It("should include container name in error message", func() {
		mounts := []corev1.VolumeMount{
			{Name: "vol-a", MountPath: "/etc/dup"},
			{Name: "vol-b", MountPath: "/etc/dup"},
		}
		err := validateVolumeMountPaths("trpc-init", mounts)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(`"trpc-init"`))
	})
})

var _ = Describe("buildDirectConfigMap", func() {
	It("should produce ConfigMap with correct data and volume mounts", func() {
		files := []cfgrender.RenderedMountableFile{
			{FileName: "app.conf", FilePath: "/etc/app", FileContent: "key=value"},
			{FileName: "extra.conf", FilePath: "/etc/extra", FileContent: "foo=bar"},
		}

		result := buildDirectConfigMap("my-app-plain-direct", files)

		Expect(result).NotTo(BeNil())

		// Verify ConfigMap
		Expect(result.configMap.Name).To(Equal("my-app-plain-direct"))
		Expect(result.configMap.Data).To(HaveLen(2))
		Expect(result.configMap.Data["00-app.conf"]).To(Equal("key=value"))
		Expect(result.configMap.Data["01-extra.conf"]).To(Equal("foo=bar"))

		// Verify volume mounts
		mounts, volumes, err := result.Storage(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(mounts).To(HaveLen(2))
		Expect(mounts[0].Name).To(Equal(plainDirectVolumeName))
		Expect(mounts[0].MountPath).To(Equal("/etc/app/app.conf"))
		Expect(mounts[0].SubPath).To(Equal("00-app.conf"))
		Expect(mounts[1].MountPath).To(Equal("/etc/extra/extra.conf"))
		Expect(mounts[1].SubPath).To(Equal("01-extra.conf"))

		// Verify volumes
		Expect(volumes).To(HaveLen(1))
		Expect(volumes[0].Name).To(Equal(plainDirectVolumeName))
		Expect(volumes[0].ConfigMap).NotTo(BeNil())
		Expect(volumes[0].ConfigMap.Name).To(Equal("my-app-plain-direct"))
		Expect(volumes[0].ConfigMap.Items).To(HaveLen(2))
	})

	It("should keep volume name within DNS-1123 label limit when ConfigMap name is long", func() {
		cmName := strings.Repeat("a", 63) + "-plain-direct"
		result := buildDirectConfigMap(cmName, []cfgrender.RenderedMountableFile{
			{FileName: "nginx.conf", FilePath: "/etc/nginx", FileContent: "worker_processes 1;\n"},
		})
		Expect(result.configMap.Name).To(Equal(cmName))
		Expect(len(result.configMap.Name)).To(BeNumerically(">", 63))

		mounts, volumes, err := result.Storage(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(volumes[0].Name).To(Equal(plainDirectVolumeName))
		Expect(len(volumes[0].Name)).To(BeNumerically("<=", 63))
		Expect(mounts[0].Name).To(Equal(plainDirectVolumeName))
		Expect(volumes[0].ConfigMap.Name).To(Equal(cmName))
	})

	It("should produce valid ExtraResources", func() {
		files := []cfgrender.RenderedMountableFile{
			{FileName: "test.conf", FilePath: "/etc/test", FileContent: "data"},
		}

		result := buildDirectConfigMap("test-cm", files)
		extras, err := result.ExtraResources(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(extras).To(HaveLen(1))
		Expect(extras[0].GetName()).To(Equal("test-cm"))
		Expect(extras[0].GetKind()).To(Equal("ConfigMap"))
	})
})
