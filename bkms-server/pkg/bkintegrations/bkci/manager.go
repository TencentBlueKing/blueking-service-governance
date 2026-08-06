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
	"slices"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/tof"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// ErrProjectCodeAlreadyUsed 项目 Code 已存在但不属于指定工作空间（项目与工作空间是一对一绑定的关系）
var ErrProjectCodeAlreadyUsed = errors.New("project code already used")

// ProjectManager 蓝盾项目管理
type ProjectManager struct {
	workspaceID string
}

// NewProjectManager ...
func NewProjectManager(workspaceID string) *ProjectManager {
	return &ProjectManager{workspaceID: workspaceID}
}

// Initialize 为工作空间新建 / 绑定蓝盾项目
// 重要：
//  1. ctx 中应该携带 username 和 ticket
//  2. 如果没指定 projectCode，则使用默认规则生成（bkms-{workspaceID})
func (m *ProjectManager) Initialize(
	ctx context.Context,
	projectCode, obsProductID, obsProductName string,
) (*Project, error) {
	// 如果没有指定 projectCode（使用已有项目），则使用默认规则生成
	if projectCode == "" {
		projectCode = m.genDefaultProjectCode(m.workspaceID)
	}

	store, err := NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrapf(err, "get project store")
	}

	// 1. 检查指定的工作空间是否有关联记录
	if proj, gErr := store.GetByWorkspace(ctx, m.workspaceID); gErr != nil && !errors.Is(gErr, ErrProjectNotFound) {
		// DB 查询失败（非记录不存在）需要返回异常
		return nil, errors.Wrapf(gErr, "find workspace %s project", m.workspaceID)
	} else if proj != nil {
		if proj.Code == projectCode {
			// 空间已经关联指定的蓝盾项目，直接返回
			return proj, nil
		}
		// 空间关联的是其他项目，报错
		return nil, errors.Errorf("workspace %s already has project %s", m.workspaceID, proj.Code)
	}

	// 2. 检查 projectCode 是否已经被占用
	if proj, gErr := store.GetByCode(ctx, projectCode); gErr != nil && !errors.Is(gErr, ErrProjectNotFound) {
		return nil, errors.Wrapf(gErr, "get project %s", projectCode)
	} else if proj != nil {
		return nil, errors.Wrapf(
			ErrProjectCodeAlreadyUsed, "project %s used by workspace %s", projectCode, proj.WorkspaceID,
		)
	}

	// 3. 创建蓝盾项目 & 数据入库
	proj, err := m.createProject(ctx, projectCode, obsProductID, obsProductName)
	if err != nil {
		return nil, errors.Wrapf(err, "create bkci project %s", projectCode)
	}

	return proj, nil
}

// createProject 创建蓝盾项目 & 入库
func (m *ProjectManager) createProject(
	ctx context.Context,
	projectCode, obsProductID, obsProductName string,
) (*Project, error) {
	user := auth.MustGetUser(ctx)

	// 1. 获取蓝盾项目的具体信息（若不存在则创建）
	client, err := bkci.New(user)
	if err != nil {
		return nil, errors.Wrapf(err, "create bkci client")
	}
	bkciProj, err := client.GetProject(ctx, projectCode)
	if err != nil {
		if errors.Is(err, bkci.ObjectNotFound) {
			// 蓝盾项目不存在，走创建逻辑
			if bkciProj, err = m.createBKCIProject(ctx, projectCode, obsProductID, obsProductName); err != nil {
				// 创建还是出错，应该抛出
				return nil, errors.Wrapf(err, "create bkci project %s", projectCode)
			}
		} else {
			// 非项目不存在的错误需要被抛出
			return nil, errors.Wrapf(err, "get bkci project %s", projectCode)
		}
	}

	proj := &Project{
		ID:          bkciProj.ID,
		Code:        bkciProj.Code,
		WorkspaceID: m.workspaceID,
		Creator:     user.ID,
	}

	// 2. 使用 Store 关联工作空间 - 蓝盾项目记录
	store, err := NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create project store")
	}
	if err = store.Create(ctx, proj); err != nil {
		return nil, errors.Wrap(err, "insert bkci project")
	}
	return proj, nil
}

