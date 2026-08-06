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

package bkci

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	// projectCollName 蓝盾项目表名
	projectCollName = "bkci_projects"
	// pipelineTemplateCollName 流水线模板表名
	pipelineTemplateCollName = "bkci_pipeline_templates"
	// pipelineCollName 流水线表名
	pipelineCollName = "bkci_pipelines"
	// repositoryCollName 制品库仓库表名
	repositoryCollName = "bkci_repositories"
)

var (
	// ErrProjectNotFound 蓝盾项目未找到
	ErrProjectNotFound = errors.New("bkci project not found")
	// ErrPipelineTemplateNotFound 流水线模板未找到
	ErrPipelineTemplateNotFound = errors.New("pipeline template not found")
	// ErrPipelineNotFound 流水线未找到
	ErrPipelineNotFound = errors.New("pipeline not found")
	// ErrRepositoryNotFound 代码仓库未找到
	ErrRepositoryNotFound = errors.New("repository not found")
)

// ProjectStore 蓝盾项目存储接口
type ProjectStore interface {
	// Create 创建蓝盾制品库仓库
	Create(ctx context.Context, project *Project) error

	// GetByWorkspace 使用 WorkspaceID 获取 Project
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
	// - 唯一：code
	// - 唯一：workspaceID
	return &ProjectStoreMongo{collection: coll}, nil
}

// Create 创建蓝盾制品库仓库
func (s *ProjectStoreMongo) Create(ctx context.Context, project *Project) error {
	project.CreatedAt = time.Now()
	if _, err := s.collection.InsertOne(ctx, project); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("project with the same id or code or workspaceID already exists")
		}
		return err
	}
	return nil
}

// GetByWorkspace 使用 WorkspaceID 获取 Project
func (s *ProjectStoreMongo) GetByWorkspace(ctx context.Context, workspaceID string) (*Project, error) {
	return s.findOne(ctx, bson.M{"workspaceID": workspaceID})
}

// GetByCode 使用 Code 获取 Project
func (s *ProjectStoreMongo) GetByCode(ctx context.Context, code string) (*Project, error) {
	return s.findOne(ctx, bson.M{"code": code})
}

// findOne 使用自定义 filter 获取 Project
func (s *ProjectStoreMongo) findOne(ctx context.Context, filter bson.M) (*Project, error) {
	var repo Project
	err := s.collection.FindOne(ctx, filter).Decode(&repo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return &repo, nil
}

// PipelineTemplateStore 是 PipelineTemplate 的存储接口
type PipelineTemplateStore interface {
	// Upsert 创建或更新 PipelineTemplate
	Upsert(ctx context.Context, tmpl *PipelineTemplate) error
	// GetByType 根据 pipelineType 获取 PipelineTemplate
	GetByType(ctx context.Context, pipelineType string) (*PipelineTemplate, error)
}

var _ PipelineTemplateStore = &DBPipelineTemplateStore{}

// DBPipelineTemplateStore 是 PipelineTemplate 的具体实现（基于 MongoDB）
type DBPipelineTemplateStore struct {
	collection *mongo.Collection
}

// NewDBPipelineTemplateStore 创建 DBPipelineTemplateStore 实例
func NewDBPipelineTemplateStore(client *mongo.Client, dbName string) (*DBPipelineTemplateStore, error) {
	coll := client.Database(dbName).Collection(pipelineTemplateCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：id
	// - 唯一：type
	return &DBPipelineTemplateStore{collection: coll}, nil
}

// Upsert 创建或更新 PipelineTemplate。
// 调用方需在传入前保证 tmpl.Version 合法（load 时已校验）。
func (s *DBPipelineTemplateStore) Upsert(ctx context.Context, tmpl *PipelineTemplate) error {
	filter := bson.M{"type": tmpl.Type}
	opts := options.Replace().SetUpsert(true)
	_, err := s.collection.ReplaceOne(ctx, filter, tmpl, opts)
	return err
}

// GetByType 根据 Type 获取 PipelineTemplate
func (s *DBPipelineTemplateStore) GetByType(ctx context.Context, pipelineType string) (*PipelineTemplate, error) {
	var tmpl PipelineTemplate
	filter := bson.M{"type": pipelineType}
	if err := s.collection.FindOne(ctx, filter).Decode(&tmpl); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPipelineTemplateNotFound
		}
		return nil, err
	}
	return &tmpl, nil
}

