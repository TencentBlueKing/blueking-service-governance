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

// Package build 提供 Helm Chart 构建触发和构建记录管理功能
package build

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci/pipelineparam"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/chart/semver"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
)

// ChartBuildResult 触发构建的结果
type ChartBuildResult struct {
	// ChartVersion 本次构建的 Chart 版本号（semver 格式：major.minor.patch）
	ChartVersion string
	// PipelineID 流水线 ID
	PipelineID string
	// BuildID 蓝盾构建 ID
	BuildID string
}

// ChartBuildService Helm Chart 构建服务，收敛所有外部依赖
type ChartBuildService struct {
	semverStore     semver.CounterStore
	credStore       credential.HelmRepoCredentialStore
	recordStore     RecordStore
	bkciProjStore   bkci.ProjectStore
	bkrepoProjStore bkrepo.ProjectStore
	pipelineStore   bkci.PipelineStore
}

// NewChartBuildService 创建 ChartBuildService 实例
func NewChartBuildService(
	semverStore semver.CounterStore,
	credStore credential.HelmRepoCredentialStore,
	recordStore RecordStore,
	bkciProjStore bkci.ProjectStore,
	bkrepoProjStore bkrepo.ProjectStore,
	pipelineStore bkci.PipelineStore,
) *ChartBuildService {
	return &ChartBuildService{
		semverStore:     semverStore,
		credStore:       credStore,
		recordStore:     recordStore,
		bkciProjStore:   bkciProjStore,
		bkrepoProjStore: bkrepoProjStore,
		pipelineStore:   pipelineStore,
	}
}

// validate 校验阶段：纯检查，无副作用
func (s *ChartBuildService) validate(app *bkmsapp.Application) error {
	appInfo := app.String()

	if !bkmsapp.IsHelmBasedType(app.Type) {
		return errors.Errorf("app %s type is not helm based, but %s", appInfo, app.Type)
	}
	if app.HelmSpec == nil || app.HelmSpec.HelmSource == nil {
		return errors.Errorf("app %s has no helm spec or helm source", appInfo)
	}
	if app.HelmSpec.HelmSource.RepoType != bkmsapp.HelmSourceRepoTypeGit {
		return errors.Errorf(
			"app %s helm source repo type is %s, not HelmGitRepo",
			appInfo, app.HelmSpec.HelmSource.RepoType,
		)
	}
	if app.HelmSpec.HelmSource.GitRepoConfig == nil {
		return errors.Errorf("app %s has no git repo config", appInfo)
	}
	return nil
}

// prepare 准备阶段：确保基础设施就绪（幂等操作）
func (s *ChartBuildService) prepare(ctx context.Context, app *bkmsapp.Application) error {
	appInfo := app.String()
	gitRepoConfig := app.HelmSpec.HelmSource.GitRepoConfig

	// 1. 确保 Helm Git Build 流水线已初始化
	if _, err := bkci.NewPipelineManager(app.WorkspaceID).Initialize(
		ctx, string(bkci.PipelineTypeHelmGitBuild),
	); err != nil {
		return errors.Wrapf(err, "ensure helm-git-build pipeline for workspace %s", app.WorkspaceID)
	}

	// 2. 确保代码库已关联到蓝盾
	if _, err := bkci.NewRepositoryManager(app.WorkspaceID).Initialize(
		ctx, gitRepoConfig.RepoURL, gitRepoConfig.RepoAlias,
	); err != nil {
		return errors.Wrapf(err, "ensure bkci repository for app %s", appInfo)
	}

	// 3. 确保 Helm 仓库凭证已初始化（兜底检查）
	bkciProj, err := s.bkciProjStore.GetByWorkspace(ctx, app.WorkspaceID)
	if err != nil {
		return errors.Wrapf(err, "get bkci project for workspace %s", app.WorkspaceID)
	}
	bkrepoProj, err := s.bkrepoProjStore.GetByWorkspace(ctx, app.WorkspaceID)
	if err != nil {
		return errors.Wrapf(err, "get bkrepo project for workspace %s", app.WorkspaceID)
	}
	if err = credential.EnsureCredential(
		ctx, s.credStore, app.WorkspaceID, bkciProj.Code, bkrepoProj.Username, bkrepoProj.Password,
	); err != nil {
		return errors.Wrapf(err, "ensure helm repo credential for workspace %s", app.WorkspaceID)
	}

	return nil
}