// createBKCIProject 调用蓝盾 API 创建项目，并在项目已存在时容忍、重新拉取项目信息返回。
func (m *ProjectManager) createBKCIProject(
	ctx context.Context,
	projectCode, obsProductID, obsProductName string,
) (*bkci.Project, error) {
	user := auth.MustGetUser(ctx)
	// 1. 先查当前用户的组织信息
	organization, err := tof.GetUserOrganization(ctx, user.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to fetch user %s organization", user.ID)
	}

	// 2. 调用蓝盾 API 创建项目
	client, err := bkci.New(user)
	if err != nil {
		return nil, errors.Wrapf(err, "create bkci client")
	}
	if err = client.CreateProject(ctx, projectCode, obsProductID, obsProductName, organization); err != nil {
		// 可能存在下面的情况，都应该容忍
		// 1. RPC 超时取消，导致项目创建后没有记录到 DB
		// 2. 用户指定的是已经存在的蓝盾项目
		if !errors.Is(err, bkci.ProjectAlreadyExist) {
			return nil, errors.Wrapf(err, "create bkci project %s", projectCode)
		}
	}

	// 3. 由于创建项目 API 不会返回对象，因此重新调 API 获取
	project, err := client.GetProject(ctx, projectCode)
	if err != nil {
		return nil, errors.Wrapf(err, "get bkci project after create")
	}
	return project, nil
}

// genDefaultProjectCode 生成默认的蓝盾项目 Code
func (m *ProjectManager) genDefaultProjectCode(workspaceID string) string {
	return "bkms-" + workspaceID
}

// PipelineManager 蓝盾流水线管理
type PipelineManager struct {
	workspaceID string
}

// NewPipelineManager ...
func NewPipelineManager(workspaceID string) *PipelineManager {
	return &PipelineManager{workspaceID: workspaceID}
}

// Initialize 初始化流水线，如果已存在则跳过
func (m *PipelineManager) Initialize(ctx context.Context, pipelineType string) (*Pipeline, error) {
	store, err := NewPipelineStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create pipeline store")
	}
	pipeline, err := store.GetByWorkspaceAndType(ctx, m.workspaceID, pipelineType)
	if err != nil {
		if errors.Is(err, ErrPipelineNotFound) {
			// 不存在，需要创建
			return m.createPipeline(ctx, pipelineType)
		}
		return nil, errors.Wrapf(err, "get workspace %s pipeline %s", m.workspaceID, pipelineType)
	}
	if !m.isBuiltinPipelineType(pipelineType) {
		return pipeline, nil
	}

	return m.ensureBuiltinPipelineTemplateVersion(ctx, store, pipeline)
}

// isBuiltinPipelineType 是否为内置流水线类型
func (m *PipelineManager) isBuiltinPipelineType(pipelineType string) bool {
	return slices.Contains(builtinPipelineTypes, PipelineType(pipelineType))
}

// shouldUpdateBuiltinPipelineFromTemplate 判断内置流水线是否需要根据模板更新。
// 空版本或非法 currentVersion 都视为需要更新，允许从模板自愈到合法版本；
// 非法 templateVersion 由调用方处理，避免纯判断函数产生日志等副作用。
func shouldUpdateBuiltinPipelineFromTemplate(currentVersion, templateVersion string) (bool, error) {
	current, err := semver.StrictNewVersion(currentVersion)
	if err != nil {
		return true, nil // nolint: nilerr
	}
	target, err := semver.StrictNewVersion(templateVersion)
	if err != nil {
		return false, err
	}
	return current.LessThan(target), nil
}

// ensureBuiltinPipelineTemplateVersion 确保内置流水线的已应用模板版本与当前模板一致，
// 如果模板版本更新，则同步更新蓝盾上的流水线及本地存储。
func (m *PipelineManager) ensureBuiltinPipelineTemplateVersion(
	ctx context.Context,
	store PipelineStore,
	pipeline *Pipeline,
) (*Pipeline, error) {
	tmpl, err := m.getPipelineTemplate(ctx, pipeline.Type)
	if err != nil {
		return nil, err
	}
	shouldUpdate, err := shouldUpdateBuiltinPipelineFromTemplate(pipeline.TemplateVersion, tmpl.Version)
	if err != nil {
		log.Warnf(ctx, "invalid bkci pipeline template version %q: %v", tmpl.Version, err)
		return pipeline, nil
	}
	if !shouldUpdate {
		return pipeline, nil
	}
	return m.updateBuiltinPipelineFromTemplate(ctx, store, pipeline, tmpl)
}

