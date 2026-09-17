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
	"net/url"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/gitremote"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// 账号密码方式关联代码库时可由用户修正的错误，接口层据此返回可读的提示，见 RepositoryErrToBkErr
var (
	// ErrRepoCredentialRequired 首次关联代码库时未填写仓库账号或密码
	ErrRepoCredentialRequired = errors.New("repository username and password are required")
	// ErrRepoPasswordRequired 修改仓库账号时未重新填写仓库密码
	ErrRepoPasswordRequired = errors.New("repository password is required when username changes")
	// ErrRepoCredentialInvalid 仓库账号或密码不正确，或该账号无权访问仓库
	ErrRepoCredentialInvalid = errors.New("repository username or password is incorrect")
	// ErrRepoUnreachable 无法通过 HTTP(S) 访问代码库（地址错误、不是 Git 仓库、网络不通等）
	ErrRepoUnreachable = errors.New("repository is unreachable")
)

const (
	// repoCredentialIDPrefix 代码库凭证 ID 前缀
	repoCredentialIDPrefix = "bkms_repo_" // #nosec G101
	// repoCredentialIDMaxLen 代码库凭证 ID 最大长度，与触发流水线凭证保持一致
	repoCredentialIDMaxLen = buildTriggerCredentialIDMaxLen
)

// RefKind 代码库引用类型
type RefKind string

const (
	// RefKindBranch 分支
	RefKindBranch RefKind = "branch"
	// RefKindTag Tag
	RefKindTag RefKind = "tag"
)

// RepositoryErrToBkErr 把代码库相关错误转换为接口错误：用户可修正的问题返回业务错误与可读提示，
// 其余视为内部错误，以 action 描述正在执行的操作
func RepositoryErrToBkErr(err error, action string) error {
	switch {
	case errors.Is(err, ErrRepoCredentialInvalid):
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, "仓库账号或密码不正确，无法访问该代码库")
	case errors.Is(err, ErrRepoCredentialRequired):
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, "首次关联该代码库需要填写仓库账号和仓库密码")
	case errors.Is(err, ErrRepoPasswordRequired):
		return bkerrs.New(bkerrs.ErrCodeInvalidRequest, "修改仓库账号时需要重新填写仓库密码")
	case errors.Is(err, ErrRepoUnreachable):
		return bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "无法访问该代码库，请确认填写的是可访问的 HTTPS 仓库地址")
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, action)
	}
}

// VerifyRepositoryCredential 使用仓库账号密码访问代码库，校验地址与账号密码是否正确，不关联代码库
func VerifyRepositoryCredential(ctx context.Context, repoURL, username, password string) error {
	if username == "" || password == "" {
		return errors.Wrapf(ErrRepoCredentialRequired, "verify repository %s", repoURL)
	}
	_, err := gitremote.ListRefs(ctx, repoURL, gitremote.Credential{Username: username, Password: password})
	return convertGitRemoteErr(err, repoURL)
}

// convertGitRemoteErr 把 Git 远端访问错误归类为代码库错误
func convertGitRemoteErr(err error, repoURL string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gitremote.ErrAuthFailed):
		return errors.Wrapf(ErrRepoCredentialInvalid, "access repository %s: %v", repoURL, err)
	default:
		// 地址非法、仓库不存在、非 Smart HTTP 响应或网络错误，均需用户检查仓库地址
		return errors.Wrapf(ErrRepoUnreachable, "access repository %s: %v", repoURL, err)
	}
}

// repositoryCredentialID 生成代码库凭证 ID：前缀 + 净化后的别名，超长时改用别名摘要；同一别名总是得到同一 ID
func repositoryCredentialID(alias string) string {
	sanitized := strings.Trim(nonCredentialIDCharPattern.ReplaceAllString(alias, "_"), "_")
	if id := repoCredentialIDPrefix + sanitized; sanitized != "" && len(id) <= repoCredentialIDMaxLen {
		return id
	}
	sum := sha256.Sum256([]byte(alias))
	digestLen := repoCredentialIDMaxLen - len(repoCredentialIDPrefix)
	return repoCredentialIDPrefix + hex.EncodeToString(sum[:digestLen/2])
}

// repositoryCredentialDesc 代码库凭证在蓝盾侧的描述
func repositoryCredentialDesc(alias string) string {
	return "[bkms] 代码库 " + alias + " 的仓库账号密码"
}

// repositoryAuthType 按仓库地址协议选择蓝盾代码库授权方式（账号密码凭证）
func repositoryAuthType(repoURL string) string {
	if u, err := url.Parse(repoURL); err == nil && u.Scheme == "http" {
		return bkci.AuthTypeHTTP
	}
	return bkci.AuthTypeHTTPS
}

// ensureRepositoryCredential 创建代码库凭证，创建失败时（如蓝盾残留同 ID 凭证）改为更新
func ensureRepositoryCredential(
	ctx context.Context, client bkci.Client, projectCode, credentialID, alias, username, password string,
) error {
	desc := repositoryCredentialDesc(alias)
	createErr := client.CreateCredential(ctx, projectCode, credentialID, desc, username, password)
	if createErr == nil {
		return nil
	}
	if err := client.UpdateCredential(ctx, projectCode, credentialID, desc, username, password); err != nil {
		return errors.Wrapf(createErr, "create credential %s for repository %s (update also failed: %v)",
			credentialID, alias, err)
	}
	return nil
}

