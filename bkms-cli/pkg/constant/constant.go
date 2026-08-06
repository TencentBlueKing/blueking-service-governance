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

// Package constant 集中定义 bkms-cli 中使用的应用类型、构建来源等常量
package constant

const (
	// AppTypeTrpc trpc 应用类型
	AppTypeTrpc = "trpc"
	// AppTypeTaf taf 应用类型
	AppTypeTaf = "taf"
	// AppTypeHelm helm 应用类型
	AppTypeHelm = "helm"
	// AppTypeAgones agones 应用类型
	AppTypeAgones = "agones"
)

// 构建来源类型
const (
	// BuildSourceImageRegistry 镜像仓库
	BuildSourceImageRegistry = "imageRegistry"
	// BuildSourceCodeRepository 代码仓库
	BuildSourceCodeRepository = "codeRepository"
	// BuildSourcePipeline 流水线
	BuildSourcePipeline = "pipeline"
)

// Helm 仓库类型
const (
	// HelmRepoTypeHelmRepo Helm 仓库
	HelmRepoTypeHelmRepo = "HelmRepo"
	// HelmRepoTypeBCSRepo BCS 仓库
	HelmRepoTypeBCSRepo = "BCSRepo"
	// HelmRepoTypeGitRepo Git 仓库
	HelmRepoTypeGitRepo = "GitRepo"
)

// 代码仓库类型
const (
	// RepoTypeTGit 工蜂仓库
	RepoTypeTGit = "TGit"
	// RepoTypeGitHub GitHub 仓库
	RepoTypeGitHub = "GitHub"
)

// Git 仓库类型（Helm GitRepo 使用）
const (
	// GitTypeTGit 工蜂
	GitTypeTGit = "TGit"
	// GitTypeGitHub GitHub
	GitTypeGitHub = "GitHub"
)

// tRPC 语言类型
const (
	// TrpcLanguageGo Go 语言
	TrpcLanguageGo = "go"
	// TrpcLanguageCpp C++ 语言
	TrpcLanguageCpp = "cpp"
)

// Helm 默认值
const (
	// DefaultHelmValueFile Helm 默认 values 文件名
	DefaultHelmValueFile = "values.yaml"
)

// 环境类型
const (
	// EnvTypeProduction 正式环境类型
	EnvTypeProduction = "production"
)
