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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RepositoryInitSpec 关联代码库所需的参数
type RepositoryInitSpec struct {
	// URL 代码库地址
	URL string
	// Alias 代码库在蓝盾侧的别名，同一工作空间内唯一
	Alias string
	// Type 代码库类型，如 TGit
	Type string
	// AuthType 鉴权方式：OAUTH / PASSWORD，为空时取该代码库类型的默认鉴权方式
	AuthType string
	// Username 仓库账号，仅账号密码方式关联时使用
	Username string
	// Password 仓库密码，仅账号密码方式关联时使用；代码库已关联时为空表示沿用已保存的密码
	Password string
}

// EnsureWorkspaceRepositories 确保请求中涉及的代码库均已关联到工作空间，同一请求内同一代码库只处理一次
func EnsureWorkspaceRepositories(
	ctx context.Context, db *mongo.Database, workspaceID string, specs []RepositoryInitSpec,
) error {
	type repoKey struct{ url, alias string }

	manager := NewRepositoryManager(db, workspaceID)
	seen := make(map[repoKey]struct{}, len(specs))
	for _, spec := range specs {
		if spec.URL == "" || spec.Alias == "" {
			continue
		}
		key := repoKey{url: spec.URL, alias: spec.Alias}
		if _, ok := seen[key]; ok {
			continue
		}
		if _, err := manager.Ensure(ctx, spec); err != nil {
			return errors.Wrapf(err, "ensure repository %s", spec.Alias)
		}
		seen[key] = struct{}{}
	}
	return nil
}
