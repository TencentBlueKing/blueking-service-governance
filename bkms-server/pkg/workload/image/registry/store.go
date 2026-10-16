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

package registry

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ImageRegistryStore 镜像仓库存储
type ImageRegistryStore = store.GenericStore[ImageRegistry]

// ImageRegistryMeta 镜像仓库的存储元数据，密码在库中对称加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + type
var ImageRegistryMeta = store.Meta[ImageRegistry]{
	Collection: "image_registries",
	NotFound:   ErrImageRegistryNotFound,
	Encode:     []func(*ImageRegistry) error{encryptPassword},
	Decode:     []func(*ImageRegistry) error{decryptPassword},
}

func encryptPassword(r *ImageRegistry) (err error) {
	r.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, r.Password)
	return errors.Wrap(err, "encrypt password")
}

func decryptPassword(r *ImageRegistry) (err error) {
	r.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESDecrypt, r.Password)
	return errors.Wrap(err, "decrypt password")
}

// ByWorkspace 按工作空间 ID 匹配镜像仓库
func ByWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ByWorkspaceAndType 按工作空间 ID 与镜像仓库类型匹配镜像仓库（至多一条）
func ByWorkspaceAndType(workspaceID string, registryType ImageRegistryType) store.Opt {
	return store.Combine(ByWorkspace(workspaceID), store.Where("type", registryType))
}
