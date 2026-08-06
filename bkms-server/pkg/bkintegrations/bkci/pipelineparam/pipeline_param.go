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

package pipelineparam

const (
	// 公共流水线变量

	// RepoURL 代码库地址
	RepoURL = "BKMS_REPO_URL"
	// RepoAlias 代码库别名
	RepoAlias = "BKMS_REPO_ALIAS"
	// RepoCheckoutBy 代码库检出方式
	RepoCheckoutBy = "BKMS_REPO_CHECKOUT_BY"
	// RepoRevision 代码库版本（分支名 / Tag / Commit ID）
	RepoRevision = "BKMS_REPO_REVISION"

	// 基于 Dockerfile 构建容器镜像流水线变量

	// ImageRegistry 镜像源仓库（如 mirrors.tencent.com/example）
	ImageRegistry = "BKMS_IMAGE_REGISTRY"
	// ImageName 镜像名称
	ImageName = "BKMS_IMAGE_NAME"
	// ImageTag 镜像标签
	ImageTag = "BKMS_IMAGE_TAG"
	// ImageCredential 镜像凭证
	ImageCredential = "BKMS_IMAGE_CREDENTIAL" // nolint: gosec
	// DockerBuildDir Docker Build 目录
	DockerBuildDir = "BKMS_DOCKER_BUILD_DIR"
	// DockerfilePath Dockerfile 路径。
	// repository 模式下表示仓库内已有 Dockerfile 的路径；generated 模式下表示蓝盾侧生成 Dockerfile
	// 写入并供后续 Docker 构建步骤读取的目标路径。
	DockerfilePath = "BKMS_DOCKERFILE_PATH"
	// DockerBuildArgs 构建参数
	DockerBuildArgs = "BKMS_DOCKER_BUILD_ARGS"
	// DockerBuildArgNames 构建参数名称列表，仅供 Dockerfile Generator 声明 ARG 使用
	DockerBuildArgNames = "BKMS_DOCKER_BUILD_ARG_NAMES"

	// 平台生成 Dockerfile 模式流水线变量（由 linuxScript 启动器传递给 Dockerfile Generator CLI）

	// DockerfileSourceType Dockerfile 来源类型（repository / bkms_generated），
	// 两种模式都需要传入，供蓝盾侧根据来源类型分支执行
	DockerfileSourceType = "BKMS_DOCKERFILE_SOURCE_TYPE"
	// DockerfileLanguage 平台生成 Dockerfile 模式下的 tRPC 应用语言类型
	DockerfileLanguage = "BKMS_DOCKERFILE_LANGUAGE"
	// ImageBuildToolchainBaseURL 镜像构建工具链下载基础 URL
	ImageBuildToolchainBaseURL = "BKMS_IMAGE_BUILD_TOOLCHAIN_BASE_URL"
	// DockerfileBuilderImage 平台生成 Dockerfile 模式下的构建阶段基础镜像
	DockerfileBuilderImage = "BKMS_DOCKERFILE_BUILDER_IMAGE"
	// DockerfileRunnerImage 平台生成 Dockerfile 模式下的运行阶段基础镜像
	DockerfileRunnerImage = "BKMS_DOCKERFILE_RUNNER_IMAGE"
	// DockerfilePreBuildCommands 平台生成 Dockerfile 模式下的编译前置命令列表
	// 当前使用 JSON 字符串数组传递，需要 Dockerfile Generator 按相同协议解析
	DockerfilePreBuildCommands = "BKMS_DOCKERFILE_PRE_BUILD_COMMANDS"
	// DockerfileBuildCommands 平台生成 Dockerfile 模式下的编译命令列表
	// 当前使用 JSON 字符串数组传递，需要 Dockerfile Generator 按相同协议解析
	DockerfileBuildCommands = "BKMS_DOCKERFILE_BUILD_COMMANDS"
	// DockerfileRuntimeEnvCommands 平台生成 Dockerfile 模式下的运行环境命令列表
	// 当前使用 JSON 字符串数组传递，需要 Dockerfile Generator 按相同协议解析
	DockerfileRuntimeEnvCommands = "BKMS_DOCKERFILE_RUNTIME_ENV_COMMANDS"
	// DockerfileStartCommand 平台生成 Dockerfile 模式下的启动命令
	DockerfileStartCommand = "BKMS_DOCKERFILE_START_COMMAND"

	// 基于 Git 源码仓库构建 HelmChart 流水线变量

	// HelmRepoURL bkrepo HELM 仓库地址
	HelmRepoURL = "BKMS_HELM_REPO_URL"
	// HelmChartName Helm Chart 名称
	HelmChartName = "BKMS_HELM_CHART_NAME"
	// HelmChartVersion Helm Chart 版本号（semver 格式，如 0.0.1）
	HelmChartVersion = "BKMS_HELM_CHART_VERSION"
	// HelmChartBuildDir 构建执行目录（Chart 源码目录）
	HelmChartBuildDir = "BKMS_HELM_CHART_BUILD_DIR"
	// HelmToolchainBaseURL Helm, Helmify, Kustomize 等二进制下载基础 URL
	HelmToolchainBaseURL = "BKMS_HELM_TOOLCHAIN_BASE_URL"
)
