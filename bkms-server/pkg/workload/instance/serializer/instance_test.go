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
	"errors"
	"math"

	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
	instancelogsvc "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/instancelog"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/instance/serializer"
)

func int64Ptr(v int64) *int64 {
	return &v
}

var _ = Describe("Instance serializer", func() {
	Describe("ListAppInstancesQueryInput.Validate", func() {
		It("accepts all=true without pagination", func() {
			err := (&serializer.ListAppInstancesQueryInput{All: true}).Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects all=true together with page", func() {
			err := (&serializer.ListAppInstancesQueryInput{All: true, Page: int64Ptr(1)}).Validate()
			Expect(err).To(HaveOccurred())
			var bkErr *bkerrs.Error
			Expect(errors.As(err, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
			Expect(err.Error()).To(ContainSubstring("cannot be used together with page or pageSize"))
		})

		It("accepts pagination mode with a legal pageSize", func() {
			err := (&serializer.ListAppInstancesQueryInput{Page: int64Ptr(1), PageSize: int64Ptr(5)}).Validate()
			Expect(err).NotTo(HaveOccurred())
		})

		// page 上界拦的是脏参数：放进去只会算出一个越过尾部的空窗口，不如直接报错
		It("rejects a page beyond the upper bound", func() {
			err := (&serializer.ListAppInstancesQueryInput{
				Page: int64Ptr(math.MaxInt64), PageSize: int64Ptr(100),
			}).Validate()
			Expect(err).To(HaveOccurred())
			var bkErr *bkerrs.Error
			Expect(errors.As(err, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
			Expect(err.Error()).To(ContainSubstring("page must be between"))
		})
	})

	// 返回值直接被当作下标切片，越尾必须收敛成空区间
	DescribeTable("ListAppInstancesQueryInput.ProjectionRange",
		func(query serializer.ListAppInstancesQueryInput, total, wantStart, wantEnd int64) {
			start, end := query.ProjectionRange(total)

			Expect(start).To(Equal(wantStart))
			Expect(end).To(Equal(wantEnd))
		},
		Entry("all mode covers everything",
			serializer.ListAppInstancesQueryInput{All: true}, int64(3), int64(0), int64(3)),
		Entry("middle page takes a full window",
			serializer.ListAppInstancesQueryInput{Page: int64Ptr(2), PageSize: int64Ptr(5)},
			int64(12), int64(5), int64(10)),
		Entry("last page is partially filled",
			serializer.ListAppInstancesQueryInput{Page: int64Ptr(3), PageSize: int64Ptr(5)},
			int64(12), int64(10), int64(12)),
		Entry("page fully past the tail yields an empty range",
			serializer.ListAppInstancesQueryInput{Page: int64Ptr(4), PageSize: int64Ptr(5)},
			int64(12), int64(12), int64(12)),
		Entry("pagination on an empty list",
			serializer.ListAppInstancesQueryInput{Page: int64Ptr(1), PageSize: int64Ptr(5)},
			int64(0), int64(0), int64(0)),
	)

	Describe("WatchAppInstancesQueryInput.Validate", func() {
		It("rejects empty resourceVersion", func() {
			err := (&serializer.WatchAppInstancesQueryInput{}).Validate()
			Expect(err).To(HaveOccurred())
			var bkErr *bkerrs.Error
			Expect(errors.As(err, &bkErr)).To(BeTrue())
			Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
			Expect(err.Error()).To(ContainSubstring("resourceVersion is required"))
		})
	})

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
			Expect(output.Resources).To(Equal(serializer.AppInstanceResourcesObj{}))
		})

		It("extracts main-container cpu and memory resources", func() {
			manifest := map[string]any{
				"metadata": map[string]any{
					"name":              "pod-res",
					"creationTimestamp": "2026-05-29T00:00:00Z",
				},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "sidecar",
							"image": "sidecar:v1",
							"resources": map[string]any{
								"limits": map[string]any{"cpu": "100m", "memory": "128Mi"},
							},
						},
						map[string]any{
							"name":  "main",
							"image": "example:v1",
							"resources": map[string]any{
								"limits": map[string]any{
									"cpu":    "2",
									"memory": "4Gi",
								},
								"requests": map[string]any{
									"cpu":    "1",
									"memory": "2Gi",
								},
							},
						},
					},
				},
				"status": map[string]any{
					"phase": "Running",
					"conditions": []any{
						map[string]any{"type": "Ready", "status": "True"},
					},
				},
			}

			output, err := new(serializer.AppInstanceOutputObj).FromPodManifest(manifest, "deploy-id")

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Image).To(Equal("sidecar:v1"))
			Expect(output.Resources).To(Equal(serializer.AppInstanceResourcesObj{
				CPULimits:      "2",
				CPURequests:    "1",
				MemoryLimits:   "4Gi",
				MemoryRequests: "2Gi",
			}))
		})

		It("leaves resources empty when the main container is missing", func() {
			manifest := map[string]any{
				"metadata": map[string]any{
					"name":              "pod-no-main",
					"creationTimestamp": "2026-05-29T00:00:00Z",
				},
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "sidecar",
							"image": "sidecar:v1",
							"resources": map[string]any{
								"limits": map[string]any{"cpu": "100m"},
							},
						},
					},
				},
				"status": map[string]any{"phase": "Running"},
			}

			output, err := new(serializer.AppInstanceOutputObj).FromPodManifest(manifest, "deploy-id")

			Expect(err).NotTo(HaveOccurred())
			Expect(output.Resources).To(Equal(serializer.AppInstanceResourcesObj{}))
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

		// 北极星侧返回顺序不保证稳定，Watch 补拉要靠前后两次结果比对差异
		// 顺序漂移会被误判成变化，每轮重复推一遍 MODIFIED
		It("orders matched instances by service coordinates regardless of input order", func() {
			appInstances := []*serializer.AppInstanceOutputObj{{ID: "pod-1", IP: "127.0.0.1"}}
			svcInstances := []*polaris.PolarisServiceInstances{
				{
					ServiceNamespace: "Production",
					ServiceName:      "svc-b",
					ServicePort:      8080,
					Instances:        []*polarisInfra.Instance{{IP: "127.0.0.1", Port: 8080}},
				},
				{
					ServiceNamespace: "Development",
					ServiceName:      "svc-a",
					ServicePort:      9090,
					Instances:        []*polarisInfra.Instance{{IP: "127.0.0.1", Port: 9090}},
				},
				{
					ServiceNamespace: "Production",
					ServiceName:      "svc-a",
					ServicePort:      8080,
					Instances:        []*polarisInfra.Instance{{IP: "127.0.0.1", Port: 8080}},
				},
			}

			serializer.MergePolarisInfoToAppInstances(appInstances, svcInstances)

			infos := appInstances[0].PolarisInfos
			Expect(infos).To(HaveLen(3))
			Expect(infos[0].ServiceNamespace).To(Equal("Development"))
			Expect(infos[0].ServiceName).To(Equal("svc-a"))
			Expect(infos[1].ServiceNamespace).To(Equal("Production"))
			Expect(infos[1].ServiceName).To(Equal("svc-a"))
			Expect(infos[2].ServiceNamespace).To(Equal("Production"))
			Expect(infos[2].ServiceName).To(Equal("svc-b"))
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
