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

var _ = Describe("BCS Serializer", func() {
	Describe("ListBCSAuthorizedProjectsOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"id": "proj0001proj0001proj0001proj0001",
						"name": "Test Project A",
						"code": "test-project-a",
						"description": "This is a test project",
						"bizID": "100001",
						"isOffline": false,
						"isBoundWorkspace": true
					},
					{
						"id": "proj0002proj0002proj0002proj0002",
						"name": "Test Project B",
						"code": "test-project-b",
						"description": "",
						"bizID": "100002",
						"isOffline": true,
						"isBoundWorkspace": false
					}
				]
			}`

			var resp serializer.ListBCSAuthorizedProjectsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(2))
			Expect(resp.Data[0].ID).To(Equal("proj0001proj0001proj0001proj0001"))
			Expect(resp.Data[0].Name).To(Equal("Test Project A"))
			Expect(resp.Data[0].Code).To(Equal("test-project-a"))
			Expect(resp.Data[0].BizID).To(Equal("100001"))
			Expect(resp.Data[0].IsOffline).To(BeFalse())
			Expect(resp.Data[0].IsBoundWorkspace).To(BeTrue())

			Expect(resp.Data[1].IsOffline).To(BeTrue())
			Expect(resp.Data[1].IsBoundWorkspace).To(BeFalse())
		})

		It("should parse empty data list", func() {
			rawJSON := `{"data": []}`

			var resp serializer.ListBCSAuthorizedProjectsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeEmpty())
		})
	})

	Describe("GetBCSProjectOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": {
					"id": "proj0001proj0001proj0001proj0001",
					"name": "Test Project",
					"code": "test-project",
					"description": "Project description",
					"bizID": "100001",
					"isOffline": false,
					"isBoundWorkspace": true
				}
			}`

			var resp serializer.GetBCSProjectOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ID).To(Equal("proj0001proj0001proj0001proj0001"))
			Expect(resp.Data.Name).To(Equal("Test Project"))
			Expect(resp.Data.Code).To(Equal("test-project"))
		})

		It("should parse JSON with null data", func() {
			rawJSON := `{"data": null}`

			var resp serializer.GetBCSProjectOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeNil())
		})
	})

	Describe("ListClustersByProjectOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"id": "BCS-K8S-00001",
						"name": "Production Cluster",
						"type": "k8s",
						"environment": "prod",
						"isShared": false,
						"description": "Production environment cluster",
						"status": "RUNNING"
					},
					{
						"id": "BCS-K8S-00002",
						"name": "Test Cluster",
						"type": "k8s",
						"environment": "stag",
						"isShared": true,
						"description": "",
						"status": "RUNNING"
					}
				]
			}`

			var resp serializer.ListClustersByProjectOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(2))
			Expect(resp.Data[0].ID).To(Equal("BCS-K8S-00001"))
			Expect(resp.Data[0].Name).To(Equal("Production Cluster"))
			Expect(resp.Data[0].Type).To(Equal("k8s"))
			Expect(resp.Data[0].IsShared).To(BeFalse())
			Expect(resp.Data[0].Status).To(Equal("RUNNING"))

			Expect(resp.Data[1].IsShared).To(BeTrue())
			Expect(resp.Data[1].Environment).To(Equal("stag"))
		})
	})

	Describe("ListNamespacesByClusterOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{"name": "default", "status": "Active"},
					{"name": "kube-system", "status": "Active"},
					{"name": "bkms-prod", "status": "Active"}
				]
			}`

			var resp serializer.ListNamespacesByClusterOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(3))
			Expect(resp.Data[0].Name).To(Equal("default"))
			Expect(resp.Data[0].Status).To(Equal("Active"))
			Expect(resp.Data[2].Name).To(Equal("bkms-prod"))
		})

		It("should parse empty namespace list", func() {
			rawJSON := `{"data": []}`

			var resp serializer.ListNamespacesByClusterOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeEmpty())
		})
	})
})
