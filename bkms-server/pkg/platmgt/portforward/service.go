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

package portforward

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrPermissionDenied 表示用户无 port-forward 白名单权限。
	ErrPermissionDenied = errors.New("no port-forward permission")

	// ErrProductionEnvNotAllowed 表示不允许将 production 类型的环境加入白名单。
	ErrProductionEnvNotAllowed = errors.New("production environment is not allowed in port-forward whitelist")

	// ErrEnvNotFound 表示指定的环境 ID 不存在。
	ErrEnvNotFound = errors.New("environment not found")
)

// Service 提供 port-forward 白名单业务操作。
type Service struct {
	whitelist *Store
	envStore  *envmodel.EnvironmentStore
}

// NewService 创建白名单 Service 实例。
func NewService(db *mongo.Database) *Service {
	return &Service{whitelist: store.Of(db, Meta), envStore: store.Of(db, envmodel.EnvMeta)}
}

// Add 将环境 ID 添加到白名单中，添加前校验环境存在性和非 production 类型；已在白名单中的环境 ID 会被跳过。
// 环境可以来自不同租户，每条记录归属其环境所在的租户
func (s *Service) Add(ctx context.Context, envIDs []string) error {
	envs, err := s.getEnvs(ctx, lo.Uniq(envIDs))
	if err != nil {
		return err
	}
	for tenantID, tenantEnvs := range lo.GroupBy(envs, func(env *envmodel.Environment) string { return env.TenantID }) {
		entries := lo.Map(tenantEnvs, func(env *envmodel.Environment, _ int) *WhitelistEntry {
			return &WhitelistEntry{EnvID: env.ID.Hex()}
		})
		if err = s.whitelist.CreateManyIfAbsent(ctx, entries, store.TenantOf(tenantID)); err != nil {
			return errors.Wrapf(err, "add env IDs of tenant %s to port-forward whitelist", tenantID)
		}
	}
	return nil
}

// Remove 从白名单中移除指定的环境 ID（不限租户），不存在的环境 ID 会被忽略。
func (s *Service) Remove(ctx context.Context, envIDs []string) error {
	if len(envIDs) == 0 {
		return nil
	}
	if _, err := s.whitelist.DeleteMany(ctx, ByEnvIDs(lo.Uniq(envIDs)), store.TenantAll()); err != nil {
		return errors.Wrap(err, "remove env IDs from port-forward whitelist")
	}
	return nil
}

// List 返回白名单中全部租户的环境 ID。
func (s *Service) List(ctx context.Context) ([]string, error) {
	entries, err := s.whitelist.List(ctx, store.TenantAll())
	if err != nil {
		return nil, errors.Wrap(err, "list port-forward whitelist")
	}
	return lo.Map(entries, func(e WhitelistEntry, _ int) string { return e.EnvID }), nil
}

// CheckPermission 校验指定环境 ID 是否在白名单中。
// 返回 nil 表示有权限，返回 ErrPermissionDenied 表示无权限。
func (s *Service) CheckPermission(ctx context.Context, envID string) error {
	ok, err := s.whitelist.Exists(ctx, store.ByID(envID))
	if err != nil {
		return errors.Wrapf(err, "check port-forward whitelist for env %s", envID)
	}
	if !ok {
		return ErrPermissionDenied
	}
	return nil
}

// getEnvs 跨租户查出 envIDs 对应的环境，并校验每个环境：
// 1. 环境是否存在（通过 envID 查询数据库）
// 2. 环境类型是否为非 production
func (s *Service) getEnvs(ctx context.Context, envIDs []string) ([]*envmodel.Environment, error) {
	envs := make([]*envmodel.Environment, 0, len(envIDs))
	for _, envID := range envIDs {
		objID, err := bson.ObjectIDFromHex(envID)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid environment ID %q", envID)
		}

		env, err := s.envStore.GetByID(ctx, objID, store.TenantAll())
		if err != nil {
			if errors.Is(err, envmodel.ErrEnvNotFound) {
				return nil, errors.Wrapf(ErrEnvNotFound, "environment %q", envID)
			}
			return nil, err
		}

		// 校验环境类型是否为非 production
		if bkmsenv.IsProductionType(bkmsenv.Type(env.Type)) {
			return nil, ErrProductionEnvNotAllowed
		}
		envs = append(envs, env)
	}

	return envs, nil
}
