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

package client

// AppMinimal 应用简要信息
type AppMinimal struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	DisplayName string `json:"displayName" yaml:"displayName"`
	Type        string `json:"type" yaml:"type"`
	Creator     string `json:"creator" yaml:"creator"`
}

// ListAppsRespData 获取应用列表返回数据
type ListAppsRespData struct {
	Data []AppMinimal `json:"data"`
}

// CreateAppRespData 创建应用返回数据
type CreateAppRespData struct {
	Data AppMinimal `json:"data"`
}

// GetAppIDAutoSuffixRespData 获取应用 ID 自动后缀返回数据
// 后端直接返回 {"suffix": "..."} 格式，无 data 包装
type GetAppIDAutoSuffixRespData struct {
	Suffix string `json:"suffix"`
}

// DevModeConfig 开发模式配置
type DevModeConfig struct {
	// Enabled 是否启用开发模式
	Enabled bool `json:"enabled" yaml:"enabled"`

	// WorkPath 开发模式根目录
	WorkPath string `json:"workPath" yaml:"workPath"`

	// MountPath 脚本挂载路径
	MountPath string `json:"mountPath" yaml:"mountPath"`
}

// DevModePreflightData 开发模式 Publish 预检响应数据
type DevModePreflightData struct {
	// Token 用户 Token
	Token string `json:"token"`
	// Address 已组装好的集群完整地址
	Address string `json:"address"`
	// Namespace 目标命名空间
	Namespace string `json:"namespace"`
	// InstanceIDs 校验通过的实例 ID 列表
	InstanceIDs []string `json:"instanceIDs"`
	// DevMode 开发模式路径配置
	DevMode *DevModeConfig `json:"devMode"`
}

// DevModePreflightRespData 开发模式 Publish 预检响应包装
type DevModePreflightRespData struct {
	Data *DevModePreflightData `json:"data"`
}

// DevModePublishReportOptions 开发模式发布结果上报参数
type DevModePublishReportOptions struct {
	// BinaryName 发布的二进制名称
	BinaryName string `json:"binaryName"`
	// FileSize 文件大小（字节）
	FileSize int64 `json:"fileSize"`
	// MD5 文件 MD5 值
	MD5 string `json:"md5"`
	// Results 逐实例发布结果
	Results []DevModePublishResult `json:"results"`
}

// DevModePublishResult 单个实例的发布结果
type DevModePublishResult struct {
	// Instance 目标实例（pod）名称
	Instance string `json:"instance"`
	// Status 发布状态：success / failed
	Status string `json:"status"`
	// Message 失败原因等附加信息
	Message string `json:"message"`
}

