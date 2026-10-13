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

package bkrepo

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrProjectNotFound 蓝盾制品库项目未找到
	ErrProjectNotFound = errors.New("bkrepo project not found")
	// ErrRepositoryNotFound 蓝盾制品库仓库未找到
	ErrRepositoryNotFound = errors.New("bkrepo repository not found")
)

type (
	// ProjectStore 蓝盾制品库项目存储
	ProjectStore = store.GenericStore[Project]
	// RepositoryStore 蓝盾制品库仓库存储
	RepositoryStore = store.GenericStore[Repository]
)

// ProjectMeta 蓝盾制品库项目的存储元数据，公共账号密码在库中对称加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
// - 唯一：workspaceID
var ProjectMeta = store.Meta[Project]{
	Collection: "bkrepo_projects",
	NotFound:   ErrProjectNotFound,
	Encode:     []func(*Project) error{encryptProjectPassword},
	Decode:     []func(*Project) error{decryptProjectPassword},
}

func encryptProjectPassword(p *Project) (err error) {
	p.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, p.Password)
	return errors.Wrap(err, "encrypt password")
}

func decryptProjectPassword(p *Project) (err error) {
	p.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESDecrypt, p.Password)
	return errors.Wrap(err, "decrypt password")
}

// RepositoryMeta 蓝盾制品库仓库的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + type
var RepositoryMeta = store.Meta[Repository]{
	Collection: "bkrepo_repositories",
	NotFound:   ErrRepositoryNotFound,
}

// ByWorkspace 按工作空间 ID 匹配，适用于 Project 与 Repository
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByWorkspaceAndType 按工作空间 ID 与仓库类型匹配 Repository
func ByWorkspaceAndType(workspaceID string, repoType RepoType) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("type", repoType))
}
