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

package bkci

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("extractErrorMessage", func() {
	It("should extract message from 'message' field", func() {
		result := map[string]any{"message": "something went wrong"}
		Expect(extractErrorMessage(result)).To(Equal("something went wrong"))
	})

	It("should extract message from 'data.message' field when 'message' is empty", func() {
		result := map[string]any{
			"message": "",
			"data":    map[string]any{"message": "nested error"},
		}
		Expect(extractErrorMessage(result)).To(Equal("nested error"))
	})

	It("should extract message from 'error.message' field", func() {
		result := map[string]any{
			"error": map[string]any{"message": "error detail"},
		}
		Expect(extractErrorMessage(result)).To(Equal("error detail"))
	})

	It("should extract message from 'error' field when it's a string", func() {
		result := map[string]any{
			"error": "raw error string",
		}
		Expect(extractErrorMessage(result)).To(Equal("raw error string"))
	})

	It("should extract message from 'msg' field", func() {
		result := map[string]any{
			"msg": "short message",
		}
		Expect(extractErrorMessage(result)).To(Equal("short message"))
	})

	It("should extract message from 'data.msg' field", func() {
		result := map[string]any{
			"data": map[string]any{"msg": "data msg"},
		}
		Expect(extractErrorMessage(result)).To(Equal("data msg"))
	})

	It("should return fallback when all fields are empty", func() {
		result := map[string]any{}
		Expect(extractErrorMessage(result)).To(Equal(""))
	})

	It("should return fallback for nil result", func() {
		Expect(extractErrorMessage(nil)).To(Equal(""))
	})
})

var _ = Describe("newHTTPError", func() {
	It("should include opName, status code and message", func() {
		result := map[string]any{"message": "unauthorized"}
		err := newHTTPError("test_api", 401, result)
		Expect(err.Error()).To(ContainSubstring("test_api"))
		Expect(err.Error()).To(ContainSubstring("401"))
		Expect(err.Error()).To(ContainSubstring("unauthorized"))
	})

	It("should use fallback message when result is empty", func() {
		err := newHTTPError("test_api", 500, map[string]any{})
		Expect(err.Error()).To(ContainSubstring(""))
	})
})

var _ = Describe("newBusinessError", func() {
	It("should include opName, code, status and message", func() {
		result := map[string]any{"message": "pipeline not found"}
		err := newBusinessError("pipeline_get", 1, 2101001, result)
		Expect(err.Error()).To(ContainSubstring("pipeline_get"))
		Expect(err.Error()).To(ContainSubstring("code: 1"))
		Expect(err.Error()).To(ContainSubstring("status: 2101001"))
		Expect(err.Error()).To(ContainSubstring("pipeline not found"))
	})
})

var _ = Describe("isRepoRefPropertyType", func() {
	It("should return true only for supported repo ref property types", func() {
		Expect(isRepoRefPropertyType(RepoRefPropertyTypeGitRef)).To(BeTrue())
		Expect(isRepoRefPropertyType(RepoRefPropertyTypeSvnTag)).To(BeTrue())
		Expect(isRepoRefPropertyType(RepoRefPropertyTypeRepoRef)).To(BeTrue())
		Expect(isRepoRefPropertyType("enum")).To(BeFalse())
		Expect(isRepoRefPropertyType("")).To(BeFalse())
	})
})

var _ = Describe("parseRepoRefProperties", func() {
	It("should keep only git_ref svn_tag and repo_ref properties", func() {
		data := map[string]any{
			"properties": []any{
				map[string]any{
					"id":            "branch",
					"name":          "branch",
					"label":         "Code Branch",
					"type":          RepoRefPropertyTypeGitRef,
					"valueNotEmpty": true,
					"defaultValue":  "main",
					"value":         "release",
				},
				map[string]any{
					"id":    "tag",
					"name":  "tag",
					"label": "Code Tag",
					"type":  RepoRefPropertyTypeSvnTag,
				},
				map[string]any{
					"id":    "repo",
					"name":  "repo",
					"label": "Repository Ref",
					"type":  RepoRefPropertyTypeRepoRef,
				},
				map[string]any{
					"id":    "env",
					"name":  "env",
					"label": "Env",
					"type":  "enum",
				},
			},
		}

		properties, err := parseRepoRefProperties(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(properties).To(HaveLen(3))
		Expect(properties[0].ID).To(Equal("branch"))
		Expect(properties[0].Required).To(BeTrue())
		Expect(properties[0].DefaultValue).To(Equal("main"))
		Expect(properties[0].Value).To(Equal("release"))
		Expect(properties[1].Type).To(Equal(RepoRefPropertyTypeSvnTag))
		Expect(properties[2].Type).To(Equal(RepoRefPropertyTypeRepoRef))
	})

	It("should ignore properties with unsupported types", func() {
		data := map[string]any{
			"properties": []any{
				map[string]any{
					"id":    "env",
					"name":  "env",
					"label": "Env",
					"type":  "enum",
				},
				map[string]any{
					"id":    "branch",
					"name":  "branch",
					"label": "Code Branch",
					"type":  RepoRefPropertyTypeGitRef,
				},
			},
		}

		properties, err := parseRepoRefProperties(data)
		Expect(err).NotTo(HaveOccurred())
		Expect(properties).To(HaveLen(1))
		Expect(properties[0].ID).To(Equal("branch"))
		Expect(properties[0].Type).To(Equal(RepoRefPropertyTypeGitRef))
	})

	It("should return error when a property item is not a map", func() {
		data := map[string]any{
			"properties": []any{
				"not-a-map",
			},
		}

		properties, err := parseRepoRefProperties(data)
		Expect(err).To(HaveOccurred())
		Expect(properties).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("invalid repo ref property"))
	})
})

