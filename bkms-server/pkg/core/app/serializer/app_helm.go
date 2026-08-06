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

// Package serializer 定义 Helm 应用相关的 Gin input/output 序列化结构和转换方法。
package serializer

import (
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
)

// HelmSpecInput is the Helm spec input.
type HelmSpecInput struct {
	// Helm 源配置
	HelmSource *HelmSourceInput `json:"helmSource" binding:"required"`
}

// HelmSourceInput is the Helm source input.
type HelmSourceInput struct {
	// 仓库类型
	RepoType string `json:"repoType" binding:"required,oneof=GitRepo HelmRepo BCSRepo"`
	// Value 文件列表
	ValueFiles []string `json:"valueFiles"`
	// Helm 仓库配置。如果 repoType 为 HelmRepo，helmRepoConfig 需要有有效值
	HelmRepoConfig *HelmRepoConfigInput `json:"helmRepoConfig"`
	// BCS 仓库配置。如果 repoType 为 BCSRepo，bcsRepoConfig 需要有有效值
	BCSRepoConfig *BCSRepoConfigInput `json:"bcsRepoConfig"`
	// Git 仓库配置。如果 repoType 为 GitRepo，gitRepoConfig 需要有有效值
	GitRepoConfig *HelmGitRepoConfigInput `json:"gitRepoConfig"`
}

// HelmRepoConfigInput is the Helm repo config input.
type HelmRepoConfigInput struct {
	// 仓库地址
	RepoURL string `json:"repoURL" binding:"required,url"`
	// Chart 名称
	ChartName string `json:"chartName" binding:"required"`
	// 用户名（可选）
	Username *string `json:"username"`
	// 密码（可选）
	Password *string `json:"password"`
}

// BCSRepoConfigInput is the BCS repo config input.
type BCSRepoConfigInput struct {
	// 项目 Code
	ProjectCode string `json:"projectCode" binding:"required"`
	// 仓库名称
	RepoName string `json:"repoName" binding:"required"`
	// Chart 名称
	ChartName string `json:"chartName" binding:"required"`
}

// HelmGitRepoConfigInput is the Helm Git repo config input.
type HelmGitRepoConfigInput struct {
	// 代码库类型
	Type string `json:"type" binding:"required,oneof=TGit GitHub"`
	// 代码库别名
	RepoAlias string `json:"repoAlias" binding:"required"`
	// 代码库地址
	RepoURL string `json:"repoURL" binding:"required,url"`
	// 代码库分支
	Revision string `json:"revision" binding:"required"`
	// Helm Chart 目录
	SourceDir string `json:"sourceDir" binding:"required"`
}

// ToModel 将 HelmSpecInput 转换为领域模型 HelmSpec
func (input *HelmSpecInput) ToModel() *bkmsapp.HelmSpec {
	if input == nil || input.HelmSource == nil {
		return nil
	}
	src := input.HelmSource
	spec := &bkmsapp.HelmSpec{
		HelmSource: &bkmsapp.HelmSource{
			RepoType:   bkmsapp.HelmSourceRepoType(src.RepoType),
			ValueFiles: src.ValueFiles,
		},
	}
	if src.HelmRepoConfig != nil {
		spec.HelmSource.HelmRepoConfig = &bkmsapp.HelmRepoConfig{
			RepoURL:   src.HelmRepoConfig.RepoURL,
			ChartName: src.HelmRepoConfig.ChartName,
		}
		if src.HelmRepoConfig.Username != nil {
			spec.HelmSource.HelmRepoConfig.Username = *src.HelmRepoConfig.Username
		}
		if src.HelmRepoConfig.Password != nil {
			spec.HelmSource.HelmRepoConfig.Password = *src.HelmRepoConfig.Password
		}
	}
	if src.BCSRepoConfig != nil {
		spec.HelmSource.BCSRepoConfig = &bkmsapp.BCSRepoConfig{
			ProjectCode: src.BCSRepoConfig.ProjectCode,
			RepoName:    src.BCSRepoConfig.RepoName,
			ChartName:   src.BCSRepoConfig.ChartName,
		}
	}
	if src.GitRepoConfig != nil {
		spec.HelmSource.GitRepoConfig = &bkmsapp.GitRepoConfig{
			Type:      bkmsapp.GitRepoType(src.GitRepoConfig.Type),
			RepoAlias: src.GitRepoConfig.RepoAlias,
			RepoURL:   src.GitRepoConfig.RepoURL,
			Revision:  src.GitRepoConfig.Revision,
			SourceDir: src.GitRepoConfig.SourceDir,
		}
	}
	return spec
}

