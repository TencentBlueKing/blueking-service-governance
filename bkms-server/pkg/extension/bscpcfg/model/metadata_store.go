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

// Package model 定义了应用配置管理相关的纯数据模型。
package model

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// metadataCollectionName MongoDB 集合名称，用于存储 Metadata。
const metadataCollectionName = "bscpcfg_metadata"

// 错误定义
var (
	// ErrMetadataNotFound 配置管理元信息不存在
	ErrMetadataNotFound = errors.New("bscpcfg metadata not found")

	// ErrMetadataAlreadyExists 配置管理元信息已存在（同一 app 只能有一条记录）
	ErrMetadataAlreadyExists = errors.New("bscpcfg metadata already exists")
)

var _ MetadataStore = new(MetadataStoreMongo)

// MetadataStore Metadata 存储接口
type MetadataStore interface {
	// Create 创建 Metadata（一个 app 一条记录）
	Create(ctx context.Context, meta *Metadata) error

	// Get 获取指定 app 的 Metadata
	Get(ctx context.Context, appID string) (*Metadata, error)

	// Update 更新 Metadata（可更新 mountPath、token、credentialID、credential）
	Update(ctx context.Context, appID string, updateData *MetadataUpdate) error

	// Delete 删除指定 app 的 Metadata
	Delete(ctx context.Context, appID string) error
}

// MetadataStoreMongo Metadata 的 MongoDB 存储实现
type MetadataStoreMongo struct {
	collection *mongo.Collection
}

// NewMetadataStoreMongo 创建 Metadata 存储
func NewMetadataStoreMongo(client *mongo.Client, dbName string) (MetadataStore, error) {
	coll := client.Database(dbName).Collection(metadataCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID
	return &MetadataStoreMongo{collection: coll}, nil
}

// Create 创建 Metadata
func (s *MetadataStoreMongo) Create(ctx context.Context, meta *Metadata) error {
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(meta); err != nil {
		return errors.Wrap(err, "metadata validation failed")
	}
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now()
	}
	if meta.UpdatedAt.IsZero() {
		meta.UpdatedAt = time.Now()
	}

	if _, err := s.collection.InsertOne(ctx, meta); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrMetadataAlreadyExists
		}
		return errors.Wrap(err, "insert metadata")
	}
	return nil
}

// Get 获取指定 app 的 Metadata
func (s *MetadataStoreMongo) Get(ctx context.Context, appID string) (*Metadata, error) {
	meta := new(Metadata)

	if err := s.collection.FindOne(ctx, bson.M{"appID": appID}).Decode(meta); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrMetadataNotFound
		}
		return nil, errors.Wrap(err, "find metadata")
	}

	return meta, nil
}

// Update 更新 Metadata（支持 mountPath、token、credentialID、credential）
func (s *MetadataStoreMongo) Update(
	ctx context.Context,
	appID string,
	updateData *MetadataUpdate,
) error {
	if updateData == nil {
		return nil
	}
	needUpdate := false
	updateSet := bson.M{}

	if updateData.MountPath != nil {
		updateSet["mountPath"] = *updateData.MountPath
		needUpdate = true
	}
	if updateData.CredentialID != nil {
		updateSet["credentialID"] = *updateData.CredentialID
		needUpdate = true
	}
	if updateData.CredentialName != nil {
		updateSet["credential"] = *updateData.CredentialName
		needUpdate = true
	}
	if updateData.Token != nil {
		updateSet["token"] = *updateData.Token
		needUpdate = true
	}
	if updateData.WorkloadName != nil {
		updateSet["workloadName"] = *updateData.WorkloadName
		needUpdate = true
	}
	if updateData.WorkloadKind != nil {
		updateSet["workloadKind"] = *updateData.WorkloadKind
		needUpdate = true
	}
	if !needUpdate {
		return nil
	}
	updateSet["updatedAt"] = time.Now()

	filter := bson.M{"appID": appID}
	result, err := s.collection.UpdateOne(ctx, filter, bson.M{"$set": updateSet})
	if err != nil {
		return errors.Wrap(err, "update metadata")
	}
	if result.MatchedCount == 0 {
		return ErrMetadataNotFound
	}

	return nil
}

// Delete 删除指定 app 的 Metadata
func (s *MetadataStoreMongo) Delete(ctx context.Context, appID string) error {
	filter := bson.M{"appID": appID}
	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "delete metadata")
	}
	if result.DeletedCount == 0 {
		return ErrMetadataNotFound
	}
	return nil
}