var _ = Describe("StubApiClient repo refs", func() {
	It("should return repo ref properties from startup info", func() {
		client := NewStub(auth.User{ID: "tester"})

		properties, err := client.ListPipelineRepoRefProperties(context.Background(), "demo", "pipe-1")
		Expect(err).NotTo(HaveOccurred())
		Expect(properties).NotTo(BeEmpty())
		Expect(properties[0].Type).To(Equal(RepoRefPropertyTypeGitRef))
	})

	It("should return repo ref options for a property", func() {
		client := NewStub(auth.User{ID: "tester"})

		options, err := client.ListPipelineRepoRefOptions(context.Background(), "demo", "pipe-1", "branch", "release")
		Expect(err).NotTo(HaveOccurred())
		Expect(options).NotTo(BeEmpty())
		Expect(options[0].Key).To(ContainSubstring("release"))
	})
})

var _ = Describe("parseBuildLog", func() {
	It("keeps nil data payload backward compatible as an empty build log", func() {
		result := map[string]any{
			"status": 0,
			"data":   nil,
		}

		buildLog, err := parseBuildLog(context.Background(), result)

		Expect(err).NotTo(HaveOccurred())
		Expect(buildLog).NotTo(BeNil())
		Expect(buildLog.Status).To(Equal(0))
		Expect(buildLog.Logs).To(BeNil())
	})

	It("keeps EMPTY log status as a non-error response", func() {
		result := map[string]any{
			"status": 0,
			"data": map[string]any{
				"status":   1,
				"message":  "no logs yet",
				"finished": false,
				"hasMore":  false,
				"logs":     []map[string]any{},
			},
		}

		buildLog, err := parseBuildLog(context.Background(), result)

		Expect(err).NotTo(HaveOccurred())
		Expect(buildLog.Status).To(Equal(1))
		Expect(buildLog.Message).To(Equal("no logs yet"))
		Expect(buildLog.Logs).To(BeEmpty())
	})

	It("keeps old responses without status field compatible with finished state checks", func() {
		result := map[string]any{
			"status": 0,
			"data": map[string]any{
				"finished": true,
				"hasMore":  false,
				"logs":     []map[string]any{},
			},
		}

		buildLog, err := parseBuildLog(context.Background(), result)

		Expect(err).NotTo(HaveOccurred())
		Expect(buildLog).NotTo(BeNil())
		Expect(buildLog.Status).To(Equal(0))
		Expect(buildLog.IsComplete()).To(BeTrue())
	})

	It("returns an error when BKCI reports the build logs are expired", func() {
		result := map[string]any{
			"status": 0,
			"data": map[string]any{
				"status":  2,
				"message": "log expired",
			},
		}

		buildLog, err := parseBuildLog(context.Background(), result)

		Expect(buildLog).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("expired"))
		Expect(err.Error()).To(ContainSubstring("log expired"))
	})

	It("returns an error when BKCI reports the build logs are cleaned", func() {
		result := map[string]any{
			"status": 0,
			"data": map[string]any{
				"status":  3,
				"message": "log cleaned",
			},
		}

		buildLog, err := parseBuildLog(context.Background(), result)

		Expect(buildLog).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("cleaned"))
		Expect(err.Error()).To(ContainSubstring("log cleaned"))
	})

	It("returns an error when BKCI reports log query failure", func() {
		result := map[string]any{
			"status": 0,
			"data": map[string]any{
				"status":  999,
				"message": "query failed",
			},
		}

		buildLog, err := parseBuildLog(context.Background(), result)

		Expect(buildLog).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("query failed"))
	})

	It("returns the base error unchanged when BKCI status message is empty", func() {
		result := map[string]any{
			"status": 0,
			"data": map[string]any{
				"status": 2,
			},
		}

		buildLog, err := parseBuildLog(context.Background(), result)

		Expect(buildLog).To(BeNil())
		Expect(err).To(MatchError(BuildLogExpired))
		Expect(err.Error()).To(Equal(BuildLogExpired.Error()))
	})
})