// rollbackRepositoryCredential 关联代码库失败时回收刚创建的凭证，失败只记日志
func rollbackRepositoryCredential(ctx context.Context, client bkci.Client, projectCode, credentialID string) {
	if err := client.DeleteCredential(ctx, projectCode, credentialID); err != nil &&
		!errors.Is(err, bkci.ObjectNotFound) {
		log.Warnf(ctx, "rollback bkci credential %s in project %s failed: %v", credentialID, projectCode, err)
	}
}

// 代码仓库标识类型，与蓝盾分支 / Tag 查询接口的 repositoryType 参数一致
const (
	// RepositoryIdentityTypeID 按蓝盾代码库 Hash ID 标识
	RepositoryIdentityTypeID = "ID"
	// RepositoryIdentityTypeName 按代码库别名标识
	RepositoryIdentityTypeName = "NAME"
)

// RepositoryRefsQuery 代码库分支 / Tag 查询参数
type RepositoryRefsQuery struct {
	// Kind 引用类型：分支或 Tag
	Kind RefKind
	// RepositoryID 代码库 Hash ID 或别名，由 RepositoryType 决定
	RepositoryID string
	// RepositoryType 代码库标识类型：ID 或 NAME
	RepositoryType string
	// Search 名称关键字
	Search string
	// Page 页码，从 1 开始
	Page int64
	// PageSize 每页数量
	PageSize int64
}

// ListRepositoryRefs 列出工作空间代码库的分支或 Tag
//
// 以账号密码关联的代码库直接使用仓库账号密码从 Git 仓库查询（此类环境的蓝盾通常未提供查询接口），
// 其余代码库（OAuth 关联或本地无记录）通过蓝盾接口查询。
func ListRepositoryRefs(
	ctx context.Context, db *mongo.Database, workspaceID string, query RepositoryRefsQuery,
) ([]bkci.RepositoryRef, error) {
	repoOpt := ByWorkspaceAndAlias(workspaceID, query.RepositoryID)
	if query.RepositoryType == RepositoryIdentityTypeID {
		repoOpt = ByWorkspaceAndRepositoryID(workspaceID, query.RepositoryID)
	}
	repository, err := store.Of(db, RepositoryMeta).Get(ctx, repoOpt)
	switch {
	case err == nil && repository.UsesCredential():
		return ListRepositoryRefsWithCredential(
			ctx, repository, query.Kind, query.Search, query.Page, query.PageSize,
		)
	case err != nil && !errors.Is(err, ErrRepositoryNotFound):
		return nil, errors.Wrapf(err, "get workspace %s repository %s", workspaceID, query.RepositoryID)
	}

	project, err := store.Of(db, ProjectMeta).Get(ctx, ByWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "get project by workspace %s", workspaceID)
	}
	client, err := bkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}
	listFunc := client.ListRepositoryBranches
	if query.Kind == RefKindTag {
		listFunc = client.ListRepositoryTags
	}
	refs, err := listFunc(
		ctx, project.Code, query.RepositoryID, query.RepositoryType, query.Search, query.Page, query.PageSize,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list %s of repository %s via bkci", query.Kind, query.RepositoryID)
	}
	return refs, nil
}

// ListRepositoryRefsWithCredential 使用代码库记录中的账号密码直接从 Git 仓库列出分支或 Tag，按关键字过滤后分页
//
// 排序：分支默认分支在前、其余按名称；Tag 按语义化版本从新到旧，非语义化版本的 Tag 排在其后按名称倒序
func ListRepositoryRefsWithCredential(
	ctx context.Context, repository *Repository, kind RefKind, search string, page, pageSize int64,
) ([]bkci.RepositoryRef, error) {
	refs, err := gitremote.ListRefs(ctx, repository.URL, gitremote.Credential{
		Username: repository.Username, Password: repository.Password,
	})
	if err != nil {
		return nil, convertGitRemoteErr(err, repository.URL)
	}

	var items []gitremote.Ref
	if kind == RefKindTag {
		items = refs.Tags
		sortTagsDesc(items)
	} else {
		items = refs.Branches
		sortBranches(items, refs.DefaultBranch)
	}

	if search = strings.ToLower(strings.TrimSpace(search)); search != "" {
		items = slices.DeleteFunc(slices.Clone(items), func(ref gitremote.Ref) bool {
			return !strings.Contains(strings.ToLower(ref.Name), search)
		})
	}

	start := (page - 1) * pageSize
	if page < 1 || pageSize < 1 || start >= int64(len(items)) {
		return []bkci.RepositoryRef{}, nil
	}
	end := min(start+pageSize, int64(len(items)))

	result := make([]bkci.RepositoryRef, 0, end-start)
	for _, ref := range items[start:end] {
		result = append(result, bkci.RepositoryRef{Name: ref.Name, Path: ref.Path, SHA: ref.SHA})
	}
	return result, nil
}

// sortBranches 默认分支排在最前，其余按名称排序
func sortBranches(branches []gitremote.Ref, defaultBranch string) {
	slices.SortStableFunc(branches, func(a, b gitremote.Ref) int {
		switch {
		case a.Name == defaultBranch && b.Name != defaultBranch:
			return -1
		case b.Name == defaultBranch && a.Name != defaultBranch:
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
}

// sortTagsDesc 语义化版本 Tag 按版本从新到旧排在前，其余按名称倒序排在后
func sortTagsDesc(tags []gitremote.Ref) {
	slices.SortStableFunc(tags, func(a, b gitremote.Ref) int {
		va, errA := semver.NewVersion(a.Name)
		vb, errB := semver.NewVersion(b.Name)
		switch {
		case errA == nil && errB == nil:
			return vb.Compare(va)
		case errA == nil:
			return -1
		case errB == nil:
			return 1
		default:
			return strings.Compare(b.Name, a.Name)
		}
	})
}
