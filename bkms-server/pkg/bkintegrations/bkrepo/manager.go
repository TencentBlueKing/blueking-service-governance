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

package bkrepo

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/passwd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	bkciapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkrepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	helmchartcred "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/helmcore/credential"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
)

// 蓝盾制品库公共账户密码长度
const bkRepoAccountPasswordLength = 32

// ProjectManager 蓝盾制品项目管理
type ProjectManager struct {
	workspaceID string
	operator    string
	// 公共账户（如：g_bkms_test）
	username string
	password string
}

// NewProjectManager ...
func NewProjectManager(workspaceID, operator string) *ProjectManager {
	return &ProjectManager{
		workspaceID: workspaceID,
		operator:    operator,
		username:    fmt.Sprintf("g_bkms_%s", strings.ReplaceAll(workspaceID, "-", "_")),
		password:    passwd.New(bkRepoAccountPasswordLength),
	}
}

// Initialize 执行蓝盾制品库初始化，确保工作空间对应的项目 & 仓库存在
func (m *ProjectManager) Initialize(ctx context.Context) error {
	// 1. 查询数据库以获取蓝盾项目 Code
	bkciProjStore, err := bkci.NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrapf(err, "create bkci project store")
	}
	proj, err := bkciProjStore.GetByWorkspace(ctx, m.workspaceID)
	if err != nil {
		return errors.Wrapf(err, "get bkci project for workspace %s to initialize bkrepo", m.workspaceID)
	}

	client, err := bkrepo.New(m.operator)
	if err != nil {
		return errors.Wrapf(err, "create bkrepo client")
	}
	// 2. 初始化 & 记录蓝盾制品库项目
	if err = m.initProject(ctx, client, proj.Code); err != nil {
		return errors.Wrap(err, "init bkrepo project")
	}
	// 3. 初始化 & 记录蓝盾制品库仓库
	if err = m.initRepositories(ctx, client, proj.Code); err != nil {
		return errors.Wrap(err, "init bkrepo repositories")
	}
	// 4. 将镜像源凭证初始化到蓝盾凭证管理 & 记录引用信息到数据库中
	if err = m.initImageRegistryCredentials(ctx, proj.Code); err != nil {
		return errors.Wrap(err, "init image registry credentials")
	}
	// 5. 将 Helm 仓库凭证初始化到蓝盾凭证管理 & 记录引用信息到数据库中
	if err = m.initHelmRepoCredentials(ctx, proj.Code); err != nil {
		return errors.Wrap(err, "init helm repo credentials")
	}

	return nil
}

// 初始化蓝盾制品库项目
func (m *ProjectManager) initProject(
	ctx context.Context, client bkrepo.Client, projectID string,
) error {
	store, err := NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrap(err, "create bkrepo project store")
	}

	// 1. 查询 DB，确认是否有关联记录
	if _, err = store.GetByWorkspace(ctx, m.workspaceID); err == nil {
		return nil
	} else if !errors.Is(err, ErrProjectNotFound) {
		return err
	}

	// 2. 调用蓝盾制品库 API 创建项目
	if err = client.CreateProject(ctx, projectID); err != nil {
		return errors.Wrapf(err, "create bkrepo %s", projectID)
	}

	// 3. 将制品库项目信息记录到数据库
	project := &Project{
		ID:          projectID,
		WorkspaceID: m.workspaceID,
		Username:    m.username,
		Password:    m.password,
		Creator:     m.operator,
	}
	if err = store.Create(ctx, project); err != nil {
		return errors.Wrap(err, "insert bkrepo project")
	}

	// 4. 创建公共账号用户 & 关联到蓝盾制品库项目，并且为操作人添加关联
	if err = client.CreateUserToProject(
		ctx, projectID, m.username, m.password, []string{m.operator},
	); err != nil {
		return errors.Wrapf(err, "create bkrepo user to project %s", m.username)
	}

	return nil
}

