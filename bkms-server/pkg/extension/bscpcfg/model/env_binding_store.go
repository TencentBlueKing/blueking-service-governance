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

// envBindingCollectionName MongoDB 集合名称，用于存储 EnvBinding。
const envBindingCollectionName = "bscpcfg_env_bindings"

// 错误定义
var (
	// ErrEnvBindingNotFound 配置管理环境绑定不存在
	ErrEnvBindingNotFound = errors.New("bscpcfg env binding not found")

	// ErrEnvBindingAlreadyExists 配置管理环境绑定已存在（同一 app+env 只能有一条记录）
	ErrEnvBindingAlreadyExists = errors.New("bscpcfg env binding already exists")
)

var _ EnvBindingStore = new(EnvBindingStoreMongo)

// EnvBindingStore EnvBinding存储接口
type EnvBindingStore interface {
	// Create 创建 EnvBinding（一个 app+env 一条记录）
	Create(ctx context.Context, binding *EnvBinding) error

	// Delete 删除指定 app+env 的绑定
	Delete(ctx context.Context, appID, envName string) error
	// DeleteByApp 删除应用下所有绑定（应用删除时级联）
	DeleteByApp(ctx context.Context, appID string) error

	// Update 更新绑定（可更新 services 数组）
	Update(ctx context.Context, appID, envName string, updateData *EnvBindingUpdate) error

	// Get 获取指定 app+env 的绑定
	Get(ctx context.Context, appID, envName string) (*EnvBinding, error)
	// ListByApp 获取应用下所有环境的绑定列表
	ListByApp(ctx context.Context, appID string) ([]*EnvBinding, error)
}

// EnvBindingStoreMongo EnvBinding的 MongoDB 存储实现
type EnvBindingStoreMongo struct {
	collection *mongo.Collection
}

// NewEnvBindingStoreMongo 创建 EnvBinding存储
func NewEnvBindingStoreMongo(client *mongo.Client, dbName string) (EnvBindingStore, error) {
	coll := client.Database(dbName).Collection(envBindingCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + envName
	return &EnvBindingStoreMongo{collection: coll}, nil
}

// Create 创建 EnvBinding
func (s *EnvBindingStoreMongo) Create(ctx context.Context, binding *EnvBinding) error {
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(binding); err != nil {
		return errors.Wrap(err, "env binding validation failed")
	}
	if binding.Services == nil {
		binding.Services = make([]ServiceRef, 0)
	}
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = time.Now()
	}
	if binding.UpdatedAt.IsZero() {
		binding.UpdatedAt = time.Now()
	}

	if _, err := s.collection.InsertOne(ctx, binding); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrEnvBindingAlreadyExists
		}
		return errors.Wrap(err, "insert env binding")
	}
	return nil
}

// Get 获取指定 app+env 的绑定
func (s *EnvBindingStoreMongo) Get(ctx context.Context, appID, envName string) (*EnvBinding, error) {
	binding := new(EnvBinding)

	if err := s.collection.FindOne(ctx, bson.M{"appID": appID, "envName": envName}).Decode(binding); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrEnvBindingNotFound
		}
		return nil, errors.Wrap(err, "find env binding")
	}

	return binding, nil
}

// ListByApp 获取应用下所有环境的绑定列表
func (s *EnvBindingStoreMongo) ListByApp(ctx context.Context, appID string) ([]*EnvBinding, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"appID": appID})
	if err != nil {
		return nil, errors.Wrap(err, "find env bindings")
	}
	defer cursor.Close(ctx)

	bindings := make([]*EnvBinding, 0)
	if err = cursor.All(ctx, &bindings); err != nil {
		return nil, errors.Wrap(err, "decode env bindings")
	}

	return bindings, nil
}

// Update 更新绑定（支持 services 全量替换）
func (s *EnvBindingStoreMongo) Update(
	ctx context.Context,
	appID, envName string,
	updateData *EnvBindingUpdate,
) error {
	if updateData == nil {
		return nil
	}
	needUpdate := false
	updateSet := bson.M{}

	if updateData.Services != nil {
		updateSet["bscpApps"] = *updateData.Services
		needUpdate = true
	}
	if !needUpdate {
		return nil
	}
	updateSet["updatedAt"] = time.Now()

	filter := bson.M{"appID": appID, "envName": envName}
	result, err := s.collection.UpdateOne(ctx, filter, bson.M{"$set": updateSet})
	if err != nil {
		return errors.Wrap(err, "update env binding")
	}
	if result.MatchedCount == 0 {
		return ErrEnvBindingNotFound
	}

	return nil
}

// Delete 删除指定 app+env 的绑定
func (s *EnvBindingStoreMongo) Delete(ctx context.Context, appID, envName string) error {
	filter := bson.M{"appID": appID, "envName": envName}
	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "delete env binding")
	}
	if result.DeletedCount == 0 {
		return ErrEnvBindingNotFound
	}
	return nil
}

// DeleteByApp 删除应用下所有绑定
func (s *EnvBindingStoreMongo) DeleteByApp(ctx context.Context, appID string) error {
	if _, err := s.collection.DeleteMany(ctx, bson.M{"appID": appID}); err != nil {
		return errors.Wrap(err, "delete env bindings by app")
	}
	return nil
}
