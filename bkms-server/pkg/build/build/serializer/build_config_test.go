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
	"strconv"
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
					ExtraFiles: []string{"data/key.pem", "certs"},
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
			ExtraFiles: []string{"data/key.pem", "certs"},
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

	Describe("Platform build extra files validation", func() {
		platformRepoInput := func(extraFiles []string) *serializer.RepositoryBuildConfigInput {
			return &serializer.RepositoryBuildConfigInput{
				Type:           string(imagebuild.RepositoryTypeTGit),
				DefaultBranch:  "main",
				RepoAlias:      "demo",
				RepoURL:        "https://git.example.com/demo",
				ImageBuildMode: string(imagebuild.ImageBuildModePlatform),
				PlatformBuildConfig: &serializer.PlatformBuildConfigInput{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
					ExtraFiles:   extraFiles,
				},
			}
		}

		It("accepts a valid extra files list", func() {
			err := platformRepoInput([]string{
				"data/key.pem",
				"certs",
				"data/*.pem",
				"*.json",
			}).ValidatePlatformBuildConfig()

			Expect(err).NotTo(HaveOccurred())
		})

		It("accepts empty extra files", func() {
			err := platformRepoInput(nil).ValidatePlatformBuildConfig()

			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects parent path segments", func() {
			err := platformRepoInput([]string{"data/key.pem", "foo/../bar"}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[1] must not contain '..'",
			)))
		})

		It("rejects absolute paths", func() {
			err := platformRepoInput([]string{"/etc/passwd"}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[0] must not start with '/'",
			)))
		})

		It("rejects duplicated paths after trim", func() {
			err := platformRepoInput([]string{"data/key.pem", "  data/key.pem  "}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[1] is duplicated",
			)))
		})

		It("rejects extra files exceeding the max count", func() {
			extraFiles := make([]string, imagebuild.MaxPlatformBuildExtraFileCount+1)
			for i := range extraFiles {
				extraFiles[i] = "file-" + strconv.Itoa(i)
			}
			err := platformRepoInput(extraFiles).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles length must not exceed",
			)))
		})

		It("rejects an extra file longer than the max length", func() {
			err := platformRepoInput([]string{strings.Repeat("a", imagebuild.MaxPlatformBuildExtraFileLen+1)}).
				ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[0] length must not exceed",
			)))
		})

		It("rejects extra files containing newline characters", func() {
			err := platformRepoInput([]string{"data/key.pem\ncerts"}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[0] must not contain newline characters",
			)))
		})

		It("rejects extra files containing whitespace", func() {
			err := platformRepoInput([]string{"my key.pem"}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[0] " +
					"must not contain whitespace or Dockerfile-unsafe characters",
			)))
		})

		It("rejects extra files that look like COPY options", func() {
			err := platformRepoInput([]string{"--from=builder"}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[0] must not start with '-'",
			)))
		})

		It("rejects copying the entire build context", func() {
			err := platformRepoInput([]string{"."}).ValidatePlatformBuildConfig()

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.extraFiles[0] must not copy the entire build context",
			)))
		})

		It("accepts extra files whose rune count is within the max length", func() {
			err := platformRepoInput([]string{strings.Repeat("文", imagebuild.MaxPlatformBuildExtraFileLen)}).
				ValidatePlatformBuildConfig()

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
