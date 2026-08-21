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
	"strings"
	"time"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

var _ = Describe("Image serializers", func() {
	DescribeTable(
		"AppImageTagURIInput validation",
		func(input serializer.AppImageTagURIInput, expectedErrSubstrings []string) {
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
		Entry("valid app and tag path", serializer.AppImageTagURIInput{
			AppID: "app-valid-1",
			Tag:   "v1.0.0",
		}, nil),
		Entry("missing tag", serializer.AppImageTagURIInput{
			AppID: "app-valid-1",
		}, []string{
			"AppImageTagURIInput.Tag",
			"failed on the 'required' tag",
		}),
		Entry("tag too long", serializer.AppImageTagURIInput{
			AppID: "app-valid-1",
			Tag:   strings.Repeat("a", 129),
		}, []string{
			"AppImageTagURIInput.Tag",
			"failed on the 'max' tag",
		}),
		Entry("invalid app id", serializer.AppImageTagURIInput{
			AppID: "app/test",
			Tag:   "v1.0.0",
		}, []string{
			"AppImageTagURIInput.AppID",
			"failed on the 'uri_slug' tag",
		}),
	)

	It("marshals image count and size as strings", func() {
		builtAt := time.Date(2026, 5, 27, 15, 0, 0, 0, time.UTC)
		payload := serializer.ListAppImagesOutput{
			Data: &serializer.PaginatedAppImagesOutputObjs{
				Count: 1,
				Results: []*serializer.AppImageOutputObj{
					{
						Repository: "repo/test",
						Tag:        "v1.0.0",
						Size:       1024,
						BuiltAt:    &builtAt,
					},
				},
				SnapshotStatus: &serializer.SnapshotStatusInfoOutputObj{
					RefreshStatus: string(snapshot.RefreshStatusIdle),
				},
			},
		}

		body, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(ContainSubstring(`"count":"1"`))
		Expect(string(body)).To(ContainSubstring(`"size":"1024"`))
	})

	It("marshals refresh counters as strings", func() {
		payload := serializer.RefreshAppImagesOutput{
			Data: &serializer.RefreshResultInfoOutputObj{
				Status:        "success",
				Message:       "done",
				AddedTagCnt:   2,
				RemovedTagCnt: 1,
			},
		}

		body, err := json.Marshal(payload)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(ContainSubstring(`"addedTagCnt":"2"`))
		Expect(string(body)).To(ContainSubstring(`"removedTagCnt":"1"`))
	})

	It("returns idle snapshot status when model is nil", func() {
		output := new(serializer.SnapshotStatusInfoOutputObj).FromModel(nil)
		Expect(output.RefreshStatus).To(Equal(string(snapshot.RefreshStatusIdle)))
	})

	It("validates deployable image tag pagination input", func() {
		input := serializer.ListDeployableImageTagsQueryInput{
			Keyword:  "v1",
			Page:     1,
			PageSize: 20,
		}

		err := binding.Validator.ValidateStruct(input)
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects invalid deployable image tag page size", func() {
		input := serializer.ListDeployableImageTagsQueryInput{
			Page:     1,
			PageSize: 7,
		}

		err := binding.Validator.ValidateStruct(input)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ListDeployableImageTagsQueryInput.PageSize"))
		Expect(err.Error()).To(ContainSubstring(strings.TrimSpace("failed on the 'oneof' tag")))
	})

	DescribeTable(
		"ListCustomRuntimeImageTagsQueryInput name validation",
		func(name string, expectValid bool) {
			input := serializer.ListCustomRuntimeImageTagsQueryInput{
				Name:     name,
				Page:     1,
				PageSize: 10,
			}
			err := binding.Validator.ValidateStruct(input)
			if expectValid {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ListCustomRuntimeImageTagsQueryInput.Name"))
			Expect(err.Error()).To(ContainSubstring("failed on the 'custom_image_repo' tag"))
		},
		Entry("accepts registry host and path", "docker.bkrepo.example.com/demo/repo/my-golang", true),
		Entry("rejects short docker hub name", "nginx", false),
		Entry("rejects name without registry host", "library/nginx", false),
		Entry("rejects tagged name", "registry.example.com/team/runtime:latest", false),
	)

	DescribeTable(
		"RefreshCustomRuntimeImageTagsInput name validation",
		func(name string, expectValid bool) {
			input := serializer.RefreshCustomRuntimeImageTagsInput{Name: name}
			err := binding.Validator.ValidateStruct(input)
			if expectValid {
				Expect(err).NotTo(HaveOccurred())
				return
			}
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("RefreshCustomRuntimeImageTagsInput.Name"))
			Expect(err.Error()).To(ContainSubstring("failed on the 'custom_image_repo' tag"))
		},
		Entry("accepts registry host and path", "registry.example.com:5000/team/runtime/base", true),
		Entry("rejects short docker hub name", "nginx", false),
	)
})
