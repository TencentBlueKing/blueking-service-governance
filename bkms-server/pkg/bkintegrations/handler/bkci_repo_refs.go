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

package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkciproject "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	build "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// repositoryTypeName 按代码库别名查询分支 / 标签
const repositoryTypeName = "NAME"

// repositoryRefsLister 与 bkci.Client.ListRepositoryBranches / ListRepositoryTags 的方法表达式签名一致
type repositoryRefsLister func(
	client bkci.Client, ctx context.Context,
	projectCode, repositoryID, repositoryType, search string, page, pageSize int64,
) ([]bkci.RepositoryRef, error)

// listBkCIRepositoryRefs 查询代码库分支 / 标签。
// 蓝盾按别名找不到代码库时（存量应用从未关联，或在蓝盾上被删除），用当前用户身份重新关联后重试一次；
// 失败时已写入错误响应，返回 false
func (h *Handler) listBkCIRepositoryRefs(
	c *gin.Context, action string, list repositoryRefsLister,
) ([]bkci.RepositoryRef, bool) {
	var uriInput slz.BkCIWorkspaceURIInput
	var queryInput slz.BkCIRepositoryRefsQueryInput
	if err := ginutils.BindURIQuery(c, &uriInput, &queryInput); err != nil {
		bkerrs.AbortWithErr(c, err)
		return nil, false
	}

	ctx := c.Request.Context()
	project, err := h.bkciProjects().Get(ctx, bkciproject.ByWorkspace(uriInput.WorkspaceID))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "get workspace project"))
		return nil, false
	}

	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		bkerrs.AbortWithErr(c, bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "initial bkci client"))
		return nil, false
	}

	doList := func() ([]bkci.RepositoryRef, error) {
		return list(
			client, ctx, project.Code,
			queryInput.RepositoryID, queryInput.RepositoryType, queryInput.Search,
			queryInput.Page, queryInput.PageSize,
		)
	}

	refs, err := doList()
	if errors.Is(err, bkci.ObjectNotFound) && queryInput.RepositoryType == repositoryTypeName {
		if rebindErr := h.rebindRepository(ctx, uriInput.WorkspaceID, queryInput.RepositoryID); rebindErr != nil {
			bkerrs.AbortWithErr(c, rebindErr)
			return nil, false
		}
		refs, err = doList()
	}
	if err != nil {
		if errors.Is(err, bkci.ObjectNotFound) {
			bkerrs.AbortWithErr(
				c, bkerrs.WrapBkCIRepositoryNotBound(err, uriInput.WorkspaceID, queryInput.RepositoryID),
			)
		} else {
			bkerrs.AbortWithErr(c, bkciproject.WrapRepositoryErr(err, action))
		}
		return nil, false
	}
	return refs, true
}

// rebindRepository 按别名找到仓库地址，在蓝盾重新关联代码库
func (h *Handler) rebindRepository(ctx context.Context, workspaceID, alias string) error {
	url, err := h.lookupRepositoryURL(ctx, workspaceID, alias)
	if err != nil {
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "lookup repository url")
	}
	if url == "" {
		return bkerrs.WrapBkCIRepositoryNotBound(
			errors.Errorf("no repository url configured for alias %s", alias), workspaceID, alias,
		)
	}

	if _, err = bkciproject.NewRepositoryManager(h.registry.DB, workspaceID).Rebind(ctx, url, alias); err != nil {
		return bkciproject.WrapRepositoryErr(err, "rebind bkci repository")
	}
	log.Infof(ctx, "rebind bkci repository for workspace %s, alias: %s, url: %s", workspaceID, alias, url)
	return nil
}

// lookupRepositoryURL 按别名查找工作空间内的仓库地址，找不到时返回空字符串。
// 依次查找：已关联的蓝盾代码库记录、Helm Chart 的 Git 来源、代码库来源的构建配置。
// 涉及模型：bkci.Repository、app.Application、image.Config
func (h *Handler) lookupRepositoryURL(ctx context.Context, workspaceID, alias string) (string, error) {
	repo, err := store.Of(h.registry.DB, bkciproject.RepositoryMeta).Get(
		ctx, bkciproject.ByWorkspaceAndAlias(workspaceID, alias),
	)
	if err != nil && !errors.Is(err, bkciproject.ErrRepositoryNotFound) {
		return "", errors.Wrapf(err, "get workspace %s bkci repository %s", workspaceID, alias)
	}
	if repo != nil && repo.URL != "" {
		return repo.URL, nil
	}

	apps := store.Of(h.registry.DB, bkmsapp.AppMeta)
	app, err := apps.Get(
		ctx,
		bkmsapp.InWorkspace(workspaceID),
		store.Where("helmSpec.helmSource.repoType", bkmsapp.HelmSourceRepoTypeGit),
		store.Where("helmSpec.helmSource.gitRepoConfig.repoAlias", alias),
	)
	if err != nil && !errors.Is(err, bkmsapp.ErrAppNotFound) {
		return "", errors.Wrapf(err, "get helm app in workspace %s by repo alias %s", workspaceID, alias)
	}
	if app != nil && app.HelmSpec != nil && app.HelmSpec.HelmSource != nil &&
		app.HelmSpec.HelmSource.GitRepoConfig != nil && app.HelmSpec.HelmSource.GitRepoConfig.RepoURL != "" {
		return app.HelmSpec.HelmSource.GitRepoConfig.RepoURL, nil
	}

	workspaceApps, err := apps.List(ctx, bkmsapp.InWorkspace(workspaceID), store.SelectFields("id"))
	if err != nil {
		return "", errors.Wrapf(err, "list apps in workspace %s", workspaceID)
	}
	if len(workspaceApps) == 0 {
		return "", nil
	}
	appIDs := lo.Map(workspaceApps, func(a bkmsapp.Application, _ int) string { return a.ID })
	cfg, err := store.Of(h.registry.DB, build.ConfigMeta).Get(
		ctx,
		store.Filter(bson.M{"appID": bson.M{"$in": appIDs}}),
		store.Where("sourceType", build.SourceTypeCodeRepository),
		store.Where("codeRepo.repoAlias", alias),
	)
	if err != nil && !errors.Is(err, build.ErrConfigNotFound) {
		return "", errors.Wrapf(err, "get build config in workspace %s by repo alias %s", workspaceID, alias)
	}
	if cfg != nil && cfg.CodeRepo != nil {
		return cfg.CodeRepo.RepoURL, nil
	}
	return "", nil
}
