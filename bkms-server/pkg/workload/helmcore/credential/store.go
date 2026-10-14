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

// Package credential 提供 Helm 仓库凭证管理功能
package credential

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrHelmRepoCredentialNotFound Helm 仓库凭证未找到时返回固定错误
var ErrHelmRepoCredentialNotFound = errors.New("helm repo credential not found")

// Store Helm 仓库凭证存储
type Store = store.GenericStore[HelmRepoCredential]

// Meta Helm 仓库凭证的存储元数据，密码在库中对称加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID
var Meta = store.Meta[HelmRepoCredential]{
	Collection: "helm_repo_credentials",
	NotFound:   ErrHelmRepoCredentialNotFound,
	Encode:     []func(*HelmRepoCredential) error{encryptPassword},
	Decode:     []func(*HelmRepoCredential) error{decryptPassword},
}

// ByWorkspace 按工作空间 ID 匹配凭证
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

func encryptPassword(c *HelmRepoCredential) (err error) {
	c.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, c.Password)
	return errors.Wrap(err, "encrypt password")
}

func decryptPassword(c *HelmRepoCredential) (err error) {
	c.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESDecrypt, c.Password)
	return errors.Wrap(err, "decrypt password")
}
