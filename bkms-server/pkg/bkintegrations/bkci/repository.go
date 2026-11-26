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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/coderepo"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	repoCredentialIDPrefix = "bkms_repo_" // #nosec G101
	repoCredentialIDMaxLen = buildTriggerCredentialIDMaxLen
	repoCredentialDescFmt  = "bkms repository credential for %s" // #nosec G101
)

// 代码库标识类型，即蓝盾分支 / Tag 接口的 repositoryType 参数
const (
	// RepositoryIdentityTypeID 蓝盾代码库 Hash ID
	RepositoryIdentityTypeID = "ID"
	// RepositoryIdentityTypeName 代码库别名
	RepositoryIdentityTypeName = "NAME"
)

// bkciRepoTypes 各接入方式同步到蓝盾时的代码库类型
var bkciRepoTypes = map[string]string{
	config.RepoSourceSpecBkCIManaged: bkci.RepoTypeCodeGit,
	config.RepoSourceSpecBasic:       bkci.RepoTypeCodeTGit,
}

// repositorySyncer 同步代码库到工作空间的蓝盾项目
type repositorySyncer struct {
	db      *mongo.Database
	client  bkci.Client
	project *Project
	user    auth.User
}

// managedController 蓝盾托管方式的 Controller，通过蓝盾接口访问代码库
type managedController struct {
	db   *mongo.Database
	repo *coderepo.CodeRepository
}

func init() {
	coderepo.RegisterControllerFactory(config.RepoSourceSpecBkCIManaged, newManagedController)
}

// SaveWorkspaceRepositories 保存代码库；蓝盾托管方式立即同步到蓝盾
func SaveWorkspaceRepositories(ctx context.Context, db *mongo.Database, specs []coderepo.SaveSpec) error {
	specs = lo.Uniq(lo.Filter(specs, func(spec coderepo.SaveSpec, _ int) bool { return spec.URL != "" }))
	for _, spec := range specs {
		repo, err := coderepo.Save(ctx, db, spec)
		if err != nil {
			return errors.Wrapf(err, "save repository %s", spec.URL)
		}
		sourceType, err := coderepo.ResolveSourceType(repo.SourceType)
		if err != nil {
			return errors.Wrapf(err, "repository %s", repo.Alias)
		}
		if sourceType.Spec != config.RepoSourceSpecBkCIManaged {
			continue
		}
		if _, err = SyncRepository(ctx, db, repo); err != nil {
			return errors.Wrapf(err, "sync repository %s to bkci", repo.Alias)
		}
	}
	return nil
}

// SyncBuildRepository 构建前同步代码库到蓝盾，bkms 中没有记录时（历史数据）先按构建配置保存
func SyncBuildRepository(
	ctx context.Context, db *mongo.Database, workspaceID, repoType, repoURL, alias string,
) (*Repository, error) {
	repo, err := coderepo.Get(ctx, db, workspaceID, alias)
	if errors.Is(err, coderepo.ErrRepositoryNotFound) {
		repo, err = coderepo.Save(ctx, db, coderepo.SaveSpec{
			WorkspaceID: workspaceID, Type: repoType, URL: repoURL, Alias: alias,
		})
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get repository %s", alias)
	}
	return SyncRepository(ctx, db, repo)
}

// SyncRepository 同步代码库到蓝盾：蓝盾托管方式以 OAuth 关联，basic 方式先用 bkms 的账号密码覆盖蓝盾凭证
func SyncRepository(ctx context.Context, db *mongo.Database, repo *coderepo.CodeRepository) (*Repository, error) {
	sourceType, err := coderepo.ResolveSourceType(repo.SourceType)
	if err != nil {
		return nil, errors.Wrapf(err, "repository %s", repo.Alias)
	}
	syncer, err := newRepositorySyncer(ctx, db, repo.WorkspaceID)
	if err != nil {
		return nil, err
	}

	switch sourceType.Spec {
	case config.RepoSourceSpecBkCIManaged:
		return syncer.ensure(ctx, repo, bkciRepoTypes[sourceType.Spec], "")
	case config.RepoSourceSpecBasic:
		if repo.BasicAuth == nil {
			return nil, errors.Wrapf(coderepo.ErrCredentialRequired, "repository %s", repo.Alias)
		}
		credentialID := repositoryCredentialID(repo.Alias)
		if err = syncer.upsertCredential(ctx, credentialID, repo); err != nil {
			return nil, err
		}
		return syncer.ensure(ctx, repo, bkciRepoTypes[sourceType.Spec], credentialID)
	default:
		return nil, errors.Wrapf(coderepo.ErrUnsupported, "sync %s repository %s to bkci", sourceType.Spec, repo.Alias)
	}
}

