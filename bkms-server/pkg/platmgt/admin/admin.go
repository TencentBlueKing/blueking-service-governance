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
	"errors"
	"strings"

	"github.com/samber/lo"
)

// Service provides platform administrator role binding operations.
type Service struct {
	store Store
}

// NewService creates a platform administrator role binding manager backed by the given store.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List lists platform administrator role bindings with optional keyword filtering.
func (a *Service) List(ctx context.Context, keyword string) ([]RoleBinding, error) {
	return a.store.List(ctx, &ListOptions{Keyword: strings.TrimSpace(keyword)})
}

// GetRole returns the current platform role of the given username.
func (a *Service) GetRole(ctx context.Context, username string) (RoleCode, bool, error) {
	roleBinding, err := a.store.Get(ctx, username)
	if err == nil {
		return roleBinding.RoleCode, true, nil
	}
	if errors.Is(err, ErrRoleBindingNotFound) {
		return "", false, nil
	}
	return "", false, err
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

	roleBindings := lo.Map(usernames, func(username string, _ int) *RoleBinding {
		return &RoleBinding{
			Username: username,
			RoleCode: roleCode,
			Creator:  operator,
			Updater:  operator,
		}
	})
	return a.store.CreateMany(ctx, roleBindings)
}

// RevokeRole removes a platform administrator by username.
func (a *Service) RevokeRole(ctx context.Context, username string) error {
	return a.store.Delete(ctx, username)
}
