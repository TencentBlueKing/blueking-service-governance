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

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
)

var _ = Describe("Service build validation", func() {
	var (
		ctx          context.Context
		app          *bkmsapp.Application
		cfg          *imagebuild.Config
		oldConfig    *config.Config
		buildService *Service
	)

	BeforeEach(func() {
		ctx = context.Background()
		oldConfig = config.G
		config.G = &config.Config{}
		config.G.ImageBuild.ToolchainBaseURL = "https://bkrepo.example.com/image-build-toolchain"
		app = &bkmsapp.Application{
			ID:          "test-app",
			Name:        "test-app",
			WorkspaceID: "test-workspace",
			Type:        bkmsapp.AppTypeTRPC,
			TrpcSpec:    &bkmsapp.TrpcSpec{Language: appmodel.LanguageGo},
		}
		cfg = &imagebuild.Config{
			SourceType: imagebuild.SourceTypeCodeRepository,
			CodeRepo: &imagebuild.RepositoryConfig{
				ImageBuildMode: imagebuild.ImageBuildModePlatform,
				PlatformBuildConfig: &imagebuild.PlatformBuildConfig{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
				},
			},
		}
		buildService = &Service{imageReferenceValidator: &workloadruntime.ImageReferenceValidator{}}
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		config.G = oldConfig
	})

	It("should validate platform generated Dockerfile build before start", func() {
		mockey.Mock(ValidatePlatformBuildImages).Return(nil).Build()

		err := buildService.validateBeforeBuild(ctx, app, cfg)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should reject platform generated Dockerfile build without toolchain base URL", func() {
		config.G.ImageBuild.ToolchainBaseURL = ""

		err := buildService.validateBeforeBuild(ctx, app, cfg)

		Expect(err).To(MatchError(ContainSubstring("imageBuild.toolchainBaseURL is required")))
	})

	It("should reject platform generated Dockerfile build for non-Go TRPC applications", func() {
		app.TrpcSpec.Language = appmodel.LanguageCpp

		err := buildService.validateBeforeBuild(ctx, app, cfg)

		Expect(err).To(MatchError(ContainSubstring("only supports Go language")))
	})

	It("should reject platform generated Dockerfile build for non-TRPC app types", func() {
		app.Type = bkmsapp.AppTypeTAF
		app.TrpcSpec = nil

		err := buildService.validateBeforeBuild(ctx, app, cfg)

		Expect(err).To(MatchError(ContainSubstring("does not support app type taf")))
	})

	It("should wrap image validation errors", func() {
		mockey.Mock(ValidatePlatformBuildImages).Return(errors.New("runtime image unavailable")).Build()

		err := buildService.validateBeforeBuild(ctx, app, cfg)

		Expect(err).To(MatchError(ContainSubstring("validate platform generated Dockerfile build images")))
	})

	It("should skip repository Dockerfile mode", func() {
		cfg.CodeRepo.ImageBuildMode = imagebuild.ImageBuildModeRepositoryDockerfile
		cfg.CodeRepo.PlatformBuildConfig = nil

		err := buildService.validateBeforeBuild(ctx, app, cfg)

		Expect(err).NotTo(HaveOccurred())
	})

	// validateBeforeBuild 用 pkg/errors 逐层包装校验错误，handler 依赖包装后仍能用 errors.Is
	// 区分「镜像填错」与「镜像源故障」来选 HTTP 错误码，因此断言真实调用链上的归因没有丢
	DescribeTable("keeps the failure attribution through the wrap chain",
		func(validateErr error, wantReferenceInvalid, wantRegistryFailure bool) {
			const (
				customImageName = "docker.bkrepo.example.com/demo/repo/my-golang"
				customImage     = customImageName + ":1.0"
			)
			cfg.CodeRepo.PlatformBuildConfig.BuilderImage = customImage
			buildService.customImageChecker = &stubCustomChecker{
				matches:      map[string]bool{customImageName: true},
				validateErrs: map[string]error{customImage: validateErr},
			}

			err := buildService.validateBeforeBuild(ctx, app, cfg)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("validate platform generated Dockerfile build images")))
			Expect(IsImageReferenceInvalid(err)).To(Equal(wantReferenceInvalid))
			Expect(IsImageRegistryFailure(err)).To(Equal(wantRegistryFailure))
		},
		Entry("image name not found is a user input problem",
			customruntime.ErrImageNameNotFound, true, false),
		Entry("image tag not found is a user input problem",
			customruntime.ErrImageTagNotFound, true, false),
		Entry("registry auth failure is not attributable to the user",
			customruntime.ErrRegistryAccessDenied, false, true),
		Entry("registry unreachable is not attributable to the user",
			customruntime.ErrRegistryAccessFailed, false, true),
	)

	Describe("custom image credential precheck", func() {
		const (
			customImageName = "docker.bkrepo.example.com/demo/repo/my-golang"
			customImage     = customImageName + ":1.0"
		)

		BeforeEach(func() {
			mockey.Mock(ValidatePlatformBuildImages).Return(nil).Build()
			cfg.CodeRepo.PlatformBuildConfig.BuilderImage = customImage
			buildService.customImageChecker = &stubCustomChecker{
				matches: map[string]bool{customImageName: true},
			}
		})

		It("rejects custom image build when bkCICredentialID is missing", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry: "docker.bkrepo.example.com/demo/repo",
				Username: "robot",
			}, nil).Build()

			err := buildService.validateBeforeBuild(ctx, app, cfg)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("complete workspace image registry setup")))
			Expect(errors.Is(err, ErrWorkspaceImageCredentialMissing)).To(BeTrue())
		})

		It("allows custom image build when bkCICredentialID exists", func() {
			mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&registry.ImageRegistry{
				Registry:         "docker.bkrepo.example.com/demo/repo",
				Username:         "robot",
				BkCICredentialID: "cred-123",
			}, nil).Build()

			err := buildService.validateBeforeBuild(ctx, app, cfg)

			Expect(err).NotTo(HaveOccurred())
		})

		It("does not block official images when bkCICredentialID is missing", func() {
			cfg.CodeRepo.PlatformBuildConfig.BuilderImage = "golang:1.24"
			cfg.CodeRepo.PlatformBuildConfig.RunnerImage = "debian:12"
			buildService.customImageChecker = &stubCustomChecker{matches: map[string]bool{}}

			err := buildService.validateBeforeBuild(ctx, app, cfg)

			Expect(err).NotTo(HaveOccurred())
		})
	})
})
