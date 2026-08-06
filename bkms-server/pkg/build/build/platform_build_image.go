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

	"github.com/pkg/errors"

	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	workloadruntime "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/runtime"
)

const (
	platformBuildConfigField  = "buildConfig.repoBuildConfig.platformBuildConfig"
	platformBuilderImageField = platformBuildConfigField + ".builderImage"
	platformRunnerImageField  = platformBuildConfigField + ".runnerImage"
)

// ImageReferenceValidator 校验运行时镜像引用与快照 tag 是否存在
type ImageReferenceValidator interface {
	// ValidateTaggedReference 校验完整镜像引用是否属于指定类型的运行时镜像且 tag 已存在于快照中
	ValidateTaggedReference(
		ctx context.Context,
		imageType workloadruntime.ImageType,
		image string,
	) (*workloadruntime.ImageReference, error)
}

// ValidatePlatformBuildImages 校验平台通用构建配置中的 builderImage 和 runnerImage
func ValidatePlatformBuildImages(
	ctx context.Context,
	validator ImageReferenceValidator,
	cfg *imagebuild.Config,
) error {
	if cfg == nil || cfg.SourceType != imagebuild.SourceTypeCodeRepository || cfg.CodeRepo == nil {
		return nil
	}
	if cfg.CodeRepo.EffectiveImageBuildMode() != imagebuild.ImageBuildModePlatform {
		return nil
	}
	platformCfg := cfg.CodeRepo.PlatformBuildConfig
	if platformCfg == nil {
		return errors.New(platformBuildConfigField + " is required")
	}
	if validator == nil {
		return errors.New("platform build image validator is required")
	}

	// 校验 builder 镜像
	if err := validatePlatformBuildImage(
		ctx,
		validator,
		platformBuilderImageField,
		workloadruntime.ImageTypeBuilder,
		platformCfg.BuilderImage,
	); err != nil {
		return errors.Wrap(err, "validate platform build builder image")
	}
	// 校验 runner 镜像
	if err := validatePlatformBuildImage(
		ctx,
		validator,
		platformRunnerImageField,
		workloadruntime.ImageTypeRunner,
		platformCfg.RunnerImage,
	); err != nil {
		return errors.Wrap(err, "validate platform build runner image")
	}
	return nil
}

// validatePlatformBuildImage 校验平台构建相关镜像（builder/runner）
func validatePlatformBuildImage(
	ctx context.Context,
	validator ImageReferenceValidator,
	field string,
	imageType workloadruntime.ImageType,
	image string,
) error {
	ref, err := validator.ValidateTaggedReference(ctx, imageType, image)
	if err == nil {
		return nil
	}
	if ref == nil {
		return errors.Wrapf(err, "%s is invalid", field)
	}

	switch {
	case errors.Is(err, workloadruntime.ErrRuntimeImageNotFound):
		return errors.Errorf("%s runtime image %s does not exist", field, ref.Name)
	case errors.Is(err, workloadruntime.ErrRuntimeImageTagNotFound):
		return errors.Errorf("%s tag %s does not exist in runtime image %s snapshot", field, ref.Tag, ref.Name)
	default:
		return errors.Wrapf(err, "validate %s", field)
	}
}