// createPipeline 在数据库中添加流水线
func (m *PipelineManager) createPipeline(ctx context.Context, pipelineType string) (*Pipeline, error) {
	var pipelineID string
	var tmpl *PipelineTemplate
	var tmplVersion string
	var err error

	// 创建 bkci API 客户端
	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}

	// 如果需要创建流水线，且是内置的流水线类型，说明蓝盾上不存在，此时需要调用蓝盾 API 创建
	if m.isBuiltinPipelineType(pipelineType) {
		tmpl, err = m.getPipelineTemplate(ctx, pipelineType)
		if err != nil {
			return nil, errors.Wrapf(err, "get pipeline template with type: %s", pipelineType)
		}
		pipelineID, err = m.createBKCIPipeline(ctx, tmpl)
		if err != nil {
			return nil, errors.Wrapf(err, "create bkci pipeline with type: %s", pipelineType)
		}
		tmplVersion = tmpl.Version
	} else {
		// 用户自定义的流水线，其 ID 与 Type 相同
		pipelineID = pipelineType
		tmplVersion = ""
	}

	// 获取项目信息
	projectStore, err := NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create project store")
	}
	project, err := projectStore.GetByWorkspace(ctx, m.workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "get project by workspace %s", m.workspaceID)
	}
	// 调用 bkci API 获取流水线信息，确保流水线确实存在
	bkciPipeline, err := client.GetPipeline(ctx, project.Code, pipelineID)
	if err != nil {
		return nil, errors.Wrapf(err, "get bkci pipeline %s to verify existence", pipelineID)
	}

	// 构建 Pipeline 对象
	pipeline := &Pipeline{
		ID:              pipelineID,
		Type:            pipelineType,
		WorkspaceID:     m.workspaceID,
		ProjectCode:     project.Code,
		Name:            bkciPipeline.Name,
		Description:     bkciPipeline.Description,
		TemplateVersion: tmplVersion,
		Creator:         auth.MustGetUser(ctx).ID,
	}

	// 入库
	store, err := NewPipelineStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create pipeline store")
	}
	if err = store.Create(ctx, pipeline); err != nil {
		return nil, errors.Wrap(err, "insert pipeline to db")
	}

	return pipeline, nil
}

// getPipelineTemplate 根据流水线类型从模板存储中获取对应的流水线模板。
func (m *PipelineManager) getPipelineTemplate(ctx context.Context, pipelineType string) (*PipelineTemplate, error) {
	tmplStore, err := NewDBPipelineTemplateStore(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create pipeline template store")
	}
	tmpl, err := tmplStore.GetByType(ctx, pipelineType)
	if err != nil {
		return nil, errors.Wrapf(err, "get pipeline template by type %s", pipelineType)
	}
	return tmpl, nil
}

// updateBuiltinPipelineFromTemplate 将蓝盾上的内置流水线同步为指定模板的最新状态，
// 并更新本地存储中的名称、描述及模板版本。
func (m *PipelineManager) updateBuiltinPipelineFromTemplate(
	ctx context.Context,
	store PipelineStore,
	pipeline *Pipeline,
	tmpl *PipelineTemplate,
) (*Pipeline, error) {
	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}
	if err = client.UpdatePipeline(
		ctx, pipeline.ProjectCode, pipeline.ID, tmpl.Name, tmpl.Description, tmpl.Stages,
	); err != nil {
		return nil, errors.Wrapf(
			err, "update workspace %s builtin pipeline %s in project %s from template version %s to %s",
			m.workspaceID, pipeline.Type, pipeline.ProjectCode, pipeline.TemplateVersion, tmpl.Version,
		)
	}

	pipeline.Name = tmpl.Name
	pipeline.Description = tmpl.Description
	pipeline.TemplateVersion = tmpl.Version
	if err = store.UpdateBuiltinTemplateVersion(ctx, pipeline); err != nil {
		return nil, errors.Wrapf(
			err, "update workspace %s pipeline %s template version to %s",
			m.workspaceID, pipeline.Type, tmpl.Version,
		)
	}
	return pipeline, nil
}

