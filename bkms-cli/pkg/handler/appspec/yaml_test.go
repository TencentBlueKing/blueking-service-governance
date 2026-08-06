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

package appspec

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("YAML Parsing", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "appspec-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("ParseYAMLFile", func() {
		It("should return error when file does not exist", func() {
			var target map[string]any
			err := ParseYAMLFile("/nonexistent/path.yaml", &target)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("file not found"))
		})

		It("should return error when file is empty", func() {
			emptyFile := filepath.Join(tmpDir, "empty.yaml")
			err := os.WriteFile(emptyFile, []byte(""), 0o644)
			Expect(err).NotTo(HaveOccurred())

			var target map[string]any
			err = ParseYAMLFile(emptyFile, &target)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("file is empty"))
		})

		It("should return error when YAML format is invalid", func() {
			badFile := filepath.Join(tmpDir, "bad.yaml")
			err := os.WriteFile(badFile, []byte("invalid: [yaml: content"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			var target map[string]any
			err = ParseYAMLFile(badFile, &target)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse YAML file"))
		})
	})

	Describe("ParseResourcesFile", func() {
		It("should parse resources YAML correctly", func() {
			content := `replicas: 2
cpuRequests: "0.5"
cpuLimits: "2"
memoryRequests: "256Mi"
memoryLimits: "2Gi"
`
			file := filepath.Join(tmpDir, "resources.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseResourcesFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(*result.Replicas).To(Equal(int32(2)))
			Expect(*result.CPURequests).To(Equal("0.5"))
			Expect(*result.CPULimits).To(Equal("2"))
			Expect(*result.MemoryRequests).To(Equal("256Mi"))
			Expect(*result.MemoryLimits).To(Equal("2Gi"))
		})

		It("should handle partial fields", func() {
			content := `replicas: 3
cpuRequests: "1"
`
			file := filepath.Join(tmpDir, "resources-partial.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseResourcesFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(*result.Replicas).To(Equal(int32(3)))
			Expect(*result.CPURequests).To(Equal("1"))
			Expect(result.CPULimits).To(BeNil())
			Expect(result.MemoryRequests).To(BeNil())
			Expect(result.MemoryLimits).To(BeNil())
		})
	})

	Describe("ParseUpdateStrategyFile", func() {
		It("should parse update-strategy YAML correctly", func() {
			content := `maxSurge: "25%"
maxUnavailable: "25%"
`
			file := filepath.Join(tmpDir, "update-strategy.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseUpdateStrategyFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(*result.MaxSurge).To(Equal("25%"))
			Expect(*result.MaxUnavailable).To(Equal("25%"))
		})
	})

	Describe("ParseLifecycleFile", func() {
		It("should parse lifecycle YAML with EXEC type", func() {
			content := `preStop:
  type: EXEC
  exec:
    sleepSeconds: 30
terminationGracePeriodSeconds: 60
`
			file := filepath.Join(tmpDir, "lifecycle.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseLifecycleFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.PreStop).NotTo(BeNil())
			Expect(result.PreStop.Type).To(Equal("EXEC"))
			Expect(result.PreStop.Exec).NotTo(BeNil())
			Expect(*result.PreStop.Exec.SleepSeconds).To(Equal(int64(30)))
			Expect(*result.TerminationGracePeriodSeconds).To(Equal(int64(60)))
			Expect(result.PostStart).To(BeNil())
		})

		It("should parse lifecycle YAML with HTTP type", func() {
			content := `postStart:
  type: HTTP
  http:
    url: /ready
    headers:
      X-Custom: "value"
`
			file := filepath.Join(tmpDir, "lifecycle-http.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseLifecycleFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.PostStart).NotTo(BeNil())
			Expect(result.PostStart.Type).To(Equal("HTTP"))
			Expect(result.PostStart.HTTP).NotTo(BeNil())
			Expect(result.PostStart.HTTP.URL).To(Equal("/ready"))
			Expect(result.PostStart.HTTP.Headers).To(HaveKeyWithValue("X-Custom", "value"))
		})
	})

	Describe("ParseProbeFile", func() {
		It("should parse probe YAML correctly", func() {
			content := `liveness:
  handler:
    type: HTTP
    port: 8080
    url: /healthz
  initialDelaySeconds: 10
  periodSeconds: 30
  failureThreshold: 3
readiness:
  handler:
    type: TCP
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
`
			file := filepath.Join(tmpDir, "probe.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseProbeFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())

			// liveness
			Expect(result.Liveness).NotTo(BeNil())
			Expect(result.Liveness.Handler).NotTo(BeNil())
			Expect(result.Liveness.Handler.Type).To(Equal("HTTP"))
			Expect(result.Liveness.Handler.Port).To(Equal(int32(8080)))
			Expect(result.Liveness.Handler.URL).To(Equal("/healthz"))
			Expect(*result.Liveness.InitialDelaySeconds).To(Equal(int32(10)))
			Expect(*result.Liveness.PeriodSeconds).To(Equal(int32(30)))
			Expect(*result.Liveness.FailureThreshold).To(Equal(int32(3)))

			// readiness
			Expect(result.Readiness).NotTo(BeNil())
			Expect(result.Readiness.Handler).NotTo(BeNil())
			Expect(result.Readiness.Handler.Type).To(Equal("TCP"))
			Expect(result.Readiness.Handler.Port).To(Equal(int32(8080)))
			Expect(*result.Readiness.InitialDelaySeconds).To(Equal(int32(5)))
			Expect(*result.Readiness.PeriodSeconds).To(Equal(int32(10)))

			// startup
			Expect(result.Startup).To(BeNil())
		})
	})

	Describe("ParseLabelsFile", func() {
		It("should parse labels YAML correctly", func() {
			content := `labels:
  app.kubernetes.io/team: "platform"
  app.kubernetes.io/version: "v1.0"
`
			file := filepath.Join(tmpDir, "labels.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseLabelsFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Labels).To(HaveLen(2))
			Expect(result.Labels).To(HaveKeyWithValue("app.kubernetes.io/team", "platform"))
			Expect(result.Labels).To(HaveKeyWithValue("app.kubernetes.io/version", "v1.0"))
		})
	})

	Describe("ParseAnnotationsFile", func() {
		It("should parse annotations YAML correctly", func() {
			content := `annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
`
			file := filepath.Join(tmpDir, "annotations.yaml")
			err := os.WriteFile(file, []byte(content), 0o644)
			Expect(err).NotTo(HaveOccurred())

			result, err := ParseAnnotationsFile(file)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Annotations).To(HaveLen(2))
			Expect(result.Annotations).To(HaveKeyWithValue("prometheus.io/scrape", "true"))
			Expect(result.Annotations).To(HaveKeyWithValue("prometheus.io/port", "9090"))
		})
	})
})
