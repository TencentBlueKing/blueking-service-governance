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

var _ = Describe("parseRepositoryRefs", func() {
	It("should parse name path sha and linkUrl fields", func() {
		refs, err := parseRepositoryRefs([]any{
			map[string]any{
				"name":    "main",
				"path":    "refs/heads/main",
				"sha":     "abc123",
				"linkUrl": "https://git.example.com/repo/commits/abc123",
			},
			map[string]any{
				"name":    "v1.0.0",
				"path":    "refs/tags/v1.0.0",
				"sha":     "def456",
				"linkUrl": "https://git.example.com/repo/tags/v1.0.0",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(refs).To(Equal([]RepositoryRef{
			{
				Name:    "main",
				Path:    "refs/heads/main",
				SHA:     "abc123",
				LinkURL: "https://git.example.com/repo/commits/abc123",
			},
			{
				Name:    "v1.0.0",
				Path:    "refs/tags/v1.0.0",
				SHA:     "def456",
				LinkURL: "https://git.example.com/repo/tags/v1.0.0",
			},
		}))
	})

	It("should return error when an item is not a map", func() {
		refs, err := parseRepositoryRefs([]any{"not-a-map"})
		Expect(err).To(HaveOccurred())
		Expect(refs).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("invalid repository ref"))
	})
})

var _ = Describe("StubApiClient repository refs", func() {
	It("should return branch refs", func() {
		client := NewStub(auth.User{ID: "tester"})

		refs, err := client.ListRepositoryBranches(
			context.Background(), "demo", "repo-1", "NAME", "release", 1, 20,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(refs).NotTo(BeEmpty())
		Expect(refs[0].Path).To(HavePrefix("refs/heads/"))
	})

	It("should return tag refs", func() {
		client := NewStub(auth.User{ID: "tester"})

		refs, err := client.ListRepositoryTags(context.Background(), "demo", "repo-1", "ID", "v1", 1, 20)
		Expect(err).NotTo(HaveOccurred())
		Expect(refs).NotTo(BeEmpty())
		Expect(refs[0].Path).To(HavePrefix("refs/tags/"))
	})

	It("should return stable unique repository ids for different repositories", func() {
		client := NewStub(auth.User{ID: "tester"})

		repoID1, err := client.CreateRepository(
			context.Background(), "demo", "https://example.com/team/repo-1.git", "repo-1",
		)
		Expect(err).NotTo(HaveOccurred())

		repoID2, err := client.CreateRepository(
			context.Background(), "demo", "https://example.com/team/repo-2.git", "repo-2",
		)
		Expect(err).NotTo(HaveOccurred())

		repoID1Again, err := client.CreateRepository(
			context.Background(), "demo", "https://example.com/team/repo-1.git", "repo-1",
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(repoID1).NotTo(Equal(repoID2))
		Expect(repoID1Again).To(Equal(repoID1))

		repo1, err := client.GetRepository(context.Background(), "demo", repoID1)
		Expect(err).NotTo(HaveOccurred())
		Expect(repo1.Alias).To(Equal("repo-1"))
		Expect(repo1.Url).To(Equal("https://example.com/team/repo-1.git"))
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
