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

package serializer_test

import (
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
	instancelogsvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

var _ = Describe("Instance serializer", func() {
	Describe("FromPodManifest", func() {
		It("converts a pod manifest into an app instance output", func() {
			manifest := map[string]any{
				"metadata": map[string]any{
					"name":              "pod-1",
					"creationTimestamp": "2026-05-29T00:00:00Z",
				},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{"image": "example:v1"},
					},
				},
				"status": map[string]any{
					"podIP":  "127.0.0.1",
					"hostIP": "127.0.0.1",
					"phase":  "Running",
					"containerStatuses": []any{
						map[string]any{"restartCount": int64(1)},
						map[string]any{"restartCount": int64(3)},
					},
					"conditions": []any{
						map[string]any{"type": "Ready", "status": "True"},
					},
				},
			}

			output, err := new(serializer.AppInstanceOutputObj).FromPodManifest(manifest, "deploy-id")

			Expect(err).NotTo(HaveOccurred())
			Expect(output.ID).To(Equal("pod-1"))
			Expect(output.DeployID).To(Equal("deploy-id"))
			Expect(output.IP).To(Equal("127.0.0.1"))
			Expect(output.NodeIP).To(Equal("127.0.0.1"))
			Expect(output.Image).To(Equal("example:v1"))
			Expect(output.RestartCount).To(Equal(int64(3)))
			Expect(output.Status).To(Equal("Running"))
			Expect(output.IsHealthy).To(BeTrue())
			Expect(output.Age).NotTo(BeEmpty())
		})

		It("returns an error when pod name is missing", func() {
			_, err := new(serializer.AppInstanceOutputObj).FromPodManifest(map[string]any{}, "deploy-id")

			Expect(err).To(MatchError("pod name is empty"))
		})
	})

	Describe("MergePolarisInfoToAppInstances", func() {
		It("attaches matched Polaris instances by pod IP and service port", func() {
			appInstances := []*serializer.AppInstanceOutputObj{{ID: "pod-1", IP: "127.0.0.1"}}
			svcInstances := []*polaris.PolarisServiceInstances{
				{
					ServiceNamespace: "Production",
					ServiceName:      "svc-a",
					ServicePort:      8080,
					Instances: []*polarisInfra.Instance{
						{
							IP:                "127.0.0.1",
							Port:              8080,
							Weight:            100,
							IsHealthy:         true,
							EnableHealthCheck: true,
							Metadata:          map[string]string{"k": "v"},
						},
						{IP: "127.0.0.1", Port: 9090},
					},
				},
			}

			serializer.MergePolarisInfoToAppInstances(appInstances, svcInstances)

			Expect(appInstances[0].PolarisInfos).To(HaveLen(1))
			Expect(appInstances[0].PolarisInfos[0].ServiceNamespace).To(Equal("Production"))
			Expect(appInstances[0].PolarisInfos[0].ServiceName).To(Equal("svc-a"))
			Expect(appInstances[0].PolarisInfos[0].Port).To(Equal(uint32(8080)))
			Expect(appInstances[0].PolarisInfos[0].Weight).To(Equal(int64(100)))
			Expect(appInstances[0].PolarisInfos[0].Metadata).To(Equal(map[string]string{"k": "v"}))
		})
	})

	Describe("LogEntryOutputObj", func() {
		It("converts a log entry model", func() {
			output := new(serializer.LogEntryOutputObj).FromModel(&instancelogsvc.LogEntry{
				Timestamp: "2026-05-29T00:00:00Z",
				Content:   "hello",
			})

			Expect(output.Timestamp).To(Equal("2026-05-29T00:00:00Z"))
			Expect(output.Content).To(Equal("hello"))
		})
	})

	Describe("PortForwardQueryInput", func() {
		var validate *validator.Validate

		BeforeEach(func() {
			validate = validator.New()
			validate.SetTagName("binding")
		})

		It("passes validation with required ports", func() {
			input := serializer.PortForwardQueryInput{RemotePort: 8080, LocalPort: 18080}

			Expect(validate.Struct(input)).To(Succeed())
		})

		It("fails when remotePort is out of range", func() {
			input := serializer.PortForwardQueryInput{RemotePort: 65536, LocalPort: 18080}

			Expect(validate.Struct(input)).NotTo(Succeed())
		})

		It("fails when localPort is out of range", func() {
			input := serializer.PortForwardQueryInput{RemotePort: 8080, LocalPort: 0}

			Expect(validate.Struct(input)).NotTo(Succeed())
		})
	})
})
