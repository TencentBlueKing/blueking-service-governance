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

var _ = Describe("buildCreateAppRequest", func() {
	Describe("buildAppID", func() {
		It("should concatenate name and suffix when total length is within limit", func() {
			id, err := buildAppID("my-app", "-abc123")
			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal("my-app-abc123"))
		})

		It("should return error when name + suffix exceeds maxIDLength", func() {
			longName := strings.Repeat("a", 60)
			_, err := buildAppID(longName, "-abcdef")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exceeds"))
		})
	})

	// ==================== trpc 类型请求组装 ====================
	Describe("trpc type", func() {
		It("should assemble request with correct fields", func() {
			spec := &AppCreateSpec{
				Name: "my-trpc-app",
				Type: "trpc",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/my-trpc-app",
					},
				},
				AppModelSpec: &AppModelSpecSpec{
					Command: []string{"/app/bin/server"},
					Args:    []string{"--config", "/app/conf/trpc_go.yaml"},
					EnvVars: []VariableSpec{
						{Key: "ENV", Value: "prod", Description: "环境"},
					},
					TrpcSpec: &TrpcSpecSpec{
						Language: "go",
						FileName: "trpc_go.yaml",
						FilePath: "/app/conf",
					},
				},
			}

			req, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).NotTo(HaveOccurred())

			Expect(req.ID).To(Equal("test-app-id"))
			Expect(req.Name).To(Equal("my-trpc-app"))
			Expect(req.Type).To(Equal("trpc"))

			// buildConfig 直接透传
			Expect(req.BuildConfig).To(Equal(spec.BuildConfig))
			Expect(req.BuildConfig.SourceType).To(Equal("imageRegistry"))
			Expect(req.BuildConfig.ImageBuildConfig.Name).To(Equal("mirrors.tencent.com/myns/my-trpc-app"))

			// appModelSpec 直接透传
			Expect(req.AppModelSpec).To(Equal(spec.AppModelSpec))
			Expect(req.AppModelSpec.TrpcSpec.Language).To(Equal("go"))
			Expect(req.AppModelSpec.TrpcSpec.FileName).To(Equal("trpc_go.yaml"))
			Expect(req.AppModelSpec.Command).To(Equal([]string{"/app/bin/server"}))
			Expect(req.AppModelSpec.EnvVars).To(HaveLen(1))
			Expect(req.AppModelSpec.EnvVars[0].Key).To(Equal("ENV"))

			// 不应有 helmSpec
			Expect(req.HelmSpec).To(BeNil())
		})
	})

	// ==================== taf 类型请求组装 ====================
	Describe("taf type", func() {
		It("should assemble request with correct fields", func() {
			spec := &AppCreateSpec{
				Name: "my-taf-app",
				Type: "taf",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/my-taf-app",
					},
				},
				AppModelSpec: &AppModelSpecSpec{
					TafSpec: &TafSpecSpec{
						FileName: "config.conf",
						FilePath: "/app/conf",
					},
				},
			}

			req, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).NotTo(HaveOccurred())

			Expect(req.AppModelSpec).To(Equal(spec.AppModelSpec))
			Expect(req.AppModelSpec.TafSpec.FileName).To(Equal("config.conf"))
			Expect(req.AppModelSpec.TrpcSpec).To(BeNil())
			Expect(req.HelmSpec).To(BeNil())
		})
	})

	// ==================== helm 类型请求组装 ====================
	Describe("helm type", func() {
		It("should fill default valueFiles when not specified", func() {
			spec := &AppCreateSpec{
				Name: "my-helm-app",
				Type: "helm",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/my-helm-app",
					},
				},
				HelmSpec: &HelmSpecSpec{
					HelmSource: &HelmSourceSpec{
						RepoType: "HelmRepo",
						HelmRepoConfig: &HelmRepoConfigSpec{
							RepoURL:   "https://charts.example.com",
							ChartName: "my-chart",
						},
					},
				},
			}

			req, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).NotTo(HaveOccurred())

			Expect(req.HelmSpec.HelmSource.ValueFiles).To(Equal([]string{"values.yaml"}))
			Expect(req.HelmSpec.HelmSource.RepoType).To(Equal("HelmRepo"))
			Expect(req.HelmSpec.HelmSource.HelmRepoConfig.ChartName).To(Equal("my-chart"))
			Expect(req.AppModelSpec).To(BeNil())
		})

		It("should use user-specified valueFiles", func() {
			spec := &AppCreateSpec{
				Name: "my-helm-app",
				Type: "helm",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/my-helm-app",
					},
				},
				HelmSpec: &HelmSpecSpec{
					HelmSource: &HelmSourceSpec{
						RepoType:   "GitRepo",
						ValueFiles: []string{"values.yaml", "values-prod.yaml"},
						GitRepoConfig: &GitRepoConfigSpec{
							Type:      "TGit",
							RepoAlias: "myrepo",
							RepoURL:   "https://git.example.com/charts.git",
							Revision:  "main",
							SourceDir: "./charts/my-chart",
						},
					},
				},
			}

			req, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).NotTo(HaveOccurred())

			Expect(req.HelmSpec.HelmSource.ValueFiles).To(Equal([]string{"values.yaml", "values-prod.yaml"}))
			Expect(req.HelmSpec.HelmSource.GitRepoConfig.Type).To(Equal("TGit"))
			Expect(req.HelmSpec.HelmSource.GitRepoConfig.RepoAlias).To(Equal("myrepo"))
		})

		It("should support agones type", func() {
			spec := &AppCreateSpec{
				Name: "my-agones-app",
				Type: "agones",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "mirrors.tencent.com/myns/my-agones-app",
					},
				},
				HelmSpec: &HelmSpecSpec{
					HelmSource: &HelmSourceSpec{
						RepoType: "BCSRepo",
						BCSRepoConfig: &BCSRepoConfigSpec{
							ProjectCode: "my-project",
							RepoName:    "my-repo",
							ChartName:   "my-chart",
						},
					},
				},
			}

			req, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).NotTo(HaveOccurred())
			Expect(req.Type).To(Equal("agones"))
			Expect(req.HelmSpec.HelmSource.BCSRepoConfig.ProjectCode).To(Equal("my-project"))
		})
	})

	// ==================== 构建方式透传 ====================
	Describe("build config passthrough", func() {
		It("should pass through codeRepository build config", func() {
			spec := &AppCreateSpec{
				Name: "my-repo-app",
				Type: "trpc",
				BuildConfig: &BuildConfigSpec{
					SourceType: "codeRepository",
					RepoBuildConfig: &RepoBuildConfigSpec{
						Type:            "TGit",
						RepoAlias:       "myrepo",
						RepoURL:         "https://git.example.com/myrepo.git",
						DefaultBranch:   "main",
						SourceDir:       "./src",
						Dockerfile:      "Dockerfile.prod",
						DockerBuildArgs: map[string]string{"GO_VERSION": "1.21"},
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

			req, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).NotTo(HaveOccurred())

			Expect(req.BuildConfig.SourceType).To(Equal("codeRepository"))
			Expect(req.BuildConfig.RepoBuildConfig.Type).To(Equal("TGit"))
			Expect(req.BuildConfig.RepoBuildConfig.RepoAlias).To(Equal("myrepo"))
			Expect(req.BuildConfig.RepoBuildConfig.DockerBuildArgs).To(Equal(map[string]string{"GO_VERSION": "1.21"}))
		})

		It("should pass through pipeline build config", func() {
			spec := &AppCreateSpec{
				Name: "my-pipe-app",
				Type: "trpc",
				BuildConfig: &BuildConfigSpec{
					SourceType: "pipeline",
					PipelineBuildConfig: &PipelineBuildConfigSpec{
						PipelineID: "p-xxxx",
						Params:     map[string]string{"env": "prod"},
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

			req, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).NotTo(HaveOccurred())

			Expect(req.BuildConfig.SourceType).To(Equal("pipeline"))
			Expect(req.BuildConfig.PipelineBuildConfig.PipelineID).To(Equal("p-xxxx"))
			Expect(req.BuildConfig.PipelineBuildConfig.Params).To(Equal(map[string]string{"env": "prod"}))
		})
	})

	// ==================== 未知类型错误处理 ====================
	Describe("unknown type handling", func() {
		It("should return error for unsupported app type", func() {
			spec := &AppCreateSpec{
				Name: "my-app",
				Type: "unknown",
				BuildConfig: &BuildConfigSpec{
					SourceType: "imageRegistry",
					ImageBuildConfig: &ImageBuildConfigSpec{
						Name: "img",
					},
				},
			}
			_, err := buildCreateAppRequest("test-app-id", spec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported app type"))
		})
	})
})
