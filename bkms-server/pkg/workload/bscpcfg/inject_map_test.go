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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

var _ = Describe("MergePodSpecMap", func() {
	// buildFragment 构建测试用的 PodFragment
	buildFragment := func() *bscpcfg.PodFragment {
		return bscpcfg.Build(bscpcfg.Params{
			BscpBizID: "100",
			AppNames:  "order-svc",
			MountPath: "/data/bscp",
			FeedAddr:  "feed.example.com:9510",
			Token:     "test-token",
		})
	}

	// buildPodSpecMap 构建一个基本的 podSpecMap
	buildPodSpecMap := func() map[string]any {
		return map[string]any{
			"containers": []any{
				map[string]any{
					"name":  "main",
					"image": "my-app:latest",
				},
			},
		}
	}

	Describe("when fragment is nil", func() {
		It("should not modify the podSpecMap and return nil", func() {
			podSpecMap := buildPodSpecMap()

			err := bscpcfg.MergePodSpecMap(podSpecMap, nil, "main")

			Expect(err).NotTo(HaveOccurred())
			// podSpecMap 应保持不变
			containers := podSpecMap["containers"].([]any)
			Expect(containers).To(HaveLen(1))
			_, hasInitContainers := podSpecMap["initContainers"]
			Expect(hasInitContainers).To(BeFalse())
		})
	})

	Describe("when podSpecMap is nil", func() {
		It("should return ErrPodSpecNil", func() {
			fragment := buildFragment()

			err := bscpcfg.MergePodSpecMap(nil, fragment, "main")

			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(bscpcfg.ErrPodSpecNil))
		})
	})

	Describe("when main container is not found", func() {
		It("should return ErrMainContainerNotFound with container name", func() {
			podSpecMap := buildPodSpecMap()
			fragment := buildFragment()

			err := bscpcfg.MergePodSpecMap(podSpecMap, fragment, "non-existent")

			Expect(err).To(HaveOccurred())
			Expect(errors.Cause(err)).To(Equal(bscpcfg.ErrMainContainerNotFound))
			Expect(err.Error()).To(ContainSubstring("non-existent"))
		})

		It("should return ErrMainContainerNotFound when containers is empty", func() {
			podSpecMap := map[string]any{
				"containers": []any{},
			}
			fragment := buildFragment()

			err := bscpcfg.MergePodSpecMap(podSpecMap, fragment, "main")

			Expect(err).To(HaveOccurred())
			Expect(errors.Cause(err)).To(Equal(bscpcfg.ErrMainContainerNotFound))
		})
	})

	Describe("normal injection", func() {
		It("should inject initContainers, containers, volumes and volumeMounts", func() {
			podSpecMap := buildPodSpecMap()
			fragment := buildFragment()

			err := bscpcfg.MergePodSpecMap(podSpecMap, fragment, "main")

			Expect(err).NotTo(HaveOccurred())

			By("initContainers should be created and added")
			initContainers := podSpecMap["initContainers"].([]any)
			Expect(initContainers).To(HaveLen(1))
			ic := initContainers[0].(map[string]any)
			Expect(ic["name"]).To(Equal(bscpcfg.InitContainerName))

			By("sidecar container should be appended")
			containers := podSpecMap["containers"].([]any)
			Expect(containers).To(HaveLen(2))
			sidecar := containers[1].(map[string]any)
			Expect(sidecar["name"]).To(Equal(bscpcfg.SidecarContainerName))

			By("volumes should be created and added")
			volumes := podSpecMap["volumes"].([]any)
			Expect(volumes).To(HaveLen(2))
			Expect(volumes[0].(map[string]any)["name"]).To(Equal(bscpcfg.VolumeName))
			Expect(volumes[1].(map[string]any)["name"]).To(Equal(bscpcfg.ShareVolumeName))

			By("main container should have volumeMount (bscp-share -> MountPath)")
			mainContainer := containers[0].(map[string]any)
			mounts := mainContainer["volumeMounts"].([]any)
			Expect(mounts).To(HaveLen(1))
			mount := mounts[0].(map[string]any)
			Expect(mount["name"]).To(Equal(bscpcfg.ShareVolumeName))
			Expect(mount["mountPath"]).To(Equal("/data/bscp"))
		})
	})

	Describe("idempotency - already injected", func() {
		It("should skip injection when already present", func() {
			podSpecMap := map[string]any{
				"initContainers": []any{
					map[string]any{"name": bscpcfg.InitContainerName, "image": bscpcfg.InitImage},
				},
				"containers": []any{
					map[string]any{
						"name":  "main",
						"image": "my-app:latest",
						"volumeMounts": []any{
							map[string]any{"name": bscpcfg.ShareVolumeName, "mountPath": "/data/bscp"},
						},
					},
					map[string]any{"name": bscpcfg.SidecarContainerName, "image": bscpcfg.SidecarImage},
				},
				"volumes": []any{
					map[string]any{"name": bscpcfg.VolumeName, "emptyDir": map[string]any{}},
					map[string]any{"name": bscpcfg.ShareVolumeName, "emptyDir": map[string]any{}},
				},
			}
			fragment := buildFragment()

			err := bscpcfg.MergePodSpecMap(podSpecMap, fragment, "main")

			Expect(err).NotTo(HaveOccurred())
			// 不应重复追加
			Expect(podSpecMap["initContainers"].([]any)).To(HaveLen(1))
			Expect(podSpecMap["containers"].([]any)).To(HaveLen(2))
			Expect(podSpecMap["volumes"].([]any)).To(HaveLen(2))
			mainContainer := podSpecMap["containers"].([]any)[0].(map[string]any)
			Expect(mainContainer["volumeMounts"].([]any)).To(HaveLen(1))
		})
	})

	Describe("multi-container - volumeMount only injected into main container", func() {
		It("should inject volumeMount only into the specified main container", func() {
			podSpecMap := map[string]any{
				"containers": []any{
					map[string]any{
						"name":  "primary",
						"image": "primary:latest",
						"volumeMounts": []any{
							map[string]any{"name": "app-config", "mountPath": "/etc/config"},
						},
					},
					map[string]any{"name": "secondary", "image": "secondary:latest"},
				},
			}
			fragment := buildFragment()

			err := bscpcfg.MergePodSpecMap(podSpecMap, fragment, "primary")

			Expect(err).NotTo(HaveOccurred())

			containers := podSpecMap["containers"].([]any)
			// primary 应有 volumeMount（原有 + 新注入）
			primary := containers[0].(map[string]any)
			mounts := primary["volumeMounts"].([]any)
			Expect(mounts).To(HaveLen(2))
			Expect(mounts[0].(map[string]any)["name"]).To(Equal("app-config"))
			Expect(mounts[1].(map[string]any)["name"]).To(Equal(bscpcfg.ShareVolumeName))

			// secondary 不应有 volumeMount
			secondary := containers[1].(map[string]any)
			_, hasMounts := secondary["volumeMounts"]
			Expect(hasMounts).To(BeFalse())
		})
	})

	Describe("preserves newer k8s version fields unknown to our library", func() {
		It(
			"should preserve schedulingGates, resourceClaims, hostnameOverride and other newer fields after injection",
			func() {
				// 模拟用户使用了较新 k8s 版本（如 1.32+）的 PodSpec 字段，
				// 这些字段在我们 go.mod 引用的 k8s 库版本中可能不存在。
				// MergePodSpecMap 基于原生 map 操作，不会丢失这些字段。
				podSpecMap := map[string]any{
					// k8s 1.26+ schedulingGates（调度门控）
					"schedulingGates": []any{
						map[string]any{"name": "example.com/wait-for-gpu"},
						map[string]any{"name": "example.com/network-ready"},
					},
					// k8s 1.26+ resourceClaims（动态资源分配）
					"resourceClaims": []any{
						map[string]any{
							"name":              "gpu-claim",
							"resourceClaimName": "shared-gpu",
						},
					},
					// k8s 1.35+ hostnameOverride（Pod 主机名覆盖）
					"hostnameOverride": "custom-hostname",
					// 容器中使用 resources.claims（DRA 相关）
					"containers": []any{
						map[string]any{
							"name":  "main",
							"image": "my-app:latest",
							"resources": map[string]any{
								"claims": []any{
									map[string]any{"name": "gpu-claim"},
								},
								"limits": map[string]any{
									"cpu":    "2",
									"memory": "4Gi",
								},
							},
							// k8s 1.29+ resizePolicy（容器资源调整策略）
							"resizePolicy": []any{
								map[string]any{
									"resourceName":  "cpu",
									"restartPolicy": "NotRequired",
								},
							},
						},
					},
					// 常规字段
					"serviceAccountName": "my-sa",
					"nodeSelector": map[string]any{
						"disktype": "ssd",
					},
				}
				fragment := buildFragment()

				err := bscpcfg.MergePodSpecMap(podSpecMap, fragment, "main")

				Expect(err).NotTo(HaveOccurred())

				By("schedulingGates should be fully preserved")
				gates := podSpecMap["schedulingGates"].([]any)
				Expect(gates).To(HaveLen(2))
				Expect(gates[0].(map[string]any)["name"]).To(Equal("example.com/wait-for-gpu"))
				Expect(gates[1].(map[string]any)["name"]).To(Equal("example.com/network-ready"))

				By("resourceClaims should be fully preserved")
				claims := podSpecMap["resourceClaims"].([]any)
				Expect(claims).To(HaveLen(1))
				Expect(claims[0].(map[string]any)["name"]).To(Equal("gpu-claim"))
				Expect(claims[0].(map[string]any)["resourceClaimName"]).To(Equal("shared-gpu"))

				By("hostnameOverride should be preserved")
				Expect(podSpecMap["hostnameOverride"]).To(Equal("custom-hostname"))

				By("serviceAccountName should be preserved")
				Expect(podSpecMap["serviceAccountName"]).To(Equal("my-sa"))

				By("nodeSelector should be preserved")
				nodeSelector := podSpecMap["nodeSelector"].(map[string]any)
				Expect(nodeSelector["disktype"]).To(Equal("ssd"))

				By("main container resources.claims should be preserved")
				mainContainer := podSpecMap["containers"].([]any)[0].(map[string]any)
				resources := mainContainer["resources"].(map[string]any)
				resClaims := resources["claims"].([]any)
				Expect(resClaims).To(HaveLen(1))
				Expect(resClaims[0].(map[string]any)["name"]).To(Equal("gpu-claim"))

				By("main container resources.limits should be preserved")
				limits := resources["limits"].(map[string]any)
				Expect(limits["cpu"]).To(Equal("2"))
				Expect(limits["memory"]).To(Equal("4Gi"))

				By("main container resizePolicy should be preserved")
				resizePolicy := mainContainer["resizePolicy"].([]any)
				Expect(resizePolicy).To(HaveLen(1))
				Expect(resizePolicy[0].(map[string]any)["resourceName"]).To(Equal("cpu"))
				Expect(resizePolicy[0].(map[string]any)["restartPolicy"]).To(Equal("NotRequired"))

				By("injection should still work correctly")
				initContainers := podSpecMap["initContainers"].([]any)
				Expect(initContainers).To(HaveLen(1))
				containers := podSpecMap["containers"].([]any)
				Expect(containers).To(HaveLen(2))
				volumes := podSpecMap["volumes"].([]any)
				Expect(volumes).To(HaveLen(2))
			},
		)
	})
})
