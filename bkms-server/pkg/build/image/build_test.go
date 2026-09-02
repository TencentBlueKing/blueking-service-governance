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
	"context"
	"encoding/json"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

var _ = Describe("Build Functions", func() {
	var (
		ctx       context.Context
		testApp   *bkmsapp.Application
		testCfg   *Config
		branch    string
		imageTag  string
		oldConfig *config.Config
	)

	BeforeEach(func() {
		ctx = context.Background()
		branch = "main"
		imageTag = "v1.0.0"
		oldConfig = config.G
		config.G = &config.Config{}
		config.G.ImageBuild.ToolchainBaseURL = "https://bkrepo.example.com/image-build-toolchain"

		// Mock auth.MustGetUser
		mockey.Mock(auth.MustGetUser).Return(auth.User{ID: "test-user"}).Build()

		// 创建测试应用对象
		testApp = &bkmsapp.Application{
			ID:          "test-app-id",
			Name:        "bkms-server",
			WorkspaceID: "test-workspace-id",
			TrpcSpec:    &bkmsapp.TrpcSpec{Language: appmodel.LanguageGo},
		}

		// 创建测试配置对象（代码仓库类型）
		testCfg = &Config{
			AppID:        testApp.ID,
			SourceType:   SourceTypeCodeRepository,
			PipelineType: "Dockerfile",
			CodeRepo: &RepositoryConfig{
				Type:          RepositoryTypeTGit,
				RepoAlias:     "bkms/bkms-server",
				RepoURL:       "https://git.example.com/bkms/bkms-server",
				DefaultBranch: "main",
				SourceDir:     "src",
				Dockerfile:    "Dockerfile",
				DockerBuildArgs: map[string]string{
					"BUILD_ENV": "production",
					"APP_NAME":  "bkms-server",
				},
			},
		}
	})

	AfterEach(func() {
		// 清理所有 Mock
		mockey.UnPatchAll()
		// 恢复 config.G，避免全局配置污染后续测试
		config.G = oldConfig
	})

	Context("ExecuteBKCIPipelineBuild", func() {
		It("should build from code repository successfully", func() {
			// Mock RepositoryManager.Initialize 方法
			mockey.Mock((*bkci.RepositoryManager).Initialize).Return(
				&bkci.Repository{ID: "repo-hash-123", Alias: "bkms/bkms-server"}, nil,
			).Build()

			// Mock PipelineManager.Initialize：代码仓库场景会重新初始化内置 Dockerfile 流水线
			mockey.Mock((*bkci.PipelineManager).Initialize).Return(
				&bkci.Pipeline{ID: "pipeline-123", Type: testCfg.PipelineType}, nil,
			).Build()

			// Mock PipelineStore
			mockStore := &bkci.PipelineStoreMongo{}
			mockey.Mock(bkci.NewPipelineStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock(
				(*bkci.PipelineStoreMongo).GetByWorkspaceAndType,
			).Return(&bkci.Pipeline{
				ID:          "pipeline-123",
				ProjectCode: "project-code",
				Type:        testCfg.PipelineType,
			}, nil).Build()

			// Mock bkciapi.New
			mockey.Mock(bkciapi.New).Return(&bkciapi.ApiClient{}, nil).Build()

			// Mock BKCI api client 的方法
			mockey.Mock((*bkciapi.ApiClient).CreatePipelineBuild).Return(
				&bkciapi.PipelineBuildReference{ID: "build-123", Num: "1"}, nil,
			).Build()

			mockey.Mock((*bkciapi.ApiClient).GetPipelineBuildState).Return(
				&bkciapi.PipelineBuildState{Status: "RUNNING"}, nil,
			).Build()

			// Mock 工作空间镜像仓库
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(
				&registry.ImageRegistry{Registry: "hub.example.com", BkCICredentialID: "cred-123"}, nil,
			).Build()

			// 执行构建
			buildState, params, err := ExecuteBKCIPipelineBuild(ctx, testApp, testCfg, branch, imageTag)

			// 验证结果
			Expect(err).NotTo(HaveOccurred())
			Expect(buildState).NotTo(BeNil())
			Expect(buildState.Status).To(Equal("RUNNING"))
			Expect(params).NotTo(BeEmpty())
			Expect(params[pipelineparam.RepoURL]).To(Equal(testCfg.CodeRepo.RepoURL))
		})

		It("should build from pipeline with param merging", func() {
			// 修改配置为流水线类型
			testCfg.SourceType = SourceTypePipeline
			testCfg.CodeRepo = nil
			testCfg.Pipeline = &PipelineConfig{
				PipelineID: "p-123456",
				Params: map[string]string{
					"CUSTOM_PARAM":         "custom-value",
					pipelineparam.ImageTag: "should-be-overridden",
				},
			}

			// Mock PipelineManager.Initialize 方法
			mockey.Mock((*bkci.PipelineManager).Initialize).Return(
				&bkci.Pipeline{ID: "p-123456", Name: "test-pipeline"}, nil,
			).Build()

			// Mock PipelineStore
			mockStore := &bkci.PipelineStoreMongo{}
			mockey.Mock(bkci.NewPipelineStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock(
				(*bkci.PipelineStoreMongo).GetByWorkspaceAndType,
			).Return(&bkci.Pipeline{
				ID:          "pipeline-456",
				ProjectCode: "project-code",
				Type:        testCfg.PipelineType,
			}, nil).Build()

			// Mock bkciapi.New 返回一个非 nil 的 ApiClient
			mockey.Mock(bkciapi.New).Return(&bkciapi.ApiClient{}, nil).Build()

			// Mock BKCI 客户端的方法
			mockey.Mock((*bkciapi.ApiClient).CreatePipelineBuild).Return(
				&bkciapi.PipelineBuildReference{ID: "build-456", Num: "1"}, nil,
			).Build()

			mockey.Mock((*bkciapi.ApiClient).GetPipelineBuildState).Return(
				&bkciapi.PipelineBuildState{Status: "RUNNING"}, nil,
			).Build()

			// Mock 工作空间镜像仓库
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(
				&registry.ImageRegistry{Registry: "hub.example.com", BkCICredentialID: "cred-123"}, nil,
			).Build()

			// 执行构建
			buildState, params, err := ExecuteBKCIPipelineBuild(ctx, testApp, testCfg, branch, imageTag)

			// 验证结果
			Expect(err).NotTo(HaveOccurred())
			Expect(buildState).NotTo(BeNil())
			Expect(buildState.Status).To(Equal("RUNNING"))
			Expect(params).NotTo(BeEmpty())
			// 验证自定义参数被合并
			Expect(params["CUSTOM_PARAM"]).To(Equal("custom-value"))
			// 验证系统参数优先级更高
			Expect(params[pipelineparam.ImageTag]).To(Equal(imageTag))
		})

		It("should return error when code repo config is missing", func() {
			// 修改配置：缺少代码仓库配置
			testCfg.SourceType = SourceTypeCodeRepository
			testCfg.CodeRepo = nil

			_, _, err := ExecuteBKCIPipelineBuild(ctx, testApp, testCfg, branch, imageTag)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("code repo config not found"))
		})

		It("should return error when pipeline config is missing", func() {
			// 修改配置：缺少流水线配置
			testCfg.SourceType = SourceTypePipeline
			testCfg.CodeRepo = nil
			testCfg.Pipeline = nil

			_, _, err := ExecuteBKCIPipelineBuild(ctx, testApp, testCfg, branch, imageTag)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("pipeline config not found"))
		})

		It("should return error when build trigger fails", func() {
			// Mock 仓库初始化成功
			mockey.Mock((*bkci.RepositoryManager).Initialize).Return(
				&bkci.Repository{ID: "repo-123", Alias: "bkms/bkms-server"}, nil,
			).Build()

			// Mock PipelineManager.Initialize：代码仓库场景会重新初始化内置 Dockerfile 流水线
			mockey.Mock((*bkci.PipelineManager).Initialize).Return(
				&bkci.Pipeline{ID: "pipeline-789", Type: testCfg.PipelineType}, nil,
			).Build()

			// Mock PipelineStore
			mockStore := &bkci.PipelineStoreMongo{}
			mockey.Mock(bkci.NewPipelineStoreMongo).Return(mockStore, nil).Build()
			mockey.Mock(
				(*bkci.PipelineStoreMongo).GetByWorkspaceAndType,
			).Return(&bkci.Pipeline{
				ID:          "pipeline-789",
				ProjectCode: "project-code",
				Type:        testCfg.PipelineType,
			}, nil).Build()

			// Mock BKCI 客户端
			mockey.Mock(bkciapi.New).Return(&bkciapi.ApiClient{}, nil).Build()

			// Mock 构建触发失败
			mockey.Mock((*bkciapi.ApiClient).CreatePipelineBuild).Return(
				nil, errors.New("build trigger failed"),
			).Build()

			// Mock 工作空间镜像仓库
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(
				&registry.ImageRegistry{Registry: "hub.example.com", BkCICredentialID: "cred-123"}, nil,
			).Build()

			// 执行构建
			buildState, params, err := ExecuteBKCIPipelineBuild(ctx, testApp, testCfg, branch, imageTag)

			// 验证错误
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("create pipeline build"))
			Expect(buildState).To(BeNil())
			Expect(params).To(BeNil())
		})
	})

	Context("genPlatformBuildParams", func() {
		It("should generate repository Dockerfile defaults", func() {
			params, err := genPlatformBuildParams(testCfg.CodeRepo, testApp)

			Expect(err).NotTo(HaveOccurred())
			Expect(params).To(Equal(map[string]string{
				pipelineparam.DockerfileSourceType:         bkciDockerfileSourceRepository,
				pipelineparam.ImageBuildToolchainBaseURL:   config.G.ImageBuild.ToolchainBaseURL,
				pipelineparam.DockerfileLanguage:           "",
				pipelineparam.DockerfileBuilderImage:       "",
				pipelineparam.DockerfileRunnerImage:        "",
				pipelineparam.DockerfilePreBuildCommands:   "",
				pipelineparam.DockerfileBuildCommands:      "",
				pipelineparam.DockerfileRuntimeEnvCommands: "",
				pipelineparam.DockerfileStartCommand:       "",
			}))
		})

		It("should generate platform build params with commands", func() {
			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &BuildCommands{
					PreBuild:   []string{"go mod download", "go env"},
					Build:      []string{"go build -o app ./cmd/server"},
					RuntimeEnv: []string{"apt-get update", "apt-get install -y ca-certificates"},
					Start:      "./app",
				},
			}

			params, err := genPlatformBuildParams(testCfg.CodeRepo, testApp)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.DockerfileSourceType]).To(Equal(bkciDockerfileSourceGenerated))
			Expect(params[pipelineparam.ImageBuildToolchainBaseURL]).To(Equal(config.G.ImageBuild.ToolchainBaseURL))
			Expect(params[pipelineparam.DockerfileLanguage]).To(Equal(appmodel.LanguageGo))
			Expect(params[pipelineparam.DockerfileBuilderImage]).To(Equal("golang:1.24"))
			Expect(params[pipelineparam.DockerfileRunnerImage]).To(Equal("debian:12"))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerfilePreBuildCommands])).To(Equal(
				[]string{"go mod download", "go env"},
			))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerfileBuildCommands])).To(Equal(
				[]string{"go build -o app ./cmd/server"},
			))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerfileRuntimeEnvCommands])).To(Equal(
				[]string{"apt-get update", "apt-get install -y ca-certificates"},
			))
			Expect(params[pipelineparam.DockerfileStartCommand]).To(Equal("./app"))
		})

		It("should encode platform commands with special characters", func() {
			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &BuildCommands{
					PreBuild: []string{
						`go env -w GOPROXY="https://goproxy.cn,direct"`,
						`go env -w GOPRIVATE=""`,
						`printf "path=/data/workspace"`,
						"",
					},
					Build: []string{},
					RuntimeEnv: []string{
						"apk --update --no-cache add bash",
					},
					Start: "./app",
				},
			}

			params, err := genPlatformBuildParams(testCfg.CodeRepo, testApp)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.DockerfilePreBuildCommands]).NotTo(ContainSubstring("\n"))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerfilePreBuildCommands])).To(Equal([]string{
				`go env -w GOPROXY="https://goproxy.cn,direct"`,
				`go env -w GOPRIVATE=""`,
				`printf "path=/data/workspace"`,
				"",
			}))
			Expect(params[pipelineparam.DockerfileBuildCommands]).To(Equal(""))
			Expect(
				decodeDockerfileCommandsParam(params[pipelineparam.DockerfileRuntimeEnvCommands]),
			).To(Equal([]string{
				"apk --update --no-cache add bash",
			}))
		})

		It("should return error when app trpc spec is missing", func() {
			testApp.TrpcSpec = nil
			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
			}

			params, err := genPlatformBuildParams(testCfg.CodeRepo, testApp)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing TrpcSpec for platform image build"))
			Expect(params).To(BeNil())
		})
	})

	Context("genPipelineBuildParams", func() {
		It("should generate all required params correctly", func() {
			// Mock 工作空间镜像仓库
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "hub.example.com",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			// 生成参数
			params, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)

			// 验证结果
			Expect(err).NotTo(HaveOccurred())
			Expect(params).NotTo(BeEmpty())
			// 验证代码库信息
			Expect(params[pipelineparam.RepoURL]).To(Equal(testCfg.CodeRepo.RepoURL))
			Expect(params[pipelineparam.RepoAlias]).To(Equal(testCfg.CodeRepo.RepoAlias))
			Expect(params[pipelineparam.RepoCheckoutBy]).To(Equal("BRANCH"))
			Expect(params[pipelineparam.RepoRevision]).To(Equal(branch))
			// 验证 Docker 构建参数
			Expect(params[pipelineparam.DockerBuildDir]).To(Equal("src"))
			Expect(params[pipelineparam.DockerfilePath]).To(Equal("src/Dockerfile"))
			Expect(params[pipelineparam.DockerBuildArgs]).To(ContainSubstring("APP_NAME=bkms-server"))
			Expect(params[pipelineparam.DockerBuildArgs]).To(ContainSubstring("BUILD_ENV=production"))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerBuildArgNames])).To(Equal(
				[]string{"APP_NAME", "BUILD_ENV"},
			))
			// 验证镜像信息
			Expect(params[pipelineparam.ImageCredential]).To(Equal("cred-123"))
			Expect(params[pipelineparam.ImageRegistry]).To(Equal("hub.example.com"))
			Expect(params[pipelineparam.ImageRegistryHost]).To(BeEmpty())
			Expect(params[pipelineparam.ImageName]).To(Equal(testApp.Name))
			Expect(params[pipelineparam.ImageTag]).To(Equal(imageTag))
			// 验证 platform 构建相关参数：repositoryDockerfile 模式下参数集固定，除 sourceType 和 ToolchainBaseURL 外其余为空字符串
			Expect(params[pipelineparam.DockerfileSourceType]).To(Equal(bkciDockerfileSourceRepository))
			Expect(params[pipelineparam.DockerfileLanguage]).To(Equal(""))
			Expect(params[pipelineparam.ImageBuildToolchainBaseURL]).To(Equal(config.G.ImageBuild.ToolchainBaseURL))
			Expect(params[pipelineparam.DockerfileBuilderImage]).To(Equal(""))
			Expect(params[pipelineparam.DockerfileRunnerImage]).To(Equal(""))
			Expect(params[pipelineparam.DockerfilePreBuildCommands]).To(Equal(""))
			Expect(params[pipelineparam.DockerfileBuildCommands]).To(Equal(""))
			Expect(params[pipelineparam.DockerfileRuntimeEnvCommands]).To(Equal(""))
			Expect(params[pipelineparam.DockerfileStartCommand]).To(Equal(""))
		})

		It("should fill registry host for repository Dockerfile mode", func() {
			// 仓库自带 Dockerfile 也可能引用镜像源里的私有基础镜像，凭证配对不能因构建方式被跳过
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "mirrors.tencent.com/example",
				Username:         "robot",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			params, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.DockerfileSourceType]).To(Equal(bkciDockerfileSourceRepository))
			Expect(params[pipelineparam.ImageRegistryHost]).To(Equal("mirrors.tencent.com"))
			Expect(params[pipelineparam.ImageCredential]).To(Equal("cred-123"))
		})

		It("should populate platform build params when image build mode is platform", func() {
			// Mock 工作空间镜像仓库
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "hub.example.com",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			// platform 模式下 Dockerfile 字段应为空，SourceDir 保留作为 build context。
			testCfg.CodeRepo.Dockerfile = ""
			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &BuildCommands{
					PreBuild:   []string{"go mod download", "go env"},
					Build:      []string{"go build -o app ./cmd/server"},
					RuntimeEnv: []string{"apt-get update"},
					Start:      "./app",
				},
			}

			params, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.DockerfileSourceType]).To(Equal(bkciDockerfileSourceGenerated))
			Expect(params[pipelineparam.DockerfileLanguage]).To(Equal(appmodel.LanguageGo))
			Expect(params[pipelineparam.ImageBuildToolchainBaseURL]).To(Equal(config.G.ImageBuild.ToolchainBaseURL))
			Expect(params[pipelineparam.DockerfileBuilderImage]).To(Equal("golang:1.24"))
			Expect(params[pipelineparam.DockerfileRunnerImage]).To(Equal("debian:12"))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerfilePreBuildCommands])).To(Equal(
				[]string{"go mod download", "go env"},
			))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerfileBuildCommands])).To(Equal(
				[]string{"go build -o app ./cmd/server"},
			))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerfileRuntimeEnvCommands])).To(Equal(
				[]string{"apt-get update"},
			))
			Expect(params[pipelineparam.DockerfileStartCommand]).To(Equal("./app"))
			// SourceDir 仍作为 build context 保留
			Expect(params[pipelineparam.DockerBuildDir]).To(Equal("src"))
			Expect(params[pipelineparam.DockerfilePath]).To(Equal("src/.bkms/Dockerfile.generated"))
			Expect(decodeDockerfileCommandsParam(params[pipelineparam.DockerBuildArgNames])).To(Equal(
				[]string{"APP_NAME", "BUILD_ENV"},
			))
		})

		It("should use workspace root for generated Dockerfile path when source dir is empty", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "hub.example.com",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			testCfg.CodeRepo.SourceDir = ""
			testCfg.CodeRepo.Dockerfile = ""
			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
			}

			params, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.DockerBuildDir]).To(Equal("."))
			Expect(params[pipelineparam.DockerfilePath]).To(Equal(".bkms/Dockerfile.generated"))
		})

		It("should allow empty platform build start command", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "hub.example.com",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &BuildCommands{
					PreBuild: []string{"go mod download"},
					Build:    []string{"go build -o app ./cmd/server"},
				},
			}

			params, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.DockerfileStartCommand]).To(Equal(""))
		})

		It("should return error when platform build app trpc spec is missing", func() {
			testApp.TrpcSpec = nil
			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
			}

			_, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing TrpcSpec for platform image build"))
		})

		It("should return error when platform build app language is empty", func() {
			testApp.TrpcSpec.Language = ""
			testCfg.CodeRepo.ImageBuildMode = ImageBuildModePlatform
			testCfg.CodeRepo.PlatformBuildConfig = &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
			}

			_, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing TrpcSpec language for platform image build"))
		})

		It("should return error when CodeRepo is nil", func() {
			// 设置 CodeRepo 为 nil
			testCfg.CodeRepo = nil

			_, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("code repo config not found"))
		})

		It("should return error when image params generation fails", func() {
			// Mock 工作空间镜像仓库失败
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(nil, errors.New("registry not found")).Build()

			_, err := genPipelineBuildParams(ctx, testApp, testCfg, branch, imageTag)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("generate pipeline build image params"))
		})
	})

	Context("genPipelineBuildRepoAndImageParams", func() {
		It("should generate repo and image params successfully", func() {
			// Mock 工作空间镜像仓库
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "hub.example.com",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			// 生成参数
			params, err := genPipelineBuildRepoAndImageParams(ctx, testApp, branch, imageTag)

			// 验证结果
			Expect(err).NotTo(HaveOccurred())
			Expect(params).NotTo(BeEmpty())
			// 验证固定参数
			Expect(params[pipelineparam.RepoCheckoutBy]).To(Equal("BRANCH"))
			Expect(params[pipelineparam.RepoRevision]).To(Equal(branch))
			// 验证镜像仓库信息
			Expect(params[pipelineparam.ImageRegistry]).To(Equal("hub.example.com"))
			Expect(params[pipelineparam.ImageRegistryHost]).To(BeEmpty())
			Expect(params[pipelineparam.ImageName]).To(Equal(testApp.Name))
			Expect(params[pipelineparam.ImageTag]).To(Equal(imageTag))
			Expect(params[pipelineparam.ImageCredential]).To(Equal("cred-123"))
		})

		It("should fill registry host when registry has username and credential", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "https://mirrors.tencent.com/example",
				Username:         "robot",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			params, err := genPipelineBuildRepoAndImageParams(ctx, testApp, branch, imageTag)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.ImageRegistryHost]).To(Equal("mirrors.tencent.com"))
			Expect(params[pipelineparam.ImageCredential]).To(Equal("cred-123"))
		})

		It("should leave registry host empty for public registry", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "mirrors.tencent.com/example",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			params, err := genPipelineBuildRepoAndImageParams(ctx, testApp, branch, imageTag)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.ImageRegistryHost]).To(BeEmpty())
			Expect(params[pipelineparam.ImageCredential]).To(Equal("cred-123"))
		})

		It("should leave registry host empty when bkci credential is missing", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry: "mirrors.tencent.com/example",
				Username: "robot",
			}, nil).Build()

			params, err := genPipelineBuildRepoAndImageParams(ctx, testApp, branch, imageTag)

			Expect(err).NotTo(HaveOccurred())
			Expect(params[pipelineparam.ImageRegistryHost]).To(BeEmpty())
		})

		It("should return error when getting workspace registry fails", func() {
			// Mock 工作空间镜像仓库失败
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(
				nil, errors.New("registry not found"),
			).Build()

			// 生成参数
			params, err := genPipelineBuildRepoAndImageParams(ctx, testApp, branch, imageTag)

			// 验证错误
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("get workspace image registry"))
			Expect(params).To(BeNil())
		})
	})

	DescribeTable("sourceImageRegistryHost clips registry to host",
		func(addr, want string) {
			Expect(sourceImageRegistryHost(&registry.ImageRegistry{
				Registry:         addr,
				Username:         "robot",
				BkCICredentialID: "cred-123",
			})).To(Equal(want))
		},
		Entry("path only", "mirrors.tencent.com/example", "mirrors.tencent.com"),
		Entry("https scheme", "https://mirrors.tencent.com/example", "mirrors.tencent.com"),
		Entry("host with port", "mirrors.tencent.com:5000/example", "mirrors.tencent.com:5000"),
	)
})

func decodeDockerfileCommandsParam(param string) []string {
	var commands []string
	ExpectWithOffset(1, json.Unmarshal([]byte(param), &commands)).To(Succeed())
	return commands
}