// -----------------------------------------------------------------------------
// Helm Output
// -----------------------------------------------------------------------------

// HelmSpecOutputObj is the Helm spec output.
type HelmSpecOutputObj struct {
	// Helm 源配置
	HelmSource *HelmSourceOutputObj `json:"helmSource,omitempty"`
}

// HelmSourceOutputObj is the Helm source output.
type HelmSourceOutputObj struct {
	// 仓库类型
	RepoType string `json:"repoType"`
	// Value 文件列表
	ValueFiles []string `json:"valueFiles"`
	// Helm 仓库配置
	HelmRepoConfig *HelmRepoConfigOutputObj `json:"helmRepoConfig,omitempty"`
	// BCS 仓库配置
	BCSRepoConfig *BCSRepoConfigOutputObj `json:"bcsRepoConfig,omitempty"`
	// Git 仓库配置
	GitRepoConfig *HelmGitRepoConfigOutputObj `json:"gitRepoConfig,omitempty"`
}

// HelmRepoConfigOutputObj is the Helm repo config output.
type HelmRepoConfigOutputObj struct {
	// 仓库地址
	RepoURL string `json:"repoURL"`
	// Chart 名称
	ChartName string `json:"chartName"`
	// 用户名
	Username string `json:"username"`
}

// BCSRepoConfigOutputObj is the BCS repo config output.
type BCSRepoConfigOutputObj struct {
	// 项目 Code
	ProjectCode string `json:"projectCode"`
	// 仓库名称
	RepoName string `json:"repoName"`
	// Chart 名称
	ChartName string `json:"chartName"`
}

// HelmGitRepoConfigOutputObj is the Helm Git repo config output.
type HelmGitRepoConfigOutputObj struct {
	// 代码库类型
	Type string `json:"type"`
	// 代码库别名
	RepoAlias string `json:"repoAlias"`
	// 代码库地址
	RepoURL string `json:"repoURL"`
	// 代码库分支
	Revision string `json:"revision"`
	// Helm Chart 目录
	SourceDir string `json:"sourceDir"`
}

// FromModel fills output fields from a HelmSpec model.
func (o *HelmSpecOutputObj) FromModel(spec *bkmsapp.HelmSpec) *HelmSpecOutputObj {
	if spec == nil || spec.HelmSource == nil {
		return nil
	}
	src := spec.HelmSource
	out := &HelmSourceOutputObj{
		RepoType:   string(src.RepoType),
		ValueFiles: emptySliceIfNil(src.ValueFiles),
	}
	if src.HelmRepoConfig != nil {
		out.HelmRepoConfig = &HelmRepoConfigOutputObj{
			RepoURL:   src.HelmRepoConfig.RepoURL,
			ChartName: src.HelmRepoConfig.ChartName,
			Username:  src.HelmRepoConfig.Username,
		}
	}
	if src.BCSRepoConfig != nil {
		out.BCSRepoConfig = &BCSRepoConfigOutputObj{
			ProjectCode: src.BCSRepoConfig.ProjectCode,
			RepoName:    src.BCSRepoConfig.RepoName,
			ChartName:   src.BCSRepoConfig.ChartName,
		}
	}
	if src.GitRepoConfig != nil {
		out.GitRepoConfig = &HelmGitRepoConfigOutputObj{
			Type:      string(src.GitRepoConfig.Type),
			RepoAlias: src.GitRepoConfig.RepoAlias,
			RepoURL:   src.GitRepoConfig.RepoURL,
			Revision:  src.GitRepoConfig.Revision,
			SourceDir: src.GitRepoConfig.SourceDir,
		}
	}
	o.HelmSource = out
	return o
}

// UpdateHelmSpecInput is the JSON body for updating Helm spec.
type UpdateHelmSpecInput struct {
	// 待更新的 Helm Chart 配置
	HelmSpec *HelmSpecInput `json:"helmSpec" binding:"required"`
}
