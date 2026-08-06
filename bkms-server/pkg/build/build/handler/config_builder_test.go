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

package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	buildserializer "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/serializer"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
)

var _ = Describe("buildConfigFromInput", func() {
	platformBuildConfigInput := func(
		mut func(*buildserializer.RepositoryBuildConfigInput),
	) buildserializer.UpdateBuildConfigInput {
		codeRepo := &buildserializer.RepositoryBuildConfigInput{
			Type:           string(imagebuild.RepositoryTypeTGit),
			DefaultBranch:  "main",
			RepoAlias:      "demo",
			RepoURL:        "https://git.example.com/demo",
			ImageBuildMode: string(imagebuild.ImageBuildModePlatform),
			PlatformBuildConfig: &buildserializer.PlatformBuildConfigInput{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &buildserializer.BuildCommandsInput{
					Build: []string{"go build -o app ./cmd/server"},
					Start: "./app",
				},
			},
		}
		if mut != nil {
			mut(codeRepo)
		}
		return buildserializer.UpdateBuildConfigInput{
			SourceType: string(imagebuild.SourceTypeCodeRepository),
			CodeRepo:   codeRepo,
		}
	}

	It("builds a code repository config with custom tag options", func() {
		cfg, err := buildConfigFromInput("demo-app", buildserializer.UpdateBuildConfigInput{
			SourceType: string(imagebuild.SourceTypeCodeRepository),
			TagConfig: &buildserializer.TagConfigInput{
				Type: string(imagebuild.VersionTypeCustom),
				CustomOpts: &buildserializer.CustomTagOptsInput{
					Prefix:        "pre",
					WithRevision:  true,
					WithBuildTime: true,
				},
			},
			CodeRepo: &buildserializer.RepositoryBuildConfigInput{
				Type:            string(imagebuild.RepositoryTypeTGit),
				DefaultBranch:   "main",
				RepoAlias:       "demo",
				RepoURL:         "https://git.example.com/demo",
				SourceDir:       "cmd/server",
				Dockerfile:      "Dockerfile.prod",
				DockerBuildArgs: map[string]string{"FOO": "bar"},
			},
		}, nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.AppID).To(Equal("demo-app"))
		Expect(cfg.SourceType).To(Equal(imagebuild.SourceTypeCodeRepository))
		Expect(cfg.PipelineType).To(Equal("dockerfile"))
		Expect(cfg.TagConfig.Type).To(Equal(imagebuild.VersionTypeCustom))
		Expect(cfg.TagConfig.CustomOpts).NotTo(BeNil())
		Expect(cfg.TagConfig.CustomOpts.Prefix).To(Equal("pre"))
		Expect(cfg.CodeRepo).NotTo(BeNil())
		Expect(cfg.CodeRepo.RepoAlias).To(Equal("demo"))
		Expect(cfg.CodeRepo.DockerBuildArgs).To(HaveKeyWithValue("FOO", "bar"))
	})

	DescribeTable(
		"builds code repository config with sourceDir validation",
		func(sourceDir, expectedErr string) {
			cfg, err := buildConfigFromInput("demo-app", buildserializer.UpdateBuildConfigInput{
				SourceType: string(imagebuild.SourceTypeCodeRepository),
				CodeRepo: &buildserializer.RepositoryBuildConfigInput{
					Type:          string(imagebuild.RepositoryTypeTGit),
					DefaultBranch: "main",
					RepoAlias:     "demo",
					RepoURL:       "https://git.example.com/demo",
					SourceDir:     sourceDir,
				},
			}, nil)

			if expectedErr != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(expectedErr))
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

	It("builds a platform build repository config", func() {
		cfg, err := buildConfigFromInput("demo-app", buildserializer.UpdateBuildConfigInput{
			SourceType: string(imagebuild.SourceTypeCodeRepository),
			CodeRepo: &buildserializer.RepositoryBuildConfigInput{
				Type:           string(imagebuild.RepositoryTypeTGit),
				DefaultBranch:  "main",
				RepoAlias:      "demo",
				RepoURL:        "https://git.example.com/demo",
				ImageBuildMode: string(imagebuild.ImageBuildModePlatform),
				PlatformBuildConfig: &buildserializer.PlatformBuildConfigInput{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
					Commands: &buildserializer.BuildCommandsInput{
						PreBuild:   []string{"go mod download"},
						Build:      []string{"go build -o app ./cmd/server"},
						RuntimeEnv: []string{"apt-get update"},
						Start:      "./app",
					},
				},
			},
		}, nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.CodeRepo).NotTo(BeNil())
		Expect(cfg.CodeRepo.ImageBuildMode).To(Equal(imagebuild.ImageBuildModePlatform))
		Expect(cfg.CodeRepo.PlatformBuildConfig).To(Equal(&imagebuild.PlatformBuildConfig{
			BuilderImage: "golang:1.24",
			RunnerImage:  "debian:12",
			Commands: &imagebuild.BuildCommands{
				PreBuild:   []string{"go mod download"},
				Build:      []string{"go build -o app ./cmd/server"},
				RuntimeEnv: []string{"apt-get update"},
				Start:      "./app",
			},
		}))
	})

	It("builds a platform build repository config without commands", func() {
		cfg, err := buildConfigFromInput("demo-app", platformBuildConfigInput(
			func(input *buildserializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig.Commands = nil
			},
		), nil)

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.CodeRepo.PlatformBuildConfig).To(Equal(&imagebuild.PlatformBuildConfig{
			BuilderImage: "golang:1.24",
			RunnerImage:  "debian:12",
		}))
	})

	It("requires platform build config for platform build mode", func() {
		_, err := buildConfigFromInput("demo-app", buildserializer.UpdateBuildConfigInput{
			SourceType: string(imagebuild.SourceTypeCodeRepository),
			CodeRepo: &buildserializer.RepositoryBuildConfigInput{
				Type:           string(imagebuild.RepositoryTypeTGit),
				DefaultBranch:  "main",
				RepoAlias:      "demo",
				RepoURL:        "https://git.example.com/demo",
				ImageBuildMode: string(imagebuild.ImageBuildModePlatform),
			},
		}, nil)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("buildConfig.repoBuildConfig.platformBuildConfig is required"))
	})

	It("rejects repository dockerfile mode when platform build config is provided", func() {
		_, err := buildConfigFromInput("demo-app", platformBuildConfigInput(
			func(input *buildserializer.RepositoryBuildConfigInput) {
				input.ImageBuildMode = string(imagebuild.ImageBuildModeRepositoryDockerfile)
			},
		), nil)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"buildConfig.repoBuildConfig.platformBuildConfig must be empty when imageBuildMode is repositoryDockerfile",
		))
	})

	It("rejects command lists exceeding the max count", func() {
		commands := make([]string, imagebuild.MaxPlatformBuildCommandCount+1)
		for i := range commands {
			commands[i] = "echo ok"
		}
		_, err := buildConfigFromInput("demo-app", platformBuildConfigInput(
			func(input *buildserializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig.Commands.PreBuild = commands
			},
		), nil)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"buildConfig.repoBuildConfig.platformBuildConfig.commands.preBuild length must not exceed",
		))
	})

	It("rejects commands containing only whitespace", func() {
		_, err := buildConfigFromInput("demo-app", platformBuildConfigInput(
			func(input *buildserializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig.Commands.Start = "   "
			},
		), nil)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			"buildConfig.repoBuildConfig.platformBuildConfig.commands.start must not be blank",
		))
	})

	It("reuses stored image credentials when the request omits them", func() {
		cfg, err := buildConfigFromInput("demo-app", buildserializer.UpdateBuildConfigInput{
			SourceType: string(imagebuild.SourceTypeImageRegistry),
			Image: &buildserializer.ImageBuildConfigInput{
				Name: "example/demo",
			},
		}, &imagebuild.Config{
			Image: &imagebuild.ImageConfig{
				Username: "kept-user",
				Password: "kept-pass",
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Image).NotTo(BeNil())
		Expect(cfg.Image.Username).To(Equal("kept-user"))
		Expect(cfg.Image.Password).To(Equal("kept-pass"))
	})

	It("rejects an unusable custom tag configuration", func() {
		_, err := buildConfigFromInput("demo-app", buildserializer.UpdateBuildConfigInput{
			SourceType: string(imagebuild.SourceTypePipeline),
			TagConfig: &buildserializer.TagConfigInput{
				Type:       string(imagebuild.VersionTypeCustom),
				CustomOpts: &buildserializer.CustomTagOptsInput{},
			},
			Pipeline: &buildserializer.PipelineBuildConfigInput{
				PipelineID: "pipe-1",
			},
		}, nil)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("custom tag must have at least one of"))
	})

	It("requires source-specific configuration", func() {
		_, err := buildConfigFromInput("demo-app", buildserializer.UpdateBuildConfigInput{
			SourceType: string(imagebuild.SourceTypePipeline),
		}, nil)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("pipeline field is required"))
	})
})
