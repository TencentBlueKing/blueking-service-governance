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

package snapshot

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
)

// 详情同步批次控制
const (
	// DetailSyncBatchSize 每批同步标签数量
	DetailSyncBatchSize = 10
	// DetailSyncMaxConcurrency 最大并发同步数
	DetailSyncMaxConcurrency = 3
)

// 标签常量
const (
	// TagLatest latest 标签名（唯一视为可变的标签）
	TagLatest = "latest"
)

// RefreshStatus 刷新状态
type RefreshStatus string

const (
	// RefreshStatusIdle 空闲
	RefreshStatusIdle RefreshStatus = "idle"
	// RefreshStatusRefreshing 标签刷新中
	RefreshStatusRefreshing RefreshStatus = "refreshing"
	// RefreshStatusDetailSyncing 详情同步中
	RefreshStatusDetailSyncing RefreshStatus = "detail_syncing"
)

// RefreshResult.Status 取值
const (
	// RefreshResultSuccess 本次刷新已完成
	RefreshResultSuccess = "success"
	// RefreshResultRefreshing 已有刷新在进行中，本次未重复发起远程调用，属正常响应而非错误
	RefreshResultRefreshing = "refreshing"
	// RefreshResultFailed 刷新失败，快照仍保留上一次成功的内容
	RefreshResultFailed = "failed"
)

// RefreshResult 刷新结果
type RefreshResult struct {
	// Status 刷新状态，取值见 RefreshResultSuccess / RefreshResultRefreshing / RefreshResultFailed
	Status string
	// Message 提示信息
	Message string
	// AddedTagCnt 本次新增标签数量
	AddedTagCnt int64
	// RemovedTagCnt 本次删除标签数量
	RemovedTagCnt int64
}

// ImageDetailSyncArgs 镜像快照详情同步任务参数
// 凭据字段在构造时即完成加密，对外仅通过 Username() / Password() 方法按需解密
type ImageDetailSyncArgs struct {
	RepoKey           string `json:"repoKey"`
	RepoName          string `json:"repoName"`
	EncryptedUsername string `json:"encryptedUsername"`
	EncryptedPassword string `json:"encryptedPassword"`
}

// NewImageDetailSyncArgs 创建 ImageDetailSyncArgs，构造时即对凭据进行 AES-GCM 加密
func NewImageDetailSyncArgs(repoKey, repoName, username, password string) (*ImageDetailSyncArgs, error) {
	encUser, err := encryptCredential(username)
	if err != nil {
		return nil, errors.Wrap(err, "encrypt username")
	}
	encPass, err := encryptCredential(password)
	if err != nil {
		return nil, errors.Wrap(err, "encrypt password")
	}
	return &ImageDetailSyncArgs{
		RepoKey:           repoKey,
		RepoName:          repoName,
		EncryptedUsername: encUser,
		EncryptedPassword: encPass,
	}, nil
}

// String 参数内容字符串化（不暴露凭据）
func (args ImageDetailSyncArgs) String() string {
	return fmt.Sprintf("<repoKey: %s, repoName: %s>", args.RepoKey, args.RepoName)
}

// Username 解密并返回用户名明文
func (args ImageDetailSyncArgs) Username() (string, error) {
	return decryptCredential(args.EncryptedUsername)
}

// Password 解密并返回密码明文
func (args ImageDetailSyncArgs) Password() (string, error) {
	return decryptCredential(args.EncryptedPassword)
}

// encryptCredential 对单个凭据进行 AES-GCM 加密，空值直接返回
func encryptCredential(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	return crypto.AESEncrypt(config.G.Encrypt.Secret, plaintext)
}

// decryptCredential 对单个凭据进行 AES-GCM 解密，空值直接返回
func decryptCredential(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	return crypto.AESDecrypt(config.G.Encrypt.Secret, ciphertext)
}
