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
	"strings"

	"github.com/gin-gonic/gin/binding"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/serializer"
	deploystatus "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/status"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
)

var _ = Describe("App deploy status serializers", func() {
	It("maps deploy status rows to output objects", func() {
		output := new(serializer.AppDeployedEnvOutputObj).FromModel(deploystatus.AppDeployStatus{
			EnvID:           "env-id",
			EnvName:         "prod",
			EnvDisplayName:  "Production",
			EnvType:         "production",
			EnvKind:         "standard",
			TrafficLaneName: "gray",
			DeployStatus:    "success",
			ImageTag:        "v1.2.3",
		})

		Expect(output).To(Equal(&serializer.AppDeployedEnvOutputObj{
			ID:              "env-id",
			Name:            "prod",
			DisplayName:     "Production",
			Type:            "production",
			Kind:            "standard",
			TrafficLaneName: "gray",
			DeployStatus:    "success",
			ImageTag:        "v1.2.3",
		}))
	})

	DescribeTable(
		"AppURIInput validation",
		func(input serializer.AppURIInput, expectedErrSubstrings []string) {
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
		Entry("valid app id", serializer.AppURIInput{
			AppID: "app-123",
		}, nil),
		Entry("missing app id", serializer.AppURIInput{}, []string{
			"AppURIInput.AppID",
			"failed on the 'required' tag",
		}),
		Entry("invalid uri slug", serializer.AppURIInput{
			AppID: "app.invalid",
		}, []string{
			"AppURIInput.AppID",
			"failed on the 'uri_slug' tag",
		}),
	)
})

var _ = Describe("App serializers", func() {
	It("maps app detail output with app model fields", func() {
		app := &bkmsapp.Application{
			ID:          "app-id",
			WorkspaceID: "workspace-1",
			Name:        "demo-app",
			Type:        bkmsapp.AppTypeTRPC,
			DisplayName: "Demo App",
			Creator:     "tester",
		}
		buildConfig := &build.Config{
			SourceType: build.SourceTypePipeline,
			TagConfig: build.TagConfig{
				Type: build.VersionTypeCustom,
				CustomOpts: &build.CustomTagOpts{
					Prefix:        "rel",
					WithRevision:  true,
					WithBuildTime: true,
				},
			},
			Pipeline: &build.PipelineConfig{
				PipelineID: "p-123",
				Params:     map[string]string{"region": "gz"},
			},
		}
		appModel := &appmodel.AppModel{
			Workload: appmodel.Workload{
				Command: []string{"/start"},
				Args:    []string{"--debug"},
				EnvVars: []appmodel.Variable{{Key: "ENV", Value: "prod", Description: "runtime env"}},
				TrpcConfig: appmodel.TrpcConfig{
					Language:    "go",
					FileName:    "trpc_go.yaml",
					FilePath:    "/data/trpc_go.yaml",
					FileContent: "content",
				},
			},
		}
		components := []serializer.ComponentOutputObj{{
			Name:                 "resource-limits",
			Type:                 "ResourceLimits",
			Version:              "v1.0.0",
			Properties:           map[string]any{"cpu": "500m"},
			RefWorkspaceCompName: "shared-config",
			ScopeType:            "environment",
			ScopeEnvNames:        []string{"prod"},
		}}

		output := new(serializer.AppDetailOutputObj).FromModel(app, buildConfig, appModel, components)

		Expect(output).To(Equal(&serializer.AppDetailOutputObj{
			ID:          "app-id",
			WorkspaceID: "workspace-1",
			Name:        "demo-app",
			Type:        "trpc",
			DisplayName: "Demo App",
			Creator:     "tester",
			BuildConfig: &serializer.BuildConfigOutputObj{
				SourceType: "pipeline",
				TagConfig: &serializer.TagConfigOutputObj{
					Type: "custom",
					CustomOpts: &serializer.CustomTagOptsOutputObj{
						Prefix:        "rel",
						WithRevision:  true,
						WithBuildTime: true,
					},
				},
				PipelineBuildConfig: &serializer.PipelineBuildConfigOutputObj{
					PipelineID: "p-123",
					Params:     map[string]string{"region": "gz"},
				},
			},
			AppModelSpec: &serializer.AppModelSpecOutputObj{
				Command: []string{"/start"},
				Args:    []string{"--debug"},
				EnvVars: []serializer.VariableOutputObj{{Key: "ENV", Value: "prod", Description: "runtime env"}},
				Components: []serializer.ComponentOutputObj{{
					Name:                 "resource-limits",
					Type:                 "ResourceLimits",
					Version:              "v1.0.0",
					Properties:           map[string]any{"cpu": "500m"},
					RefWorkspaceCompName: "shared-config",
					ScopeType:            "environment",
					ScopeEnvNames:        []string{"prod"},
				}},
				TrpcSpec: &serializer.TrpcSpecOutputObj{
					Language:    "go",
					FileName:    "trpc_go.yaml",
					FilePath:    "/data/trpc_go.yaml",
					FileContent: "content",
				},
			},
		}))
	})

	It("keeps optional app detail fields nil when models are absent", func() {
		app := &bkmsapp.Application{
			ID:          "helm-app",
			WorkspaceID: "workspace-1",
			Name:        "helm-demo",
			Type:        bkmsapp.AppTypeHelm,
			DisplayName: "Helm Demo",
			Creator:     "tester",
		}

		output := new(serializer.AppDetailOutputObj).FromModel(app, nil, nil, nil)

		Expect(output.BuildConfig).To(BeNil())
		Expect(output.HelmSpec).To(BeNil())
		Expect(output.AppModelSpec).To(BeNil())
	})

	It("maps build config output from code repository source", func() {
		output := new(serializer.BuildConfigOutputObj).FromModel(&build.Config{
			SourceType: build.SourceTypeCodeRepository,
			TagConfig: build.TagConfig{
				Type: build.VersionTypeCustom,
			},
			CodeRepo: &build.RepositoryConfig{
				Type:            build.RepositoryTypeTGit,
				RepoAlias:       "demo-repo",
				RepoURL:         "https://example.com/repo.git",
				DefaultBranch:   "main",
				SourceDir:       "cmd/server",
				Dockerfile:      "build/Dockerfile",
				DockerBuildArgs: map[string]string{"GO_ENV": "prod"},
			},
		})

		Expect(output).To(Equal(&serializer.BuildConfigOutputObj{
			SourceType: "codeRepository",
			TagConfig: &serializer.TagConfigOutputObj{
				Type: "custom",
			},
			RepoBuildConfig: &serializer.RepoBuildConfigOutputObj{
				Type:            "TGit",
				RepoAlias:       "demo-repo",
				RepoURL:         "https://example.com/repo.git",
				DefaultBranch:   "main",
				SourceDir:       "cmd/server",
				Dockerfile:      "build/Dockerfile",
				DockerBuildArgs: map[string]string{"GO_ENV": "prod"},
				ImageBuildMode:  "repositoryDockerfile",
			},
		}))
	})

	It("maps platform build config output", func() {
		output := new(serializer.BuildConfigOutputObj).FromModel(&build.Config{
			SourceType: build.SourceTypeCodeRepository,
			TagConfig:  build.TagConfig{Type: build.VersionTypeCustom},
			CodeRepo: &build.RepositoryConfig{
				Type:           build.RepositoryTypeTGit,
				ImageBuildMode: build.ImageBuildModePlatform,
				PlatformBuildConfig: &build.PlatformBuildConfig{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
					Commands: &build.BuildCommands{
						PreBuild:   []string{"go mod download"},
						Build:      []string{"go build -o app ./cmd/server"},
						RuntimeEnv: []string{"apt-get update"},
						Start:      "./app",
					},
				},
			},
		})

		Expect(output.RepoBuildConfig.ImageBuildMode).To(Equal("platform"))
		Expect(output.RepoBuildConfig.PlatformBuildConfig).To(Equal(&serializer.PlatformBuildConfigOutputObj{
			BuilderImage: "golang:1.24",
			RunnerImage:  "debian:12",
			Commands: &serializer.BuildCommandsOutputObj{
				PreBuild:   []string{"go mod download"},
				Build:      []string{"go build -o app ./cmd/server"},
				RuntimeEnv: []string{"apt-get update"},
				Start:      "./app",
			},
		}))
	})

	It("maps helm spec output from git repo source", func() {
		output := new(serializer.HelmSpecOutputObj).FromModel(&bkmsapp.HelmSpec{
			HelmSource: &bkmsapp.HelmSource{
				RepoType:   bkmsapp.HelmSourceRepoTypeGit,
				ValueFiles: []string{"values.yaml", "prod.yaml"},
				GitRepoConfig: &bkmsapp.GitRepoConfig{
					Type:      bkmsapp.GitRepoTypeTGit,
					RepoAlias: "charts",
					RepoURL:   "https://example.com/charts.git",
					Revision:  "main",
					SourceDir: "deploy/chart",
				},
			},
		})

		Expect(output).To(Equal(&serializer.HelmSpecOutputObj{
			HelmSource: &serializer.HelmSourceOutputObj{
				RepoType:   "GitRepo",
				ValueFiles: []string{"values.yaml", "prod.yaml"},
				GitRepoConfig: &serializer.HelmGitRepoConfigOutputObj{
					Type:      "TGit",
					RepoAlias: "charts",
					RepoURL:   "https://example.com/charts.git",
					Revision:  "main",
					SourceDir: "deploy/chart",
				},
			},
		}))
	})

	It("normalizes empty slices in app detail outputs", func() {
		appModel := &appmodel.AppModel{}
		components := []serializer.ComponentOutputObj{{
			Name:      "resource-limits",
			ScopeType: "global",
		}}

		output := new(serializer.AppModelSpecOutputObj).FromModel(appModel, components)

		Expect(output.Command).To(Equal([]string{}))
		Expect(output.Args).To(Equal([]string{}))
		Expect(output.EnvVars).To(Equal([]serializer.VariableOutputObj{}))
		Expect(output.Components).To(Equal([]serializer.ComponentOutputObj{{
			Name:          "resource-limits",
			ScopeType:     "global",
			ScopeEnvNames: []string{},
		}}))
	})

	It("normalizes empty slices in helm outputs", func() {
		output := new(serializer.HelmSpecOutputObj).FromModel(&bkmsapp.HelmSpec{
			HelmSource: &bkmsapp.HelmSource{
				RepoType: bkmsapp.HelmSourceRepoTypeGit,
			},
		})

		Expect(output.HelmSource.ValueFiles).To(Equal([]string{}))
	})

	It("maps app model spec output with taf config", func() {
		appModel := &appmodel.AppModel{
			Workload: appmodel.Workload{
				Command: []string{"run"},
				Args:    []string{"--taf"},
				EnvVars: []appmodel.Variable{{Key: "MODE", Value: "taf"}},
				TafConfig: appmodel.TafConfig{
					FileName:    "taf.conf",
					FilePath:    "/data/taf.conf",
					FileContent: "taf-content",
				},
			},
		}

		output := new(serializer.AppModelSpecOutputObj).FromModel(appModel, nil)

		Expect(output).To(Equal(&serializer.AppModelSpecOutputObj{
			Command:    []string{"run"},
			Args:       []string{"--taf"},
			EnvVars:    []serializer.VariableOutputObj{{Key: "MODE", Value: "taf", Description: ""}},
			Components: []serializer.ComponentOutputObj{},
			TafSpec: &serializer.TafSpecOutputObj{
				FileName:    "taf.conf",
				FilePath:    "/data/taf.conf",
				FileContent: "taf-content",
			},
		}))
	})

	DescribeTable(
		"WorkspaceURIInput validation",
		func(input serializer.WorkspaceURIInput, expectedErrSubstrings []string) {
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
		Entry("valid workspace id", serializer.WorkspaceURIInput{WorkspaceID: "workspace-1"}, nil),
		Entry("missing workspace id", serializer.WorkspaceURIInput{}, []string{
			"WorkspaceURIInput.WorkspaceID",
			"failed on the 'required' tag",
		}),
		Entry("invalid uri slug", serializer.WorkspaceURIInput{WorkspaceID: "workspace.invalid"}, []string{
			"WorkspaceURIInput.WorkspaceID",
			"failed on the 'uri_slug' tag",
		}),
	)

	DescribeTable(
		"VariableInput validation",
		func(input serializer.VariableInput, expectedErrSubstrings []string) {
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
		Entry("valid env var key", serializer.VariableInput{
			Key:   "VALID_KEY",
			Value: "v1",
		}, nil),
		Entry("invalid env var key", serializer.VariableInput{
			Key:   "INVALID-KEY",
			Value: "v1",
		}, []string{
			"VariableInput.Key",
			"failed on the 'envvar_key' tag",
		}),
		Entry("too long env var key", serializer.VariableInput{
			Key:   strings.Repeat("A", 257),
			Value: "v1",
		}, []string{
			"VariableInput.Key",
			"failed on the 'envvar_key' tag",
		}),
	)

	DescribeTable(
		"CreateAppInput validation",
		func(input serializer.CreateAppInput, expectedErrSubstrings []string) {
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
		Entry("valid trpc app input", serializer.CreateAppInput{
			Name: "demo-app",
			ID:   "demo-app",
			Type: bkmsapp.AppTypeTRPC,
			BuildConfig: &serializer.BuildConfigInput{
				SourceType: "pipeline",
				PipelineBuildConfig: &serializer.PipelineBuildConfigInput{
					PipelineID: "pipeline-id",
				},
			},
			AppModelSpec: &serializer.AppModelSpecInput{
				TrpcSpec: &serializer.TrpcSpecInput{
					Language: "go",
					FileName: "trpc_go.yaml",
					FilePath: "/data/trpc_go.yaml",
				},
			},
		}, nil),
		Entry("duplicate env var keys in trpc create input", serializer.CreateAppInput{
			Name: "demo-app",
			ID:   "demo-app",
			Type: bkmsapp.AppTypeTRPC,
			BuildConfig: &serializer.BuildConfigInput{
				SourceType: "pipeline",
				PipelineBuildConfig: &serializer.PipelineBuildConfigInput{
					PipelineID: "pipeline-id",
				},
			},
			AppModelSpec: &serializer.AppModelSpecInput{
				EnvVars: []serializer.VariableInput{
					{Key: "DUP_KEY", Value: "v1"},
					{Key: "DUP_KEY", Value: "v2"},
				},
				TrpcSpec: &serializer.TrpcSpecInput{
					Language: "go",
					FileName: "trpc_go.yaml",
					FilePath: "/data/trpc_go.yaml",
				},
			},
		}, []string{
			"CreateAppInput.AppModelSpec.EnvVars",
			"failed on the 'env_var_key_unique' tag",
		}),
		Entry("too long app name", serializer.CreateAppInput{
			Name: "demo-app-abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdef",
			ID:   "demo-app",
			Type: bkmsapp.AppTypeTRPC,
			BuildConfig: &serializer.BuildConfigInput{
				SourceType:          "pipeline",
				PipelineBuildConfig: &serializer.PipelineBuildConfigInput{PipelineID: "pipeline-id"},
			},
			AppModelSpec: &serializer.AppModelSpecInput{
				TrpcSpec: &serializer.TrpcSpecInput{
					Language: "go",
					FileName: "trpc_go.yaml",
					FilePath: "/data/trpc_go.yaml",
				},
			},
		}, []string{
			"CreateAppInput.Name",
			"failed on the 'app_id' tag",
		}),
		Entry("too long app id", serializer.CreateAppInput{
			Name: "demo-app",
			ID:   "demo-app-abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdef",
			Type: bkmsapp.AppTypeTRPC,
			BuildConfig: &serializer.BuildConfigInput{
				SourceType:          "pipeline",
				PipelineBuildConfig: &serializer.PipelineBuildConfigInput{PipelineID: "pipeline-id"},
			},
			AppModelSpec: &serializer.AppModelSpecInput{
				TrpcSpec: &serializer.TrpcSpecInput{
					Language: "go",
					FileName: "trpc_go.yaml",
					FilePath: "/data/trpc_go.yaml",
				},
			},
		}, []string{
			"CreateAppInput.ID",
			"failed on the 'app_id' tag",
		}),
		Entry("trpc app requires trpc spec", serializer.CreateAppInput{
			Name: "demo-app",
			ID:   "demo-app",
			Type: bkmsapp.AppTypeTRPC,
			BuildConfig: &serializer.BuildConfigInput{
				SourceType:          "pipeline",
				PipelineBuildConfig: &serializer.PipelineBuildConfigInput{PipelineID: "pipeline-id"},
			},
			AppModelSpec: &serializer.AppModelSpecInput{},
		}, []string{
			"Key: 'CreateAppInput.AppModelSpec.TrpcSpec'",
			"Error:Field validation for 'AppModelSpec.TrpcSpec' failed on the 'required' tag",
		}),
		Entry("helm app requires helm repo config for helm repo type", serializer.CreateAppInput{
			Name: "helm-app",
			ID:   "helm-app",
			Type: bkmsapp.AppTypeHelm,
			BuildConfig: &serializer.BuildConfigInput{
				SourceType:       "imageRegistry",
				ImageBuildConfig: &serializer.ImageBuildConfigInput{Name: "example.com/demo"},
			},
			HelmSpec: &serializer.HelmSpecInput{
				HelmSource: &serializer.HelmSourceInput{RepoType: string(bkmsapp.HelmSourceRepoTypeHelm)},
			},
		}, []string{
			"Key: 'CreateAppInput.HelmSpec.HelmSource.HelmRepoConfig'",
			"Error:Field validation for 'HelmRepoConfig' failed on the 'required' tag",
		}),
	)

	DescribeTable(
		"UpdateHelmSpecInput validation",
		func(input serializer.UpdateHelmSpecInput, expectedErrSubstrings []string) {
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
		Entry("missing helm spec", serializer.UpdateHelmSpecInput{}, []string{
			"Key: 'UpdateHelmSpecInput.HelmSpec'",
			"Error:Field validation for 'HelmSpec' failed on the 'required' tag",
		}),
		Entry(
			"helm repo type requires helm repo config",
			serializer.UpdateHelmSpecInput{
				HelmSpec: &serializer.HelmSpecInput{
					HelmSource: &serializer.HelmSourceInput{
						RepoType: string(bkmsapp.HelmSourceRepoTypeHelm),
					},
				},
			},
			[]string{
				"Key: 'UpdateHelmSpecInput.HelmSpec.HelmSource.HelmRepoConfig'",
				"Error:Field validation for 'HelmRepoConfig' failed on the 'required' tag",
			},
		),
		Entry(
			"bcs repo type requires bcs repo config",
			serializer.UpdateHelmSpecInput{
				HelmSpec: &serializer.HelmSpecInput{
					HelmSource: &serializer.HelmSourceInput{
						RepoType: string(bkmsapp.HelmSourceRepoTypeBCS),
					},
				},
			},
			[]string{
				"Key: 'UpdateHelmSpecInput.HelmSpec.HelmSource.BCSRepoConfig'",
				"Error:Field validation for 'BCSRepoConfig' failed on the 'required' tag",
			},
		),
		Entry(
			"git repo type requires git repo config",
			serializer.UpdateHelmSpecInput{
				HelmSpec: &serializer.HelmSpecInput{
					HelmSource: &serializer.HelmSourceInput{
						RepoType: string(bkmsapp.HelmSourceRepoTypeGit),
					},
				},
			},
			[]string{
				"Key: 'UpdateHelmSpecInput.HelmSpec.HelmSource.GitRepoConfig'",
				"Error:Field validation for 'GitRepoConfig' failed on the 'required' tag",
			},
		),
	)

	It("converts build config input to model for code repository source", func() {
		output, err := (&serializer.BuildConfigInput{
			SourceType: "codeRepository",
			RepoBuildConfig: &serializer.RepoBuildConfigInput{
				Type:          "TGit",
				RepoAlias:     "demo",
				RepoURL:       "https://example.com/repo.git",
				DefaultBranch: "main",
				SourceDir:     "cmd/server",
				Dockerfile:    "build/Dockerfile",
				DockerBuildArgs: map[string]string{
					"GO_ENV": "prod",
				},
			},
		}).ToModel("app-id")

		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(Equal(&build.Config{
			AppID:        "app-id",
			SourceType:   build.SourceTypeCodeRepository,
			PipelineType: "dockerfile",
			TagConfig: build.TagConfig{
				Type: build.VersionTypeCustom,
				CustomOpts: &build.CustomTagOpts{
					Prefix:        "",
					WithRevision:  false,
					WithBuildTime: true,
				},
			},
			CodeRepo: &build.RepositoryConfig{
				Type:          build.RepositoryTypeTGit,
				RepoAlias:     "demo",
				RepoURL:       "https://example.com/repo.git",
				DefaultBranch: "main",
				SourceDir:     "cmd/server",
				Dockerfile:    "build/Dockerfile",
				DockerBuildArgs: map[string]string{
					"GO_ENV": "prod",
				},
				ImageBuildMode: build.ImageBuildModeRepositoryDockerfile,
			},
		}))
	})

	DescribeTable(
		"Build config sourceDir validation",
		func(sourceDir, expectedErr string) {
			cfg, err := (&serializer.BuildConfigInput{
				SourceType: "codeRepository",
				RepoBuildConfig: &serializer.RepoBuildConfigInput{
					Type:          "TGit",
					RepoAlias:     "demo",
					RepoURL:       "https://example.com/repo.git",
					DefaultBranch: "main",
					SourceDir:     sourceDir,
				},
			}).ToModel("app-id")

			if expectedErr != "" {
				Expect(err).To(MatchError(ContainSubstring(expectedErr)))
				Expect(cfg).To(BeNil())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.CodeRepo.SourceDir).To(Equal(sourceDir))
		},
		Entry("empty sourceDir uses repository root", "", ""),
		Entry("relative sourceDir is accepted", "cmd/server", ""),
		Entry("absolute sourceDir is rejected", "/cmd/server", "must be a relative path inside repository"),
		Entry("parent segment sourceDir is rejected", "cmd/../server", "must not contain '..' path segment"),
	)

	It("converts platform build config input to model", func() {
		output, err := (&serializer.BuildConfigInput{
			SourceType: "codeRepository",
			RepoBuildConfig: &serializer.RepoBuildConfigInput{
				Type:           "TGit",
				RepoAlias:      "demo",
				RepoURL:        "https://example.com/repo.git",
				DefaultBranch:  "main",
				ImageBuildMode: "platform",
				PlatformBuildConfig: &serializer.PlatformBuildConfigInput{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
					Commands: &serializer.BuildCommandsInput{
						PreBuild:   []string{"go mod download"},
						Build:      []string{"go build -o app ./cmd/server"},
						RuntimeEnv: []string{"apt-get update"},
						Start:      "./app",
					},
				},
			},
		}).ToModel("app-id")

		Expect(err).NotTo(HaveOccurred())
		Expect(output.CodeRepo.ImageBuildMode).To(Equal(build.ImageBuildModePlatform))
		Expect(output.CodeRepo.PlatformBuildConfig).To(Equal(&build.PlatformBuildConfig{
			BuilderImage: "golang:1.24",
			RunnerImage:  "debian:12",
			Commands: &build.BuildCommands{
				PreBuild:   []string{"go mod download"},
				Build:      []string{"go build -o app ./cmd/server"},
				RuntimeEnv: []string{"apt-get update"},
				Start:      "./app",
			},
		}))
	})

	It("converts platform build config input without commands", func() {
		output, err := (&serializer.BuildConfigInput{
			SourceType: "codeRepository",
			RepoBuildConfig: &serializer.RepoBuildConfigInput{
				Type:           "TGit",
				RepoAlias:      "demo",
				RepoURL:        "https://example.com/repo.git",
				DefaultBranch:  "main",
				ImageBuildMode: "platform",
				PlatformBuildConfig: &serializer.PlatformBuildConfigInput{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
				},
			},
		}).ToModel("app-id")

		Expect(err).NotTo(HaveOccurred())
		Expect(output.CodeRepo.PlatformBuildConfig).To(Equal(&build.PlatformBuildConfig{
			BuilderImage: "golang:1.24",
			RunnerImage:  "debian:12",
		}))
	})

	DescribeTable(
		"Platform build config validation",
		func(config *serializer.PlatformBuildConfigInput, expectedErr string) {
			_, err := (&serializer.BuildConfigInput{
				SourceType: "codeRepository",
				RepoBuildConfig: &serializer.RepoBuildConfigInput{
					Type:                "TGit",
					RepoAlias:           "demo",
					RepoURL:             "https://example.com/repo.git",
					DefaultBranch:       "main",
					ImageBuildMode:      "platform",
					PlatformBuildConfig: config,
				},
			}).ToModel("app-id")

			Expect(err).To(MatchError(expectedErr))
		},
		Entry(
			"missing platform build config",
			nil,
			"buildConfig.repoBuildConfig.platformBuildConfig is required",
		),
		Entry(
			"empty command item",
			&serializer.PlatformBuildConfigInput{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &serializer.BuildCommandsInput{
					PreBuild: []string{""},
				},
			},
			"buildConfig.repoBuildConfig.platformBuildConfig.commands.preBuild[0] is required",
		),
	)

	It("converts helm spec input to model", func() {
		username := lo.ToPtr("user")
		password := lo.ToPtr("pass")

		output := (&serializer.HelmSpecInput{
			HelmSource: &serializer.HelmSourceInput{
				RepoType:   string(bkmsapp.HelmSourceRepoTypeHelm),
				ValueFiles: []string{"values.yaml"},
				HelmRepoConfig: &serializer.HelmRepoConfigInput{
					RepoURL:   "https://example.com/charts",
					ChartName: "demo",
					Username:  username,
					Password:  password,
				},
			},
		}).ToModel()

		Expect(output).To(Equal(&bkmsapp.HelmSpec{
			HelmSource: &bkmsapp.HelmSource{
				RepoType:   bkmsapp.HelmSourceRepoTypeHelm,
				ValueFiles: []string{"values.yaml"},
				HelmRepoConfig: &bkmsapp.HelmRepoConfig{
					RepoURL:   "https://example.com/charts",
					ChartName: "demo",
					Username:  "user",
					Password:  "pass",
				},
			},
		}))
	})
})

var _ = Describe("Platform build serializer edge cases", func() {
	buildRepoInput := func(mut func(r *serializer.RepoBuildConfigInput)) *serializer.BuildConfigInput {
		repo := &serializer.RepoBuildConfigInput{
			Type:           "TGit",
			RepoAlias:      "demo",
			RepoURL:        "https://example.com/repo.git",
			DefaultBranch:  "main",
			ImageBuildMode: "platform",
			PlatformBuildConfig: &serializer.PlatformBuildConfigInput{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &serializer.BuildCommandsInput{
					Build: []string{"go build -o app ./cmd/server"},
					Start: "./app",
				},
			},
		}
		if mut != nil {
			mut(repo)
		}
		return &serializer.BuildConfigInput{
			SourceType:      "codeRepository",
			RepoBuildConfig: repo,
		}
	}

	Context("mutual exclusion between platform and repository fields", func() {
		It("rejects platform mode when dockerfile field is not empty", func() {
			_, err := buildRepoInput(func(r *serializer.RepoBuildConfigInput) {
				r.Dockerfile = "build/Dockerfile"
			}).ToModel("app-id")

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.dockerfile must be empty when imageBuildMode is platform",
			)))
		})

		It("accepts platform mode with sourceDir set and empty dockerfile", func() {
			cfg, err := buildRepoInput(func(r *serializer.RepoBuildConfigInput) {
				r.SourceDir = "cmd/server"
			}).ToModel("app-id")

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.CodeRepo.SourceDir).To(Equal("cmd/server"))
			Expect(cfg.CodeRepo.ImageBuildMode).To(Equal(build.ImageBuildModePlatform))
		})

		It("rejects repository dockerfile mode when platform build config is provided", func() {
			_, err := (&serializer.BuildConfigInput{
				SourceType: "codeRepository",
				RepoBuildConfig: &serializer.RepoBuildConfigInput{
					Type:           "TGit",
					RepoAlias:      "demo",
					RepoURL:        "https://example.com/repo.git",
					DefaultBranch:  "main",
					ImageBuildMode: "repositoryDockerfile",
					PlatformBuildConfig: &serializer.PlatformBuildConfigInput{
						BuilderImage: "golang:1.24",
						RunnerImage:  "debian:12",
						Commands: &serializer.BuildCommandsInput{
							Start: "./app",
						},
					},
				},
			}).ToModel("app-id")

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig " +
					"must be empty when imageBuildMode is repositoryDockerfile",
			)))
		})
	})

	Context("length and content validations for commands", func() {
		It("rejects command list exceeding the max count", func() {
			overflow := make([]string, 33)
			for i := range overflow {
				overflow[i] = "echo cmd"
			}
			_, err := buildRepoInput(func(r *serializer.RepoBuildConfigInput) {
				r.PlatformBuildConfig.Commands.PreBuild = overflow
			}).ToModel("app-id")

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.commands.preBuild length must not exceed",
			)))
		})

		It("rejects a command item containing newline characters", func() {
			_, err := buildRepoInput(func(r *serializer.RepoBuildConfigInput) {
				r.PlatformBuildConfig.Commands.Build = []string{"go build\nrm -rf /"}
			}).ToModel("app-id")

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig" +
					".commands.build[0] must not contain newline characters",
			)))
		})
	})
})
