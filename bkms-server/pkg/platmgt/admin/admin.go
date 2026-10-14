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

// Package admin 提供平台级管理员角色绑定与查询能力，作为 platmgt 模块的一部分
package admin

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// Service provides platform administrator role binding operations.
type Service struct {
	roleBindings *Store
}

// NewService creates a platform administrator role binding manager backed by the given database.
func NewService(db *mongo.Database) *Service {
	return &Service{roleBindings: store.Of(db, RoleBindingMeta)}
}

// List lists platform administrator role bindings sorted by username, keyword fuzzy matches
// usernames case-insensitively and is ignored when empty.
func (a *Service) List(ctx context.Context, keyword string) ([]RoleBinding, error) {
	roleBindings, err := a.roleBindings.List(ctx, UsernameContains(keyword), store.Asc("username"))
	if err != nil {
		return nil, errors.Wrap(err, "list platform administrator role bindings")
	}
	return roleBindings, nil
}

// GetRole returns the current platform role of the given username.
func (a *Service) GetRole(ctx context.Context, username string) (RoleCode, bool, error) {
	roleBinding, err := a.roleBindings.Get(ctx, ByUsername(username))
	if err == nil {
		return roleBinding.RoleCode, true, nil
	}
	if errors.Is(err, ErrRoleBindingNotFound) {
		return "", false, nil
	}
	return "", false, errors.Wrapf(err, "get platform role of user %s", username)
}

// AssignRoles adds platform administrator roles for multiple users.
// Existing role bindings are skipped silently to keep the operation idempotent.
func (a *Service) AssignRoles(
	ctx context.Context,
	usernames []string,
	roleCode RoleCode,
	operator string,
) error {
	if !isValidRoleCode(roleCode) {
		return ErrInvalidRoleCode
	}

	roleBindings := lo.Map(lo.Uniq(usernames), func(username string, _ int) *RoleBinding {
		return &RoleBinding{
			Username: username,
			RoleCode: roleCode,
			Creator:  operator,
			Updater:  operator,
		}
	})
	if err := a.roleBindings.CreateManyIfAbsent(ctx, roleBindings); err != nil {
		return errors.Wrap(err, "create platform administrator role bindings")
	}
	return nil
}

// RevokeRole removes a platform administrator by username.
// Revoking a user without platform role succeeds silently.
func (a *Service) RevokeRole(ctx context.Context, username string) error {
	if err := a.roleBindings.DeleteIfExists(ctx, ByUsername(username)); err != nil {
		return errors.Wrapf(err, "delete platform administrator role binding of user %s", username)
	}
	return nil
}
