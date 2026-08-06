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
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
)

// projectCollName 蓝盾制品库项目表名
const projectCollName = "bkrepo_projects"

// repositoryCollName 蓝盾制品库仓库表名
const repositoryCollName = "bkrepo_repositories"

var (
	// ErrProjectNotFound 蓝盾制品库项目未找到
	ErrProjectNotFound = errors.New("bkrepo project not found")
	// ErrRepositoryNotFound 蓝盾制品库仓库未找到
	ErrRepositoryNotFound = errors.New("bkrepo repository not found")
)

// ProjectStore 蓝盾制品库项目存储接口
type ProjectStore interface {
	// Create 创建蓝盾制品库项目
	Create(ctx context.Context, project *Project) error

	// Get 使用 ID 获取 Project
	Get(ctx context.Context, id string) (*Project, error)

	// GetByWorkspace 获取工作空间关联的 Project
	GetByWorkspace(ctx context.Context, workspaceID string) (*Project, error)
}

var _ ProjectStore = &ProjectStoreMongo{}

// ProjectStoreMongo 是 ProjectStore 的具体实现（基于 MongoDB）
type ProjectStoreMongo struct {
	collection *mongo.Collection
}

// NewProjectStoreMongo 创建 ProjectStoreMongo 实例
func NewProjectStoreMongo(client *mongo.Client, dbName string) (*ProjectStoreMongo, error) {
	coll := client.Database(dbName).Collection(projectCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：id
	// - 唯一：workspaceID
	return &ProjectStoreMongo{collection: coll}, nil
}

// Create 创建蓝盾制品库项目
func (s *ProjectStoreMongo) Create(ctx context.Context, project *Project) error {
	project.CreatedAt = time.Now()
	// 入库前对敏感字段进行加密
	if err := s.handleSensitiveFields(project, crypto.AESEncrypt); err != nil {
		return err
	}
	if _, err := s.collection.InsertOne(ctx, project); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("project with the same workspaceID already exists")
		}
		return err
	}
	return nil
}

// Get 使用 ID 获取 Project
func (s *ProjectStoreMongo) Get(ctx context.Context, id string) (*Project, error) {
	return s.findOne(ctx, bson.M{"id": id})
}

// GetByWorkspace 获取工作空间关联的 Project
func (s *ProjectStoreMongo) GetByWorkspace(ctx context.Context, workspaceID string) (*Project, error) {
	return s.findOne(ctx, bson.M{"workspaceID": workspaceID})
}

// findOne 通过指定的过滤器获取单个 Project
func (s *ProjectStoreMongo) findOne(ctx context.Context, filter bson.M) (*Project, error) {
	var proj Project
	err := s.collection.FindOne(ctx, filter).Decode(&proj)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	// 出库前对敏感字段进行解密
	if err = s.handleSensitiveFields(&proj, crypto.AESDecrypt); err != nil {
		return nil, err
	}
	return &proj, err
}

// handleSensitiveFields 用于加密或解密 Project 中的敏感字段
// 其中 handleFunc 是一个用于加密或解密字段的函数
func (s *ProjectStoreMongo) handleSensitiveFields(
	project *Project, handleFunc func(key, data string) (string, error),
) error {
	// 如果有公共账户密码，则需要进行加/解密
	if project.Password != "" {
		password, err := handleFunc(config.G.Encrypt.Secret, project.Password)
		if err != nil {
			return err
		}
		project.Password = password
	}
	return nil
}

// RepositoryStore 蓝盾制品库仓库存储接口
type RepositoryStore interface {
	// Create 创建蓝盾制品库仓库
	Create(ctx context.Context, repository *Repository) error

	// GetByWorkspaceAndType 使用 WorkspaceID + Type 获取 Repository
	GetByWorkspaceAndType(ctx context.Context, workspaceID string, repoType RepoType) (*Repository, error)
}

var _ RepositoryStore = &RepositoryStoreMongo{}

// RepositoryStoreMongo 是 RepositoryStore 的具体实现（基于 MongoDB）
type RepositoryStoreMongo struct {
	collection *mongo.Collection
}

// NewRepositoryStoreMongo 创建 RepositoryStoreMongo 实例
func NewRepositoryStoreMongo(client *mongo.Client, dbName string) (*RepositoryStoreMongo, error) {
	coll := client.Database(dbName).Collection(repositoryCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID + type
	return &RepositoryStoreMongo{collection: coll}, nil
}

// Create 创建蓝盾制品库仓库
func (s *RepositoryStoreMongo) Create(ctx context.Context, repository *Repository) error {
	repository.CreatedAt = time.Now()
	if _, err := s.collection.InsertOne(ctx, repository); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("repository with the same workspaceID and type already exists")
		}
		return err
	}
	return nil
}

// GetByWorkspaceAndType 使用 WorkspaceID + Type 获取 Repository
func (s *RepositoryStoreMongo) GetByWorkspaceAndType(
	ctx context.Context,
	workspaceID string,
	repoType RepoType,
) (*Repository, error) {
	var repo Repository
	filter := bson.M{"workspaceID": workspaceID, "type": string(repoType)}
	err := s.collection.FindOne(ctx, filter).Decode(&repo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	return &repo, nil
}
