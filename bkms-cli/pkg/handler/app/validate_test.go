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

// Package app 提供应用创建相关的处理逻辑
package app

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// validTrpcSpec 返回一个合法的 trpc 类型 AppCreateSpec
func validTrpcSpec() *AppCreateSpec {
	return &AppCreateSpec{
		Name: "my-app",
		Type: "trpc",
		BuildConfig: &BuildConfigSpec{
			SourceType: "imageRegistry",
			ImageBuildConfig: &ImageBuildConfigSpec{
				Name: "mirrors.tencent.com/myns/my-app",
			},
		},
		AppModelSpec: &AppModelSpecSpec{
			TrpcSpec: &TrpcSpecSpec{
				Language: "go",
				FileName: "trpc_go.yaml",
				FilePath: "/app/conf",
			},
		},
	}
}

var _ = Describe("AppCreateSpec Validate", func() {
	// ==================== name 校验（app_name 正则） ====================
	Describe("name field", func() {
		It("should return error when name is empty", func() {
			spec := validTrpcSpec()
			spec.Name = ""
			Expect(spec.Validate()).To(MatchError(ContainSubstring("'name' is required")))
		})

		DescribeTable("name format validation (app_name regex)",
			func(appName string, expectErr bool) {
				spec := validTrpcSpec()
				spec.Name = appName
				err := spec.Validate()
				if expectErr {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			},
			Entry("valid - single char", "a", false),
			Entry("valid - with digits and hyphens", "my-app-123", false),
			Entry("valid - max length 63", strings.Repeat("a", 63), false),
			Entry("invalid - starts with uppercase", "MyApp", true),
			Entry("invalid - starts with digit", "1app", true),
			Entry("invalid - ends with hyphen", "my-app-", true),
			Entry("invalid - contains underscore", "my_app", true),
			Entry("invalid - exceeds 63 chars", strings.Repeat("a", 64), true),
			Entry("invalid - contains uppercase", "myApp", true),
			Entry("invalid - starts with hyphen", "-my-app", true),
		)
	})

	// ==================== type 校验（oneof 枚举） ====================
	Describe("type field", func() {
		It("should return error when type is empty", func() {
			spec := validTrpcSpec()
			spec.Type = ""
			Expect(spec.Validate()).To(MatchError(ContainSubstring("'type' is required")))
		})

		It("should return error when type is invalid", func() {
			spec := validTrpcSpec()
			spec.Type = "unknown"
			Expect(spec.Validate()).To(MatchError(ContainSubstring("'type' is invalid")))
		})
	})

	// ==================== buildConfig.sourceType 校验（required + oneof） ====================
	Describe("buildConfig field", func() {
		It("should return error when buildConfig is nil", func() {
			spec := validTrpcSpec()
			spec.BuildConfig = nil
			Expect(spec.Validate()).To(MatchError(ContainSubstring("'buildConfig' is required")))
		})

		It("should return error when buildConfig.sourceType is empty", func() {
			spec := validTrpcSpec()
			spec.BuildConfig.SourceType = ""
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is required")))
		})

		It("should return error when buildConfig.sourceType is invalid", func() {
			spec := validTrpcSpec()
			spec.BuildConfig.SourceType = "invalidSource"
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is invalid")))
		})
	})

	// ==================== appModelSpec.trpcSpec.language 校验（required + oneof） ====================
	Describe("appModelSpec.trpcSpec.language field", func() {
		It("should return error when trpcSpec.language is empty", func() {
			spec := validTrpcSpec()
			spec.AppModelSpec.TrpcSpec.Language = ""
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is required")))
		})

		It("should return error when trpcSpec.language is invalid", func() {
			spec := validTrpcSpec()
			spec.AppModelSpec.TrpcSpec.Language = "java"
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is invalid")))
		})
	})

	// ==================== helmSpec.helmSource.repoType 校验（required + oneof） ====================
	Describe("helmSpec.helmSource.repoType field", func() {
		It("should return error when helmSource.repoType is empty", func() {
			spec := &AppCreateSpec{
				Name: "my-helm-app",
				Type: "helm",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/app",
					},
				},
				HelmSpec: &HelmSpecSpec{
					HelmSource: &HelmSourceSpec{
						RepoType: "",
					},
				},
			}
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is required")))
		})

		It("should return error when helmSource.repoType is invalid", func() {
			spec := &AppCreateSpec{
				Name: "my-helm-app",
				Type: "helm",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/app",
					},
				},
				HelmSpec: &HelmSpecSpec{
					HelmSource: &HelmSourceSpec{
						RepoType: "InvalidRepo",
					},
				},
			}
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is invalid")))
		})
	})

	// ==================== buildConfig.repoBuildConfig.type 校验（required + oneof） ====================
	Describe("buildConfig.repoBuildConfig.type field", func() {
		It("should return error when repoBuildConfig.type is invalid", func() {
			spec := validTrpcSpec()
			spec.BuildConfig.SourceType = "codeRepository"
			spec.BuildConfig.ImageBuildConfig = nil
			spec.BuildConfig.RepoBuildConfig = &RepoBuildConfigSpec{
				Type:          "InvalidType",
				RepoAlias:     "my-repo",
				RepoURL:       "https://git.example.com/repo.git",
				DefaultBranch: "main",
			}
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is invalid")))
		})
	})

	// ==================== helmSpec.helmSource.gitRepoConfig.type 校验（required + oneof） ====================
	Describe("helmSpec.helmSource.gitRepoConfig.type field", func() {
		It("should return error when gitRepoConfig.type is invalid", func() {
			spec := &AppCreateSpec{
				Name: "my-git-app",
				Type: "helm",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/app",
					},
				},
				HelmSpec: &HelmSpecSpec{
					HelmSource: &HelmSourceSpec{
						RepoType: "GitRepo",
						GitRepoConfig: &GitRepoConfigSpec{
							Type:      "InvalidGit",
							RepoAlias: "my-repo",
							RepoURL:   "https://git.example.com/charts.git",
							Revision:  "main",
							SourceDir: "./charts",
						},
					},
				},
			}
			Expect(spec.Validate()).To(MatchError(ContainSubstring("is invalid")))
		})
	})

	// ==================== 合法 spec 通过校验 ====================
	Describe("valid specs", func() {
		It("should pass validation for valid spec", func() {
			spec := validTrpcSpec()
			Expect(spec.Validate()).NotTo(HaveOccurred())
		})
	})
})
