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

package serializer

import (
	"strings"
	"time"

	"github.com/pkg/errors"

	imagebuild "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

const (
	platformBuildConfigField       = "buildConfig.repoBuildConfig.platformBuildConfig"
	platformBuildCommandsField     = platformBuildConfigField + ".commands"
	repoBuildConfigDockerfileField = "buildConfig.repoBuildConfig.dockerfile"
	repoBuildConfigSourceDirField  = "buildConfig.repoBuildConfig.sourceDir"
)

// AppURIInput is the path input for APIs scoped by application.
type AppURIInput struct {
	// 应用 ID
	AppID string `uri:"appID" binding:"required,max=63,uri_slug"`
}

// CustomTagOptsInput is the JSON representation of custom tag options.
type CustomTagOptsInput struct {
	// 自定义前缀
	Prefix string `json:"prefix" binding:"omitempty,max=16"`
	// 是否拼接代码版本
	WithRevision bool `json:"withRevision"`
	// 是否拼接构建时间
	WithBuildTime bool `json:"withBuildTime"`
}

// TagConfigInput is the JSON representation of image tag config.
type TagConfigInput struct {
	// Tag 生成策略：semver 或 custom
	Type string `json:"type" binding:"required,oneof=semver custom"`
	// 自定义 Tag 配置，仅当 type=custom 时生效
	CustomOpts *CustomTagOptsInput `json:"customOpts,omitempty"`
}

// RepositoryBuildConfigInput is the JSON representation of source repo build config.
type RepositoryBuildConfigInput struct {
	// 代码仓库类型：TGit 或 GitHub
	Type string `json:"type" binding:"required,oneof=TGit GitHub"`
	// 默认构建分支
	DefaultBranch string `json:"defaultBranch" binding:"required,min=1,max=255"`
	// 蓝盾侧仓库别名
	RepoAlias string `json:"repoAlias" binding:"required,min=1,max=256"`
	// 代码仓库地址
	RepoURL string `json:"repoURL" binding:"required,min=1,max=512,url"`
	// 构建目录，空表示仓库根目录；非空时必须是仓库内相对路径，不能以 / 开头，也不能包含 .. 路径段
	SourceDir string `json:"sourceDir" binding:"omitempty,max=64"`
	// Dockerfile 路径，空表示默认路径。仅 imageBuildMode=repositoryDockerfile 时有效
	Dockerfile string `json:"dockerfile" binding:"omitempty,max=64"`
	// DockerBuildArgs Docker 构建参数
	DockerBuildArgs map[string]string `json:"dockerBuildArgs"`
	// ImageBuildMode 镜像构建方式：repositoryDockerfile 表示使用仓库 Dockerfile，platform 表示平台通用构建
	ImageBuildMode string `json:"imageBuildMode" binding:"omitempty,oneof=platform repositoryDockerfile"`
	// PlatformBuildConfig 平台通用构建配置，仅 imageBuildMode=platform 时有效；Dockerfile 是流水线内部中间产物
	PlatformBuildConfig *PlatformBuildConfigInput `json:"platformBuildConfig"`
}

// UsesNewImageBuildMode returns true when the request uses the new image build mode field.
func (r *RepositoryBuildConfigInput) UsesNewImageBuildMode() bool {
	return r != nil && r.ImageBuildMode != ""
}

// EffectiveImageBuildMode returns the image build mode selected by current input.
func (r *RepositoryBuildConfigInput) EffectiveImageBuildMode() imagebuild.ImageBuildMode {
	if r == nil {
		return imagebuild.ImageBuildModeRepositoryDockerfile
	}
	if r.ImageBuildMode != "" {
		return imagebuild.NormalizeImageBuildMode(imagebuild.ImageBuildMode(r.ImageBuildMode))
	}
	if r.PlatformBuildConfig != nil {
		return imagebuild.ImageBuildModePlatform
	}
	return imagebuild.ImageBuildModeRepositoryDockerfile
}

// ValidatePlatformBuildConfig validates image build mode related input fields.
func (r *RepositoryBuildConfigInput) ValidatePlatformBuildConfig() error {
	mode := r.EffectiveImageBuildMode()
	switch mode {
	case imagebuild.ImageBuildModePlatform:
		if r.Dockerfile != "" {
			return errors.Errorf(
				"%s must be empty when imageBuildMode is platform",
				repoBuildConfigDockerfileField,
			)
		}
		if r.PlatformBuildConfig == nil {
			return errors.New(platformBuildConfigField + " is required")
		}
		return r.PlatformBuildConfig.Validate()
	case imagebuild.ImageBuildModeRepositoryDockerfile:
		if r.PlatformBuildConfig != nil {
			return errors.Errorf(
				"%s must be empty when imageBuildMode is repositoryDockerfile",
				platformBuildConfigField,
			)
		}
		return nil
	default:
		return errors.Errorf("unsupported image build mode: %s", mode)
	}
}

// ToModel converts RepositoryBuildConfigInput to imagebuild.RepositoryConfig.
func (r *RepositoryBuildConfigInput) ToModel() (*imagebuild.RepositoryConfig, error) {
	if err := ValidateRepositorySourceDir(repoBuildConfigSourceDirField, r.SourceDir); err != nil {
		return nil, err
	}
	if err := r.ValidatePlatformBuildConfig(); err != nil {
		return nil, err
	}
	imageBuildMode := r.EffectiveImageBuildMode()
	var platformBuildConfig *imagebuild.PlatformBuildConfig
	if r.PlatformBuildConfig != nil {
		platformBuildConfig = r.PlatformBuildConfig.ToModel()
	}
	return &imagebuild.RepositoryConfig{
		Type:                imagebuild.RepositoryType(r.Type),
		RepoAlias:           r.RepoAlias,
		RepoURL:             r.RepoURL,
		SourceDir:           r.SourceDir,
		Dockerfile:          r.Dockerfile,
		DockerBuildArgs:     r.DockerBuildArgs,
		DefaultBranch:       r.DefaultBranch,
		ImageBuildMode:      imageBuildMode,
		PlatformBuildConfig: platformBuildConfig,
	}, nil
}

// ValidateRepositorySourceDir 校验代码仓库构建目录必须位于仓库内部
//
// 空字符串表示仓库根目录；非空时只允许仓库内相对路径，不允许绝对路径或包含 .. 路径段，避免构建路径逃逸
func ValidateRepositorySourceDir(field, sourceDir string) error {
	if sourceDir == "" {
		return nil
	}
	if strings.HasPrefix(sourceDir, "/") {
		return errors.Errorf("%s must be a relative path inside repository", field)
	}
	if strings.Contains(sourceDir, "..") {
		return errors.Errorf("%s must not contain '..' path segment", field)
	}
	return nil
}

// PlatformBuildConfigInput is the JSON representation of platform build config.
type PlatformBuildConfigInput struct {
	// 构建阶段基础镜像
	BuilderImage string `json:"builderImage"`
	// 运行阶段基础镜像
	RunnerImage string `json:"runnerImage"`
	// 命令配置
	Commands *BuildCommandsInput `json:"commands"`
}

// Validate validates platform build input fields.
func (c *PlatformBuildConfigInput) Validate() error {
	if c == nil {
		return errors.New(platformBuildConfigField + " is required")
	}
	return c.Commands.Validate()
}

// ToModel converts PlatformBuildConfigInput to imagebuild.PlatformBuildConfig.
func (c *PlatformBuildConfigInput) ToModel() *imagebuild.PlatformBuildConfig {
	if c == nil {
		return nil
	}
	var commands *imagebuild.BuildCommands
	if c.Commands != nil {
		commands = c.Commands.ToModel()
	}
	return &imagebuild.PlatformBuildConfig{
		BuilderImage: c.BuilderImage,
		RunnerImage:  c.RunnerImage,
		Commands:     commands,
	}
}

// BuildCommandsInput is the JSON representation of platform build commands.
type BuildCommandsInput struct {
	// 编译前置命令列表
	PreBuild []string `json:"preBuild"`
	// 编译命令列表
	Build []string `json:"build"`
	// 运行环境命令列表
	RuntimeEnv []string `json:"runtimeEnv"`
	// 启动命令
	Start string `json:"start"`
}

// Validate validates platform build command input fields.
func (c *BuildCommandsInput) Validate() error {
	if c == nil {
		return nil
	}
	if err := ValidatePlatformBuildCommands(platformBuildCommandsField, "preBuild", c.PreBuild); err != nil {
		return err
	}
	if err := ValidatePlatformBuildCommands(platformBuildCommandsField, "build", c.Build); err != nil {
		return err
	}
	if err := ValidatePlatformBuildCommands(platformBuildCommandsField, "runtimeEnv", c.RuntimeEnv); err != nil {
		return err
	}
	return ValidatePlatformBuildStart(platformBuildCommandsField+".start", c.Start)
}

// ToModel converts BuildCommandsInput to imagebuild.BuildCommands.
func (c *BuildCommandsInput) ToModel() *imagebuild.BuildCommands {
	if c == nil {
		return nil
	}
	return &imagebuild.BuildCommands{
		PreBuild:   c.PreBuild,
		Build:      c.Build,
		RuntimeEnv: c.RuntimeEnv,
		Start:      c.Start,
	}
}

// ValidatePlatformBuildCommands validates platform build command list input.
func ValidatePlatformBuildCommands(prefix, field string, commands []string) error {
	if len(commands) > imagebuild.MaxPlatformBuildCommandCount {
		return errors.Errorf(
			"%s.%s length must not exceed %d",
			prefix, field, imagebuild.MaxPlatformBuildCommandCount,
		)
	}
	for i, command := range commands {
		if strings.TrimSpace(command) == "" {
			return errors.Errorf("%s.%s[%d] is required", prefix, field, i)
		}
		if len(command) > imagebuild.MaxPlatformBuildCommandLen {
			return errors.Errorf(
				"%s.%s[%d] length must not exceed %d",
				prefix, field, i, imagebuild.MaxPlatformBuildCommandLen,
			)
		}
		if strings.ContainsAny(command, "\r\n") {
			return errors.Errorf("%s.%s[%d] must not contain newline characters", prefix, field, i)
		}
	}
	return nil
}

// ValidatePlatformBuildStart validates platform build start command input.
func ValidatePlatformBuildStart(field, command string) error {
	if command == "" {
		return nil
	}
	if strings.TrimSpace(command) == "" {
		return errors.New(field + " must not be blank")
	}
	if len(command) > imagebuild.MaxPlatformBuildCommandLen {
		return errors.Errorf("%s length must not exceed %d", field, imagebuild.MaxPlatformBuildCommandLen)
	}
	if strings.ContainsAny(command, "\r\n") {
		return errors.Errorf("%s must not contain newline characters", field)
	}
	return nil
}

// ImageBuildConfigInput is the JSON representation of image build config.
type ImageBuildConfigInput struct {
	// 镜像名称，不包含 Tag
	Name string `json:"name" binding:"required,min=1,max=512"`
	// 镜像仓库用户名
	Username *string `json:"username,omitempty" binding:"omitempty,max=64"`
	// 镜像仓库密码
	Password *string `json:"password,omitempty" binding:"omitempty,max=64"`
}

// PipelineBuildConfigInput is the JSON representation of pipeline build config.
type PipelineBuildConfigInput struct {
	// 蓝盾流水线 ID
	PipelineID string `json:"pipelineID" binding:"required,min=1"`
	// 流水线额外参数
	Params map[string]string `json:"params"`
}

// UpdateBuildConfigInput is the JSON body for updating build config.
type UpdateBuildConfigInput struct {
	// 构建来源：codeRepository / imageRegistry / pipeline
	SourceType string `json:"sourceType" binding:"required,oneof=codeRepository imageRegistry pipeline"`
	// 镜像 Tag 配置
	TagConfig *TagConfigInput `json:"tagConfig,omitempty"`
	// 代码仓库构建配置
	CodeRepo *RepositoryBuildConfigInput `json:"codeRepo,omitempty"`
	// 镜像仓库配置
	Image *ImageBuildConfigInput `json:"image,omitempty"`
	// 蓝盾流水线配置
	Pipeline *PipelineBuildConfigInput `json:"pipeline,omitempty"`
}

// CustomTagOptsOutput is the JSON representation of custom tag options.
type CustomTagOptsOutput struct {
	// 自定义前缀
	Prefix string `json:"prefix"`
	// 是否拼接代码版本
	WithRevision bool `json:"withRevision"`
	// 是否拼接构建时间
	WithBuildTime bool `json:"withBuildTime"`
}

// TagConfigOutput is the JSON representation of image tag config.
type TagConfigOutput struct {
	// Tag 生成策略
	Type string `json:"type"`
	// 自定义 Tag 配置
	CustomOpts *CustomTagOptsOutput `json:"customOpts,omitempty"`
}

// RepositoryBuildConfigOutputObj is the JSON representation of source repo build config.
type RepositoryBuildConfigOutputObj struct {
	// 代码仓库类型
	Type string `json:"type"`
	// 蓝盾侧仓库别名
	RepoAlias string `json:"repoAlias"`
	// 代码仓库地址
	RepoURL string `json:"repoURL"`
	// 构建目录
	SourceDir string `json:"sourceDir"`
	// Dockerfile 路径，仅 imageBuildMode=repositoryDockerfile 时有效
	Dockerfile string `json:"dockerfile"`
	// DockerBuildArgs Docker 构建参数
	DockerBuildArgs map[string]string `json:"dockerBuildArgs,omitempty"`
	// 默认构建分支
	DefaultBranch string `json:"defaultBranch"`
	// ImageBuildMode 镜像构建方式：repositoryDockerfile 表示使用仓库 Dockerfile，platform 表示平台通用构建
	ImageBuildMode string `json:"imageBuildMode"`
	// PlatformBuildConfig 平台通用构建配置，仅 imageBuildMode=platform 时返回
	PlatformBuildConfig *PlatformBuildConfigOutputObj `json:"platformBuildConfig,omitempty"`
}

// PlatformBuildConfigOutputObj is the JSON representation of platform build config.
type PlatformBuildConfigOutputObj struct {
	// 构建阶段基础镜像
	BuilderImage string `json:"builderImage"`
	// 运行阶段基础镜像
	RunnerImage string `json:"runnerImage"`
	// 命令配置
	Commands *BuildCommandsOutputObj `json:"commands,omitempty"`
}

// BuildCommandsOutputObj is the JSON representation of platform build commands.
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

// ImageBuildConfigOutputObj is the JSON representation of image build config.
type ImageBuildConfigOutputObj struct {
	// 镜像名称
	Name string `json:"name"`
	// 镜像仓库用户名
	Username string `json:"username"`
}

// BuildConfigOutputObj is the JSON representation of one build config.
type BuildConfigOutputObj struct {
	// 应用 ID
	AppID string `json:"appID"`
	// 构建来源
	SourceType string `json:"sourceType"`
	// 实际执行的流水线类型
	PipelineType string `json:"pipelineType"`
	// 镜像 Tag 配置
	TagConfig *TagConfigOutput `json:"tagConfig,omitempty"`
	// 代码仓库构建配置
	CodeRepo *RepositoryBuildConfigOutputObj `json:"codeRepo,omitempty"`
	// 镜像仓库配置
	Image *ImageBuildConfigOutputObj `json:"image,omitempty"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromModel fills output fields from a build config model.
func (o *BuildConfigOutputObj) FromModel(cfg *imagebuild.Config) *BuildConfigOutputObj {
	if cfg == nil {
		return nil
	}

	*o = BuildConfigOutputObj{
		AppID:        cfg.AppID,
		SourceType:   string(cfg.SourceType),
		PipelineType: cfg.PipelineType,
		TagConfig:    new(TagConfigOutput).FromModel(cfg.TagConfig),
		CreatedAt:    cfg.CreatedAt,
		UpdatedAt:    cfg.UpdatedAt,
	}
	if cfg.CodeRepo != nil {
		imageBuildMode := cfg.CodeRepo.EffectiveImageBuildMode()
		o.CodeRepo = &RepositoryBuildConfigOutputObj{
			Type:            string(cfg.CodeRepo.Type),
			RepoAlias:       cfg.CodeRepo.RepoAlias,
			RepoURL:         cfg.CodeRepo.RepoURL,
			SourceDir:       cfg.CodeRepo.SourceDir,
			Dockerfile:      cfg.CodeRepo.Dockerfile,
			DockerBuildArgs: cfg.CodeRepo.DockerBuildArgs,
			DefaultBranch:   cfg.CodeRepo.DefaultBranch,
			ImageBuildMode:  string(imageBuildMode),
		}
		if platformCfg := cfg.CodeRepo.PlatformBuildConfig; platformCfg != nil {
			o.CodeRepo.PlatformBuildConfig = &PlatformBuildConfigOutputObj{
				BuilderImage: platformCfg.BuilderImage,
				RunnerImage:  platformCfg.RunnerImage,
			}
			if commands := platformCfg.Commands; commands != nil {
				o.CodeRepo.PlatformBuildConfig.Commands = &BuildCommandsOutputObj{
					PreBuild:   emptySliceIfNil(commands.PreBuild),
					Build:      emptySliceIfNil(commands.Build),
					RuntimeEnv: emptySliceIfNil(commands.RuntimeEnv),
					Start:      commands.Start,
				}
			}
		}
	}
	if cfg.Image != nil {
		o.Image = &ImageBuildConfigOutputObj{
			Name:     cfg.Image.Name,
			Username: cfg.Image.Username,
		}
	}
	return o
}

// FromModel fills output fields from a tag config model.
func (o *TagConfigOutput) FromModel(cfg imagebuild.TagConfig) *TagConfigOutput {
	*o = TagConfigOutput{Type: string(cfg.Type)}
	if cfg.CustomOpts != nil {
		o.CustomOpts = &CustomTagOptsOutput{
			Prefix:        cfg.CustomOpts.Prefix,
			WithRevision:  cfg.CustomOpts.WithRevision,
			WithBuildTime: cfg.CustomOpts.WithBuildTime,
		}
	}
	return o
}

// UpdateBuildConfigOutput is the JSON response for updating build config.
type UpdateBuildConfigOutput struct {
	// 构建配置详情
	Data *BuildConfigOutputObj `json:"data"`
}

func emptySliceIfNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
