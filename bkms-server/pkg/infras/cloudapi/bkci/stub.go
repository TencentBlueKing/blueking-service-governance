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
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/tof"
)

// StubApiClient 测试用的蓝盾 API 客户端实现，返回模拟数据
type StubApiClient struct {
	user auth.User
}

var _ Client = &StubApiClient{}

var stubCreatedPipelines sync.Map

// NewStub 创建 StubApiClient
func NewStub(user auth.User) *StubApiClient {
	return &StubApiClient{user: user}
}

// ------------------------------------------ 蓝盾项目相关 API ------------------------------------------

// ListProjects 返回模拟的项目列表
func (s *StubApiClient) ListProjects(ctx context.Context) ([]Project, error) {
	log.Info(ctx, "Stub: ListProjects request")
	return []Project{
		{
			ID:            "stub-project-id-1",
			Code:          "stub-project-1",
			Name:          "Stub Project 1",
			Creator:       s.user.ID,
			HasManagePerm: true,
		},
		{
			ID:            "stub-project-id-2",
			Code:          "stub-project-2",
			Name:          "Stub Project 2",
			Creator:       s.user.ID,
			HasManagePerm: true,
		},
	}, nil
}

// GetProject 返回模拟的项目详情
func (s *StubApiClient) GetProject(ctx context.Context, projectCode string) (*Project, error) {
	log.Infof(ctx, "Stub: GetProject request: %s", projectCode)
	return &Project{
		ID:            "stub-project-id-" + projectCode,
		Code:          projectCode,
		Name:          "Stub Project",
		Creator:       s.user.ID,
		HasManagePerm: true,
	}, nil
}

// CreateProject 模拟创建项目，总是返回成功
func (s *StubApiClient) CreateProject(
	ctx context.Context,
	projectCode, obsProductID, obsProductName string,
	_ *tof.Organization,
) error {
	log.Infof(ctx, "Stub: CreateProject request: %s, obsProductID: %s, obsProductName: %s",
		projectCode,
		obsProductID,
		obsProductName,
	)
	return nil
}

// ------------------------------------------ 蓝盾代码库 & OAuth API ------------------------------------------

// ListOAuthGitProjects 返回模拟的 Git 项目列表
func (s *StubApiClient) ListOAuthGitProjects(ctx context.Context, projectCode, keyword string) ([]GitProject, error) {
	log.Infof(ctx, "Stub: ListOAuthGitProjects request: %s, keyword: %s", projectCode, keyword)
	return []GitProject{
		{
			ID:    "stub-git-project-1",
			Name:  "stub-repo-1",
			Alias: "stub-group/stub-repo-1",
			Url:   "https://git.example.com/stub-group/stub-repo-1.git",
		},
		{
			ID:    "stub-git-project-2",
			Name:  "stub-repo-2",
			Alias: "stub-group/stub-repo-2",
			Url:   "https://git.example.com/stub-group/stub-repo-2.git",
		},
	}, nil
}

// GetOAuthUrl 返回模拟的 OAuth 授权地址
func (s *StubApiClient) GetOAuthUrl(ctx context.Context, projectCode string) (string, error) {
	log.Infof(ctx, "Stub: GetOAuthUrl request: %s", projectCode)
	return "https://example.com/oauth/authorize?project=" + projectCode, nil
}

// ------------------------------------------ 蓝盾凭证管理 API ------------------------------------------

// CreateCredential 模拟创建凭证，总是返回成功
func (s *StubApiClient) CreateCredential(
	ctx context.Context, projectCode, credentialID, _, _, _ string,
) error {
	log.Infof(ctx, "Stub: CreateCredential request: %s, %s", projectCode, credentialID)
	return nil
}

// ------------------------------------------ 蓝盾流水线 API ------------------------------------------

