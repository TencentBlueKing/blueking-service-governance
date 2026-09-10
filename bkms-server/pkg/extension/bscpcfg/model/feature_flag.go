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
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const featureFlagCollectionName = "bscpcfg_feature_flags"

// ErrFeatureFlagNotFound FeatureFlag 不存在
var ErrFeatureFlagNotFound = errors.New("bscpcfg feature flag not found")

var _ FeatureFlagStore = new(FeatureFlagStoreMongo)

// FeatureFlag 记录某个 app 是否已激活 bscpcfg 能力
type FeatureFlag struct {
	// AppID 应用 ID（唯一键）
	AppID string `bson:"appID" validate:"required"`
	// Enabled 是否启用新版 BSCP 配置管理
	Enabled bool `bson:"enabled"`
	// Operator 操作人
	Operator string `bson:"operator"`
	// CreatedAt 创建时间
	CreatedAt time.Time `bson:"createdAt"`
	// UpdatedAt 更新时间
	UpdatedAt time.Time `bson:"updatedAt"`
}

// FeatureFlagStore FeatureFlag 存储接口
type FeatureFlagStore interface {
	// Get 查询单个应用的 FeatureFlag
	Get(ctx context.Context, appID string) (*FeatureFlag, error)
	// List 查询所有已启用的应用
	List(ctx context.Context) ([]*FeatureFlag, error)
	// Upsert 创建或更新 FeatureFlag（幂等）
	Upsert(ctx context.Context, flag *FeatureFlag) error
}

// FeatureFlagStoreMongo FeatureFlag 的 MongoDB 存储实现
type FeatureFlagStoreMongo struct {
	collection *mongo.Collection
}

// NewFeatureFlagStoreMongo 创建 FeatureFlag 存储
func NewFeatureFlagStoreMongo(client *mongo.Client, dbName string) FeatureFlagStore {
	coll := client.Database(dbName).Collection(featureFlagCollectionName)
	return &FeatureFlagStoreMongo{collection: coll}
}

// Get 查询单个应用的 FeatureFlag
func (s *FeatureFlagStoreMongo) Get(ctx context.Context, appID string) (*FeatureFlag, error) {
	flag := new(FeatureFlag)
	if err := s.collection.FindOne(ctx, bson.M{"appID": appID}).Decode(flag); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrFeatureFlagNotFound
		}
		return nil, errors.Wrap(err, "find feature flag")
	}
	return flag, nil
}

// List 查询所有已启用的应用
func (s *FeatureFlagStoreMongo) List(ctx context.Context) ([]*FeatureFlag, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, errors.Wrap(err, "find feature flags")
	}
	defer cursor.Close(ctx)

	var flags []*FeatureFlag
	if err = cursor.All(ctx, &flags); err != nil {
		return nil, errors.Wrap(err, "decode feature flags")
	}
	return flags, nil
}

// Upsert 创建或更新 FeatureFlag（幂等，按 appID 唯一键）
func (s *FeatureFlagStoreMongo) Upsert(ctx context.Context, flag *FeatureFlag) error {
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(flag); err != nil {
		return errors.Wrap(err, "feature flag validation failed")
	}
	now := time.Now()
	if flag.CreatedAt.IsZero() {
		flag.CreatedAt = now
	}
	flag.UpdatedAt = now

	_, err := s.collection.UpdateOne(
		ctx,
		bson.M{"appID": flag.AppID},
		bson.M{
			"$set": bson.M{
				"enabled":   flag.Enabled,
				"operator":  flag.Operator,
				"updatedAt": flag.UpdatedAt,
			},
			"$setOnInsert": bson.M{
				"createdAt": flag.CreatedAt,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return errors.Wrap(err, "upsert feature flag")
	}
	return nil
}
