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

// Package app 提供应用创建相关的处理逻辑
package app

import (
	"reflect"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/stringx"
)

const (
	// maxIDLength 应用 ID 最大长度
	maxIDLength = 63
)

// AppCreateSpec 面向用户的 YAML 结构体，用于创建应用
// 结构与后端 CreateAppInput 一致，可直接序列化为后端 API 请求体
type AppCreateSpec struct {
	// ID 应用 ID（可选，不填时由 name + 随机后缀自动生成）
	ID string `yaml:"id" json:"id" validate:"omitempty,app_name"`
	// Name 应用名称（必填）
	Name string `yaml:"name" json:"name" validate:"required,app_name"`
	// Type 应用类型：trpc | taf | helm | agones（必填）
	Type string `yaml:"type" json:"type" validate:"required,oneof=trpc taf helm agones"`
	// BuildConfig 构建配置（必填）
	BuildConfig *BuildConfigSpec `yaml:"buildConfig" json:"buildConfig" validate:"required"`
	// AppModelSpec 应用模型规范（type=trpc/taf 时需要）
	AppModelSpec *AppModelSpecSpec `yaml:"appModelSpec,omitempty" json:"appModelSpec,omitempty"`
	// HelmSpec Helm 应用描述规范（type=helm/agones 时需要）
	HelmSpec *HelmSpecSpec `yaml:"helmSpec,omitempty" json:"helmSpec,omitempty"`
}

// BuildConfigSpec 构建配置
type BuildConfigSpec struct {
	// SourceType 来源类型：imageRegistry | codeRepository | pipeline（必填）
	SourceType string `yaml:"sourceType" json:"sourceType" validate:"required,oneof=imageRegistry codeRepository pipeline"`
	// ImageBuildConfig 镜像仓库配置（sourceType=imageRegistry 时需要）
	ImageBuildConfig *ImageBuildConfigSpec `yaml:"imageBuildConfig,omitempty" json:"imageBuildConfig,omitempty"`
	// RepoBuildConfig 代码仓库配置（sourceType=codeRepository 时需要）
	RepoBuildConfig *RepoBuildConfigSpec `yaml:"repoBuildConfig,omitempty" json:"repoBuildConfig,omitempty"`
	// PipelineBuildConfig 流水线配置（sourceType=pipeline 时需要）
	PipelineBuildConfig *PipelineBuildConfigSpec `yaml:"pipelineBuildConfig,omitempty" json:"pipelineBuildConfig,omitempty"`
}

