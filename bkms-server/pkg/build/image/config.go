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
	"time"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/credentials"
)

// SourceType 应用来源
type SourceType string

const (
	// SourceTypeCodeRepository 由代码仓库构建的镜像
	SourceTypeCodeRepository SourceType = "codeRepository"
	// SourceTypeImageRegistry 由镜像仓库拉取的镜像, 仅 helm 使用
	SourceTypeImageRegistry SourceType = "imageRegistry"
	// SourceTypePipeline 由流水线构建的镜像, 仅 trpc 应用使用
	SourceTypePipeline SourceType = "pipeline"
)

// VersionType 推荐版本号类型
type VersionType string

const (
	// VersionTypeSemver 语义化版本（如 v1.0.1）
	VersionTypeSemver VersionType = "semver"
	// VersionTypeCustom 自定义版本（分支/Tag + 构建时间）
	VersionTypeCustom VersionType = "custom"
)

// MaxCustomTagPrefixLength 自定义 Tag 前缀最大长度
const MaxCustomTagPrefixLength = 16

// 平台通用构建配置校验阈值。
const (
	// MaxPlatformBuildCommandCount 命令列表允许的最大条数。
	MaxPlatformBuildCommandCount = 32
	// MaxPlatformBuildCommandLen 单条命令允许的最大长度。
	MaxPlatformBuildCommandLen = 4096
)

// CustomTagOpts 自定义 Tag 选项，仅当 TagConfig.Type 为 custom 时有效
type CustomTagOpts struct {
	// Prefix 自定义前缀（可选，最长 MaxCustomTagPrefixLength 个字符）
	Prefix string `bson:"prefix" json:"prefix"`
	// WithRevision 是否包含分支/Tag 名称
	WithRevision bool `bson:"withRevision" json:"withRevision"`
	// WithBuildTime 是否包含构建时间戳
	WithBuildTime bool `bson:"withBuildTime" json:"withBuildTime"`
}

// TagConfig 镜像 Tag 配置
type TagConfig struct {
	// Type 版本号类型: semver / custom / 空字符串
	Type VersionType `bson:"type" json:"type"`
	// CustomOpts 自定义 Tag 选项，仅当 Type 为 custom 时有效
	CustomOpts *CustomTagOpts `bson:"customOpts,omitempty" json:"customOpts,omitempty"`
}

// IsAutoGenerateEnabled 是否已配置自动生成镜像 Tag（semver / custom）。
// Type 为空表示未开启推荐版本号，自动触发构建前须拒绝
func (c TagConfig) IsAutoGenerateEnabled() bool {
	return c.Type == VersionTypeSemver || c.Type == VersionTypeCustom
}

// RepositoryType 代码库类型
type RepositoryType string

const (
	// RepositoryTypeTGit TGit（工蜂）仓库
	RepositoryTypeTGit RepositoryType = "TGit"
	// RepositoryTypeGitHub GitHub 仓库
	RepositoryTypeGitHub RepositoryType = "GitHub"
)

// ImageBuildMode 镜像构建方式。
type ImageBuildMode string

const (
	// ImageBuildModeRepositoryDockerfile 使用代码仓库中的 Dockerfile 构建镜像。
	ImageBuildModeRepositoryDockerfile ImageBuildMode = "repositoryDockerfile"
	// ImageBuildModePlatform 使用平台通用构建配置构建镜像。
	ImageBuildModePlatform ImageBuildMode = "platform"
)

// RepositoryConfig 代码库配置
type RepositoryConfig struct {
	// Type 代码库类型
	Type RepositoryType `bson:"type"`
	// RepoAlias 代码库别名
	RepoAlias string `bson:"repoAlias"`
	// RepoURL 代码仓库地址
	RepoURL string `bson:"repoURL"`
	// DefaultBranch 默认分支，指构建时不手动挑选任何分支时的默认值
	DefaultBranch string `bson:"defaultBranch"`
	// SourceDir 应用源码目录（构建目录），为空表示仓库根目录；非空时必须是仓库内相对路径
	SourceDir string `bson:"sourceDir"`
	// Dockerfile Dockerfile 文件路径，为空表示使用默认路径（根目录下的 Dockerfile）
	Dockerfile string `bson:"dockerfile"`
	// FIXME: 当前前端仍使用 dockerBuildArgs，后续需要切换为 BuildArgs
	// DockerBuildArgs Docker 构建参数
	DockerBuildArgs map[string]string `bson:"dockerBuildArgs,omitempty"`
	// ImageBuildMode 镜像构建方式；空值表示 repositoryDockerfile。
	ImageBuildMode ImageBuildMode `bson:"imageBuildMode,omitempty"`
	// PlatformBuildConfig 平台通用构建配置，仅平台通用构建方式有效。
	PlatformBuildConfig *PlatformBuildConfig `bson:"platformBuildConfig,omitempty"`
}

