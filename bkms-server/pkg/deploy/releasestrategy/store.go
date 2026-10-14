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

package releasestrategy

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CollectionName 发布策略表名
const CollectionName = "release_strategies"

// Store 发布策略存储接口
type Store interface {
	// List 列出空间内全部发布策略（按创建时间升序，不分页）
	List(ctx context.Context, workspaceID string) ([]ReleaseStrategy, error)
	// Get 获取空间内指定 ID 的发布策略，不存在（含 ID 不合法）时返回 ErrStrategyNotFound
	Get(ctx context.Context, workspaceID, id string) (*ReleaseStrategy, error)
	// Create 创建发布策略，同一空间内名称重复时返回 ErrStrategyNameConflict
	Create(ctx context.Context, strategy *ReleaseStrategy) error
	// Update 整体替换发布策略的可编辑内容，名称冲突返回 ErrStrategyNameConflict，不存在返回 ErrStrategyNotFound
	Update(ctx context.Context, strategy *ReleaseStrategy) error
	// Delete 删除并返回空间内指定 ID 的发布策略，不存在时返回 ErrStrategyNotFound
	Delete(ctx context.Context, workspaceID, id string) (*ReleaseStrategy, error)
}

var _ Store = (*StoreMongo)(nil)

// StoreMongo Store 实现（基于 MongoDB）
type StoreMongo struct {
	collection *mongo.Collection
}

// NewStoreMongo 新建 StoreMongo 实例
func NewStoreMongo(client *mongo.Client, dbName string) (*StoreMongo, error) {
	coll := client.Database(dbName).Collection(CollectionName)
	// 索引
	// - 唯一：workspaceID + name，约束同一空间内策略名称唯一
	return &StoreMongo{collection: coll}, nil
}

// List 列出空间内全部发布策略
func (s *StoreMongo) List(ctx context.Context, workspaceID string) ([]ReleaseStrategy, error) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})
	cursor, err := s.collection.Find(ctx, bson.M{"workspaceID": workspaceID}, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "find release strategies in workspace %s", workspaceID)
	}
	defer cursor.Close(ctx)

	strategies := make([]ReleaseStrategy, 0)
	if err = cursor.All(ctx, &strategies); err != nil {
		return nil, errors.Wrapf(err, "decode release strategies in workspace %s", workspaceID)
	}
	return strategies, nil
}

// Get 获取空间内指定 ID 的发布策略
func (s *StoreMongo) Get(ctx context.Context, workspaceID, id string) (*ReleaseStrategy, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(ErrStrategyNotFound, "invalid release strategy id %q", id)
	}

	strategy := new(ReleaseStrategy)
	err = s.collection.FindOne(ctx, bson.M{"_id": objID, "workspaceID": workspaceID}).Decode(strategy)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Wrapf(ErrStrategyNotFound, "release strategy %s in workspace %s", id, workspaceID)
		}
		return nil, errors.Wrapf(err, "find release strategy %s in workspace %s", id, workspaceID)
	}
	return strategy, nil
}

// Create 创建发布策略
func (s *StoreMongo) Create(ctx context.Context, strategy *ReleaseStrategy) error {
	strategy.ID = bson.NewObjectID()
	now := time.Now()
	strategy.CreatedAt, strategy.UpdatedAt = now, now
	if _, err := s.collection.InsertOne(ctx, strategy); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Wrapf(
				ErrStrategyNameConflict, "name %q in workspace %s", strategy.Name, strategy.WorkspaceID,
			)
		}
		return errors.Wrapf(err, "insert release strategy %q", strategy.Name)
	}
	return nil
}

// Update 整体替换发布策略的可编辑内容（名称、描述、批次）及更新人
func (s *StoreMongo) Update(ctx context.Context, strategy *ReleaseStrategy) error {
	strategy.UpdatedAt = time.Now()
	ret, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": strategy.ID, "workspaceID": strategy.WorkspaceID},
		bson.M{"$set": bson.M{
			"name":        strategy.Name,
			"description": strategy.Description,
			"batches":     strategy.Batches,
			"updater":     strategy.Updater,
			"updatedAt":   strategy.UpdatedAt,
		}},
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Wrapf(
				ErrStrategyNameConflict, "name %q in workspace %s", strategy.Name, strategy.WorkspaceID,
			)
		}
		return errors.Wrapf(err, "update release strategy %s", strategy.ID.Hex())
	}
	if ret.MatchedCount == 0 {
		return errors.Wrapf(
			ErrStrategyNotFound, "release strategy %s in workspace %s", strategy.ID.Hex(), strategy.WorkspaceID,
		)
	}
	return nil
}

// Delete 删除并返回空间内指定 ID 的发布策略
func (s *StoreMongo) Delete(ctx context.Context, workspaceID, id string) (*ReleaseStrategy, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(ErrStrategyNotFound, "invalid release strategy id %q", id)
	}

	strategy := new(ReleaseStrategy)
	err = s.collection.FindOneAndDelete(ctx, bson.M{"_id": objID, "workspaceID": workspaceID}).Decode(strategy)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Wrapf(ErrStrategyNotFound, "release strategy %s in workspace %s", id, workspaceID)
		}
		return nil, errors.Wrapf(err, "delete release strategy %s in workspace %s", id, workspaceID)
	}
	return strategy, nil
}
