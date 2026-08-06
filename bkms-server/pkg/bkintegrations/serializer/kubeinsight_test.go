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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
)

var _ = Describe("KubeInsight Serializer", func() {
	Describe("GetLatestEnvReportOutput", func() {
		It("should parse raw JSON with full report into struct correctly", func() {
			rawJSON := `{
				"data": {
					"clusterID": "BCS-K8S-00001",
					"startTime": "2026-06-20T00:00:00Z",
					"endTime": "2026-06-20T23:59:59Z",
					"clusterInfo": {
						"clusterID": "BCS-K8S-00001",
						"clusterName": "test-cluster",
						"clusterType": "k8s",
						"provider": "tencentCloud",
						"manageType": "MANAGED_CLUSTER",
						"creator": "admin",
						"businessID": "100001",
						"projectID": "proj-001",
						"clusterVersion": "1.24.8",
						"networkParams": {
							"cidrs": ["10.0.0.0/16", "172.16.0.0/16"],
							"maxNodePodNum": 256,
							"maxServiceNum": 4096
						},
						"osRuntimeInfo": {
							"osImage": "Ubuntu 22.04",
							"runtime": "containerd",
							"runtimeVersion": "1.6.20"
						},
						"region": "ap-guangzhou",
						"vpcID": "vpc-12345",
						"nodeCount": 10,
						"masterCount": 3
					},
					"abnormalItems": [
						{
							"timestamp": "2026-06-20T10:00:00Z",
							"lastUpdateTimestamp": "2026-06-20T10:30:00Z",
							"key": "pod-restart-check",
							"description": "Pod frequent restarts",
							"category": "workload",
							"contextMsg": "pod nginx-xxx restarted 5 times in the past 1 hour",
							"level": "warning",
							"resourceKey": "nginx-xxx",
							"resourceType": "Pod",
							"errorDetail": "OOMKilled",
							"solutions": "Increase memory limit",
							"recovered": false,
							"recordCount": 5
						}
					],
					"score": 85
				}
			}`

			var resp serializer.GetLatestEnvReportOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ClusterID).To(Equal("BCS-K8S-00001"))
			Expect(resp.Data.StartTime).To(Equal("2026-06-20T00:00:00Z"))
			Expect(resp.Data.EndTime).To(Equal("2026-06-20T23:59:59Z"))
			Expect(resp.Data.Score).To(Equal(int32(85)))

			// 验证 clusterInfo
			Expect(resp.Data.ClusterInfo).NotTo(BeNil())
			Expect(resp.Data.ClusterInfo.ClusterID).To(Equal("BCS-K8S-00001"))
			Expect(resp.Data.ClusterInfo.ClusterName).To(Equal("test-cluster"))
			Expect(resp.Data.ClusterInfo.ClusterVersion).To(Equal("1.24.8"))
			Expect(resp.Data.ClusterInfo.NodeCount).To(Equal(int32(10)))
			Expect(resp.Data.ClusterInfo.MasterCount).To(Equal(int32(3)))

			// 验证 networkParams
			Expect(resp.Data.ClusterInfo.NetworkParams).NotTo(BeNil())
			Expect(resp.Data.ClusterInfo.NetworkParams.Cidrs).To(HaveLen(2))
			Expect(resp.Data.ClusterInfo.NetworkParams.MaxNodePodNum).To(Equal(int32(256)))

			// 验证 osRuntimeInfo
			Expect(resp.Data.ClusterInfo.OSRuntimeInfo).NotTo(BeNil())
			Expect(resp.Data.ClusterInfo.OSRuntimeInfo.OSImage).To(Equal("Ubuntu 22.04"))
			Expect(resp.Data.ClusterInfo.OSRuntimeInfo.Runtime).To(Equal("containerd"))

			// 验证 abnormalItems
			Expect(resp.Data.AbnormalItems).To(HaveLen(1))
			Expect(resp.Data.AbnormalItems[0].Key).To(Equal("pod-restart-check"))
			Expect(resp.Data.AbnormalItems[0].Level).To(Equal("warning"))
			Expect(resp.Data.AbnormalItems[0].Recovered).To(BeFalse())
			Expect(resp.Data.AbnormalItems[0].RecordCount).To(Equal(int32(5)))
		})

		It("should parse JSON with null data", func() {
			rawJSON := `{"data": null}`

			var resp serializer.GetLatestEnvReportOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeNil())
		})

		It("should parse JSON with empty abnormalItems", func() {
			rawJSON := `{
				"data": {
					"clusterID": "BCS-K8S-00002",
					"startTime": "2026-06-20T00:00:00Z",
					"endTime": "2026-06-20T23:59:59Z",
					"clusterInfo": null,
					"abnormalItems": [],
					"score": 100
				}
			}`

			var resp serializer.GetLatestEnvReportOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data.Score).To(Equal(int32(100)))
			Expect(resp.Data.AbnormalItems).To(BeEmpty())
			Expect(resp.Data.ClusterInfo).To(BeNil())
		})
	})
})