// NormalizeImageBuildMode 返回规范化后的镜像构建方式。
//
// ImageBuildMode 为空时按 repositoryDockerfile 处理。
func NormalizeImageBuildMode(mode ImageBuildMode) ImageBuildMode {
	if mode == "" {
		return ImageBuildModeRepositoryDockerfile
	}
	return mode
}

// EffectiveImageBuildMode 返回有效的镜像构建方式。
func (c *RepositoryConfig) EffectiveImageBuildMode() ImageBuildMode {
	if c == nil {
		return ImageBuildModeRepositoryDockerfile
	}
	if c.ImageBuildMode != "" {
		return NormalizeImageBuildMode(c.ImageBuildMode)
	}
	if c.PlatformBuildConfig != nil {
		return ImageBuildModePlatform
	}
	return ImageBuildModeRepositoryDockerfile
}

// PlatformBuildConfig 平台通用构建配置。
type PlatformBuildConfig struct {
	// BuilderImage 构建阶段基础镜像
	BuilderImage string `bson:"builderImage"`
	// RunnerImage 运行阶段基础镜像
	RunnerImage string `bson:"runnerImage"`
	// Commands 命令配置
	Commands *BuildCommands `bson:"commands,omitempty"`
}

// BuildCommands 平台通用构建命令配置。
type BuildCommands struct {
	// PreBuild 编译前置命令列表
	PreBuild []string `bson:"preBuild,omitempty"`
	// Build 编译命令列表
	Build []string `bson:"build,omitempty"`
	// RuntimeEnv 运行环境命令列表
	RuntimeEnv []string `bson:"runtimeEnv,omitempty"`
	// Start 启动命令
	Start string `bson:"start,omitempty"`
}

// ImageConfig 镜像配置
type ImageConfig struct {
	// Name 镜像名称（不包含镜像 Tag）
	// e.g. "myapp" or "hub.bktencent.com/blueking/myapp"
	Name string `bson:"name"`
	// Username 用户名
	Username string `bson:"username"`
	// Password 密码
	Password string `bson:"password"`
}

// PipelineConfig 蓝盾流水线配置
type PipelineConfig struct {
	// PipelineID 流水线 ID
	PipelineID string `bson:"pipelineID"`
	// Params 构建流水线参数
	Params map[string]string `bson:"params,omitempty"`
}

// Config 包含应用构建相关配置
type Config struct {
	// AppID 应用 ID
	AppID string `bson:"appID"`

	// SourceType 应用来源 类型
	SourceType SourceType `bson:"sourceType"`
	// PipelineType 流水线类型
	PipelineType string `bson:"pipelineType"`
	// TagConfig 镜像 Tag 配置
	TagConfig TagConfig `bson:"tagConfig,omitempty"`
	// CodeRepo 代码库配置
	CodeRepo *RepositoryConfig `bson:"codeRepo,omitempty" mapstructure:"repoBuildConfig"`
	// Image 镜像配置， 目前仅 helm 类型 app 支持
	Image *ImageConfig `bson:"image,omitempty" mapstructure:"imageBuildConfig"`
	// Pipeline 流水线配置, 目前仅 trpc 类型 app 支持
	Pipeline *PipelineConfig `bson:"pipeline,omitempty" mapstructure:"pipelineBuildConfig"`

	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// SetUserPass 设置 ImageConfig 的 username 和 password 字段，值为空时用 existingConfig 的值
// 设置完成后自动校验 Username 和 Password 的合法性
func (c *ImageConfig) SetUserPass(existingConfig *ImageConfig, username, password *string) error {
	if existingConfig != nil {
		c.Username = existingConfig.Username
		c.Password = existingConfig.Password
	}
	if username != nil {
		c.Username = *username
	}
	if password != nil {
		c.Password = *password
	}
	return credentials.ValidateOptionalUserPass(c.Username, c.Password)
}