// PipelineStore 定义流水线的持久化操作
type PipelineStore interface {
	// Create 创建 Pipeline
	Create(ctx context.Context, pipeline *Pipeline) error
	// GetByWorkspaceAndType 根据 WorkspaceID 和 pipelineType 获取 Pipeline
	GetByWorkspaceAndType(ctx context.Context, workspaceID, pipelineType string) (*Pipeline, error)
	// UpdateBuiltinTemplateVersion 更新内置流水线已应用的模板版本、名称与描述。
	// 注意：仅适用于内置流水线（每个 Workspace 每种内置类型只有一条 Pipeline 记录，且
	// 名称/描述由模板决定）。禁止用于用户自定义流水线，否则会用本地陈旧的 name/description
	// 覆盖蓝盾侧用户维护的最新值。
	UpdateBuiltinTemplateVersion(ctx context.Context, pipeline *Pipeline) error
}

var _ PipelineStore = &PipelineStoreMongo{}

// PipelineStoreMongo 是 PipelineStore 的具体实现（基于 MongoDB）
type PipelineStoreMongo struct {
	collection *mongo.Collection
}

// NewPipelineStoreMongo 创建 PipelineStoreMongo 实例
func NewPipelineStoreMongo(client *mongo.Client, dbName string) (*PipelineStoreMongo, error) {
	coll := client.Database(dbName).Collection(pipelineCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：id
	// - 唯一：workspaceID + type
	return &PipelineStoreMongo{collection: coll}, nil
}

// Create 创建 Pipeline
func (s *PipelineStoreMongo) Create(ctx context.Context, pipeline *Pipeline) error {
	pipeline.CreatedAt = time.Now()
	if _, err := s.collection.InsertOne(ctx, pipeline); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("pipeline with the same id or (workspaceID, type) already exists")
		}
		return err
	}
	return nil
}

// GetByWorkspaceAndType 根据 WorkspaceID 和 Type 获取 Pipeline
func (s *PipelineStoreMongo) GetByWorkspaceAndType(
	ctx context.Context, workspaceID, pipelineType string,
) (*Pipeline, error) {
	var pipeline Pipeline
	filter := bson.M{"workspaceID": workspaceID, "type": pipelineType}
	if err := s.collection.FindOne(ctx, filter).Decode(&pipeline); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPipelineNotFound
		}
		return nil, err
	}
	return &pipeline, nil
}

// UpdateBuiltinTemplateVersion 更新内置流水线已应用的模板版本、名称与描述。
// 注意：仅适用于内置流水线，禁止用于用户自定义流水线（详见接口定义）。
func (s *PipelineStoreMongo) UpdateBuiltinTemplateVersion(ctx context.Context, pipeline *Pipeline) error {
	filter := bson.M{"workspaceID": pipeline.WorkspaceID, "type": pipeline.Type}
	update := bson.M{"$set": bson.M{
		"name":            pipeline.Name,
		"description":     pipeline.Description,
		"templateVersion": pipeline.TemplateVersion,
	}}
	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return ErrPipelineNotFound
	}
	return nil
}

// RepositoryStore 代码仓库存储接口
type RepositoryStore interface {
	// Create 创建 Repository
	Create(ctx context.Context, repository *Repository) error
	// GetByWorkspaceAndAlias 根据 WorkspaceID 和 Alias 获取 Repository
	GetByWorkspaceAndAlias(ctx context.Context, workspaceID, alias string) (*Repository, error)
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
	// - 唯一：id + projectCode
	// - 唯一：workspaceID + alias
	return &RepositoryStoreMongo{collection: coll}, nil
}

// Create 创建 Repository
func (s *RepositoryStoreMongo) Create(ctx context.Context, repository *Repository) error {
	repository.CreatedAt = time.Now()
	if _, err := s.collection.InsertOne(ctx, repository); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("repository with the same id or (workspaceID, alias) already exists")
		}
		return err
	}
	return nil
}

// GetByWorkspaceAndAlias 根据 WorkspaceID 和 Alias 获取 Repository
func (s *RepositoryStoreMongo) GetByWorkspaceAndAlias(
	ctx context.Context, workspaceID, alias string,
) (*Repository, error) {
	var repository Repository
	filter := bson.M{"workspaceID": workspaceID, "alias": alias}
	if err := s.collection.FindOne(ctx, filter).Decode(&repository); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}
	return &repository, nil
}