// ListWorkspaceRepositoryRefs 列出代码库分支或 Tag：bkms 中有记录的按接入方式查询，否则走蓝盾接口
func ListWorkspaceRepositoryRefs(
	ctx context.Context,
	db *mongo.Database,
	workspaceID, repositoryID, repositoryType string,
	query coderepo.RefsQuery,
) ([]coderepo.Ref, error) {
	alias := repositoryID
	if repositoryType == RepositoryIdentityTypeID {
		record, err := store.Of(db, RepositoryMeta).Get(ctx, ByWorkspaceAndRepositoryID(workspaceID, repositoryID))
		switch {
		case err == nil:
			alias = record.Alias
		case errors.Is(err, ErrRepositoryNotFound):
			alias = ""
		default:
			return nil, errors.Wrapf(err, "get workspace %s bkci repository %s", workspaceID, repositoryID)
		}
	}

	if alias != "" {
		repo, err := coderepo.Get(ctx, db, workspaceID, alias)
		switch {
		case err == nil:
			controller, ctrlErr := coderepo.NewController(ctx, db, repo)
			if ctrlErr != nil {
				return nil, ctrlErr
			}
			refs, listErr := controller.ListRefs(ctx, query)
			if listErr != nil {
				return nil, errors.Wrapf(listErr, "list %s of repository %s", query.Kind, alias)
			}
			return refs, nil
		case !errors.Is(err, coderepo.ErrRepositoryNotFound):
			return nil, err
		}
	}
	return listRefsViaBkCI(ctx, db, workspaceID, repositoryID, repositoryType, query)
}

// Touch 通过列出分支校验能否访问
func (c *managedController) Touch(ctx context.Context) error {
	_, err := c.ListRefs(ctx, coderepo.RefsQuery{Kind: coderepo.RefKindBranch, Page: 1, PageSize: 1})
	return err
}

// ListRefs 按别名通过蓝盾接口列出分支或 Tag
func (c *managedController) ListRefs(ctx context.Context, query coderepo.RefsQuery) ([]coderepo.Ref, error) {
	return listRefsViaBkCI(ctx, c.db, c.repo.WorkspaceID, c.repo.Alias, RepositoryIdentityTypeName, query)
}

func newRepositorySyncer(ctx context.Context, db *mongo.Database, workspaceID string) (*repositorySyncer, error) {
	project, err := store.Of(db, ProjectMeta).Get(ctx, ByWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get project by workspace %s", workspaceID)
	}
	user := auth.MustGetUser(ctx)
	client, err := bkci.New(user)
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}
	return &repositorySyncer{db: db, client: client, project: project, user: user}, nil
}

// ensure 蓝盾代码库不存在时创建；credentialID 为空表示以 OAuth 关联
func (s *repositorySyncer) ensure(
	ctx context.Context, repo *coderepo.CodeRepository, repoType, credentialID string,
) (*Repository, error) {
	repos := store.Of(s.db, RepositoryMeta)
	existing, err := repos.Get(ctx, ByWorkspaceAndAlias(repo.WorkspaceID, repo.Alias))
	switch {
	case err == nil && existing.CredentialID == credentialID:
		return existing, nil
	case err == nil:
		return nil, errors.Errorf("bkci repository %s is bound with credential %q, expect %q",
			repo.Alias, existing.CredentialID, credentialID)
	case !errors.Is(err, ErrRepositoryNotFound):
		return nil, errors.Wrapf(err, "get workspace %s bkci repository %s", repo.WorkspaceID, repo.Alias)
	}

	opts := bkci.CreateRepositoryOptions{
		RepoType: repoType, URL: repo.URL, Alias: repo.Alias, CredentialID: credentialID,
	}
	hashID, err := s.createOrReuse(ctx, opts)
	if err != nil {
		return nil, err
	}
	bkciRepo, err := s.client.GetRepository(ctx, s.project.Code, hashID)
	if err != nil {
		return nil, errors.Wrapf(err, "get bkci repository %s in bkci project %s", hashID, s.project.Code)
	}
	record := &Repository{
		ID:           hashID,
		Alias:        repo.Alias,
		URL:          bkciRepo.Url,
		Type:         bkciRepo.Type,
		WorkspaceID:  repo.WorkspaceID,
		ProjectCode:  s.project.Code,
		CredentialID: credentialID,
		Creator:      s.user.ID,
	}
	if err = repos.Create(ctx, record); err != nil {
		return nil, errors.Wrapf(err, "insert bkci repository %s to db", repo.Alias)
	}
	return record, nil
}

