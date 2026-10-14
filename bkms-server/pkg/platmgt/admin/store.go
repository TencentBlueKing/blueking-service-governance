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

package admin

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrRoleBindingNotFound indicates the target platform administrator role binding does not exist.
	ErrRoleBindingNotFound = errors.New("platform administrator role binding not found")
	// ErrRoleBindingAlreadyExists indicates the target platform administrator role binding already exists.
	ErrRoleBindingAlreadyExists = errors.New("platform administrator role binding already exists")
	// ErrPermissionDenied indicates the current user is not a platform administrator.
	ErrPermissionDenied = errors.New("platform administrator permission denied")
)

// Store persists platform administrator role bindings.
type Store = store.GenericStore[RoleBinding]

// RoleBindingMeta is the storage metadata of platform administrator role bindings.
//
// 索引（由 golang-migrate 维护）：
// - 唯一：username
var RoleBindingMeta = store.Meta[RoleBinding]{
	Collection: "plat_admin_role_bindings",
	NotFound:   ErrRoleBindingNotFound,
	Duplicate:  ErrRoleBindingAlreadyExists,
	BeforeSave: []func(*RoleBinding) error{defaultUpdater},
}

// defaultUpdater 未指定更新人时，以创建人作为更新人
func defaultUpdater(rb *RoleBinding) error {
	if rb.Updater == "" {
		rb.Updater = rb.Creator
	}
	return nil
}

// ByUsername matches the role binding of the given username.
func ByUsername(username string) store.Opt {
	return store.Where("username", username)
}

// UsernameContains fuzzy matches usernames containing keyword case-insensitively,
// it does not filter when keyword is blank.
func UsernameContains(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "username")
}
