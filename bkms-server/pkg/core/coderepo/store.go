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
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// CodeRepositoryMeta 代码库存储元数据
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + alias
// - 普通：tenantID
var CodeRepositoryMeta = store.Meta[CodeRepository]{
	Collection:   "code_repositories",
	TenantScoped: true,
	NotFound:     ErrRepositoryNotFound,
	Encode:       []func(*CodeRepository) error{encryptPassword},
	Decode:       []func(*CodeRepository) error{decryptPassword},
}

// ByWorkspace 按工作空间 ID 匹配
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByWorkspaceAndAlias 按工作空间 ID 与代码库别名匹配
func ByWorkspaceAndAlias(workspaceID, alias string) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("alias", alias))
}

// 复制 BasicAuth 后再改写，避免修改调用方的对象
func encryptPassword(r *CodeRepository) error {
	return transformPassword(r, crypto.AESEncrypt, "encrypt")
}

func decryptPassword(r *CodeRepository) error {
	return transformPassword(r, crypto.AESDecrypt, "decrypt")
}

func transformPassword(r *CodeRepository, fn func(key, data string) (string, error), action string) error {
	if r.BasicAuth == nil {
		return nil
	}
	basicAuth := *r.BasicAuth
	password, err := crypto.OnDemand(config.G.Encrypt.Secret, fn, basicAuth.Password)
	if err != nil {
		return errors.Wrapf(err, "%s repository password", action)
	}
	basicAuth.Password = password
	r.BasicAuth = &basicAuth
	return nil
}
