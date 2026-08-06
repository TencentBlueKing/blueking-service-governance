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

// Package build 提供 Helm Chart 构建触发和构建记录管理功能
package build

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// helmChartBuildRecordCollName Helm Chart 构建记录表名
const helmChartBuildRecordCollName = "helm_chart_build_records"

// helmChartBuildCounterCollName Helm Chart 构建序号计数器表名
const helmChartBuildCounterCollName = "helm_chart_build_counters"

// ErrRecordNotFound Helm Chart 构建记录未找到时返回固定错误
var ErrRecordNotFound = errors.New("helm chart build record not found")

// RecordStore Helm Chart 构建记录存储接口
type RecordStore interface {
	// Create 创建新的构建记录（自动分配构建序号）
	Create(ctx context.Context, record *Record) error
	// Update 更新已存在的构建记录（更新状态、extras、结束时间）
	Update(ctx context.Context, record *Record) error
	// Get 通过 AppID 和 BuildID 获取构建记录
	Get(ctx context.Context, appID, buildID string) (*Record, error)
	// List 按分页/关键字列出构建记录（按 startedAt 倒序），keyword 模糊匹配 chartVersion / operator
	List(ctx context.Context, appID, keyword string, page, pageSize int64) ([]Record, int64, error)
}

var _ RecordStore = &RecordStoreMongo{}

// RecordStoreMongo RecordStore 的 MongoDB 实现
type RecordStoreMongo struct {
	collection  *mongo.Collection
	counterColl *mongo.Collection
}

// NewRecordStoreMongo 创建 RecordStoreMongo 实例
func NewRecordStoreMongo(client *mongo.Client, dbName string) (*RecordStoreMongo, error) {
	db := client.Database(dbName)
	coll := db.Collection(helmChartBuildRecordCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + buildID
	counterColl := db.Collection(helmChartBuildCounterCollName)
	return &RecordStoreMongo{collection: coll, counterColl: counterColl}, nil
}

// buildCounter 构建序号计数器文档
type buildCounter struct {
	AppID string `bson:"_id"`
	Seq   int64  `bson:"seq"`
}

// nextNum 获取指定 AppID 的下一个构建序号（原子操作，并发安全）
func (s *RecordStoreMongo) nextNum(ctx context.Context, appID string) (int64, error) {
	filter := bson.M{"_id": appID}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var c buildCounter
	if err := s.counterColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&c); err != nil {
		return 0, errors.Wrapf(err, "get next build num for app %s", appID)
	}
	return c.Seq, nil
}

// Create 创建新的构建记录（自动分配构建序号）
func (s *RecordStoreMongo) Create(ctx context.Context, record *Record) error {
	num, err := s.nextNum(ctx, record.AppID)
	if err != nil {
		return err
	}
	record.Num = num
	record.StartedAt = time.Now()

	if _, err = s.collection.InsertOne(ctx, record); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Errorf("helm chart build record with the same appID & buildID already exists")
		}
		return err
	}
	return nil
}

// Update 更新已存在的构建记录（状态、extras、结束时间）
func (s *RecordStoreMongo) Update(ctx context.Context, record *Record) error {
	filter := bson.M{
		"appID":   record.AppID,
		"buildID": record.BuildID,
	}
	updateDoc := bson.M{"$set": bson.M{
		"status":  record.Status,
		"extras":  record.Extras,
		"endedAt": record.EndedAt,
	}}
	ret, err := s.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return errors.Wrapf(err, "update helm chart build record for app %s build %s", record.AppID, record.BuildID)
	}
	if ret.MatchedCount == 0 {
		return errors.Wrapf(ErrRecordNotFound, "app %s build %s", record.AppID, record.BuildID)
	}
	return nil
}

// Get 通过 AppID 和 BuildID 获取构建记录
func (s *RecordStoreMongo) Get(ctx context.Context, appID, buildID string) (*Record, error) {
	var record Record
	if err := s.collection.FindOne(ctx, bson.M{"appID": appID, "buildID": buildID}).Decode(&record); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Wrapf(ErrRecordNotFound, "app %s build %s", appID, buildID)
		}
		return nil, err
	}
	return &record, nil
}

// keywordFilter 构造按 chartVersion / operator / num 模糊匹配的 $or 条件
func keywordFilter(keyword string) []bson.M {
	keyword = regexp.QuoteMeta(keyword)
	return []bson.M{
		{"chartVersion": bson.M{"$regex": keyword, "$options": "i"}},
		{"operator": bson.M{"$regex": keyword, "$options": "i"}},
	}
}

// List 按分页/关键字列出构建记录（按 startedAt 倒序）
func (s *RecordStoreMongo) List(
	ctx context.Context, appID, keyword string, page, pageSize int64,
) ([]Record, int64, error) {
	filter := bson.M{"appID": appID}
	if keyword != "" {
		filter["$or"] = keywordFilter(keyword)
	}

	total, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "count helm chart build records for app %s", appID)
	}

	opts := options.Find().
		SetLimit(pageSize).
		SetSkip((page - 1) * pageSize).
		SetSort(bson.D{{Key: "startedAt", Value: -1}})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "find helm chart build records for app %s", appID)
	}
	defer cursor.Close(ctx)

	var records []Record
	if err = cursor.All(ctx, &records); err != nil {
		return nil, 0, errors.Wrapf(err, "decode helm chart build records for app %s", appID)
	}
	return records, total, nil
}
