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

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg/serializer"
)

var _ = Describe("App config file serializers", func() {
	DescribeTable(
		"validates create input",
		func(input serializer.CreateAppConfigFileInput, expectedErrSubstrings []string) {
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
		Entry("valid input", serializer.CreateAppConfigFileInput{
			Name:              "default-config",
			Type:              "normal",
			ContentSourceType: "local",
			FileFormat:        "yaml",
		}, nil),
		Entry("invalid name", serializer.CreateAppConfigFileInput{
			Name:              "bad name",
			Type:              "normal",
			ContentSourceType: "local",
			FileFormat:        "yaml",
		}, []string{
			"CreateAppConfigFileInput.Name",
			"failed on the 'app_config_file_name' tag",
		}),
	)

	It("accepts numeric int64 currentVersion", func() {
		var input serializer.UpdateAppConfigFileInput
		err := json.Unmarshal([]byte(`{"name":"demo","currentVersion":42}`), &input)

		Expect(err).NotTo(HaveOccurred())
		Expect(input.CurrentVersion).NotTo(BeNil())
		Expect(*input.CurrentVersion).To(Equal(int64(42)))
	})

	It("marshals file output currentVersion as JSON number", func() {
		payload, err := json.Marshal(serializer.AppConfigFileOutputObj{
			ID:             "abc",
			Name:           "demo",
			Type:           "normal",
			FileFormat:     "yaml",
			CurrentVersion: 12,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"id": "abc",
			"name": "demo",
			"type": "normal",
			"contentSourceType": "",
			"envName": "",
			"fileFormat": "yaml",
			"currentVersion": 12,
			"updater": "",
			"updatedAt": ""
		}`))
	})

	It("marshals version list numeric fields as JSON numbers", func() {
		baseVersion := int64(2)
		rollbackFromVersion := int64(5)
		payload, err := json.Marshal(serializer.ListAppConfigFileVersionsOutput{
			Data: &serializer.PaginatedAppConfigFileVersionOutputObjs{
				Count: 3,
				Results: []*serializer.AppConfigFileVersionOutputObj{
					{
						ID:                  "v1",
						Version:             7,
						BaseVersion:         &baseVersion,
						RollbackFromVersion: &rollbackFromVersion,
					},
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"data": {
				"count": 3,
				"results": [
					{
						"id": "v1",
						"appConfigFileID": "",
						"appID": "",
						"envName": "",
						"name": "",
						"version": 7,
						"description": "",
						"type": "",
						"contentSourceType": "",
						"fileFormat": "",
						"baseVersion": 2,
						"operationType": "",
						"rollbackFromVersion": 5,
						"creator": "",
						"createdAt": "",
						"isDeleted": false
					}
				]
			}
		}`))
	})
})