// DevModePublishRecord 开发模式发布记录
type DevModePublishRecord struct {
	// ID 发布记录 ID
	ID string `json:"id" yaml:"id"`
	// Instance 目标实例（pod）名称
	Instance string `json:"instance" yaml:"instance"`
	// BinaryName 发布的二进制名称
	BinaryName string `json:"binaryName" yaml:"binaryName"`
	// FileSize 文件大小（字节）
	FileSize int64 `json:"fileSize" yaml:"fileSize"`
	// MD5 文件 MD5 值
	MD5 string `json:"md5" yaml:"md5"`
	// Status 发布状态
	Status string `json:"status" yaml:"status"`
	// Message 失败原因等附加信息
	Message string `json:"message" yaml:"message"`
	// Operator 操作人
	Operator string `json:"operator" yaml:"operator"`
	// CreatedAt 创建时间
	CreatedAt string `json:"createdAt" yaml:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt string `json:"updatedAt" yaml:"updatedAt"`
}

// DevModePublishRecordsRespData 开发模式发布记录列表数据
type DevModePublishRecordsRespData struct {
	// Count 总数
	Count string `json:"count"`
	// Results 发布记录
	Results []DevModePublishRecord `json:"results"`
}

// DevModePublishRecordsResp 开发模式发布记录列表响应
type DevModePublishRecordsResp struct {
	Data DevModePublishRecordsRespData `json:"data"`
}

// AppFull 应用完整定义
type AppFull struct {
	ID           string        `json:"id" yaml:"id"`
	Name         string        `json:"name" yaml:"name"`
	DisplayName  string        `json:"displayName" yaml:"displayName"`
	Type         string        `json:"type" yaml:"type"`
	BuildConfig  *BuildConfig  `json:"buildConfig" yaml:"buildConfig"`
	AppModelSpec *AppModelSpec `json:"appModelSpec,omitempty" yaml:"appModelSpec,omitempty"`
	HelmSpec     any           `json:"helmSpec,omitempty" yaml:"helmSpec,omitempty"`
}

// BuildConfig 构建配置
type BuildConfig struct {
	SourceType          string               `json:"sourceType" yaml:"sourceType"`
	TagConfig           *TagConfig           `json:"tagConfig,omitempty" yaml:"tagConfig,omitempty"`
	ImageBuildConfig    *ImageBuildConfig    `json:"imageBuildConfig,omitempty" yaml:"imageBuildConfig,omitempty"`
	RepoBuildConfig     *RepoBuildConfig     `json:"repoBuildConfig,omitempty" yaml:"repoBuildConfig,omitempty"`
	PipelineBuildConfig *PipelineBuildConfig `json:"pipelineBuildConfig,omitempty" yaml:"pipelineBuildConfig,omitempty"`
}

// TagConfig 镜像 Tag 配置
type TagConfig struct {
	Type       string         `json:"type" yaml:"type"`
	CustomOpts *CustomTagOpts `json:"customOpts,omitempty" yaml:"customOpts,omitempty"`
}

// CustomTagOpts 自定义 Tag 选项
type CustomTagOpts struct {
	Prefix        string `json:"prefix" yaml:"prefix"`
	WithRevision  bool   `json:"withRevision" yaml:"withRevision"`
	WithBuildTime bool   `json:"withBuildTime" yaml:"withBuildTime"`
}

// ImageBuildConfig 镜像仓库构建配置
type ImageBuildConfig struct {
	Name     string `json:"name" yaml:"name"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
}

// RepoBuildConfig 代码仓库构建配置
type RepoBuildConfig struct {
	Type                string               `json:"type" yaml:"type"`
	RepoAlias           string               `json:"repoAlias" yaml:"repoAlias"`
	RepoURL             string               `json:"repoURL" yaml:"repoURL"`
	DefaultBranch       string               `json:"defaultBranch" yaml:"defaultBranch"`
	SourceDir           string               `json:"sourceDir,omitempty" yaml:"sourceDir,omitempty"`
	Dockerfile          string               `json:"dockerfile,omitempty" yaml:"dockerfile,omitempty"`
	DockerBuildArgs     map[string]string    `json:"dockerBuildArgs,omitempty" yaml:"dockerBuildArgs,omitempty"`
	ImageBuildMode      string               `json:"imageBuildMode,omitempty" yaml:"imageBuildMode,omitempty"`
	PlatformBuildConfig *PlatformBuildConfig `json:"platformBuildConfig,omitempty" yaml:"platformBuildConfig,omitempty"`
}

// PlatformBuildConfig 平台通用构建配置
type PlatformBuildConfig struct {
	BuilderImage string         `json:"builderImage" yaml:"builderImage"`
	RunnerImage  string         `json:"runnerImage" yaml:"runnerImage"`
	Commands     *BuildCommands `json:"commands,omitempty" yaml:"commands,omitempty"`
}

// BuildCommands 平台构建命令配置
type BuildCommands struct {
	PreBuild   []string `json:"preBuild" yaml:"preBuild"`
	Build      []string `json:"build" yaml:"build"`
	RuntimeEnv []string `json:"runtimeEnv" yaml:"runtimeEnv"`
	Start      string   `json:"start" yaml:"start"`
}

// PipelineBuildConfig 流水线构建配置
type PipelineBuildConfig struct {
	PipelineID string            `json:"pipelineID" yaml:"pipelineID"`
	Params     map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
}

// GetAppFullRespData 获取应用完整定义返回数据
type GetAppFullRespData struct {
	Data AppFull `json:"data"`
}

// UpdateAppDisplayNameBody 更新应用显示名请求体
type UpdateAppDisplayNameBody struct {
	DisplayName string `json:"displayName"`
}

// --- 构建配置更新（PUT /apps/:appID/build-configs）输入类型 ---
// 字段名与 GET /apps/:appID 返回的 buildConfig 不同，请勿混用：
//   GET: repoBuildConfig / imageBuildConfig / pipelineBuildConfig
//   PUT: codeRepo        / image            / pipeline

