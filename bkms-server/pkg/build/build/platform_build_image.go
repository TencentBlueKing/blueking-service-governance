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
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
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

// CustomImageChecker 判定并校验工作空间自定义镜像
type CustomImageChecker interface {
	MatchesWorkspaceRegistry(ctx context.Context, workspaceID, imageName string) (bool, error)
	ValidateTaggedReference(ctx context.Context, workspaceID, image string) error
}

// 编译期接口实现检查（断言放在接口定义处：customruntime 不能反向依赖本包）
var (
	_ CustomImageChecker = (*customruntime.ExistenceChecker)(nil)
	_ CustomImageChecker = (*customruntime.PersistManager)(nil)
)

// ValidatePlatformBuildImages 校验平台通用构建配置中的 builderImage 和 runnerImage
//
// builder / runner 各自独立判定：落在工作空间生效镜像源路径下走自定义存在性确认，
// 否则维持官方 runtime_images + 快照口径。
// customChecker 或 workspaceID 为空时全部按官方口径校验，便于无工作空间上下文时直接调用
func ValidatePlatformBuildImages(
	ctx context.Context,
	validator ImageReferenceValidator,
	customChecker CustomImageChecker,
	cfg *imagebuild.Config,
	workspaceID string,
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
		customChecker,
		workspaceID,
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
		customChecker,
		workspaceID,
		platformRunnerImageField,
		workloadruntime.ImageTypeRunner,
		platformCfg.RunnerImage,
	); err != nil {
		return errors.Wrap(err, "validate platform build runner image")
	}
	return nil
}

// validatePlatformBuildImage 按路径分流后校验一条 builder / runner 镜像
func validatePlatformBuildImage(
	ctx context.Context,
	validator ImageReferenceValidator,
	customChecker CustomImageChecker,
	workspaceID, field string,
	imageType workloadruntime.ImageType,
	image string,
) error {
	// 先解析出不含 tag 的仓库名，供路径归属判断；引用非法或缺 tag 直接按字段无效返回
	ref, parseErr := workloadruntime.ParseTaggedImageReference(image)
	if parseErr != nil {
		return errors.Wrapf(parseErr, "%s is invalid", field)
	}

	// 落在工作空间生效镜像源路径下才走自定义存在性确认，未绑定镜像源或路径不匹配则继续官方口径
	if customChecker != nil && workspaceID != "" {
		matches, err := customChecker.MatchesWorkspaceRegistry(ctx, workspaceID, ref.Name)
		if err != nil {
			return errors.Wrapf(err, "classify %s", field)
		}
		if matches {
			if err = customChecker.ValidateTaggedReference(ctx, workspaceID, image); err != nil {
				return newCustomImageValidateError(field, ref, err)
			}
			return nil
		}
	}

	// 官方口径：runtime_images 有该镜像，且快照里有该 tag；错误文案保持原字段语义
	validated, err := validator.ValidateTaggedReference(ctx, imageType, image)
	if err == nil {
		return nil
	}
	if validated == nil {
		return errors.Wrapf(err, "%s is invalid", field)
	}

	// 与自定义镜像一致地保留哨兵错误，供 handler 用 errors.Is 判定归因后选错误码
	switch {
	case errors.Is(err, workloadruntime.ErrRuntimeImageNotFound):
		return errors.Wrapf(err, "%s runtime image %s does not exist", field, validated.Name)
	case errors.Is(err, workloadruntime.ErrRuntimeImageTagNotFound):
		return errors.Wrapf(
			err,
			"%s tag %s does not exist in runtime image %s snapshot",
			field,
			validated.Tag,
			validated.Name,
		)
	default:
		return errors.Wrapf(err, "validate %s", field)
	}
}

// newCustomImageValidateError 把自定义镜像校验错误映射成带字段路径的文案。
//
// 一律保留底层哨兵错误：调用方需要用 errors.Is 区分「用户填错镜像引用」与「镜像源本身故障」，
// 两者对应的 HTTP 错误码不同，而错误码只在 handler 层决定
func newCustomImageValidateError(field string, ref *workloadruntime.ImageReference, err error) error {
	switch {
	case errors.Is(err, customruntime.ErrImageNameNotFound):
		return errors.Wrapf(err, "%s custom image %s does not exist in workspace registry", field, ref.Name)
	case errors.Is(err, customruntime.ErrImageTagNotFound):
		return errors.Wrapf(err, "%s tag %s does not exist in custom image %s", field, ref.Tag, ref.Name)
	case errors.Is(err, customruntime.ErrRegistryAccessDenied):
		return errors.Wrapf(err, "%s workspace registry auth failed", field)
	case errors.Is(err, customruntime.ErrRegistryAccessFailed):
		return errors.Wrapf(err, "%s workspace registry access failed", field)
	default:
		return errors.Wrapf(err, "validate %s", field)
	}
}

// IsImageRegistryFailure 判断镜像校验失败是否源于工作空间镜像源本身（鉴权失败或不可达）。
//
// 这类失败改镜像引用也解决不了，handler 应按内部错误上报，不能归为参数问题
func IsImageRegistryFailure(err error) bool {
	return errors.Is(err, customruntime.ErrRegistryAccessDenied) ||
		errors.Is(err, customruntime.ErrRegistryAccessFailed)
}

// IsImageReferenceInvalid 判断镜像校验失败是否为用户可自行修正的镜像引用问题。
//
// 覆盖官方运行时镜像与工作空间自定义镜像两条口径，handler 据此返回参数类错误
func IsImageReferenceInvalid(err error) bool {
	return errors.Is(err, workloadruntime.ErrRuntimeImageNotFound) ||
		errors.Is(err, workloadruntime.ErrRuntimeImageTagNotFound) ||
		errors.Is(err, customruntime.ErrImageNameNotFound) ||
		errors.Is(err, customruntime.ErrImageTagNotFound)
}