// 初始化蓝盾制品库仓库
func (m *ProjectManager) initRepositories(
	ctx context.Context, client bkrepo.Client, projectID string,
) error {
	store, err := NewRepositoryStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrap(err, "create bkrepo repository store")
	}

	for _, repo := range config.G.BKRepo.InitRepos {
		// 1. 逐个仓库检查是否存在，不存在的需要创建
		if _, err = store.GetByWorkspaceAndType(ctx, m.workspaceID, RepoType(repo.Type)); err == nil {
			continue
		} else if !errors.Is(err, ErrRepositoryNotFound) {
			return err
		}

		// 2. 调用蓝盾制品库 API 创建仓库
		if err = client.CreateRepository(
			ctx, projectID, repo.Name, repo.Type, repo.Description, repo.IsPublic,
		); err != nil {
			return errors.Wrapf(err, "create repository %s", repo.Name)
		}

		// 3. 使用 Store 将仓库信息记录到数据库
		repository := &Repository{
			WorkspaceID: m.workspaceID,
			ProjectID:   projectID,
			Name:        repo.Name,
			Type:        RepoType(repo.Type),
			IsPublic:    repo.IsPublic,
			Creator:     m.operator,
		}
		if err = store.Create(ctx, repository); err != nil {
			return errors.Wrap(err, "insert bkrepo repository")
		}
	}
	return nil
}

// 镜像源账密配置添加到蓝盾凭证管理
func (m *ProjectManager) initImageRegistryCredentials(ctx context.Context, projectID string) error {
	registry, err := config.G.BKRepo.GenRepoEndpoint(projectID, string(RepoTypeDocker))
	if err != nil {
		return errors.Wrap(err, "generate image registry from config")
	}

	store, err := bkmsreg.NewImageRegistryStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrap(err, "create image registry store")
	}

	// 1. 查询 DB，确认是否有关联记录，若存在，直接返回
	if _, err = store.GetByWorkspaceAndType(ctx, m.workspaceID, bkmsreg.ImageRegistryTypeBuiltin); err == nil {
		return nil
	} else if !errors.Is(err, bkmsreg.ErrImageRegistryNotFound) {
		return err
	}

	// 2. 数据库中没有该镜像源配置的记录，重新初始化
	credentialID := strings.ReplaceAll(uuid.NewString(), "-", "")
	description := fmt.Sprintf("bkms image credential (system-bkrepo) for registry %s", registry)

	user := auth.MustGetUser(ctx)
	// 3. 在蓝盾凭证管理中创建凭证信息
	bkciClient, err := bkciapi.New(user)
	if err != nil {
		return errors.Wrap(err, "create bkci client")
	}
	if err = bkciClient.CreateCredential(
		ctx, projectID, credentialID, description, m.username, m.password,
	); err != nil {
		return errors.Wrap(err, "create bkci credential")
	}

	// 4. 使用 Store 将凭证、镜像源信息存储入库
	imageReg := &bkmsreg.ImageRegistry{
		WorkspaceID:      m.workspaceID,
		Type:             bkmsreg.ImageRegistryTypeBuiltin,
		Registry:         registry,
		Username:         m.username,
		Password:         m.password,
		BkCICredentialID: credentialID,
	}
	if _, err = store.Create(ctx, imageReg); err != nil {
		return errors.Wrap(err, "insert image registry")
	}

	return nil
}

// Helm 仓库凭证初始化到蓝盾凭证管理
func (m *ProjectManager) initHelmRepoCredentials(ctx context.Context, projectCode string) error {
	// 1. 创建 Helm 仓库凭证 store
	credStore, err := helmchartcred.NewHelmRepoCredentialStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrap(err, "create helm repo credential store")
	}

	// 2. 幂等初始化 Helm 仓库凭证
	return helmchartcred.EnsureCredential(ctx, credStore, m.workspaceID, projectCode, m.username, m.password)
}
