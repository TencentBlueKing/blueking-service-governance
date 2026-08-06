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
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
)

type platformBuildImageValidateCall struct {
	imageType workloadruntime.ImageType
	image     string
}

var _ = Describe("ValidatePlatformBuildImages", func() {
	platformConfig := func(mut func(*imagebuild.Config)) *imagebuild.Config {
		cfg := &imagebuild.Config{
			SourceType: imagebuild.SourceTypeCodeRepository,
			CodeRepo: &imagebuild.RepositoryConfig{
				ImageBuildMode: imagebuild.ImageBuildModePlatform,
				PlatformBuildConfig: &imagebuild.PlatformBuildConfig{
					BuilderImage: "golang:1.24",
					RunnerImage:  "debian:12",
				},
			},
		}
		if mut != nil {
			mut(cfg)
		}
		return cfg
	}

	mockValidateTaggedReference := func(
		calls *[]platformBuildImageValidateCall,
		errs map[string]error,
	) {
		mockey.Mock((*workloadruntime.ImageReferenceValidator).ValidateTaggedReference).To(
			func(
				_ *workloadruntime.ImageReferenceValidator,
				_ context.Context,
				imageType workloadruntime.ImageType,
				image string,
			) (*workloadruntime.ImageReference, error) {
				ref, err := workloadruntime.ParseTaggedImageReference(image)
				if err != nil {
					return nil, err
				}
				*calls = append(*calls, platformBuildImageValidateCall{imageType: imageType, image: image})
				if errs != nil {
					return ref, errs[image]
				}
				return ref, nil
			},
		).Build()
	}

	It("validates builder and runner images", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			validator := &workloadruntime.ImageReferenceValidator{}
			calls := []platformBuildImageValidateCall{}
			mockValidateTaggedReference(&calls, nil)

			err := ValidatePlatformBuildImages(context.Background(), validator, platformConfig(nil))

			Expect(err).NotTo(HaveOccurred())
			Expect(calls).To(Equal([]platformBuildImageValidateCall{
				{imageType: workloadruntime.ImageTypeBuilder, image: "golang:1.24"},
				{imageType: workloadruntime.ImageTypeRunner, image: "debian:12"},
			}))
		})
	})

	It("skips non-code-repository configs", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			validator := &workloadruntime.ImageReferenceValidator{}
			calls := []platformBuildImageValidateCall{}
			mockValidateTaggedReference(&calls, nil)

			err := ValidatePlatformBuildImages(context.Background(), validator, &imagebuild.Config{
				SourceType: imagebuild.SourceTypePipeline,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(calls).To(BeEmpty())
		})
	})

	It("skips repository dockerfile mode", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			validator := &workloadruntime.ImageReferenceValidator{}
			calls := []platformBuildImageValidateCall{}
			mockValidateTaggedReference(&calls, nil)

			err := ValidatePlatformBuildImages(
				context.Background(),
				validator,
				platformConfig(func(cfg *imagebuild.Config) {
					cfg.CodeRepo.ImageBuildMode = imagebuild.ImageBuildModeRepositoryDockerfile
					cfg.CodeRepo.PlatformBuildConfig = nil
				}),
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(calls).To(BeEmpty())
		})
	})

	It("returns field error when runtime image name does not exist", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			validator := &workloadruntime.ImageReferenceValidator{}
			calls := []platformBuildImageValidateCall{}
			mockValidateTaggedReference(&calls, map[string]error{
				"golang:1.24": workloadruntime.ErrRuntimeImageNotFound,
			})

			err := ValidatePlatformBuildImages(context.Background(), validator, platformConfig(nil))

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.builderImage runtime image golang does not exist",
			)))
		})
	})

	It("returns field error when snapshot tag does not exist", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			validator := &workloadruntime.ImageReferenceValidator{}
			calls := []platformBuildImageValidateCall{}
			mockValidateTaggedReference(&calls, map[string]error{
				"debian:12": workloadruntime.ErrRuntimeImageTagNotFound,
			})

			err := ValidatePlatformBuildImages(context.Background(), validator, platformConfig(nil))

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.runnerImage tag 12 does not exist in runtime image debian snapshot",
			)))
		})
	})

	It("wraps unexpected validator errors", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			validator := &workloadruntime.ImageReferenceValidator{}
			calls := []platformBuildImageValidateCall{}
			mockValidateTaggedReference(&calls, map[string]error{
				"golang:1.24": errors.New("store unavailable"),
			})

			err := ValidatePlatformBuildImages(context.Background(), validator, platformConfig(nil))

			Expect(err).To(MatchError(ContainSubstring(
				"validate buildConfig.repoBuildConfig.platformBuildConfig.builderImage: store unavailable",
			)))
		})
	})

	It("rejects image references without tag", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			validator := &workloadruntime.ImageReferenceValidator{}
			calls := []platformBuildImageValidateCall{}
			mockValidateTaggedReference(&calls, nil)

			err := ValidatePlatformBuildImages(
				context.Background(),
				validator,
				platformConfig(func(cfg *imagebuild.Config) {
					cfg.CodeRepo.PlatformBuildConfig.BuilderImage = "golang"
				}),
			)

			Expect(err).To(MatchError(ContainSubstring(
				"buildConfig.repoBuildConfig.platformBuildConfig.builderImage is invalid",
			)))
			Expect(calls).To(BeEmpty())
		})
	})
})
