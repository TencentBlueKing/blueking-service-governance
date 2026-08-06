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

package devmode_test

import (
	tkex "github.com/Tencent/bk-bcs/bcs-scenarios/kourse/pkg/apis/tkex/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/devmode"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

var _ = Describe("PatchGameDeployment TRPC", func() {
	var baseGD tkex.GameDeployment

	// 每个测试前重置 GameDeployment
	BeforeEach(func() {
		baseGD = tkex.GameDeployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tkex.tencent.com/v1alpha1",
				Kind:       "GameDeployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-game-deployment",
				Namespace: "default",
			},
			Spec: tkex.GameDeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    defaults.WorkloadMainContainerName,
								Image:   "test-image:latest",
								Command: []string{"./original-app"},
								Args:    []string{"--port=8080"},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "existing-volume",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
						},
					},
				},
			},
		}
	})

	// ============================================================
	// Extra Objects 测试
	// ============================================================

	Describe("Extra Objects", func() {
		var validConfig *devmode.Config

		BeforeEach(func() {
			validConfig = &devmode.Config{
				Enabled:        true,
				EnvType:        devmode.EnvTypeDevelopment,
				AppType:        bkmsapp.AppTypeTRPC,
				AppName:        "test-app",
				StartupCommand: "./test-app",
				TrpcBinaryPath: "/usr/local/trpc/bin",
			}
		})

		It("should return ConfigMap as extra object", func() {
			_, extraObjs, err := devmode.PatchGameDeployment(baseGD, validConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(extraObjs).To(HaveLen(1))

			configMapObj := extraObjs[0]
			Expect(configMapObj.GetKind()).To(Equal("ConfigMap"))
			Expect(configMapObj.GetName()).To(Equal(devmode.ConfigMapResourceName("test-app")))
		})

		It("should include all scripts in ConfigMap data", func() {
			_, extraObjs, err := devmode.PatchGameDeployment(baseGD, validConfig)
			Expect(err).NotTo(HaveOccurred())

			configMapObj := extraObjs[0]
			data, found, err := unstructured.NestedStringMap(configMapObj.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(data).To(HaveKey(devmode.KeyInitScript))
			Expect(data).To(HaveKey(devmode.KeyStartScript))
			Expect(data).To(HaveKey(devmode.KeyStopScript))
			Expect(data).To(HaveKey(devmode.KeyMonitorScript))
			Expect(data).To(HaveKey(devmode.KeyRestartScript))
		})
	})

	// ============================================================
	// 完整流程测试
	// ============================================================

	Describe("Full Integration", func() {
		It("should successfully patch GameDeployment with all dev mode components", func() {
			config := &devmode.Config{
				Enabled:        true,
				EnvType:        devmode.EnvTypeDevelopment,
				AppType:        bkmsapp.AppTypeTRPC,
				AppName:        "my-service",
				TrpcBinaryPath: "/usr/local/trpc/bin",
				WorkPath:       "/data/bkms/dev-mode/trpc",
				StartupCommand: "./my-service --config=/etc/config.yaml",
			}

			result, extraObjs, err := devmode.PatchGameDeployment(baseGD, config)
			Expect(err).NotTo(HaveOccurred())

			// 验证 ConfigMap 已添加
			Expect(extraObjs).To(HaveLen(1))
			Expect(extraObjs[0].GetName()).To(Equal(devmode.ConfigMapResourceName("my-service")))

			// 验证 Volume 已添加
			volumeFound := false
			for _, v := range result.Spec.Template.Spec.Volumes {
				if v.Name == devmode.ConfigMapResourceName("my-service") {
					volumeFound = true
					break
				}
			}
			Expect(volumeFound).To(BeTrue())

			// 验证主容器的配置
			var mainContainer *corev1.Container
			for i := range result.Spec.Template.Spec.Containers {
				if result.Spec.Template.Spec.Containers[i].Name == defaults.WorkloadMainContainerName {
					mainContainer = &result.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(mainContainer).NotTo(BeNil())

			// 验证 VolumeMount
			mountFound := false
			for _, m := range mainContainer.VolumeMounts {
				if m.Name == devmode.ConfigMapResourceName("my-service") {
					mountFound = true
					Expect(m.MountPath).To(Equal(devmode.TrpcMountPath))
					break
				}
			}
			Expect(mountFound).To(BeTrue())

			// 验证启动命令
			Expect(mainContainer.Command).To(Equal([]string{devmode.TrpcInitScriptPath}))
			Expect(mainContainer.Args).To(BeEmpty())
		})

		It("should successfully patch GameDeployment in staging environment", func() {
			config := &devmode.Config{
				Enabled:        true,
				EnvType:        devmode.EnvTypeStaging,
				AppType:        bkmsapp.AppTypeTRPC,
				AppName:        "my-service",
				TrpcBinaryPath: "/usr/local/trpc/bin",
				StartupCommand: "./my-service --config=/etc/config.yaml",
			}

			result, extraObjs, err := devmode.PatchGameDeployment(baseGD, config)
			Expect(err).NotTo(HaveOccurred())

			// 验证 ConfigMap 已添加
			Expect(extraObjs).To(HaveLen(1))
			Expect(extraObjs[0].GetName()).To(Equal(devmode.ConfigMapResourceName("my-service")))

			// 验证主容器的配置
			var mainContainer *corev1.Container
			for i := range result.Spec.Template.Spec.Containers {
				if result.Spec.Template.Spec.Containers[i].Name == defaults.WorkloadMainContainerName {
					mainContainer = &result.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(mainContainer).NotTo(BeNil())

			// 验证启动命令
			Expect(mainContainer.Command).To(Equal([]string{devmode.TrpcInitScriptPath}))
			Expect(mainContainer.Args).To(BeEmpty())
		})

		It("should use custom WorkPath for mount path calculation", func() {
			config := &devmode.Config{
				Enabled:        true,
				EnvType:        devmode.EnvTypeDevelopment,
				AppType:        bkmsapp.AppTypeTRPC,
				AppName:        "my-service",
				StartupCommand: "./my-service",
				TrpcBinaryPath: "/usr/local/trpc/bin",
			}

			result, _, err := devmode.PatchGameDeployment(baseGD, config)
			Expect(err).NotTo(HaveOccurred())

			var mainContainer *corev1.Container
			for i := range result.Spec.Template.Spec.Containers {
				if result.Spec.Template.Spec.Containers[i].Name == defaults.WorkloadMainContainerName {
					mainContainer = &result.Spec.Template.Spec.Containers[i]
					break
				}
			}
			Expect(mainContainer).NotTo(BeNil())

			// 验证 VolumeMount 路径
			var mountPath string
			for _, m := range mainContainer.VolumeMounts {
				if m.Name == devmode.ConfigMapResourceName("my-service") {
					mountPath = m.MountPath
					break
				}
			}
			Expect(mountPath).To(Equal("/data/bkms/dev-mode/trpc/configmap-scripts"))

			// 验证启动命令使用正确的路径
			Expect(mainContainer.Command).To(Equal([]string{"/data/bkms/dev-mode/trpc/configmap-scripts/init.sh"}))
		})
	})
})
