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

package build

import (
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Version", func() {
	Context("GenerateRecommendedSemverImageTag", func() {
		It("should return max patch+1 when multiple valid semver tags exist", func() {
			tags := []string{"v1.0.0", "v1.0.1", "v2.1.0", "latest", "master-202510151300"}
			Expect(GenerateRecommendedSemverImageTag(tags)).To(Equal("v2.1.1"))
		})

		It("should return default value when no valid semver tags exist", func() {
			tags := []string{"latest", "master-202510151300", "abc123"}
			Expect(GenerateRecommendedSemverImageTag(tags)).To(Equal("v1.0.0"))
		})

		It("should return default value for empty tag list", func() {
			Expect(GenerateRecommendedSemverImageTag([]string{})).To(Equal("v1.0.0"))
		})

		It("should return default value for nil tag list", func() {
			Expect(GenerateRecommendedSemverImageTag(nil)).To(Equal("v1.0.0"))
		})

		It("should handle tags with pre-release suffix", func() {
			tags := []string{"v1.0.0", "v1.0.1-beta.1", "v1.0.2-rc.1"}
			Expect(GenerateRecommendedSemverImageTag(tags)).To(Equal("v1.0.3"))
		})

		It("should handle single tag", func() {
			tags := []string{"v0.0.1"}
			Expect(GenerateRecommendedSemverImageTag(tags)).To(Equal("v0.0.2"))
		})

		It("should pick highest version across different major versions", func() {
			tags := []string{"v1.9.9", "v2.0.0", "v1.99.99"}
			Expect(GenerateRecommendedSemverImageTag(tags)).To(Equal("v2.0.1"))
		})
	})

	Context("GenerateRecommendedCustomImageTag", func() {
		var fixedTime time.Time

		BeforeEach(func() {
			fixedTime = time.Date(2025, 10, 15, 13, 0, 0, 0, time.UTC)
		})

		Context("when opts is nil (default behavior)", func() {
			It("should include branch name", func() {
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, nil)).To(Equal("master-202510151300"))
			})

			It("should return timestamp only when branch is empty", func() {
				Expect(GenerateRecommendedCustomImageTag("", fixedTime, nil)).To(Equal("202510151300"))
			})

			It("should replace slash in branch name with hyphen", func() {
				Expect(
					GenerateRecommendedCustomImageTag("feature/login", fixedTime, nil),
				).To(Equal("feature-login-202510151300"))
			})

			It("should replace multiple slashes in branch name with hyphens", func() {
				Expect(
					GenerateRecommendedCustomImageTag("feature/user/login", fixedTime, nil),
				).To(Equal("feature-user-login-202510151300"))
			})

			It("should truncate branch name to 128 characters", func() {
				branch := strings.Repeat("a", 200)
				expected := strings.Repeat("a", 128) + "-202510151300"
				Expect(GenerateRecommendedCustomImageTag(branch, fixedTime, nil)).To(Equal(expected))
			})
		})

		Context("when opts specifies custom tag fields", func() {
			It("should return only prefix when only prefix is set", func() {
				opts := &CustomTagOpts{Prefix: "release"}
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, opts)).To(Equal("release"))
			})

			It("should return only branch when only withRevision is true", func() {
				opts := &CustomTagOpts{WithRevision: true}
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, opts)).To(Equal("master"))
			})

			It("should return only timestamp when only withBuildTime is true", func() {
				opts := &CustomTagOpts{WithBuildTime: true}
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, opts)).To(Equal("202510151300"))
			})

			It("should return prefix-branch when prefix and withRevision are set", func() {
				opts := &CustomTagOpts{Prefix: "release", WithRevision: true}
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, opts)).To(Equal("release-master"))
			})

			It("should return prefix-timestamp when prefix and withBuildTime are set", func() {
				opts := &CustomTagOpts{Prefix: "release", WithBuildTime: true}
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, opts)).To(Equal("release-202510151300"))
			})

			It("should return branch-timestamp when withRevision and withBuildTime are set", func() {
				opts := &CustomTagOpts{WithRevision: true, WithBuildTime: true}
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, opts)).To(Equal("master-202510151300"))
			})

			It("should return prefix-branch-timestamp when all fields are set", func() {
				opts := &CustomTagOpts{Prefix: "release", WithRevision: true, WithBuildTime: true}
				Expect(
					GenerateRecommendedCustomImageTag("master", fixedTime, opts),
				).To(Equal("release-master-202510151300"))
			})

			It("should return empty string when no fields are enabled and prefix is empty", func() {
				opts := &CustomTagOpts{}
				Expect(GenerateRecommendedCustomImageTag("master", fixedTime, opts)).To(Equal(""))
			})

			It("should replace slash in branch name with hyphen when using opts", func() {
				opts := &CustomTagOpts{Prefix: "v1", WithRevision: true, WithBuildTime: true}
				Expect(
					GenerateRecommendedCustomImageTag("feature/login", fixedTime, opts),
				).To(Equal("v1-feature-login-202510151300"))
			})

			It("should skip branch part when withRevision is true but branch is empty", func() {
				opts := &CustomTagOpts{Prefix: "release", WithRevision: true, WithBuildTime: true}
				Expect(GenerateRecommendedCustomImageTag("", fixedTime, opts)).To(Equal("release-202510151300"))
			})
		})

		Context("when branch name contains invalid Docker tag characters", func() {
			It("should replace special symbols like ¥ with hyphen", func() {
				Expect(
					GenerateRecommendedCustomImageTag("feature¥test", fixedTime, nil),
				).To(Equal("feature-test-202510151300"))
			})

			It("should replace Chinese characters with hyphen", func() {
				Expect(
					GenerateRecommendedCustomImageTag("feature/测试分支", fixedTime, nil),
				).To(Equal("feature-202510151300"))
			})

			It("should replace multiple consecutive invalid characters with single hyphen", func() {
				Expect(
					GenerateRecommendedCustomImageTag("feat@#$%^&", fixedTime, nil),
				).To(Equal("feat-202510151300"))
			})

			It("should preserve valid characters (letters, digits, dot, hyphen, underscore)", func() {
				Expect(
					GenerateRecommendedCustomImageTag("release_v1.0-beta", fixedTime, nil),
				).To(Equal("release_v1.0-beta-202510151300"))
			})

			It("should handle branch name with only invalid characters", func() {
				opts := &CustomTagOpts{WithRevision: true, WithBuildTime: true}
				Expect(
					GenerateRecommendedCustomImageTag("¥@#", fixedTime, opts),
				).To(Equal("202510151300"))
			})
		})
	})
})
