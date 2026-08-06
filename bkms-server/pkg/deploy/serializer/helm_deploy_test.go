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
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	helmdeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
)

var _ = Describe("Helm deploy serializer validation", func() {
	DescribeTable(
		"validates deploy URI input",
		func(input serializer.HelmDeployRecordURIInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}

			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid record URI", serializer.HelmDeployRecordURIInput{
			AppID:    "demo-app",
			EnvName:  "test",
			DeployID: "deploy-id",
		}, nil),
		Entry("missing app id", serializer.HelmDeployRecordURIInput{
			EnvName:  "test",
			DeployID: "deploy-id",
		}, []string{
			"HelmDeployRecordURIInput.AppID",
			"failed on the 'required' tag",
		}),
		Entry("missing deploy id", serializer.HelmDeployRecordURIInput{
			AppID:   "demo-app",
			EnvName: "test",
		}, []string{
			"HelmDeployRecordURIInput.DeployID",
			"failed on the 'required' tag",
		}),
	)

	DescribeTable(
		"validates list pagination",
		func(input serializer.ListHelmDeployRecordsQueryInput, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}

			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid first page and page size 1", serializer.ListHelmDeployRecordsQueryInput{
			Page:     1,
			PageSize: 1,
		}, nil),
		Entry("valid optional lane and keyword", serializer.ListHelmDeployRecordsQueryInput{
			TrafficLaneName: "lane-a",
			Keyword:         "operator",
			Page:            2,
			PageSize:        100,
		}, nil),
		Entry("missing page", serializer.ListHelmDeployRecordsQueryInput{
			PageSize: 20,
		}, []string{
			"ListHelmDeployRecordsQueryInput.Page",
			"failed on the 'required' tag",
		}),
		Entry("unsupported page size", serializer.ListHelmDeployRecordsQueryInput{
			Page:     1,
			PageSize: 7,
		}, []string{
			"ListHelmDeployRecordsQueryInput.PageSize",
			"failed on the 'oneof' tag",
		}),
	)

	DescribeTable(
		"validates create and preview inputs",
		func(input any, expectedErrSubstrings []string) {
			err := binding.Validator.ValidateStruct(input)
			if len(expectedErrSubstrings) == 0 {
				Expect(err).NotTo(HaveOccurred())
				return
			}

			Expect(err).To(HaveOccurred())
			for _, expected := range expectedErrSubstrings {
				Expect(err.Error()).To(ContainSubstring(expected))
			}
		},
		Entry("valid create body with empty lane", serializer.CreateHelmDeployInput{
			ImageTag:     "v1",
			ChartVersion: "1.0.0",
			ValuesFileID: "values-id",
		}, nil),
		Entry("missing image tag", serializer.CreateHelmDeployInput{
			ChartVersion: "1.0.0",
			ValuesFileID: "values-id",
		}, []string{
			"CreateHelmDeployInput.ImageTag",
			"failed on the 'required' tag",
		}),
		Entry("valid preview query", serializer.PreviewHelmDeployQueryInput{
			ImageTag:        "v1",
			ChartVersion:    "1.0.0",
			ValuesFileID:    "values-id",
			TrafficLaneName: "lane-a",
		}, nil),
		Entry("missing values file id", serializer.PreviewHelmDeployQueryInput{
			ImageTag:     "v1",
			ChartVersion: "1.0.0",
		}, []string{
			"PreviewHelmDeployQueryInput.ValuesFileID",
			"failed on the 'required' tag",
		}),
		Entry("valid rollback empty body", serializer.RollbackHelmDeployInput{}, nil),
	)
})

var _ = Describe("Helm deploy serializer output", func() {
	It("converts Helm deploy records to output objects", func() {
		createdAt := time.Date(2026, 5, 29, 10, 20, 30, 0, time.UTC)
		updatedAt := time.Date(2026, 5, 29, 10, 30, 30, 0, time.UTC)
		recordID := bson.NewObjectID()

		output := new(serializer.HelmDeployRecordOutputObj).FromModel(helmdeploy.Record{
			ID:           recordID,
			EnvName:      "test",
			ProjectCode:  "project",
			ClusterID:    "cluster",
			Namespace:    "namespace",
			ReleaseName:  "release",
			ChartName:    "chart",
			ChartVersion: "1.0.0",
			ValuesFileID: "values-id",
			ImageTag:     "image-tag",
			Revision:     "3",
			Status:       helm.StatusDeployed,
			Message:      "deployed",
			Operator:     "operator",
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})

		Expect(output.ID).To(Equal(recordID.Hex()))
		Expect(output.Status).To(Equal("deployed"))
		Expect(output.Values).To(BeEmpty())
		Expect(output.CreatedAt).To(Equal(createdAt))
		Expect(output.UpdatedAt).To(Equal(updatedAt))
	})

	It("marshals paginated count as a JSON string", func() {
		createdAt := time.Date(2026, 5, 29, 10, 20, 30, 0, time.UTC)
		payload, err := json.Marshal(serializer.ListHelmDeployRecordsOutput{
			Data: &serializer.PaginatedHelmDeployRecordOutputObjs{
				Count: 12,
				Results: []*serializer.HelmDeployRecordOutputObj{
					{
						ID:           "record-id",
						EnvName:      "test",
						ProjectCode:  "project",
						ClusterID:    "cluster",
						Namespace:    "namespace",
						ReleaseName:  "release",
						ChartName:    "chart",
						ChartVersion: "1.0.0",
						ValuesFileID: "values-id",
						ImageTag:     "image-tag",
						Revision:     "3",
						Status:       "deployed",
						Message:      "ok",
						Operator:     "operator",
						Values:       "replica: 1",
						CreatedAt:    createdAt,
						UpdatedAt:    createdAt,
					},
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"data": {
				"count": "12",
				"results": [{
					"id": "record-id",
					"envName": "test",
					"projectCode": "project",
					"clusterID": "cluster",
					"namespace": "namespace",
					"releaseName": "release",
					"chartName": "chart",
					"chartVersion": "1.0.0",
					"valuesFileID": "values-id",
					"imageTag": "image-tag",
					"revision": "3",
					"status": "deployed",
					"message": "ok",
					"operator": "operator",
					"values": "replica: 1",
					"createdAt": "2026-05-29T10:20:30Z",
					"updatedAt": "2026-05-29T10:20:30Z"
				}]
			}
		}`))
	})

	It("marshals empty output as an empty JSON object", func() {
		payload, err := json.Marshal(serializer.EmptyOutput{})

		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{}`))
	})
})
