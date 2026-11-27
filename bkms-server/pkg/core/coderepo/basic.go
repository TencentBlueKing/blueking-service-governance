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

package coderepo

import (
	"context"
	"slices"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/repoutil/git"
)

// basicController basic 方式的 Controller，用账号密码通过 git 协议访问
type basicController struct {
	client *git.Client
}

func init() {
	RegisterControllerFactory(config.RepoSourceSpecBasic, newBasicController)
}

// Touch 校验账号密码能否访问
func (c *basicController) Touch(ctx context.Context) error {
	return convertGitErr(c.client.Touch(ctx), c.client.URL)
}

// ListRefs 列出分支或 Tag，过滤后分页
func (c *basicController) ListRefs(ctx context.Context, query RefsQuery) ([]Ref, error) {
	refs, err := c.client.ListRefs(ctx)
	if err != nil {
		return nil, convertGitErr(err, c.client.URL)
	}

	var items []git.Ref
	if query.Kind == RefKindTag {
		items = refs.Tags
		sortTagsDesc(items)
	} else {
		items = refs.Branches
		sortBranches(items, refs.DefaultBranch)
	}
	if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" {
		items = lo.Filter(items, func(ref git.Ref, _ int) bool {
			return strings.Contains(strings.ToLower(ref.Name), search)
		})
	}

	if query.Page < 1 || query.PageSize < 1 {
		return []Ref{}, nil
	}
	page := lo.Subset(items, int((query.Page-1)*query.PageSize), uint(query.PageSize))
	return lo.Map(page, func(ref git.Ref, _ int) Ref {
		return Ref{Name: ref.Name, Path: ref.Path, SHA: ref.SHA}
	}), nil
}

func newBasicController(_ context.Context, _ *mongo.Database, repo *CodeRepository) (Controller, error) {
	if repo.BasicAuth == nil || repo.BasicAuth.Username == "" || repo.BasicAuth.Password == "" {
		return nil, errors.Wrapf(ErrCredentialRequired, "repository %s", repo.URL)
	}
	client, err := git.NewClient(repo.URL, git.Credential{
		Username: repo.BasicAuth.Username, Password: repo.BasicAuth.Password,
	})
	if err != nil {
		return nil, convertGitErr(err, repo.URL)
	}
	return &basicController{client: client}, nil
}

// convertGitErr 转换 git 错误
func convertGitErr(err error, repoURL string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, git.ErrAuthFailed):
		return errors.Wrapf(ErrCredentialInvalid, "access repository %s: %v", repoURL, err)
	default:
		return errors.Wrapf(ErrUnreachable, "access repository %s: %v", repoURL, err)
	}
}

// sortBranches 默认分支在前，其余按名称
func sortBranches(branches []git.Ref, defaultBranch string) {
	slices.SortStableFunc(branches, func(a, b git.Ref) int {
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

// sortTagsDesc semver Tag 从新到旧在前，其余按名称倒序
func sortTagsDesc(tags []git.Ref) {
	slices.SortStableFunc(tags, func(a, b git.Ref) int {
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
