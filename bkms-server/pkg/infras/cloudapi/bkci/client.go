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

package bkci

import (
	"context"
	"io"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/tof"
)

// Client 蓝盾 API 客户端接口
type Client interface {
	// ------------------------------------------ 蓝盾项目相关 API ------------------------------------------

	// ListProjects 获取蓝盾项目列表
	ListProjects(ctx context.Context) ([]Project, error)
	// GetProject 获取蓝盾项目（通过项目 Code）
	GetProject(ctx context.Context, projectCode string) (*Project, error)
	// CreateProject 创建蓝盾项目
	CreateProject(
		ctx context.Context,
		projectCode, obsProductID, obsProductName string,
		userOrg *tof.Organization,
	) error

	// ------------------------------------------ 蓝盾代码库 & OAuth API ------------------------------------------

	// ListOAuthGitProjects 获取用户有 OAuth 授权给蓝盾的 Git 项目列表
	ListOAuthGitProjects(ctx context.Context, projectCode, keyword string) ([]GitProject, error)
	// GetOAuthUrl 获取用户授权 Git 项目给蓝盾的 OAuth 授权地址
	GetOAuthUrl(ctx context.Context, projectCode string) (string, error)

	// ------------------------------------------ 蓝盾凭证管理 API ------------------------------------------

	// CreateCredential 创建蓝盾凭证（USERNAME_PASSWORD 类型）
	CreateCredential(ctx context.Context, projectCode, credentialID, credentialDesc, username, password string) error
	// CreateAccessTokenCredential 创建蓝盾 AccessToken 类型凭证
	CreateAccessTokenCredential(
		ctx context.Context, projectCode, credentialID, credentialDesc, token string,
	) error
	// DeleteCredential 删除蓝盾凭证；凭证不存在时返回 ObjectNotFound
	DeleteCredential(ctx context.Context, projectCode, credentialID string) error

	// ------------------------------------------ 蓝盾流水线 API ------------------------------------------

	// ListPipelines 获取流水线列表
	// 注意：该结构提供的 Pipeline 中 Variables 字段为空，如有需要可通过 GetPipeline 获取
	ListPipelines(
		ctx context.Context, projectCode, keyword string, page, pageSize int64,
	) (int64, []Pipeline, error)
	// GetPipeline 获取流水线详情
	GetPipeline(ctx context.Context, projectCode, pipelineID string) (*Pipeline, error)
	// CreatePipeline 创建蓝盾流水线，返回流水线 ID（p-[a-z0-9]{32}）
	CreatePipeline(ctx context.Context, projectCode, name, description string, stages []map[string]any) (string, error)
	// UpdatePipeline 更新蓝盾流水线
	UpdatePipeline(
		ctx context.Context,
		projectCode, pipelineID, name, description string,
		stages []map[string]any,
	) error
	// DeletePipeline 删除蓝盾流水线；流水线不存在时返回 ObjectNotFound
	DeletePipeline(ctx context.Context, projectCode, pipelineID string) error
	// CreatePipelineBuild 创建蓝盾流水线构建，返回构建引用信息（包含构建 ID: b-[a-z0-9]{32}）
	CreatePipelineBuild(
		ctx context.Context, projectCode, pipelineID string, pipelineParams map[string]string,
	) (*PipelineBuildReference, error)
	// GetPipelineBuildState 获取蓝盾流水线构建状态
	GetPipelineBuildState(ctx context.Context, projectCode, pipelineID, buildID string) (*PipelineBuildState, error)
	// ListPipelineRepoRefProperties 获取流水线启动表单中的代码分支/Tag字段列表
	ListPipelineRepoRefProperties(ctx context.Context, projectCode, pipelineID string) ([]RepoRefProperty, error)
	// ListPipelineRepoRefOptions 获取某个代码分支/Tag字段的可选项
	ListPipelineRepoRefOptions(
		ctx context.Context, projectCode, pipelineID, propertyID, search string,
	) ([]PipelineVariableOption, error)

	// ------------------------------------------ 蓝盾代码库管理 API ------------------------------------------

	// ListRepository 获取蓝盾仓库列表
	// repoType 可选值："", CODE_SVN, CODE_GIT, CODE_GITLAB, GITHUB, CODE_TGIT, CODE_P4
	ListRepository(
		ctx context.Context, projectCode, repoType string, page, pageSize int64,
	) (int64, []Repository, error)
	// GetRepository 获取蓝盾代码库详情
	GetRepository(ctx context.Context, projectCode, repoHashID string) (*Repository, error)
	// CreateRepository 创建蓝盾代码库，返回代码库 Hash ID（目前只支持 codeGit + OAuth）
	CreateRepository(ctx context.Context, projectCode, repoURL, repoAlias string) (string, error)

	// ------------------------------------------ 蓝盾构建日志 API ------------------------------------------

	// GetInitBuildLog 获取构建日志初始内容（从末尾开始的最近日志，适用于首次加载）
	GetInitBuildLog(ctx context.Context, projectCode, pipelineID, buildID string) (*BuildLog, error)
	// GetMoreBuildLogs 获取更多构建日志（增量拉取，用于 SSE 续传和下载聚合）。
	// batchSize 由调用方按场景决定，客户端会映射为 BKCI 的 [start, start+batchSize] 查询窗口。
	GetMoreBuildLogs(
		ctx context.Context, projectCode, pipelineID, buildID string, start, batchSize int64,
	) (*BuildLog, error)
	// DownloadBuildLogs 下载构建日志原始流（适用于下载场景）。
	DownloadBuildLogs(ctx context.Context, projectCode, pipelineID, buildID string) (io.ReadCloser, error)
}
