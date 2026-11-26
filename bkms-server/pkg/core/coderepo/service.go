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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// SaveSpec 保存代码库的参数
type SaveSpec struct {
	// WorkspaceID 工作空间 ID
	WorkspaceID string
	// Type 接入方式名
	Type string
	// URL 仓库地址
	URL string
	// Alias 别名，仅蓝盾托管方式使用
	Alias string
	// Username 仓库账号，仅 basic 方式使用
	Username string
	// Password 仓库密码，仅 basic 方式使用，为空时沿用已保存的
	Password string
}

// ResolveAlias 蓝盾托管方式用传入的别名，其余由仓库地址生成
func ResolveAlias(sourceType SourceType, repoURL, alias string) (string, error) {
	if sourceType.Spec == config.RepoSourceSpecBkCIManaged {
		if alias == "" {
			return "", errors.Wrapf(ErrAliasRequired, "repository %s", repoURL)
		}
		return alias, nil
	}
	return DeriveAlias(sourceType.Name, repoURL)
}

// Save 创建或更新代码库，basic 方式账号密码变化时先校验能否访问
func Save(ctx context.Context, db *mongo.Database, spec SaveSpec) (*CodeRepository, error) {
	sourceType, err := ResolveSourceType(spec.Type)
	if err != nil {
		return nil, err
	}
	alias, err := ResolveAlias(sourceType, spec.URL, spec.Alias)
	if err != nil {
		return nil, err
	}

	repos := store.Of(db, CodeRepositoryMeta)
	existing, err := repos.Get(ctx, ByWorkspaceAndAlias(spec.WorkspaceID, alias))
	if err != nil && !errors.Is(err, ErrRepositoryNotFound) {
		return nil, errors.Wrapf(err, "get workspace %s repository %s", spec.WorkspaceID, alias)
	}

	target := &CodeRepository{WorkspaceID: spec.WorkspaceID, Alias: alias, URL: spec.URL, SourceType: sourceType.Name}
	if existing != nil {
		target.Creator, target.CreatedAt = existing.Creator, existing.CreatedAt
	} else if user, userErr := auth.GetUser(ctx); userErr == nil {
		target.Creator = user.ID
	}

	credentialChanged := false
	if sourceType.Spec == config.RepoSourceSpecBasic {
		var current *BasicAuth
		if existing != nil && existing.SourceType == sourceType.Name {
			current = existing.BasicAuth
		}
		if target.BasicAuth, credentialChanged, err = mergeBasicAuth(current, spec.Username, spec.Password); err != nil {
			return nil, errors.Wrapf(err, "repository %s", alias)
		}
	}

	if existing == nil {
		if err = touchIfBasic(ctx, db, sourceType, target); err != nil {
			return nil, err
		}
		if err = repos.Create(ctx, target); err != nil {
			return nil, errors.Wrapf(err, "create workspace %s repository %s", spec.WorkspaceID, alias)
		}
		return target, nil
	}

	if !credentialChanged && existing.URL == target.URL && existing.SourceType == target.SourceType {
		return existing, nil
	}
	if err = touchIfBasic(ctx, db, sourceType, target); err != nil {
		return nil, err
	}
	if err = repos.Update(
		ctx, target, store.UpdateSelected("url", "sourceType", "basicAuth"),
		ByWorkspaceAndAlias(spec.WorkspaceID, alias),
	); err != nil {
		return nil, errors.Wrapf(err, "update workspace %s repository %s", spec.WorkspaceID, alias)
	}
	return target, nil
}

// Get 获取代码库
func Get(ctx context.Context, db *mongo.Database, workspaceID, alias string) (*CodeRepository, error) {
	repo, err := store.Of(db, CodeRepositoryMeta).Get(ctx, ByWorkspaceAndAlias(workspaceID, alias))
	if err != nil {
		return nil, errors.Wrapf(err, "get workspace %s repository %s", workspaceID, alias)
	}
	return repo, nil
}

// Probe 按未保存的代码库配置创建 Controller，basic 方式未传密码时沿用已保存的
func Probe(ctx context.Context, db *mongo.Database, spec SaveSpec) (Controller, error) {
	sourceType, err := ResolveSourceType(spec.Type)
	if err != nil {
		return nil, err
	}
	alias, err := ResolveAlias(sourceType, spec.URL, spec.Alias)
	if err != nil {
		return nil, err
	}
	repo := &CodeRepository{WorkspaceID: spec.WorkspaceID, Alias: alias, URL: spec.URL, SourceType: sourceType.Name}
	if sourceType.Spec == config.RepoSourceSpecBasic {
		var current *BasicAuth
		existing, getErr := store.Of(db, CodeRepositoryMeta).Get(ctx, ByWorkspaceAndAlias(spec.WorkspaceID, alias))
		switch {
		case getErr == nil && existing.SourceType == sourceType.Name:
			current = existing.BasicAuth
		case getErr != nil && !errors.Is(getErr, ErrRepositoryNotFound):
			return nil, errors.Wrapf(getErr, "get workspace %s repository %s", spec.WorkspaceID, alias)
		}
		if repo.BasicAuth, _, err = mergeBasicAuth(current, spec.Username, spec.Password); err != nil {
			return nil, errors.Wrapf(err, "repository %s", alias)
		}
	}
	return NewController(ctx, db, repo)
}

// mergeBasicAuth 合并已保存与传入的账号密码，返回结果及是否变化
func mergeBasicAuth(current *BasicAuth, username, password string) (*BasicAuth, bool, error) {
	if password == "" {
		switch {
		case current == nil:
			return nil, false, ErrCredentialRequired
		case username != "" && username != current.Username:
			return nil, false, ErrPasswordRequired
		default:
			return current, false, nil
		}
	}
	if username == "" && current != nil {
		username = current.Username
	}
	if username == "" {
		return nil, false, ErrCredentialRequired
	}
	merged := &BasicAuth{Username: username, Password: password}
	changed := current == nil || *current != *merged
	return merged, changed, nil
}

// touchIfBasic basic 方式校验能否访问
func touchIfBasic(ctx context.Context, db *mongo.Database, sourceType SourceType, repo *CodeRepository) error {
	if sourceType.Spec != config.RepoSourceSpecBasic {
		return nil
	}
	controller, err := NewController(ctx, db, repo)
	if err != nil {
		return err
	}
	return controller.Touch(ctx)
}