// AppBuildConfigUpdateBody PUT /apps/:appID/build-configs 请求体
type AppBuildConfigUpdateBody struct {
	// SourceType 来源类型：codeRepository | imageRegistry | pipeline
	SourceType string `json:"sourceType" yaml:"sourceType"`
	// TagConfig 镜像 Tag 配置
	TagConfig *AppBuildTagConfig `json:"tagConfig,omitempty" yaml:"tagConfig,omitempty"`
	// CodeRepo 代码仓库构建配置（sourceType=codeRepository 时填写）
	CodeRepo *AppBuildCodeRepoConfig `json:"codeRepo,omitempty" yaml:"codeRepo,omitempty"`
	// Image 镜像仓库构建配置（sourceType=imageRegistry 时填写）
	Image *AppBuildImageConfig `json:"image,omitempty" yaml:"image,omitempty"`
	// Pipeline 流水线构建配置（sourceType=pipeline 时填写）
	Pipeline *AppBuildPipelineConfig `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
}

// AppBuildTagConfig 镜像 Tag 配置
type AppBuildTagConfig struct {
	// Type Tag 生成策略：semver | custom
	Type       string                 `json:"type" yaml:"type"`
	CustomOpts *AppBuildTagCustomOpts `json:"customOpts,omitempty" yaml:"customOpts,omitempty"`
}

// AppBuildTagCustomOpts 自定义 Tag 选项
type AppBuildTagCustomOpts struct {
	Prefix        string `json:"prefix" yaml:"prefix"`
	WithRevision  bool   `json:"withRevision" yaml:"withRevision"`
	WithBuildTime bool   `json:"withBuildTime" yaml:"withBuildTime"`
}

// AppBuildCodeRepoConfig 代码仓库构建配置
type AppBuildCodeRepoConfig struct {
	// Type 代码仓库类型：TGit | GitHub
	Type          string `json:"type" yaml:"type"`
	RepoAlias     string `json:"repoAlias" yaml:"repoAlias"`
	RepoURL       string `json:"repoURL" yaml:"repoURL"`
	DefaultBranch string `json:"defaultBranch" yaml:"defaultBranch"`
	SourceDir     string `json:"sourceDir,omitempty" yaml:"sourceDir,omitempty"`
	// Dockerfile 仅 imageBuildMode=repositoryDockerfile 时有效
	Dockerfile      string            `json:"dockerfile,omitempty" yaml:"dockerfile,omitempty"`
	DockerBuildArgs map[string]string `json:"dockerBuildArgs,omitempty" yaml:"dockerBuildArgs,omitempty"`
	// ImageBuildMode 镜像构建方式：repositoryDockerfile | platform
	ImageBuildMode string `json:"imageBuildMode,omitempty" yaml:"imageBuildMode,omitempty"`
	// PlatformBuildConfig 平台通用构建配置，仅 imageBuildMode=platform 时填写
	PlatformBuildConfig *AppBuildPlatformConfig `json:"platformBuildConfig,omitempty" yaml:"platformBuildConfig,omitempty"`
}

// AppBuildPlatformConfig 平台通用构建配置
type AppBuildPlatformConfig struct {
	BuilderImage string                    `json:"builderImage" yaml:"builderImage"`
	RunnerImage  string                    `json:"runnerImage" yaml:"runnerImage"`
	Commands     *AppBuildPlatformCommands `json:"commands,omitempty" yaml:"commands,omitempty"`
}

// AppBuildPlatformCommands 平台通用构建命令
type AppBuildPlatformCommands struct {
	PreBuild   []string `json:"preBuild" yaml:"preBuild"`
	Build      []string `json:"build" yaml:"build"`
	RuntimeEnv []string `json:"runtimeEnv" yaml:"runtimeEnv"`
	Start      string   `json:"start" yaml:"start"`
}

// AppBuildImageConfig 镜像仓库构建配置（PUT 输入）
type AppBuildImageConfig struct {
	Name     string  `json:"name" yaml:"name"`
	Username *string `json:"username,omitempty" yaml:"username,omitempty"`
	Password *string `json:"password,omitempty" yaml:"password,omitempty"`
}

// AppBuildPipelineConfig 流水线构建配置（PUT 输入）
type AppBuildPipelineConfig struct {
	PipelineID string            `json:"pipelineID" yaml:"pipelineID"`
	Params     map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
}