// buildPipelineParams 生成 semver 版本号并构建蓝盾流水线参数
func (s *ChartBuildService) buildPipelineParams(
	ctx context.Context, app *bkmsapp.Application, bumpType, branch, helmRepoURL string,
) (map[string]string, string, error) {
	// 生成 semver 版本号（原子操作，并发安全；尽量延后，避免前序失败浪费版本号）
	chartVersion, err := s.semverStore.Next(ctx, app.ID, semver.BumpType(bumpType))
	if err != nil {
		return nil, "", errors.Wrapf(err, "generate semver for app %s", app.String())
	}

	gitRepoConfig := app.HelmSpec.HelmSource.GitRepoConfig
	params := map[string]string{
		pipelineparam.RepoCheckoutBy:       "BRANCH",
		pipelineparam.RepoRevision:         branch,
		pipelineparam.RepoAlias:            gitRepoConfig.RepoAlias,
		pipelineparam.HelmRepoURL:          helmRepoURL,
		pipelineparam.HelmChartName:        app.Name,
		pipelineparam.HelmChartVersion:     chartVersion,
		pipelineparam.HelmChartBuildDir:    gitRepoConfig.SourceDir,
		pipelineparam.HelmToolchainBaseURL: config.G.Helm.ToolchainBaseURL,
	}
	return params, chartVersion, nil
}

// execute 执行阶段：获取构建信息、生成版本号、触发蓝盾流水线构建、创建构建记录
// 注意：semver 生成尽量延后，确保前序 IO 操作成功后才消费版本号，避免浪费
func (s *ChartBuildService) execute(
	ctx context.Context, app *bkmsapp.Application, bumpType, branch string,
) (*ChartBuildResult, error) {
	appInfo := app.String()

	// 1. 获取 bkrepo HELM 仓库地址
	bkciProj, err := s.bkciProjStore.GetByWorkspace(ctx, app.WorkspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "get bkci project for workspace %s", app.WorkspaceID)
	}
	helmRepoURL, err := config.G.BKRepo.GenRepoEndpoint(bkciProj.Code, string(bkrepo.RepoTypeHelm))
	if err != nil {
		return nil, errors.Wrap(err, "get helm repo endpoint")
	}

	// 2. 获取流水线信息
	pipeline, err := s.pipelineStore.GetByWorkspaceAndType(
		ctx, app.WorkspaceID, string(bkci.PipelineTypeHelmGitBuild),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s helm-git-build pipeline", app.WorkspaceID)
	}

	// 3. 生成 semver 版本号并构建流水线参数
	params, chartVersion, err := s.buildPipelineParams(ctx, app, bumpType, branch, helmRepoURL)
	if err != nil {
		return nil, err
	}

	// 4. 触发蓝盾流水线构建
	bkciClient, err := bkciapi.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}
	buildRef, err := bkciClient.CreatePipelineBuild(ctx, pipeline.ProjectCode, pipeline.ID, params)
	if err != nil {
		return nil, errors.Wrapf(err, "trigger helm-git-build pipeline for app %s", appInfo)
	}

	// 5. 创建构建记录（持久化启动参数，便于制品 / 构建详情页展示分支、commit 等信息）
	record := &Record{
		WorkspaceID:  app.WorkspaceID,
		AppID:        app.ID,
		BuildID:      buildRef.ID,
		PipelineID:   pipeline.ID,
		ChartVersion: chartVersion,
		Status:       StatusRunning,
		Operator:     auth.MustGetUser(ctx).ID,
		Params:       params,
	}
	if err = s.recordStore.Create(ctx, record); err != nil {
		return nil, errors.Wrapf(err, "create helm chart build record for app %s", appInfo)
	}

	return &ChartBuildResult{
		ChartVersion: chartVersion,
		PipelineID:   pipeline.ID,
		BuildID:      buildRef.ID,
	}, nil
}

// ExecuteChartBuild 触发 Helm Chart 构建
// 流程：校验 → 准备（确保流水线 / 代码库 / 凭证就绪）→ 执行（获取信息 → 生成 semver → 触发构建 → 创建记录）
// 注意：异步轮询任务由调用方（Handler）负责启动
func (s *ChartBuildService) ExecuteChartBuild(
	ctx context.Context, app *bkmsapp.Application, bumpType, branch string,
) (*ChartBuildResult, error) {
	// 1. 校验
	if err := s.validate(app); err != nil {
		return nil, err
	}

	// 2. 准备（确保基础设施就绪）
	if err := s.prepare(ctx, app); err != nil {
		return nil, err
	}

	// 3. 执行（生成版本号 + 触发构建 + 创建记录）
	return s.execute(ctx, app, bumpType, branch)
}
