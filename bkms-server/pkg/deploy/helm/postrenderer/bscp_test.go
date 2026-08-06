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

package postrenderer

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	wlbscpcfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/bscpcfg"
)

// buildTestFragment 构建测试用的 PodFragment
func buildTestFragment(workload string) *wlbscpcfg.PodFragment {
	f := wlbscpcfg.Build(wlbscpcfg.Params{
		BscpBizID: "100",
		AppNames:  "order-svc",
		MountPath: "/data/bscp",
		FeedAddr:  "feed.example.com:9510",
		Token:     "test-token",
	})
	f.WorkloadName = workload
	return f
}

var _ = Describe("BscpPostRenderer", func() {
	Describe("NewBscpPostRenderer", func() {
		It("should return nil when fragment is nil", func() {
			renderer := NewBscpPostRenderer(nil)
			Expect(renderer).To(BeNil())
		})

		It("should return a non-nil renderer when fragment is provided", func() {
			fragment := buildTestFragment("my-app")
			renderer := NewBscpPostRenderer(fragment)
			Expect(renderer).NotTo(BeNil())
		})
	})

	Describe("Run with nil renderer or fragment", func() {
		It("should return the original manifest unchanged", func() {
			manifest := bytes.NewBufferString("apiVersion: v1\nkind: Service\n")
			var renderer *BscpPostRenderer

			result, err := renderer.Run(manifest)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.String()).To(Equal("apiVersion: v1\nkind: Service\n"))
		})
	})

	Describe("injection content correctness", func() {
		It("should inject correct env vars, args, and images", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
        - name: main
          image: my-app:latest
`
			fragment := &wlbscpcfg.PodFragment{
				InitContainers: []corev1.Container{
					{
						Name:  wlbscpcfg.InitContainerName,
						Image: wlbscpcfg.InitImage,
						Args:  []string{"--file-cache-enabled=false"},
						Env: []corev1.EnvVar{
							{Name: "biz", Value: "100"},
							{Name: "app", Value: "order-svc"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: wlbscpcfg.VolumeName, MountPath: wlbscpcfg.BscpDownloadPath},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Name:  wlbscpcfg.SidecarContainerName,
						Image: wlbscpcfg.SidecarImage,
						Args:  []string{"--file-cache-enabled=false"},
						Env: []corev1.EnvVar{
							{Name: "biz", Value: "100"},
							{Name: "app", Value: "order-svc"},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: wlbscpcfg.VolumeName, MountPath: wlbscpcfg.BscpDownloadPath},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: wlbscpcfg.VolumeName,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				MainContainerVolumeMounts: []corev1.VolumeMount{
					{Name: wlbscpcfg.VolumeName, MountPath: wlbscpcfg.BscpDownloadPath},
				},
				WorkloadName: "my-app",
			}
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			By("should contain correct images")
			Expect(output).To(ContainSubstring(wlbscpcfg.InitImage))
			Expect(output).To(ContainSubstring(wlbscpcfg.SidecarImage))

			By("should contain env vars")
			Expect(output).To(ContainSubstring("biz"))
			Expect(output).To(ContainSubstring("order-svc"))

			By("should contain args")
			Expect(output).To(ContainSubstring("--file-cache-enabled=false"))

			By("should contain emptyDir volume")
			Expect(output).To(ContainSubstring("emptyDir"))
		})
	})

	Describe("multi-workload mixed injection", func() {
		It("should inject only into the matching workload, not Service/ConfigMap", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: deploy-app
spec:
  template:
    spec:
      containers:
        - name: web
          image: web:latest
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: sts-app
spec:
  template:
    spec:
      containers:
        - name: db
          image: db:latest
---
apiVersion: v1
kind: Service
metadata:
  name: my-svc
spec:
  ports:
    - port: 80
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-cm
data:
  key: value
`
			fragment := buildTestFragment("deploy-app")
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			docs := strings.Split(output, "\n---\n")
			Expect(docs).To(HaveLen(4))

			By("Deployment (deploy-app) should have bscp injected")
			Expect(docs[0]).To(ContainSubstring("bscp-init"))
			Expect(docs[0]).To(ContainSubstring("bscp-sidecar"))

			By("StatefulSet (sts-app) should NOT have bscp injected - name doesn't match")
			Expect(docs[1]).NotTo(ContainSubstring("bscp-init"))

			By("Service should NOT have bscp injected")
			Expect(docs[2]).NotTo(ContainSubstring("bscp-init"))

			By("ConfigMap should NOT have bscp injected")
			Expect(docs[3]).NotTo(ContainSubstring("bscp-init"))
		})
	})

	Describe("idempotency - already injected", func() {
		It("should skip injection when bscp-init already exists in initContainers", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      initContainers:
        - name: bscp-init
          image: mirrors.tencent.com/bscp/bscp-init:latest
      containers:
        - name: main
          image: my-app:latest
        - name: bscp-sidecar
          image: mirrors.tencent.com/bscp/bscp-sidecar:latest
      volumes:
        - name: bscp-temp
          emptyDir: {}
`
			fragment := buildTestFragment("my-app")
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			Expect(strings.Count(output, "name: bscp-init")).To(Equal(1))
			Expect(strings.Count(output, "name: bscp-sidecar")).To(Equal(1))
		})
	})

	Describe("error cases", func() {
		It("should return error when containers is empty", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers: []
`
			fragment := buildTestFragment("my-app")
			renderer := NewBscpPostRenderer(fragment)

			_, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("has no containers"))
		})

		It("should return error when spec.template.spec is missing", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 1
`
			fragment := buildTestFragment("my-app")
			renderer := NewBscpPostRenderer(fragment)

			_, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("has no spec.template.spec"))
		})
	})

	Describe("GameDeployment and GameStatefulSet injection", func() {
		It("should inject into GameDeployment", func() {
			manifest := `apiVersion: tkex.tencent.com/v1alpha1
kind: GameDeployment
metadata:
  name: game-deploy
spec:
  template:
    spec:
      containers:
        - name: game
          image: game:latest
`
			fragment := buildTestFragment("game-deploy")
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			Expect(output).To(ContainSubstring("bscp-init"))
			Expect(output).To(ContainSubstring("bscp-sidecar"))
		})

		It("should inject into GameStatefulSet", func() {
			manifest := `apiVersion: tkex.tencent.com/v1alpha1
kind: GameStatefulSet
metadata:
  name: game-sts
spec:
  template:
    spec:
      containers:
        - name: game
          image: game:latest
`
			fragment := buildTestFragment("game-sts")
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			Expect(output).To(ContainSubstring("bscp-init"))
			Expect(output).To(ContainSubstring("bscp-sidecar"))
		})
	})

	Describe("empty YAML documents", func() {
		It("should skip empty documents without error", func() {
			manifest := "---\n---\napiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: my-app\nspec:\n  template:\n    spec:\n      containers:\n        - name: main\n          image: main:latest\n---\n"

			fragment := buildTestFragment("my-app")
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			Expect(output).To(ContainSubstring("bscp-init"))
			Expect(output).To(ContainSubstring("bscp-sidecar"))
		})
	})

	Describe("preserves newer k8s version fields through YAML round-trip", func() {
		It(
			"should preserve schedulingGates, resourceClaims, hostnameOverride, resizePolicy and custom metadata after injection",
			func() {
				// 模拟用户使用了较新 k8s 版本（如 1.32+）的 PodSpec 字段，
				// 这些字段在我们 go.mod 引用的 k8s 库版本中可能不存在。
				// 基于原生 map 操作的 PostRenderer 不会丢失这些字段。
				manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    metadata:
      annotations:
        custom.io/inject-mesh: "true"
        prometheus.io/scrape: "true"
      labels:
        app.kubernetes.io/version: v2.1.0
    spec:
      schedulingGates:
        - name: example.com/wait-for-gpu
        - name: example.com/network-ready
      resourceClaims:
        - name: gpu-claim
          resourceClaimName: shared-gpu
      hostnameOverride: custom-hostname
      containers:
        - name: main
          image: my-app:latest
          resources:
            claims:
              - name: gpu-claim
            limits:
              cpu: "2"
              memory: 4Gi
          resizePolicy:
            - resourceName: cpu
              restartPolicy: NotRequired
          startupProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 5
          volumeDevices:
            - name: block-device
              devicePath: /dev/xvda
      serviceAccountName: my-sa
      nodeSelector:
        disktype: ssd
`
				fragment := buildTestFragment("my-app")
				renderer := NewBscpPostRenderer(fragment)

				result, err := renderer.Run(bytes.NewBufferString(manifest))

				Expect(err).NotTo(HaveOccurred())
				output := result.String()

				By("schedulingGates should be fully preserved")
				Expect(output).To(ContainSubstring("example.com/wait-for-gpu"))
				Expect(output).To(ContainSubstring("example.com/network-ready"))

				By("resourceClaims should be fully preserved")
				Expect(output).To(ContainSubstring("gpu-claim"))
				Expect(output).To(ContainSubstring("resourceClaimName"))
				Expect(output).To(ContainSubstring("shared-gpu"))

				By("hostnameOverride should be preserved")
				Expect(output).To(ContainSubstring("hostnameOverride"))
				Expect(output).To(ContainSubstring("custom-hostname"))

				By("container resizePolicy should be preserved")
				Expect(output).To(ContainSubstring("resizePolicy"))
				Expect(output).To(ContainSubstring("NotRequired"))

				By("container startupProbe should be preserved")
				Expect(output).To(ContainSubstring("startupProbe"))
				Expect(output).To(ContainSubstring("/healthz"))

				By("container volumeDevices should be preserved")
				Expect(output).To(ContainSubstring("volumeDevices"))
				Expect(output).To(ContainSubstring("block-device"))
				Expect(output).To(ContainSubstring("/dev/xvda"))

				By("custom annotations should be preserved")
				Expect(output).To(ContainSubstring("custom.io/inject-mesh"))
				Expect(output).To(ContainSubstring("prometheus.io/scrape"))

				By("custom labels should be preserved")
				Expect(output).To(ContainSubstring("app.kubernetes.io/version"))
				Expect(output).To(ContainSubstring("v2.1.0"))

				By("serviceAccountName and nodeSelector should be preserved")
				Expect(output).To(ContainSubstring("my-sa"))
				Expect(output).To(ContainSubstring("disktype"))
				Expect(output).To(ContainSubstring("ssd"))

				By("injection should still work correctly")
				Expect(output).To(ContainSubstring("bscp-init"))
				Expect(output).To(ContainSubstring("bscp-sidecar"))
				Expect(output).To(ContainSubstring(wlbscpcfg.BscpDownloadPath))
			},
		)
	})

	Describe("workload name matching", func() {
		It("should return error when no workload matches the target name", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-app
spec:
  template:
    spec:
      containers:
        - name: main
          image: main:latest
`
			fragment := buildTestFragment("non-existent-workload")
			renderer := NewBscpPostRenderer(fragment)

			_, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("target workload"))
			Expect(err.Error()).To(ContainSubstring("non-existent-workload"))
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should only inject into the workload matching the target name", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: target-deploy
spec:
  template:
    spec:
      containers:
        - name: web
          image: web:latest
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: other-deploy
spec:
  template:
    spec:
      containers:
        - name: api
          image: api:latest
`
			fragment := buildTestFragment("target-deploy")
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			docs := strings.Split(output, "\n---\n")
			Expect(docs).To(HaveLen(2))

			By("target-deploy should have bscp injected")
			Expect(docs[0]).To(ContainSubstring("bscp-init"))
			Expect(docs[0]).To(ContainSubstring("bscp-sidecar"))

			By("other-deploy should NOT have bscp injected")
			Expect(docs[1]).NotTo(ContainSubstring("bscp-init"))
			Expect(docs[1]).NotTo(ContainSubstring("bscp-sidecar"))
		})
	})

	Describe("WorkloadKind matching", func() {
		It("should match both kind and name when WorkloadKind is set", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
        - name: web
          image: web:latest
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
        - name: db
          image: db:latest
`
			fragment := buildTestFragment("my-app")
			fragment.WorkloadKind = "StatefulSet"
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			docs := strings.Split(output, "\n---\n")
			Expect(docs).To(HaveLen(2))

			By("Deployment with same name should NOT be injected")
			Expect(docs[0]).NotTo(ContainSubstring("bscp-init"))
			Expect(docs[0]).NotTo(ContainSubstring("bscp-sidecar"))

			By("StatefulSet with matching name should be injected")
			Expect(docs[1]).To(ContainSubstring("bscp-init"))
			Expect(docs[1]).To(ContainSubstring("bscp-sidecar"))
		})

		It("should match only by name when WorkloadKind is empty (backward compatible)", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
        - name: web
          image: web:latest
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: other-app
spec:
  template:
    spec:
      containers:
        - name: db
          image: db:latest
`
			fragment := buildTestFragment("my-app")
			fragment.WorkloadKind = "" // 空值，向后兼容
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			docs := strings.Split(output, "\n---\n")
			Expect(docs).To(HaveLen(2))

			By("Deployment matching name should be injected (kind not checked)")
			Expect(docs[0]).To(ContainSubstring("bscp-init"))
			Expect(docs[0]).To(ContainSubstring("bscp-sidecar"))

			By("StatefulSet with different name should NOT be injected")
			Expect(docs[1]).NotTo(ContainSubstring("bscp-init"))
		})

		It("should return error when WorkloadKind is set but no matching kind+name found", func() {
			manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  template:
    spec:
      containers:
        - name: web
          image: web:latest
`
			fragment := buildTestFragment("my-app")
			fragment.WorkloadKind = "StatefulSet" // 指定 StatefulSet 但只有 Deployment
			renderer := NewBscpPostRenderer(fragment)

			_, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("target workload"))
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should inject into DaemonSet when WorkloadKind is DaemonSet", func() {
			manifest := `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: my-daemon
spec:
  template:
    spec:
      containers:
        - name: agent
          image: agent:latest
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-daemon
spec:
  template:
    spec:
      containers:
        - name: web
          image: web:latest
`
			fragment := buildTestFragment("my-daemon")
			fragment.WorkloadKind = "DaemonSet"
			renderer := NewBscpPostRenderer(fragment)

			result, err := renderer.Run(bytes.NewBufferString(manifest))

			Expect(err).NotTo(HaveOccurred())
			output := result.String()

			docs := strings.Split(output, "\n---\n")
			Expect(docs).To(HaveLen(2))

			By("DaemonSet should be injected")
			Expect(docs[0]).To(ContainSubstring("bscp-init"))
			Expect(docs[0]).To(ContainSubstring("bscp-sidecar"))

			By("Deployment with same name should NOT be injected")
			Expect(docs[1]).NotTo(ContainSubstring("bscp-init"))
			Expect(docs[1]).NotTo(ContainSubstring("bscp-sidecar"))
		})
	})
})
