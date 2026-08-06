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

// Package serializer 定义构建配置相关的 Gin input/output 序列化结构和转换方法。
package serializer

import (
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	buildslz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/build/serializer"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
)

// BuildConfigInput is the build configuration input.
type BuildConfigInput struct {
	// 来源类型
	SourceType string `json:"sourceType" binding:"required,oneof=imageRegistry codeRepository pipeline"`
	// 镜像仓库配置
	ImageBuildConfig *ImageBuildConfigInput `json:"imageBuildConfig"`
	// 代码仓库配置
	RepoBuildConfig *RepoBuildConfigInput `json:"repoBuildConfig"`
	// 流水线配置
	PipelineBuildConfig *PipelineBuildConfigInput `json:"pipelineBuildConfig"`
}

// ToModel 将 BuildConfigInput 转换为 build.Config 领域模型
func (b *BuildConfigInput) ToModel(appID string) (*build.Config, error) {
	cfg := &build.Config{
		AppID:      appID,
		SourceType: build.SourceType(b.SourceType),
		// 添加默认镜像 Tag 配置：自定义类型 + 仅带构建时间
		TagConfig: build.TagConfig{
			Type: build.VersionTypeCustom,
			CustomOpts: &build.CustomTagOpts{
				Prefix:        "",
				WithRevision:  false,
				WithBuildTime: true,
			},
		},
	}

	switch cfg.SourceType {
	// 来源为镜像仓库时
	case build.SourceTypeImageRegistry:
		if b.ImageBuildConfig == nil {
			return nil, errors.New("buildConfig.imageBuildConfig is required")
		}
		cfg.Image = &build.ImageConfig{
			Name:     b.ImageBuildConfig.Name,
			Username: lo.FromPtr(b.ImageBuildConfig.Username),
			Password: lo.FromPtr(b.ImageBuildConfig.Password),
		}
	// 来源为代码仓库时
	case build.SourceTypeCodeRepository:
		if b.RepoBuildConfig == nil {
			return nil, errors.New("buildConfig.repoBuildConfig is required")
		}
		repoBuildConfig, err := b.RepoBuildConfig.ToModel()
		if err != nil {
			return nil, err
		}
		// 指定 PipelineType 为 Dockerfile
		cfg.PipelineType = string(bkci.PipelineTypeDockerfile)
		cfg.CodeRepo = repoBuildConfig
	// 来源为流水线时
	case build.SourceTypePipeline:
		if b.PipelineBuildConfig == nil {
			return nil, errors.New("buildConfig.pipelineBuildConfig is required")
		}
		// 特别指定 PipelineType 为流水线 ID（唯一性）
		cfg.PipelineType = b.PipelineBuildConfig.PipelineID
		cfg.Pipeline = &build.PipelineConfig{
			PipelineID: b.PipelineBuildConfig.PipelineID,
			Params:     b.PipelineBuildConfig.Params,
		}
	default:
		return nil, errors.Errorf("unsupported source type: %s", b.SourceType)
	}
	return cfg, nil
}

// ImageBuildConfigInput is the image registry build config input.
type ImageBuildConfigInput = buildslz.ImageBuildConfigInput

// RepoBuildConfigInput is the code repository build config input.
type RepoBuildConfigInput = buildslz.RepositoryBuildConfigInput

// PlatformBuildConfigInput is the platform build config input.
type PlatformBuildConfigInput = buildslz.PlatformBuildConfigInput

// BuildCommandsInput is the platform build commands input.
type BuildCommandsInput = buildslz.BuildCommandsInput

// PipelineBuildConfigInput is the pipeline build config input.
type PipelineBuildConfigInput = buildslz.PipelineBuildConfigInput

// -----------------------------------------------------------------------------
// Build Config Output
// -----------------------------------------------------------------------------

// BuildConfigOutputObj is the build config output.
type BuildConfigOutputObj struct {
	// 来源类型
	SourceType string `json:"sourceType"`
	// 镜像 Tag 配置
	TagConfig *TagConfigOutputObj `json:"tagConfig,omitempty"`
	// 镜像仓库配置
	ImageBuildConfig *ImageBuildConfigOutputObj `json:"imageBuildConfig,omitempty"`
	// 代码仓库配置
	RepoBuildConfig *RepoBuildConfigOutputObj `json:"repoBuildConfig,omitempty"`
	// 流水线配置
	PipelineBuildConfig *PipelineBuildConfigOutputObj `json:"pipelineBuildConfig,omitempty"`
}

// TagConfigOutputObj is the tag config output.
type TagConfigOutputObj struct {
	// 版本号类型
	Type string `json:"type"`
	// 自定义 Tag 选项
	CustomOpts *CustomTagOptsOutputObj `json:"customOpts,omitempty"`
}

// CustomTagOptsOutputObj is the custom tag opts output.
type CustomTagOptsOutputObj struct {
	// 自定义前缀
	Prefix string `json:"prefix"`
	// 是否包含分支/Tag 名称
	WithRevision bool `json:"withRevision"`
	// 是否包含构建时间
	WithBuildTime bool `json:"withBuildTime"`
}

// ImageBuildConfigOutputObj is the image build config output.
type ImageBuildConfigOutputObj struct {
	// 镜像名称
	Name string `json:"name"`
	// 用户名
	Username string `json:"username"`
}

