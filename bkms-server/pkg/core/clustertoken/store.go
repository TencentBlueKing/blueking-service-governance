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

package clustertoken

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// TokenStore 集群 Token 存储
type TokenStore = store.GenericStore[ClusterToken]

// TokenMeta 集群 Token 的存储元数据，Token 在库中对称加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + fingerprint，约束空间内 Token 不重复，同时支撑按空间列出 Token
// - 部分唯一：workspaceID + isDefault（仅 isDefault=true），约束空间内至多一个默认 Token
//
// 两个唯一索引共用 Duplicate 错误，写入方按写入的字段区分冲突来源，见 Manager
var TokenMeta = store.Meta[ClusterToken]{
	Collection:   "cluster_tokens",
	TenantScoped: true,
	NotFound:     ErrTokenNotFound,
	Duplicate:    ErrTokenExists,
	Encode:       []func(*ClusterToken) error{encryptToken},
	Decode:       []func(*ClusterToken) error{decryptToken},
}

// WorkspaceClusterStore 集群接入记录存储
type WorkspaceClusterStore = store.GenericStore[WorkspaceCluster]

// WorkspaceClusterMeta 集群接入记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + clusterID，约束空间内一个集群至多接入一次；不做跨空间唯一
// - workspaceID + tokenID：列出、统计 Token 被哪些集群使用
var WorkspaceClusterMeta = store.Meta[WorkspaceCluster]{
	Collection:   "workspace_clusters",
	TenantScoped: true,
	NotFound:     ErrWorkspaceClusterNotFound,
	Duplicate:    ErrWorkspaceClusterExists,
}

// InWorkspace 匹配空间内的文档，Token 与接入记录集合通用
func InWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// TokenByID 匹配空间内指定 ID 的 Token，id 不是合法的 ObjectID 时返回 ErrTokenNotFound
func TokenByID(workspaceID, id string) (store.Opt, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(ErrTokenNotFound, "invalid cluster token id %q", id)
	}
	return store.Combine(InWorkspace(workspaceID), store.ByID(objID)), nil
}

// ByFingerprint 匹配空间内指纹相同的 Token
func ByFingerprint(workspaceID, fingerprint string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("fingerprint", fingerprint))
}

// DefaultToken 匹配空间的默认 Token
func DefaultToken(workspaceID string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("isDefault", true))
}

// ByCluster 匹配空间内指定集群的接入记录
func ByCluster(workspaceID, clusterID string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("clusterID", clusterID))
}

// ByToken 匹配空间内使用指定 Token 的接入记录
func ByToken(workspaceID, tokenID string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("tokenID", tokenID))
}

func encryptToken(t *ClusterToken) (err error) {
	t.Token, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESEncrypt, t.Token)
	return errors.Wrap(err, "encrypt cluster token")
}

func decryptToken(t *ClusterToken) (err error) {
	t.Token, err = crypto.OnDemand(config.G.Encrypt.Secret, crypto.AESDecrypt, t.Token)
	return errors.Wrap(err, "decrypt cluster token")
}
