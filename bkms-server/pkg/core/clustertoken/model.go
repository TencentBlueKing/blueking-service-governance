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

// Package clustertoken 定义空间登记的集群 Token 与集群接入记录。
//
// 开关 bcs.allowWorkspaceClusterTokens 打开时，集群 API 使用空间为集群分配的 Token，
// 未接入空间的集群不可用；关闭时沿用平台 Token。设计说明见 design_notes/cluster_token.md。
package clustertoken

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	// maxTokenLength Token 的最大长度
	maxTokenLength = 1024
	// maskKeep 脱敏展示时首尾各保留的字符数
	maskKeep = 4
)

// ClusterToken 空间登记的集群 Token
type ClusterToken struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// WorkspaceID 所属工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// Token 明文 Token，在库中对称加密存储
	Token string `bson:"token" audit:"mask"`
	// Masked 脱敏后的 Token，用于展示
	Masked string `bson:"masked"`
	// Fingerprint Token 明文的 SHA-256，用于空间内去重
	Fingerprint string `bson:"fingerprint"`
	// IsDefault 是否为空间的默认 Token，空间内至多一个
	IsDefault bool `bson:"isDefault"`

	Creator   string    `bson:"creator"`
	CreatedAt time.Time `bson:"createdAt"`
	Updater   string    `bson:"updater"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// WorkspaceCluster 集群在空间内的接入记录；同一个集群可以接入多个空间，各自使用自己的 Token
type WorkspaceCluster struct {
	ID bson.ObjectID `bson:"_id,omitempty"`
	// TenantID 所属租户，由 store 层自动写入
	TenantID string `json:"-" bson:"tenantID"`
	// WorkspaceID 所属工作空间 ID
	WorkspaceID string `bson:"workspaceID"`
	// ClusterID BCS 集群 ID，空间内唯一
	ClusterID string `bson:"clusterID"`
	// ClusterName 接入时的集群名称快照
	ClusterName string `bson:"clusterName"`
	// IsShared 是否为共享集群，接入时的快照
	IsShared bool `bson:"isShared"`
	// TokenID 使用的集群 Token ID
	TokenID string `bson:"tokenID"`

	Creator   string    `bson:"creator"`
	CreatedAt time.Time `bson:"createdAt"`
	Updater   string    `bson:"updater"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// CheckStatus 集群权限检测状态
type CheckStatus string

// CheckStatusUnchecked 未检测；Todo 暂未提供权限检测方式确定前，所有集群都是该状态
const CheckStatusUnchecked CheckStatus = "unchecked"

// normalizeToken 去掉首尾空白并校验 Token：不能为空、不超过 maxTokenLength，
// 只能包含可见 ASCII 字符（Token 会放进 Authorization 请求头）
func normalizeToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.Wrap(ErrTokenInvalid, "token is empty")
	}
	if len(token) > maxTokenLength {
		return "", errors.Wrapf(ErrTokenInvalid, "token is longer than %d characters", maxTokenLength)
	}
	for _, c := range []byte(token) {
		if c < '!' || c > '~' {
			return "", errors.Wrap(ErrTokenInvalid, "token must only contain visible ASCII characters")
		}
	}
	return token, nil
}

// fingerprintOf 返回 Token 明文的 SHA-256 十六进制摘要
func fingerprintOf(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// maskToken 脱敏 Token：足够长时保留首尾各 maskKeep 位，否则只保留末尾 maskKeep 位
func maskToken(token string) string {
	switch {
	case len(token) >= maskKeep*3:
		return lo.Substring(token, 0, maskKeep) + "****" + lo.Substring(token, -maskKeep, maskKeep)
	case len(token) > maskKeep:
		return "****" + lo.Substring(token, -maskKeep, maskKeep)
	default:
		return "****"
	}
}
