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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
)

var _ = Describe("BkCI Serializer", func() {
	Describe("ListBkCIOAuthGitProjectsOutput", func() {
		It("should parse raw JSON into struct correctly", func() {
			rawJSON := `{
				"data": [
					{
						"id": "git-001",
						"name": "bkms-server",
						"alias": "BKMS Server",
						"url": "https://git.example.com/bkms/bkms-server.git"
					},
					{
						"id": "git-002",
						"name": "bkms-ui",
						"alias": "BKMS Frontend",
						"url": "https://git.example.com/bkms/bkms-ui.git"
					}
				]
			}`

			var resp serializer.ListBkCIOAuthGitProjectsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(2))
			Expect(resp.Data[0].ID).To(Equal("git-001"))
			Expect(resp.Data[0].Name).To(Equal("bkms-server"))
			Expect(resp.Data[0].Alias).To(Equal("BKMS Server"))
			Expect(resp.Data[0].Url).To(Equal("https://git.example.com/bkms/bkms-server.git"))
		})

		It("should parse empty data list", func() {
			rawJSON := `{"data": []}`

			var resp serializer.ListBkCIOAuthGitProjectsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(BeEmpty())
		})
	})

	Describe("ListBkCIPipelinesOutput", func() {
		It("should parse raw JSON with paginated pipelines", func() {
			rawJSON := `{
				"data": {
					"count": "5",
					"results": [
						{
							"id": "p-001",
							"name": "Build Pipeline",
							"description": "For building images",
							"version": "3",
							"creator": "admin",
							"updater": "admin",
							"createdAt": "2026-01-01T00:00:00Z",
							"updatedAt": "2026-06-20T10:00:00Z"
						},
						{
							"id": "p-002",
							"name": "Deploy Pipeline",
							"description": "For deploying services",
							"version": "1",
							"creator": "dev-user",
							"updater": "dev-user",
							"createdAt": "2026-03-15T08:00:00Z",
							"updatedAt": "2026-06-18T14:30:00Z"
						}
					]
				}
			}`

			var resp serializer.ListBkCIPipelinesOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.Count).To(Equal(int64(5)))
			Expect(resp.Data.Results).To(HaveLen(2))
			Expect(resp.Data.Results[0].ID).To(Equal("p-001"))
			Expect(resp.Data.Results[0].Name).To(Equal("Build Pipeline"))
			Expect(resp.Data.Results[0].Version).To(Equal(int64(3)))
			Expect(resp.Data.Results[0].Creator).To(Equal("admin"))
			Expect(resp.Data.Results[1].ID).To(Equal("p-002"))
		})

		It("should parse JSON with empty results", func() {
			rawJSON := `{"data": {"count": "0", "results": []}}`

			var resp serializer.ListBkCIPipelinesOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data.Count).To(Equal(int64(0)))
			Expect(resp.Data.Results).To(BeEmpty())
		})
	})

	Describe("GetBkCIPipelineOutput", func() {
		It("should parse raw JSON with pipeline detail and variables", func() {
			rawJSON := `{
				"data": {
					"id": "p-001",
					"name": "Build Pipeline",
					"description": "For building images",
					"version": "2",
					"creator": "admin",
					"updater": "admin",
					"createdAt": "2026-01-01T00:00:00Z",
					"updatedAt": "2026-06-20T10:00:00Z",
					"variables": [
						{
							"id": "var-001",
							"name": "IMAGE_TAG",
							"description": "Image tag",
							"required": true,
							"readOnly": false,
							"constant": false,
							"defaultValue": "latest",
							"type": "STRING",
							"options": []
						},
						{
							"id": "var-002",
							"name": "ENV",
							"description": "Deploy environment",
							"required": true,
							"readOnly": false,
							"constant": false,
							"defaultValue": "stag",
							"type": "ENUM",
							"options": [
								{"key": "stag", "value": "Staging"},
								{"key": "prod", "value": "Production"}
							]
						}
					]
				}
			}`

			var resp serializer.GetBkCIPipelineOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).NotTo(BeNil())
			Expect(resp.Data.ID).To(Equal("p-001"))
			Expect(resp.Data.Name).To(Equal("Build Pipeline"))
			Expect(resp.Data.Version).To(Equal(int64(2)))

			// 验证 variables
			Expect(resp.Data.Variables).To(HaveLen(2))
			Expect(resp.Data.Variables[0].ID).To(Equal("var-001"))
			Expect(resp.Data.Variables[0].Name).To(Equal("IMAGE_TAG"))
			Expect(resp.Data.Variables[0].Required).To(BeTrue())
			Expect(resp.Data.Variables[0].DefaultValue).To(Equal("latest"))
			Expect(resp.Data.Variables[0].Type).To(Equal("STRING"))
			Expect(resp.Data.Variables[0].Options).To(BeEmpty())

			Expect(resp.Data.Variables[1].ID).To(Equal("var-002"))
			Expect(resp.Data.Variables[1].Type).To(Equal("ENUM"))
			Expect(resp.Data.Variables[1].Options).To(HaveLen(2))
			Expect(resp.Data.Variables[1].Options[0].Key).To(Equal("stag"))
			Expect(resp.Data.Variables[1].Options[0].Value).To(Equal("Staging"))
		})
	})

	Describe("GetBkCIOAuthUrlOutput", func() {
		It("should parse raw JSON with oauth url", func() {
			rawJSON := `{"data": "https://oauth.example.com/authorize?client_id=xxx&redirect_uri=yyy"}`

			var resp serializer.GetBkCIOAuthUrlOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(Equal("https://oauth.example.com/authorize?client_id=xxx&redirect_uri=yyy"))
		})
	})

	Describe("GetBkCIPipelineVariablesOutput", func() {
		It("should parse raw JSON with variable list", func() {
			rawJSON := `{
				"data": [
					{
						"id": "var-001",
						"name": "BRANCH",
						"description": "Branch name",
						"required": false,
						"readOnly": false,
						"constant": false,
						"defaultValue": "main",
						"type": "STRING",
						"options": []
					}
				]
			}`

			var resp serializer.GetBkCIPipelineVariablesOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Data).To(HaveLen(1))
			Expect(resp.Data[0].Name).To(Equal("BRANCH"))
			Expect(resp.Data[0].DefaultValue).To(Equal("main"))
		})
	})

	Describe("ListBkCIPipelineRepoRefsOutput", func() {
		It("should parse repo ref properties", func() {
			rawJSON := `{
				"data": [
					{
						"id": "branch",
						"name": "branch",
						"label": "Code Branch",
						"type": "git_ref",
						"required": true,
						"readOnly": false,
						"defaultValue": "main",
						"value": "release"
					}
				]
			}`

			var resp serializer.ListBkCIPipelineRepoRefsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(HaveLen(1))
			Expect(resp.Data[0].Type).To(Equal("git_ref"))
			Expect(resp.Data[0].Label).To(Equal("Code Branch"))
			Expect(resp.Data[0].Value).To(Equal("release"))
		})

		It("should keep constant field when converting from model", func() {
			output := new(serializer.BkCIPipelineRepoRefOutput).FromModel(bkci.RepoRefProperty{
				ID:           "branch",
				Name:         "branch",
				Label:        "Code Branch",
				Type:         bkci.RepoRefPropertyTypeGitRef,
				Required:     true,
				ReadOnly:     false,
				Constant:     true,
				DefaultValue: "main",
				Value:        "release",
			})

			Expect(output).NotTo(BeNil())
			Expect(output.Constant).To(BeTrue())
		})
	})

	Describe("ListBkCIPipelineRepoRefOptionsOutput", func() {
		It("should parse repo ref options", func() {
			rawJSON := `{
				"data": [
					{"key": "refs/heads/release-1.0", "value": "release-1.0"},
					{"key": "refs/tags/v1.0.0", "value": "v1.0.0"}
				]
			}`

			var resp serializer.ListBkCIPipelineRepoRefOptionsOutput
			err := json.Unmarshal([]byte(rawJSON), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Data).To(HaveLen(2))
			Expect(resp.Data[0].Key).To(Equal("refs/heads/release-1.0"))
			Expect(resp.Data[1].Value).To(Equal("v1.0.0"))
		})
	})
})