// createBKCIPipeline 在蓝盾上创建流水线（通过蓝盾 API 创建），返回 pipelineID & error
func (m *PipelineManager) createBKCIPipeline(ctx context.Context, tmpl *PipelineTemplate) (string, error) {
	// 1. 获取工作空间对应的项目 Code
	projectStore, err := NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return "", errors.Wrap(err, "create project store")
	}
	project, err := projectStore.GetByWorkspace(ctx, m.workspaceID)
	if err != nil {
		return "", errors.Wrapf(err, "get project by workspace %s", m.workspaceID)
	}

	// 2. 调用 bkci API client 创建流水线
	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return "", errors.Wrapf(err, "create bkci client")
	}
	pipelineID, err := client.CreatePipeline(ctx, project.Code, tmpl.Name, tmpl.Description, tmpl.Stages)
	if err != nil {
		return "", errors.Wrapf(err, "create bkci pipeline")
	}

	return pipelineID, nil
}

// RepositoryManager 蓝盾代码库管理
type RepositoryManager struct {
	workspaceID string
}

// NewRepositoryManager ...
func NewRepositoryManager(workspaceID string) *RepositoryManager {
	return &RepositoryManager{workspaceID: workspaceID}
}

// Initialize 初始化代码库，如果已存在则跳过
func (m *RepositoryManager) Initialize(ctx context.Context, url, alias string) (*Repository, error) {
	store, err := NewRepositoryStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create repository store")
	}
	repository, err := store.GetByWorkspaceAndAlias(ctx, m.workspaceID, alias)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			// 不存在，需要创建
			return m.createRepository(ctx, url, alias)
		}
		return nil, errors.Wrapf(err, "get workspace %s repository %s", m.workspaceID, alias)
	}

	return repository, nil
}

// createRepository 在数据库中添加代码库
func (m *RepositoryManager) createRepository(ctx context.Context, url, alias string) (*Repository, error) {
	// 创建 bkci API 客户端
	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}

	// 获取项目信息
	projectStore, err := NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create project store")
	}
	project, err := projectStore.GetByWorkspace(ctx, m.workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "get project by workspace %s", m.workspaceID)
	}

	// 调用 bkci API 创建代码库
	repoID, err := client.CreateRepository(ctx, project.Code, url, alias)
	if err != nil {
		// 如果代码库已存在，尝试从蓝盾获取已有的代码库信息
		if errors.Is(err, bkci.RepoAlreadyExist) {
			repoID, err = m.getExistingRepository(ctx, client, project.Code, alias)
			if err != nil {
				return nil, errors.Wrapf(err, "get existing bkci repository with url: %s, alias: %s", url, alias)
			}
		} else {
			return nil, errors.Wrapf(err, "create bkci repository with url: %s, alias: %s", url, alias)
		}
	}

	// 调用 bkci API 获取代码库信息，确保代码库确实存在
	repo, err := client.GetRepository(ctx, project.Code, repoID)
	if err != nil {
		return nil, errors.Wrapf(err, "get bkci repository %s in bkci project %s", repoID, project.Code)
	}

	// 构建 Repository 对象
	repository := &Repository{
		ID:          repoID,
		Alias:       alias,
		URL:         repo.Url,
		Type:        repo.Type,
		WorkspaceID: m.workspaceID,
		ProjectCode: project.Code,
		Creator:     auth.MustGetUser(ctx).ID,
	}

	// 入库
	store, err := NewRepositoryStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create bkci repository store")
	}
	if err = store.Create(ctx, repository); err != nil {
		return nil, errors.Wrap(err, "insert bkci repository to db")
	}
	return repository, nil
}

// getExistingRepository 从蓝盾获取已存在的代码库信息（当创建时发现已存在）
func (m *RepositoryManager) getExistingRepository(
	ctx context.Context, client bkci.Client, projectCode, alias string,
) (string, error) {
	// 从蓝盾获取代码库列表，查找匹配的代码库
	_, repositories, err := client.ListRepository(ctx, projectCode, "", bkci.PageForAllItems, bkci.PageSizeForAllItems)
	if err != nil {
		return "", errors.Wrapf(err, "list bkci repositories for project %s", projectCode)
	}

	// 查找匹配 alias 的代码库
	for _, repo := range repositories {
		if repo.Alias == alias {
			// 找到匹配的代码库
			return repo.ID, nil
		}
	}

	return "", errors.Errorf("repository with alias %s not found in bkci project %s", alias, projectCode)
}