// ImageBuildConfigSpec 镜像仓库构建配置
type ImageBuildConfigSpec struct {
	// Name 镜像名称（必填）
	Name string `yaml:"name" json:"name" validate:"required"`
	// Username 镜像仓库用户名（可选）
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	// Password 镜像仓库密码（可选）
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// RepoBuildConfigSpec 代码仓库构建配置
type RepoBuildConfigSpec struct {
	// Type 代码仓库类型：TGit | GitHub（必填）
	Type string `yaml:"type" json:"type" validate:"required,oneof=TGit GitHub"`
	// RepoAlias 代码仓库别名（必填）
	RepoAlias string `yaml:"repoAlias" json:"repoAlias" validate:"required"`
	// RepoURL 代码仓库地址（必填）
	RepoURL string `yaml:"repoURL" json:"repoURL" validate:"required"`
	// DefaultBranch 默认分支（必填）
	DefaultBranch string `yaml:"defaultBranch" json:"defaultBranch" validate:"required"`
	// SourceDir 源码目录（可选）
	SourceDir string `yaml:"sourceDir,omitempty" json:"sourceDir,omitempty"`
	// Dockerfile Dockerfile 路径（可选）
	Dockerfile string `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	// DockerBuildArgs Docker 构建参数（可选）
	DockerBuildArgs map[string]string `yaml:"dockerBuildArgs,omitempty" json:"dockerBuildArgs,omitempty"`
}

// PipelineBuildConfigSpec 流水线构建配置
type PipelineBuildConfigSpec struct {
	// PipelineID 流水线 ID（必填）
	PipelineID string `yaml:"pipelineID" json:"pipelineID" validate:"required"`
	// Params 流水线参数（可选）
	Params map[string]string `yaml:"params,omitempty" json:"params,omitempty"`
}

// AppModelSpecSpec 应用模型规范（trpc/taf 共用）
type AppModelSpecSpec struct {
	// Command 容器启动命令（可选）
	Command []string `yaml:"command,omitempty" json:"command,omitempty"`
	// Args 容器启动参数（可选）
	Args []string `yaml:"args,omitempty" json:"args,omitempty"`
	// EnvVars 容器环境变量（可选）
	EnvVars []VariableSpec `yaml:"envVars,omitempty" json:"envVars,omitempty"`
	// TrpcSpec tRPC 框架配置（type=trpc 时需要）
	TrpcSpec *TrpcSpecSpec `yaml:"trpcSpec,omitempty" json:"trpcSpec,omitempty"`
	// TafSpec TAF 框架配置（type=taf 时需要）
	TafSpec *TafSpecSpec `yaml:"tafSpec,omitempty" json:"tafSpec,omitempty"`
}

// VariableSpec 环境变量
type VariableSpec struct {
	// Key 变量名
	Key string `yaml:"key" json:"key"`
	// Value 变量值
	Value string `yaml:"value" json:"value"`
	// Description 描述（可选）
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// TrpcSpecSpec tRPC 框架配置
type TrpcSpecSpec struct {
	// Language 编程语言：go | cpp（必填）
	Language string `yaml:"language" json:"language" validate:"required,oneof=go cpp"`
	// FileName 配置文件名（必填）
	FileName string `yaml:"fileName" json:"fileName" validate:"required"`
	// FilePath 配置文件路径（必填）
	FilePath string `yaml:"filePath" json:"filePath" validate:"required"`
	// FileContent 配置文件内容（可选）
	FileContent string `yaml:"fileContent,omitempty" json:"fileContent,omitempty"`
}

// TafSpecSpec TAF 框架配置
type TafSpecSpec struct {
	// FileName 配置文件名（必填）
	FileName string `yaml:"fileName" json:"fileName" validate:"required"`
	// FilePath 配置文件路径（必填）
	FilePath string `yaml:"filePath" json:"filePath" validate:"required"`
	// FileContent 配置文件内容（可选）
	FileContent string `yaml:"fileContent,omitempty" json:"fileContent,omitempty"`
}

// HelmSpecSpec Helm 应用描述规范
type HelmSpecSpec struct {
	// HelmSource Helm 源配置（必填）
	HelmSource *HelmSourceSpec `yaml:"helmSource" json:"helmSource" validate:"required"`
}

// HelmSourceSpec Helm 来源配置
type HelmSourceSpec struct {
	// RepoType 仓库类型：HelmRepo | BCSRepo | GitRepo（必填）
	RepoType string `yaml:"repoType" json:"repoType" validate:"required,oneof=HelmRepo BCSRepo GitRepo"`
	// ValueFiles values 文件列表（可选，默认 ["values.yaml"]）
	ValueFiles []string `yaml:"valueFiles,omitempty" json:"valueFiles,omitempty"`
	// HelmRepoConfig Helm 仓库配置（repoType=HelmRepo 时需要）
	HelmRepoConfig *HelmRepoConfigSpec `yaml:"helmRepoConfig,omitempty" json:"helmRepoConfig,omitempty"`
	// BCSRepoConfig BCS 仓库配置（repoType=BCSRepo 时需要）
	BCSRepoConfig *BCSRepoConfigSpec `yaml:"bcsRepoConfig,omitempty" json:"bcsRepoConfig,omitempty"`
	// GitRepoConfig Git 仓库配置（repoType=GitRepo 时需要）
	GitRepoConfig *GitRepoConfigSpec `yaml:"gitRepoConfig,omitempty" json:"gitRepoConfig,omitempty"`
}

// HelmRepoConfigSpec Helm 仓库配置
type HelmRepoConfigSpec struct {
	// RepoURL 仓库地址（必填）
	RepoURL string `yaml:"repoURL" json:"repoURL" validate:"required"`
	// ChartName Chart 名称（必填）
	ChartName string `yaml:"chartName" json:"chartName" validate:"required"`
	// Username 用户名（可选）
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	// Password 密码（可选）
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

// BCSRepoConfigSpec BCS 仓库配置
type BCSRepoConfigSpec struct {
	// ProjectCode 项目 Code（必填）
	ProjectCode string `yaml:"projectCode" json:"projectCode" validate:"required"`
	// RepoName 仓库名称（必填）
	RepoName string `yaml:"repoName" json:"repoName" validate:"required"`
	// ChartName Chart 名称（必填）
	ChartName string `yaml:"chartName" json:"chartName" validate:"required"`
}

// GitRepoConfigSpec Git 仓库配置
type GitRepoConfigSpec struct {
	// Type 代码库类型：TGit | GitHub（必填）
	Type string `yaml:"type" json:"type" validate:"required,oneof=TGit GitHub"`
	// RepoAlias 代码库别名（必填）
	RepoAlias string `yaml:"repoAlias" json:"repoAlias" validate:"required"`
	// RepoURL 代码库地址（必填）
	RepoURL string `yaml:"repoURL" json:"repoURL" validate:"required"`
	// Revision 代码库分支/标签（必填）
	Revision string `yaml:"revision" json:"revision" validate:"required"`
	// SourceDir Helm Chart 目录（必填）
	SourceDir string `yaml:"sourceDir" json:"sourceDir" validate:"required"`
}

// TrimSpace 对 AppCreateSpec 中所有字符串字段递归执行 strings.TrimSpace
func (s *AppCreateSpec) TrimSpace() {
	stringx.TrimSpaceRecursive(reflect.ValueOf(s))
}
