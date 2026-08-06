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
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
)

// helmRepoCredentialCollName Helm 仓库凭证表名
const helmRepoCredentialCollName = "helm_repo_credentials" // #nosec G101

// ErrHelmRepoCredentialNotFound Helm 仓库凭证未找到时返回固定错误
var ErrHelmRepoCredentialNotFound = errors.New("helm repo credential not found")

// HelmRepoCredentialStore Helm 仓库凭证存储接口
type HelmRepoCredentialStore interface {
	// GetByWorkspace 通过工作空间 ID 获取凭证
	GetByWorkspace(ctx context.Context, workspaceID string) (*HelmRepoCredential, error)
	// Create 创建新的凭证记录
	Create(ctx context.Context, cred *HelmRepoCredential) error
}

var _ HelmRepoCredentialStore = &HelmRepoCredentialStoreMongo{}

// HelmRepoCredentialStoreMongo HelmRepoCredentialStore 的 MongoDB 实现
type HelmRepoCredentialStoreMongo struct {
	collection *mongo.Collection
}

// NewHelmRepoCredentialStoreMongo 创建 HelmRepoCredentialStoreMongo 实例
func NewHelmRepoCredentialStoreMongo(
	client *mongo.Client,
	dbName string,
) (*HelmRepoCredentialStoreMongo, error) {
	coll := client.Database(dbName).Collection(helmRepoCredentialCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID
	return &HelmRepoCredentialStoreMongo{collection: coll}, nil
}

// GetByWorkspace 通过工作空间 ID 获取凭证
func (s *HelmRepoCredentialStoreMongo) GetByWorkspace(
	ctx context.Context, workspaceID string,
) (*HelmRepoCredential, error) {
	var cred HelmRepoCredential
	if err := s.collection.FindOne(ctx, bson.M{"workspaceID": workspaceID}).Decode(&cred); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Wrapf(ErrHelmRepoCredentialNotFound, "workspace %s", workspaceID)
		}
		return nil, err
	}
	if err := s.handleSensitiveFields(&cred, crypto.AESDecrypt); err != nil {
		return nil, errors.Wrap(err, "decrypt helm repo credential sensitive fields")
	}
	return &cred, nil
}

// Create 创建新的凭证记录
func (s *HelmRepoCredentialStoreMongo) Create(ctx context.Context, cred *HelmRepoCredential) error {
	credCopy := *cred
	if err := s.handleSensitiveFields(&credCopy, crypto.AESEncrypt); err != nil {
		return errors.Wrap(err, "encrypt helm repo credential sensitive fields")
	}
	if _, err := s.collection.InsertOne(ctx, &credCopy); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Errorf("helm repo credential for workspace %s already exists", cred.WorkspaceID)
		}
		return err
	}
	return nil
}

// handleSensitiveFields 对 HelmRepoCredential 的敏感字段进行加密或解密
func (s *HelmRepoCredentialStoreMongo) handleSensitiveFields(
	cred *HelmRepoCredential, handleFunc func(key, data string) (string, error),
) error {
	if cred == nil || cred.Password == "" {
		return nil
	}
	password, err := handleFunc(config.G.Encrypt.Secret, cred.Password)
	if err != nil {
		return err
	}
	cred.Password = password
	return nil
}
