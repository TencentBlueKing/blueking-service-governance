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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/serializer"
	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
)

const testOverflowCommandCount = imagebuild.MaxPlatformBuildCommandCount + 1

var _ = Describe("Build config serializers", func() {
	DescribeTable(
		"AppURIInput validation",
		func(input serializer.AppURIInput, wantErr bool) {
			err := binding.Validator.ValidateStruct(input)
			if wantErr {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed on the 'uri_slug' tag"))
				return
			}
			Expect(err).NotTo(HaveOccurred())
		},
		Entry("accepts URI slug app IDs", serializer.AppURIInput{AppID: "app_123-Test"}, false),
		Entry("rejects app IDs with slash", serializer.AppURIInput{AppID: "app/test"}, true),
	)

	It("maps platform build config from model", func() {
		output := new(serializer.BuildConfigOutputObj).FromModel(&imagebuild.Config{
			AppID:      "demo-app",
			SourceType: imagebuild.SourceTypeCodeRepository,
			CodeRepo: &imagebuild.RepositoryConfig{
				Type:           imagebuild.RepositoryTypeTGit,
				RepoAlias:      "demo",
				RepoURL:        "https://git.example.com/demo",
				DefaultBranch:  "main",
				ImageBuildMode: imagebuild.ImageBuildModePlatform,
				PlatformBuildConfig: &imagebuild.PlatformBuildConfig{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
					Commands: &imagebuild.BuildCommands{
						PreBuild: []string{"go mod download"},
						Start:    "./app",
					},
				},
			},
		})

		Expect(output.CodeRepo.ImageBuildMode).To(Equal("platform"))
		Expect(output.CodeRepo.PlatformBuildConfig).To(Equal(&serializer.PlatformBuildConfigOutputObj{
			BuilderImage: "golang:1.24",
			RunnerImage:  "debian:12",
			Commands: &serializer.BuildCommandsOutputObj{
				PreBuild:   []string{"go mod download"},
				Build:      []string{},
				RuntimeEnv: []string{},
				Start:      "./app",
			},
		}))
	})

	Describe("Platform build config validation", func() {
		platformRepoInput := func(mut func(*serializer.RepositoryBuildConfigInput)) *serializer.RepositoryBuildConfigInput {
			input := &serializer.RepositoryBuildConfigInput{
				Type:           string(imagebuild.RepositoryTypeTGit),
				DefaultBranch:  "main",
				RepoAlias:      "demo",
				RepoURL:        "https://git.example.com/demo",
				ImageBuildMode: string(imagebuild.ImageBuildModePlatform),
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
				mut(input)
			}
			return input
		}

		It("rejects missing platform build config", func() {
			err := platformRepoInput(func(input *serializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig = nil
			}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError("buildConfig.repoBuildConfig.platformBuildConfig is required"))
		})

		It("rejects repository dockerfile mode when platform build config is provided", func() {
			err := platformRepoInput(func(input *serializer.RepositoryBuildConfigInput) {
				input.ImageBuildMode = string(imagebuild.ImageBuildModeRepositoryDockerfile)
			}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(
				"buildConfig.repoBuildConfig.platformBuildConfig " +
					"must be empty when imageBuildMode is repositoryDockerfile",
			))
		})

		It("accepts missing platform build commands", func() {
			err := platformRepoInput(func(input *serializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig.Commands = nil
			}).ValidatePlatformBuildConfig()

			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts empty start command", func() {
			err := platformRepoInput(func(input *serializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig.Commands.Start = ""
			}).ValidatePlatformBuildConfig()

			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects command lists exceeding the max count", func() {
			commands := make([]string, testOverflowCommandCount)
			for i := range commands {
				commands[i] = "echo ok"
			}
			err := platformRepoInput(func(input *serializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig.Commands.PreBuild = commands
			}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.commands.preBuild length must not exceed",
			)))
		})

		It("rejects commands containing only whitespace", func() {
			err := platformRepoInput(func(input *serializer.RepositoryBuildConfigInput) {
				input.PlatformBuildConfig.Commands.Start = strings.Repeat(" ", 3)
			}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(
				"buildConfig.repoBuildConfig.platformBuildConfig.commands.start must not be blank",
			))
		})
	})
})