func (s *StubApiClient) stubPipelines() []Pipeline {
	now := time.Now()
	return []Pipeline{
		{
			ID:          "stub-pipeline-id-1",
			Name:        "stub-pipeline-1",
			Description: "Stub Pipeline 1",
			Version:     1,
			Creator:     s.user.ID,
			Updater:     s.user.ID,
			CreatedAt:   now.Add(-24 * time.Hour),
			UpdatedAt:   now,
			Variables: []PipelineVariable{
				{
					ID:           "stub-pipeline-var-id-1",
					Name:         "IMAGE_TAG",
					Description:  "Image tag for deployment",
					Required:     true,
					ReadOnly:     false,
					Constant:     false,
					DefaultValue: "latest",
					Type:         "STRING",
				},
			},
		},
		{
			ID:          "stub-pipeline-id-2",
			Name:        "stub-pipeline-2",
			Description: "Stub Pipeline 2",
			Version:     2,
			Creator:     s.user.ID,
			Updater:     s.user.ID,
			CreatedAt:   now.Add(-48 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
			Variables: []PipelineVariable{
				{
					ID:           "stub-pipeline-var-id-2",
					Name:         "ENV",
					Description:  "Deployment environment",
					Required:     false,
					ReadOnly:     false,
					Constant:     false,
					DefaultValue: "test",
					Type:         "STRING",
				},
			},
		},
	}
}

func (s *StubApiClient) newStubPipeline(id, name, description string) Pipeline {
	now := time.Now()
	return Pipeline{
		ID:          id,
		Name:        name,
		Description: description,
		Version:     1,
		Creator:     s.user.ID,
		Updater:     s.user.ID,
		CreatedAt:   now.Add(-24 * time.Hour),
		UpdatedAt:   now,
		Variables: []PipelineVariable{
			{
				ID:           "stub-pipeline-var-id-created",
				Name:         "IMAGE_TAG",
				Description:  "Image tag for deployment",
				Required:     true,
				ReadOnly:     false,
				Constant:     false,
				DefaultValue: "latest",
				Type:         "STRING",
			},
		},
	}
}

// ListPipelines 返回模拟的流水线列表
func (s *StubApiClient) ListPipelines(
	ctx context.Context, projectCode, keyword string, page, pageSize int64,
) (int64, []Pipeline, error) {
	log.Infof(ctx, "Stub: ListPipelines request: %s, keyword: %s, page: %d, pageSize: %d",
		projectCode, keyword, page, pageSize,
	)

	pipelines := s.stubPipelines()
	return int64(len(pipelines)), pipelines, nil
}

// GetPipeline 返回模拟的流水线详情
func (s *StubApiClient) GetPipeline(ctx context.Context, projectCode, pipelineID string) (*Pipeline, error) {
	log.Infof(ctx, "Stub: GetPipeline request: %s, %s", projectCode, pipelineID)
	for _, pipeline := range s.stubPipelines() {
		if pipeline.ID == pipelineID {
			p := pipeline
			return &p, nil
		}
	}
	if pipeline, ok := stubCreatedPipelines.Load(pipelineID); ok {
		p := pipeline.(Pipeline)
		p.Updater = s.user.ID
		return &p, nil
	}
	// 兼容历史 stub 行为：对任意自定义 pipelineID 返回一条可用的模拟详情，
	// 避免初始化/校验链路因严格 not found 而级联失败。
	p := s.newStubPipeline(pipelineID, "stub-pipeline", "Stub Pipeline")
	return &p, nil
}

// CreatePipeline 模拟创建流水线，返回模拟的流水线 ID
func (s *StubApiClient) CreatePipeline(
	ctx context.Context, projectCode, name, description string, _ []map[string]any,
) (string, error) {
	log.Infof(ctx, "Stub: CreatePipeline request: %s, %s", projectCode, name)
	pipelineID := "p-stub-pipeline-id-" + stringx.Random(12)
	stubCreatedPipelines.Store(pipelineID, s.newStubPipeline(pipelineID, name, description))
	return pipelineID, nil
}

// UpdatePipeline 模拟更新流水线
func (s *StubApiClient) UpdatePipeline(
	ctx context.Context,
	projectCode, pipelineID, name, description string,
	_ []map[string]any,
) error {
	log.Infof(ctx, "Stub: UpdatePipeline request: %s, %s, %s", projectCode, pipelineID, name)
	pipeline := s.newStubPipeline(pipelineID, name, description)
	if existing, ok := stubCreatedPipelines.Load(pipelineID); ok {
		pipeline = existing.(Pipeline)
		pipeline.Name = name
		pipeline.Description = description
		pipeline.Updater = s.user.ID
		pipeline.UpdatedAt = time.Now()
	}
	stubCreatedPipelines.Store(pipelineID, pipeline)
	return nil
}

// CreatePipelineBuild 模拟创建流水线构建，返回模拟的构建引用
func (s *StubApiClient) CreatePipelineBuild(
	ctx context.Context, projectCode, pipelineID string, _ map[string]string,
) (*PipelineBuildReference, error) {
	log.Infof(ctx, "Stub: CreatePipelineBuild request: %s, %s", projectCode, pipelineID)
	return &PipelineBuildReference{ID: "b-stub-build-id-" + stringx.Random(16), Num: "1"}, nil
}

// GetPipelineBuildState 返回模拟的流水线构建状态
func (s *StubApiClient) GetPipelineBuildState(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (*PipelineBuildState, error) {
	log.Infof(ctx, "Stub: GetPipelineBuildState request: %s, %s, %s", projectCode, pipelineID, buildID)
	return &PipelineBuildState{
		PipelineID:  pipelineID,
		BuildID:     buildID,
		BuildNum:    "1",
		UserID:      s.user.ID,
		Status:      "SUCCEED",
		StartTime:   time.Now().Add(-10 * time.Minute).UnixMilli(),
		EndTime:     time.Now().UnixMilli(),
		TotalTime:   600000, // 10 minutes in milliseconds
		ExecuteTime: 600000,
		Variables: map[string]string{
			"IMAGE_TAG": "v1.0.0",
		},
	}, nil
}

// ListPipelineRepoRefProperties 返回模拟的代码分支/Tag字段列表
func (s *StubApiClient) ListPipelineRepoRefProperties(
	ctx context.Context, projectCode, pipelineID string,
) ([]RepoRefProperty, error) {
	log.Infof(ctx, "Stub: ListPipelineRepoRefProperties request: %s, %s", projectCode, pipelineID)
	return []RepoRefProperty{
		{
			ID:           "branch",
			Name:         "branch",
			Label:        "Code Branch",
			Type:         RepoRefPropertyTypeGitRef,
			Required:     true,
			ReadOnly:     false,
			Constant:     false,
			DefaultValue: "main",
			Value:        "main",
		},
		{
			ID:           "tag",
			Name:         "tag",
			Label:        "Code Tag",
			Type:         RepoRefPropertyTypeSvnTag,
			Required:     false,
			ReadOnly:     false,
			Constant:     false,
			DefaultValue: "",
			Value:        "",
		},
	}, nil
}

// ListPipelineRepoRefOptions 返回模拟的代码分支/Tag选项列表
func (s *StubApiClient) ListPipelineRepoRefOptions(
	ctx context.Context, projectCode, pipelineID, propertyID, search string,
) ([]PipelineVariableOption, error) {
	log.Infof(
		ctx,
		"Stub: ListPipelineRepoRefOptions request: %s, %s, %s, search=%s",
		projectCode,
		pipelineID,
		propertyID,
		search,
	)
	if propertyID == "tag" {
		return []PipelineVariableOption{
			{Key: "refs/tags/v1.0.0", Value: "v1.0.0"},
			{Key: "refs/tags/v1.1.0", Value: "v1.1.0"},
		}, nil
	}
	if search != "" {
		return []PipelineVariableOption{
			{Key: "refs/heads/" + search + "-1.0", Value: search + "-1.0"},
			{Key: "refs/heads/" + search + "-2.0", Value: search + "-2.0"},
		}, nil
	}
	return []PipelineVariableOption{
		{Key: "refs/heads/main", Value: "main"},
		{Key: "refs/heads/release-1.0", Value: "release-1.0"},
	}, nil
}

// ------------------------------------------ 蓝盾构建日志 API ------------------------------------------

// GetInitBuildLog 返回模拟的构建日志初始内容
func (s *StubApiClient) GetInitBuildLog(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (*BuildLog, error) {
	log.Infof(ctx, "Stub: GetInitBuildLog request: %s, %s, %s", projectCode, pipelineID, buildID)
	now := time.Now()
	return &BuildLog{
		HasMore:  false,
		Finished: true,
		Logs: []LogLine{
			{LineNo: 0, Message: "[INFO] Starting build...", Timestamp: now.Add(-5 * time.Minute).UnixMilli()},
			{LineNo: 1, Message: "[INFO] Pulling source code...", Timestamp: now.Add(-4 * time.Minute).UnixMilli()},
			{LineNo: 2, Message: "[INFO] Building image...", Timestamp: now.Add(-2 * time.Minute).UnixMilli()},
			{LineNo: 3, Message: "[INFO] Pushing image...", Timestamp: now.Add(-1 * time.Minute).UnixMilli()},
			{LineNo: 4, Message: "[INFO] Build completed successfully.", Timestamp: now.UnixMilli()},
		},
	}, nil
}

// GetMoreBuildLogs 返回模拟的增量构建日志
func (s *StubApiClient) GetMoreBuildLogs(
	ctx context.Context, projectCode, pipelineID, buildID string, start, _ int64,
) (*BuildLog, error) {
	log.Infof(ctx, "Stub: GetMoreBuildLogs request: %s, %s, %s, start=%d", projectCode, pipelineID, buildID, start)

	if start == 0 {
		initLog, _ := s.GetInitBuildLog(ctx, projectCode, pipelineID, buildID)
		return initLog, nil
	}

	return &BuildLog{
		HasMore:  false,
		Finished: true,
		Logs:     nil,
	}, nil
}

// DownloadBuildLogs 返回模拟的下载日志流
func (s *StubApiClient) DownloadBuildLogs(
	ctx context.Context, projectCode, pipelineID, buildID string,
) (io.ReadCloser, error) {
	log.Infof(ctx, "Stub: DownloadBuildLogs request: %s, %s, %s", projectCode, pipelineID, buildID)
	initLog, _ := s.GetInitBuildLog(ctx, projectCode, pipelineID, buildID)

	var content string
	for _, line := range initLog.Logs {
		content += fmt.Sprintf("%s\n", line.Message)
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

// ------------------------------------------ 蓝盾代码库管理 API ------------------------------------------

// ListRepository 返回模拟的代码库列表
func (s *StubApiClient) ListRepository(
	ctx context.Context, projectCode, repoType string, page, pageSize int64,
) (int64, []Repository, error) {
	log.Infof(ctx, "Stub: ListRepository request: %s, type: %s, page: %d, pageSize: %d",
		projectCode, repoType, page, pageSize,
	)

	repositories := []Repository{
		{
			ID:        "stub-repo-" + stringx.Random(8),
			Alias:     "stub-repo-1-" + stringx.Random(8),
			Url:       "https://git.example.com/stub-group/stub-repo-1.git",
			Type:      "CODE_GIT",
			UpdatedAt: time.Now(),
		},
		{
			ID:        "stub-repo-" + stringx.Random(8),
			Alias:     "stub-repo-2-" + stringx.Random(8),
			Url:       "https://git.example.com/stub-group/stub-repo-2.git",
			Type:      "CODE_GIT",
			UpdatedAt: time.Now(),
		},
	}

	return int64(len(repositories)), repositories, nil
}

// GetRepository 返回模拟的代码库详情
func (s *StubApiClient) GetRepository(ctx context.Context, projectCode, repoHashID string) (*Repository, error) {
	log.Infof(ctx, "Stub: GetRepository request: %s, %s", projectCode, repoHashID)
	return &Repository{
		ID:        repoHashID,
		Alias:     "stub-repo",
		Url:       "https://git.example.com/stub-group/stub-repo.git",
		Type:      "CODE_GIT",
		UpdatedAt: time.Now(),
	}, nil
}

// CreateRepository 模拟创建代码库，返回模拟的代码库 Hash ID
func (s *StubApiClient) CreateRepository(ctx context.Context, projectCode, repoUrl, repoAlias string) (string, error) {
	log.Infof(ctx, "Stub: CreateRepository request: %s, %s, %s", projectCode, repoUrl, repoAlias)
	return "stub-repo-hash-id-32chars-long", nil
}