// createOrReuse 创建蓝盾代码库，别名已存在时复用
func (s *repositorySyncer) createOrReuse(ctx context.Context, opts bkci.CreateRepositoryOptions) (string, error) {
	hashID, err := s.client.CreateRepository(ctx, s.project.Code, opts)
	if err == nil {
		return hashID, nil
	}
	if !errors.Is(err, bkci.RepoAlreadyExist) {
		return "", errors.Wrapf(err, "create bkci repository with url: %s, alias: %s", opts.URL, opts.Alias)
	}
	_, repositories, err := s.client.ListRepository(
		ctx, s.project.Code, "", bkci.PageForAllItems, bkci.PageSizeForAllItems,
	)
	if err != nil {
		return "", errors.Wrapf(err, "list bkci repositories for project %s", s.project.Code)
	}
	// 蓝盾别名忽略大小写
	existing, ok := lo.Find(
		repositories,
		func(r bkci.Repository) bool { return strings.EqualFold(r.Alias, opts.Alias) },
	)
	if !ok {
		return "", errors.Errorf("repository with alias %s not found in bkci project %s", opts.Alias, s.project.Code)
	}
	return existing.ID, nil
}

// upsertCredential 用 bkms 的账号密码覆盖蓝盾凭证，不存在时创建
func (s *repositorySyncer) upsertCredential(
	ctx context.Context, credentialID string, repo *coderepo.CodeRepository,
) error {
	desc := fmt.Sprintf(repoCredentialDescFmt, repo.Alias)
	username, password := repo.BasicAuth.Username, repo.BasicAuth.Password
	err := s.client.UpdateCredential(ctx, s.project.Code, credentialID, desc, username, password)
	if errors.Is(err, bkci.ObjectNotFound) {
		err = s.client.CreateCredential(ctx, s.project.Code, credentialID, desc, username, password)
	}
	if err != nil {
		return errors.Wrapf(err, "sync credential %s of repository %s to bkci", credentialID, repo.Alias)
	}
	return nil
}

// repositoryCredentialID 由别名生成固定的凭证 ID，超长时用摘要
func repositoryCredentialID(alias string) string {
	sanitized := strings.Trim(nonCredentialIDCharPattern.ReplaceAllString(alias, "_"), "_")
	if id := repoCredentialIDPrefix + sanitized; sanitized != "" && len(id) <= repoCredentialIDMaxLen {
		return id
	}
	sum := sha256.Sum256([]byte(alias))
	digestLen := repoCredentialIDMaxLen - len(repoCredentialIDPrefix)
	return repoCredentialIDPrefix + hex.EncodeToString(sum[:digestLen/2])
}

func newManagedController(
	_ context.Context,
	db *mongo.Database,
	repo *coderepo.CodeRepository,
) (coderepo.Controller, error) {
	return &managedController{db: db, repo: repo}, nil
}

// listRefsViaBkCI 通过蓝盾接口列出分支或 Tag
func listRefsViaBkCI(
	ctx context.Context,
	db *mongo.Database,
	workspaceID, repositoryID, repositoryType string,
	query coderepo.RefsQuery,
) ([]coderepo.Ref, error) {
	project, err := store.Of(db, ProjectMeta).Get(ctx, ByWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get project by workspace %s", workspaceID)
	}
	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}
	listFunc := client.ListRepositoryBranches
	if query.Kind == coderepo.RefKindTag {
		listFunc = client.ListRepositoryTags
	}
	refs, err := listFunc(ctx, project.Code, repositoryID, repositoryType, query.Search, query.Page, query.PageSize)
	if err != nil {
		return nil, errors.Wrapf(err, "list %s of repository %s via bkci", query.Kind, repositoryID)
	}
	return lo.Map(refs, func(ref bkci.RepositoryRef, _ int) coderepo.Ref {
		return coderepo.Ref{Name: ref.Name, Path: ref.Path, SHA: ref.SHA, LinkURL: ref.LinkURL}
	}), nil
}
