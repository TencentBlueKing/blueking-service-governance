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

package autodeploy

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "build_auto_deploy_records"

// ErrRecordNotFound build auto deploy record not found error
var ErrRecordNotFound = errors.New("build auto deploy record not found")

// RecordStore build auto deploy 记录存储接口。
type RecordStore interface {
	Create(ctx context.Context, record *Record) error
	Update(ctx context.Context, record *Record) error
	GetLatest(ctx context.Context, appID, envName, trafficLaneName string) (*Record, error)
	GetByBuildID(ctx context.Context, appID, buildID string) (*Record, error)
	GetByDeployID(ctx context.Context, appID, deployID string) (*Record, error)
}

var _ RecordStore = &RecordStoreMongo{}

// RecordStoreMongo 基于 MongoDB 的实现。
type RecordStoreMongo struct {
	collection *mongo.Collection
}

// NewRecordStoreMongo 创建 build auto deploy record store。
func NewRecordStoreMongo(client *mongo.Client, dbName string) (*RecordStoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + buildID
	// - 查询提速：appID + envName + trafficLaneName + createdAt(倒序)
	// - 条件唯一：appID + deployID（仅 deployID 非空）
	return &RecordStoreMongo{collection: coll}, nil
}

// Create 创建记录
func (s *RecordStoreMongo) Create(ctx context.Context, record *Record) error {
	timeNow := time.Now()
	record.CreatedAt = timeNow
	record.UpdatedAt = timeNow
	if _, err := s.collection.InsertOne(ctx, record); err != nil {
		return errors.Wrap(err, "create build auto deploy record")
	}
	return nil
}

// Update 更新记录
func (s *RecordStoreMongo) Update(ctx context.Context, record *Record) error {
	filter := bson.M{"_id": record.ID}
	updateDoc := bson.M{"$set": bson.M{
		"deployID":  record.DeployID,
		"stage":     record.Stage,
		"status":    record.Status,
		"message":   record.Message,
		"endedAt":   record.EndedAt,
		"updatedAt": time.Now(),
	}}
	ret, err := s.collection.UpdateOne(ctx, filter, updateDoc)
	if err != nil {
		return errors.Wrapf(err, "update build auto deploy record %s", record.ID.Hex())
	}
	if ret.MatchedCount == 0 {
		return ErrRecordNotFound
	}
	return nil
}

// GetLatest 获取最新记录
func (s *RecordStoreMongo) GetLatest(
	ctx context.Context,
	appID, envName, trafficLaneName string,
) (*Record, error) {
	var record Record
	err := s.collection.FindOne(
		ctx,
		bson.M{
			"appID":           appID,
			"envName":         envName,
			"trafficLaneName": trafficLaneName,
		},
		options.FindOne().SetSort(bson.D{{"createdAt", -1}}),
	).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRecordNotFound
		}
		return nil, errors.Wrap(err, "find latest build auto deploy record")
	}
	return &record, nil
}

// GetByBuildID 根据 buildID 获取记录
func (s *RecordStoreMongo) GetByBuildID(ctx context.Context, appID, buildID string) (*Record, error) {
	record, err := s.findOne(ctx, bson.M{"appID": appID, "buildID": buildID})
	if err != nil {
		return nil, errors.Wrap(err, "find build auto deploy record by buildID")
	}
	return record, nil
}

// GetByDeployID 根据 deployID 获取记录
func (s *RecordStoreMongo) GetByDeployID(ctx context.Context, appID, deployID string) (*Record, error) {
	record, err := s.findOne(ctx, bson.M{"appID": appID, "deployID": deployID})
	if err != nil {
		return nil, errors.Wrap(err, "find build auto deploy record by deployID")
	}
	return record, nil
}

func (s *RecordStoreMongo) findOne(ctx context.Context, filter bson.M) (*Record, error) {
	var record Record
	err := s.collection.FindOne(ctx, filter).Decode(&record)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}
