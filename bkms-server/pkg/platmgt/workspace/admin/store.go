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

package admin

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	// ErrRecordNotFound 表示目标临时管理员记录不存在。
	ErrRecordNotFound = errors.New("temporary admin record not found")
	// ErrRecordAlreadyExists 表示已存在未回收的临时管理员记录。
	ErrRecordAlreadyExists = errors.New("temporary admin record already exists")
)

// Store 持久化临时管理员记录。
type Store interface {
	GetLatestActiveGrant(ctx context.Context, workspaceID, username string) (*WorkspaceTempAdmin, error)
	Create(ctx context.Context, record *WorkspaceTempAdmin) error
	Update(ctx context.Context, record *WorkspaceTempAdmin) error
	ListExpiredGrants(ctx context.Context, now time.Time) ([]WorkspaceTempAdmin, error)
}

var _ Store = (*StoreMongo)(nil)

// StoreMongo 持久化管理员临时授权记录。
type StoreMongo struct {
	collection *mongo.Collection
}

// NewStoreMongo 创建 StoreMongo。
func NewStoreMongo(client *mongo.Client, dbName string) (*StoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 条件唯一：workspaceID + username（仅 isRecycled = false）
	// - 查询提速：isRecycled + expiresAt
	return &StoreMongo{collection: coll}, nil
}

// GetLatestActiveGrant 返回某个空间/用户当前仍生效的临时管理员授权记录。
//
// 对同一个 workspaceID + username，未回收记录受 partial unique index 约束，
// 因此查询结果至多一条，这里无需额外排序。
func (s *StoreMongo) GetLatestActiveGrant(
	ctx context.Context,
	workspaceID, username string,
) (*WorkspaceTempAdmin, error) {
	filter := bson.M{
		"workspaceID": workspaceID,
		"username":    username,
		"isRecycled":  false,
	}

	var record WorkspaceTempAdmin
	// 活跃记录由唯一索引保证至多一条，按过滤条件直接查询即可
	err := s.collection.FindOne(
		ctx,
		filter,
	).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRecordNotFound
		}
		return nil, errors.Wrap(err, "find latest active temporary admin grant")
	}
	return &record, nil
}

// Create 插入一条新的临时管理员记录。
func (s *StoreMongo) Create(ctx context.Context, record *WorkspaceTempAdmin) error {
	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	result, err := s.collection.InsertOne(ctx, record)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrRecordAlreadyExists
		}
		return errors.Wrap(err, "insert temporary admin record")
	}
	insertedID, ok := result.InsertedID.(bson.ObjectID)
	if !ok {
		return errors.New("failed to get inserted ID")
	}
	record.ID = insertedID
	return nil
}

var immutableFields = map[string]struct{}{
	"_id":         {},
	"workspaceID": {},
	"username":    {},
	"creator":     {},
	"createdAt":   {},
}

// Update 按 ID 更新一条临时管理员记录的全部可变字段。
func (s *StoreMongo) Update(ctx context.Context, record *WorkspaceTempAdmin) error {
	record.UpdatedAt = time.Now()

	raw, err := bson.Marshal(record)
	if err != nil {
		return errors.Wrap(err, "marshal temporary admin record")
	}
	var doc bson.M
	if unmarshalErr := bson.Unmarshal(raw, &doc); unmarshalErr != nil {
		return errors.Wrap(unmarshalErr, "unmarshal temporary admin record to bson.M")
	}
	for key := range immutableFields {
		delete(doc, key)
	}

	result, err := s.collection.UpdateOne(ctx, bson.M{"_id": record.ID}, bson.M{"$set": doc})
	if err != nil {
		return errors.Wrap(err, "update temporary admin record")
	}
	if result.MatchedCount == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// ListExpiredGrants 列出待清理的已过期且未回收记录。
func (s *StoreMongo) ListExpiredGrants(ctx context.Context, now time.Time) ([]WorkspaceTempAdmin, error) {
	filter := bson.M{
		"isRecycled": false,
		"expiresAt": bson.M{
			"$lte": now,
			"$ne":  time.Time{},
		},
	}
	findOpts := options.Find().
		SetSort(bson.D{{Key: "expiresAt", Value: 1}, {Key: "updatedAt", Value: 1}, {Key: "_id", Value: 1}})

	cursor, err := s.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, errors.Wrap(err, "find expired temporary admin records")
	}
	defer cursor.Close(ctx)

	var records []WorkspaceTempAdmin
	if err := cursor.All(ctx, &records); err != nil {
		return nil, errors.Wrap(err, "decode expired temporary admin records")
	}
	return records, nil
}