// RepoBuildConfigOutputObj is the repo build config output.
type RepoBuildConfigOutputObj struct {
	// 代码库类型
	Type string `json:"type"`
	// 代码库别名
	RepoAlias string `json:"repoAlias"`
	// 代码库地址
	RepoURL string `json:"repoURL"`
	// 默认分支
	DefaultBranch string `json:"defaultBranch"`
	// 源码目录
	SourceDir string `json:"sourceDir"`
	// Dockerfile 文件路径，仅 imageBuildMode=repositoryDockerfile 时有效
	Dockerfile string `json:"dockerfile"`
	// DockerBuildArgs Docker 构建参数
	DockerBuildArgs map[string]string `json:"dockerBuildArgs,omitempty"`
	// ImageBuildMode 镜像构建方式：repositoryDockerfile 表示使用仓库 Dockerfile，platform 表示平台通用构建
	ImageBuildMode string `json:"imageBuildMode"`
	// PlatformBuildConfig 平台通用构建配置，仅 imageBuildMode=platform 时返回
	PlatformBuildConfig *PlatformBuildConfigOutputObj `json:"platformBuildConfig,omitempty"`
}

// PlatformBuildConfigOutputObj is the platform build config output.
type PlatformBuildConfigOutputObj struct {
	// 构建阶段基础镜像
	BuilderImage string `json:"builderImage"`
	// 运行阶段基础镜像
	RunnerImage string `json:"runnerImage"`
	// 命令配置
	Commands *BuildCommandsOutputObj `json:"commands,omitempty"`
}

// BuildCommandsOutputObj is the platform build commands output.
type BuildCommandsOutputObj struct {
	// 编译前置命令列表
	PreBuild []string `json:"preBuild"`
	// 编译命令列表
	Build []string `json:"build"`
	// 运行环境命令列表
	RuntimeEnv []string `json:"runtimeEnv"`
	// 启动命令
	Start string `json:"start"`
}

// PipelineBuildConfigOutputObj is the pipeline build config output.
type PipelineBuildConfigOutputObj struct {
	// 流水线 ID
	PipelineID string `json:"pipelineID"`
	// 构建流水线参数
	Params map[string]string `json:"params"`
}

// FromModel fills output fields from a build config model.
func (o *BuildConfigOutputObj) FromModel(cfg *build.Config) *BuildConfigOutputObj {
	if cfg == nil {
		return nil
	}
	*o = BuildConfigOutputObj{
		SourceType: string(cfg.SourceType),
	}
	// TagConfig
	o.TagConfig = &TagConfigOutputObj{Type: string(cfg.TagConfig.Type)}
	if cfg.TagConfig.CustomOpts != nil {
		o.TagConfig.CustomOpts = &CustomTagOptsOutputObj{
			Prefix:        cfg.TagConfig.CustomOpts.Prefix,
			WithRevision:  cfg.TagConfig.CustomOpts.WithRevision,
			WithBuildTime: cfg.TagConfig.CustomOpts.WithBuildTime,
		}
	}
	// Image
	if cfg.Image != nil {
		o.ImageBuildConfig = &ImageBuildConfigOutputObj{
			Name:     cfg.Image.Name,
			Username: cfg.Image.Username,
		}
	}
	// CodeRepo
	if cfg.CodeRepo != nil {
		imageBuildMode := cfg.CodeRepo.EffectiveImageBuildMode()
		o.RepoBuildConfig = &RepoBuildConfigOutputObj{
			Type:            string(cfg.CodeRepo.Type),
			RepoAlias:       cfg.CodeRepo.RepoAlias,
			RepoURL:         cfg.CodeRepo.RepoURL,
			DefaultBranch:   cfg.CodeRepo.DefaultBranch,
			SourceDir:       cfg.CodeRepo.SourceDir,
			Dockerfile:      cfg.CodeRepo.Dockerfile,
			DockerBuildArgs: cfg.CodeRepo.DockerBuildArgs,
			ImageBuildMode:  string(imageBuildMode),
		}
		if platformCfg := cfg.CodeRepo.PlatformBuildConfig; platformCfg != nil {
			o.RepoBuildConfig.PlatformBuildConfig = &PlatformBuildConfigOutputObj{
				BuilderImage: platformCfg.BuilderImage,
				RunnerImage:  platformCfg.RunnerImage,
			}
			if platformCfg.Commands != nil {
				o.RepoBuildConfig.PlatformBuildConfig.Commands = &BuildCommandsOutputObj{
					PreBuild: emptySliceIfNil(
						platformCfg.Commands.PreBuild,
					),
					Build: emptySliceIfNil(
						platformCfg.Commands.Build,
					),
					RuntimeEnv: emptySliceIfNil(
						platformCfg.Commands.RuntimeEnv,
					),
					Start: platformCfg.Commands.Start,
				}
			}
		}
	}
	// Pipeline
	if cfg.Pipeline != nil {
		o.PipelineBuildConfig = &PipelineBuildConfigOutputObj{
			PipelineID: cfg.Pipeline.PipelineID,
			Params:     cfg.Pipeline.Params,
		}
	}
	return o
}
