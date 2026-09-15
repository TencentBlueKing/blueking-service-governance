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

package devmode

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionName 开发模式发布记录表名
const publishRecordCollectionName = "devmode_publish_records"

// PublishRecordStore 开发模式发布记录存储接口
type PublishRecordStore interface {
	// Create 批量创建发布记录，返回新记录 ID 列表（hex）
	Create(ctx context.Context, records []PublishRecord) ([]string, error)

	// List 列表发布记录（支持分页）
	List(
		ctx context.Context,
		appID, envName, keyword string,
		page, pageSize int64,
	) ([]PublishRecord, int64, error)

	// ListLatestByInstance 返回指定实例列表中每个实例的最新一条发布记录
	ListLatestByInstance(
		ctx context.Context,
		appID, envName string,
		instanceIDs []string,
	) (map[string]*PublishRecord, error)
}

var _ PublishRecordStore = &PublishRecordStoreMongo{}

// PublishRecordStoreMongo PublishRecordStore 实现（基于 MongoDB）
type PublishRecordStoreMongo struct {
	collection *mongo.Collection
}

// NewPublishRecordStoreMongo 新建 PublishRecordStoreMongo 实例
func NewPublishRecordStoreMongo(client *mongo.Client, dbName string) (*PublishRecordStoreMongo, error) {
	coll := client.Database(dbName).Collection(publishRecordCollectionName)
	return &PublishRecordStoreMongo{collection: coll}, nil
}

// Create 批量创建发布记录
func (s *PublishRecordStoreMongo) Create(ctx context.Context, records []PublishRecord) ([]string, error) {
	if len(records) == 0 {
		return nil, nil
	}

	now := time.Now()
	docs := lo.Map(records, func(record PublishRecord, _ int) any {
		record.ID = bson.NewObjectID()
		record.CreatedAt, record.UpdatedAt = now, now
		return record
	})

	ret, err := s.collection.InsertMany(ctx, docs)
	if err != nil {
		return nil, err
	}

	ids := lo.FilterMap(ret.InsertedIDs, func(id any, _ int) (string, bool) {
		oid, ok := id.(bson.ObjectID)
		if !ok {
			return "", false
		}
		return oid.Hex(), true
	})
	return ids, nil
}

// List 列表发布记录（支持分页）
func (s *PublishRecordStoreMongo) List(
	ctx context.Context,
	appID, envName, keyword string,
	page, pageSize int64,
) ([]PublishRecord, int64, error) {
	filter := bson.M{
		"appID":   appID,
		"envName": envName,
	}
	if keyword != "" {
		// 转义正则表达式特殊字符，防止注入攻击和语法错误
		keyword = regexp.QuoteMeta(keyword)
		// 模糊匹配：二进制名称、操作人、实例
		filter["$or"] = []bson.M{
			{"binaryName": bson.M{"$regex": keyword, "$options": "i"}},
			{"operator": bson.M{"$regex": keyword, "$options": "i"}},
			{"instance": bson.M{"$regex": keyword, "$options": "i"}},
		}
	}

	total, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "count devmode publish records for app %s env %s", appID, envName)
	}

	opts := options.Find().
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize).
		SetSort(bson.D{{"createdAt", -1}})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "list devmode publish records for app %s env %s", appID, envName)
	}
	defer cursor.Close(ctx)

	var records []PublishRecord
	if err = cursor.All(ctx, &records); err != nil {
		return nil, 0, errors.Wrapf(err, "decode devmode publish records for app %s env %s", appID, envName)
	}
	return records, total, nil
}

// ListLatestByInstance 返回指定实例列表中每个实例的最新一条发布记录
func (s *PublishRecordStoreMongo) ListLatestByInstance(
	ctx context.Context,
	appID, envName string,
	instanceIDs []string,
) (map[string]*PublishRecord, error) {
	if len(instanceIDs) == 0 {
		return nil, nil
	}

	pipeline := bson.A{
		bson.M{"$match": bson.M{
			"appID":    appID,
			"envName":  envName,
			"instance": bson.M{"$in": instanceIDs},
		}},
		bson.M{"$sort": bson.M{"createdAt": -1}},
		bson.M{"$group": bson.M{
			"_id": "$instance",
			"doc": bson.M{"$first": "$$ROOT"},
		}},
		bson.M{"$replaceRoot": bson.M{"newRoot": "$doc"}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrapf(err, "aggregate latest publish records for app %s env %s", appID, envName)
	}
	defer cursor.Close(ctx)

	out := make(map[string]*PublishRecord)
	for cursor.Next(ctx) {
		var record PublishRecord
		if err := cursor.Decode(&record); err != nil {
			return nil, errors.Wrapf(err, "decode latest publish record for app %s env %s", appID, envName)
		}
		rec := record
		out[rec.Instance] = &rec
	}
	if err := cursor.Err(); err != nil {
		return nil, errors.Wrapf(err, "iterate latest publish records for app %s env %s", appID, envName)
	}
	return out, nil
}
