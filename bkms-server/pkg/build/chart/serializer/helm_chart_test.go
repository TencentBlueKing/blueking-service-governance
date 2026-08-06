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

	helmbuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/semver"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/serializer"
	helmrepo "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/source"
)

var _ = Describe("Helm Chart serializer validation", func() {
	DescribeTable(
		"validates build input bump type",
		func(input serializer.CreateHelmChartBuildInput, expectedErrSubstrings []string) {
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
		Entry("valid patch bump", serializer.CreateHelmChartBuildInput{
			BumpType: "patch",
			Branch:   "master",
		}, nil),
		Entry("missing bump type", serializer.CreateHelmChartBuildInput{
			Branch: "master",
		}, []string{
			"CreateHelmChartBuildInput.BumpType",
			"failed on the 'required' tag",
		}),
		Entry("unsupported bump type", serializer.CreateHelmChartBuildInput{
			BumpType: "invalid",
			Branch:   "master",
		}, []string{
			"CreateHelmChartBuildInput.BumpType",
			"failed on the 'oneof' tag",
		}),
	)

	DescribeTable(
		"validates optional semver preview bump type",
		func(input serializer.GetHelmChartSemverQueryInput, expectedErrSubstrings []string) {
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
		Entry("empty bump type", serializer.GetHelmChartSemverQueryInput{}, nil),
		Entry("valid minor bump", serializer.GetHelmChartSemverQueryInput{BumpType: "minor"}, nil),
		Entry("unsupported bump type", serializer.GetHelmChartSemverQueryInput{BumpType: "invalid"}, []string{
			"GetHelmChartSemverQueryInput.BumpType",
			"failed on the 'oneof' tag",
		}),
	)

	DescribeTable(
		"validates list pagination",
		func(input serializer.ListQueryInput, expectedErrSubstrings []string) {
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
		Entry("valid pagination", serializer.ListQueryInput{Page: 1, PageSize: 20}, nil),
		Entry("missing page", serializer.ListQueryInput{PageSize: 20}, []string{
			"ListQueryInput.Page",
			"failed on the 'required' tag",
		}),
		Entry("unsupported page size", serializer.ListQueryInput{Page: 1, PageSize: 7}, []string{
			"ListQueryInput.PageSize",
			"failed on the 'oneof' tag",
		}),
	)
})

var _ = Describe("Helm Chart serializer output", func() {
	It("marshals semver next as null when it is not provided", func() {
		payload, err := json.Marshal(serializer.GetHelmChartSemverOutput{
			Data: &serializer.GetHelmChartSemverOutputObj{
				Latest: new(serializer.SemverOutputObj).FromCounter(&semver.Counter{
					Major: 1,
					Minor: 2,
					Patch: 3,
				}),
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"data": {
				"latest": {
					"major": "1",
					"minor": "2",
					"patch": "3",
					"version": "1.2.3"
				},
				"next": null
			}
		}`))
	})

	It("marshals paginated counts and int64 fields as JSON strings", func() {
		startedAt := time.Date(2026, 5, 29, 10, 20, 30, 0, time.UTC)
		payload, err := json.Marshal(serializer.ListHelmChartBuildRecordsOutput{
			Data: &serializer.PaginatedHelmChartBuildRecordOutputObjs{
				Count: 12,
				Results: []*serializer.HelmChartBuildRecordOutputObj{
					new(serializer.HelmChartBuildRecordOutputObj).FromBuildRecord(helmbuild.Record{
						Num:          3,
						PipelineID:   "pipeline-1",
						BuildID:      "build-1",
						ChartVersion: "1.0.0",
						Status:       helmbuild.StatusRunning,
						Operator:     "tester",
						Params:       map[string]string{"branch": "master"},
						Extras:       map[string]string{"commit": "abc"},
						StartedAt:    startedAt,
					}),
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"data": {
				"count": "12",
				"results": [{
					"num": "3",
					"pipelineID": "pipeline-1",
					"buildID": "build-1",
					"chartVersion": "1.0.0",
					"status": "running",
					"operator": "tester",
					"params": {"branch": "master"},
					"extras": {"commit": "abc"},
					"startedAt": "2026-05-29T10:20:30Z"
				}]
			}
		}`))
	})

	It("converts recursive Helm Chart file nodes", func() {
		payload, err := json.Marshal(serializer.GetHelmChartFilesOutput{
			Data: &serializer.GetHelmChartFilesOutputObj{
				ChartName:    "demo",
				ChartVersion: "1.0.0",
				Root: new(serializer.HelmChartFileNode).FromRepoFileNode(&helmrepo.FileNode{
					Name:  "demo",
					Path:  "",
					IsDir: true,
					Children: []*helmrepo.FileNode{
						{
							Name:    "values.yaml",
							Path:    "values.yaml",
							Size:    11,
							Content: []byte("replica: 1\n"),
						},
						{
							Name:     "logo.png",
							Path:     "logo.png",
							Size:     5,
							IsBinary: true,
							Content:  []byte{0, 1, 2},
						},
					},
				}),
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(payload).To(MatchJSON(`{
			"data": {
				"chartName": "demo",
				"chartVersion": "1.0.0",
				"root": {
					"name": "demo",
					"path": "",
					"isDir": true,
					"size": "0",
					"isBinary": false,
					"content": "",
					"children": [
						{
							"name": "values.yaml",
							"path": "values.yaml",
							"isDir": false,
							"size": "11",
							"isBinary": false,
							"content": "replica: 1\n",
							"children": []
						},
						{
							"name": "logo.png",
							"path": "logo.png",
							"isDir": false,
							"size": "5",
							"isBinary": true,
							"content": "",
							"children": []
						}
					]
				}
			}
		}`))
	})
})
