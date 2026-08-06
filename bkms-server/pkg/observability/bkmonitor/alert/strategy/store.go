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

// Package strategy 提供蓝鲸监控告警策略相关功能
package strategy

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// collectionName bkms 中保存告警策略记录，
// 存储策略与 workspace/app 的关联范围、启用状态、strategyCode 等本地元数据，用于默认策略下发或应用部署后策略下发。
const collectionName = "bkmonitor_alert_strategy"

// ErrNotFound 告警策略未找到
var ErrNotFound = errors.New("AlertStrategy not found")

var _ Store = &StoreMongo{}

// Store defines the storage interface for alert strategy data.
type Store interface {
	Create(ctx context.Context, rule *AlertStrategy) (bson.ObjectID, error)
	Get(ctx context.Context, id bson.ObjectID) (*AlertStrategy, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]AlertStrategy, error)
	ListByApp(ctx context.Context, workspaceID, appID string) ([]AlertStrategy, error)
	ListByAppAndRemoteEnv(
		ctx context.Context,
		workspaceID, appID string,
		envID bson.ObjectID,
		trafficLaneName string,
	) ([]AlertStrategy, error)
	ListEnabledByAppMatchingEnv(
		ctx context.Context, workspaceID, appID, envType string, envID bson.ObjectID,
	) ([]AlertStrategy, error)
	Update(ctx context.Context, id bson.ObjectID, updateData bson.M) error
	Delete(ctx context.Context, id bson.ObjectID) error
}

// StoreMongo implements Store interface with MongoDB.
type StoreMongo struct {
	collection *mongo.Collection
}

// NewStoreMongo creates a new strategy store instance.
func NewStoreMongo(client *mongo.Client, dbName string) (Store, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 查询提速：workspaceID + appID + enabled
	// - 查询提速：workspaceID + appID + remoteRefs.envID + remoteRefs.trafficLaneName
	return &StoreMongo{collection: coll}, nil
}

// Create 创建告警策略记录，插入前会校验结构体与生效范围，并补全时间戳与默认值。
func (s *StoreMongo) Create(ctx context.Context, rule *AlertStrategy) (bson.ObjectID, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(rule); err != nil {
		return bson.NilObjectID, errors.Wrap(err, "AlertStrategy validation failed")
	}
	if err := rule.EffectiveScope.Validate(); err != nil {
		return bson.NilObjectID, errors.Wrap(err, "AlertStrategy effectiveScope validation failed")
	}

	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = rule.CreatedAt

	if rule.RemoteRefs == nil {
		rule.RemoteRefs = make([]RemoteStrategyRef, 0)
	}
	if rule.NoticeGroupIDs == nil {
		rule.NoticeGroupIDs = make([]int64, 0)
	}

	ret, err := s.collection.InsertOne(ctx, rule)
	if err != nil {
		return bson.NilObjectID, err
	}
	return ret.InsertedID.(bson.ObjectID), nil
}

// Get 根据 ID 查询单个告警策略，不存在时返回 ErrNotFound。
func (s *StoreMongo) Get(ctx context.Context, id bson.ObjectID) (*AlertStrategy, error) {
	return s.findOne(ctx, bson.M{"_id": id})
}

// ListByWorkspace 查询指定工作空间下的全部告警策略，按创建时间倒序返回。
func (s *StoreMongo) ListByWorkspace(ctx context.Context, workspaceID string) ([]AlertStrategy, error) {
	filter := bson.M{"workspaceID": workspaceID}
	sort := bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // nolint

	rules := make([]AlertStrategy, 0)
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// ListByApp 查询指定工作空间下某应用的全部告警策略，按创建时间倒序返回。
func (s *StoreMongo) ListByApp(ctx context.Context, workspaceID, appID string) ([]AlertStrategy, error) {
	filter := bson.M{"workspaceID": workspaceID, "appID": appID}
	sort := bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // nolint

	rules := make([]AlertStrategy, 0)
	if err = cursor.All(ctx, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

// ListByAppAndRemoteEnv 查询指定工作空间下某应用、且关联了某远程环境的告警策略，按创建时间倒序返回。
func (s *StoreMongo) ListByAppAndRemoteEnv(
	ctx context.Context,
	workspaceID, appID string,
	envID bson.ObjectID,
	trafficLaneName string,
) ([]AlertStrategy, error) {
	filter := bson.M{
		"workspaceID": workspaceID,
		"appID":       appID,
		"remoteRefs": bson.M{
			"$elemMatch": bson.M{
				"envID":           envID,
				"trafficLaneName": trafficLaneName,
			},
		},
	}
	sort := bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // nolint

	strategies := make([]AlertStrategy, 0)
	if err = cursor.All(ctx, &strategies); err != nil {
		return nil, err
	}
	return strategies, nil
}

// ListEnabledByAppMatchingEnv 查询指定工作空间下某应用中、已启用且生效范围匹配给定环境的告警策略，按创建时间倒序返回。
func (s *StoreMongo) ListEnabledByAppMatchingEnv(
	ctx context.Context, workspaceID, appID, envType string, envID bson.ObjectID,
) ([]AlertStrategy, error) {
	filter := bson.M{
		"workspaceID": workspaceID,
		"appID":       appID,
		"enabled":     true,
		"$or": bson.A{
			bson.M{"effectiveScope.type": EffectiveScopeAll},
			bson.M{"effectiveScope.type": EffectiveScopeEnvType, "effectiveScope.envTypes": envType},
			bson.M{"effectiveScope.type": EffectiveScopeSpecificEnvs, "effectiveScope.envIDs": envID},
		},
	}
	sort := bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // nolint

	strategies := make([]AlertStrategy, 0)
	if err = cursor.All(ctx, &strategies); err != nil {
		return nil, err
	}
	return strategies, nil
}

// Update 使用 $set 更新指定告警策略的部分字段，并自动刷新 updatedAt；updateData 为空时直接返回。
func (s *StoreMongo) Update(ctx context.Context, id bson.ObjectID, updateData bson.M) error {
	if len(updateData) == 0 {
		return nil
	}

	updateData["updatedAt"] = time.Now()
	filter := bson.M{"_id": id}
	return s.updateOne(ctx, filter, bson.M{"$set": updateData})
}

// Delete 删除指定 ID 的告警策略。
func (s *StoreMongo) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (s *StoreMongo) findOne(ctx context.Context, filter bson.M) (*AlertStrategy, error) {
	rule := new(AlertStrategy)
	if err := s.collection.FindOne(ctx, filter).Decode(rule); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rule, nil
}

func (s *StoreMongo) updateOne(ctx context.Context, filter, update bson.M) error {
	opts := options.UpdateOne().SetUpsert(false)
	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return ErrNotFound
	}
	return err
}
