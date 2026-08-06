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

package helmcomponent

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ErrHelmAppComponentNotFound 组件引用不存在
var ErrHelmAppComponentNotFound = errors.New("helm app component not found")

// HelmAppComponentStore 定义 Helm 应用组件引用的存储接口
type HelmAppComponentStore interface {
	// Add 添加一个组件引用
	Add(ctx context.Context, comp *HelmAppComponent) error
	// Get 根据 ID 获取组件引用
	Get(ctx context.Context, id bson.ObjectID) (*HelmAppComponent, error)
	// ListByAppAndEnv 根据应用 ID 和环境名称查询组件引用列表（按 priority 升序）
	ListByAppAndEnv(ctx context.Context, appID, envName string) ([]*HelmAppComponent, error)
	// Update 更新组件引用
	Update(ctx context.Context, id bson.ObjectID, data *UpdateData) error
	// Delete 删除组件引用
	Delete(ctx context.Context, id bson.ObjectID) error
	// DeleteByApp 删除应用下所有组件引用
	DeleteByApp(ctx context.Context, appID string) error
}

// UpdateData 定义更新组件引用时允许修改的字段
type UpdateData struct {
	Properties map[string]any
	Target     *TargetResourceSelector
	Priority   *int
}

// DbHelmAppComponentStore MongoDB 实现
type DbHelmAppComponentStore struct {
	collection *mongo.Collection
}

// 编译期接口实现检查
var _ HelmAppComponentStore = &DbHelmAppComponentStore{}

// NewDbHelmAppComponentStore 创建 MongoDB 存储实例。
func NewDbHelmAppComponentStore(client *mongo.Client, dbName string) (*DbHelmAppComponentStore, error) {
	coll := client.Database(dbName).Collection(helmAppComponentCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + envName + name
	// - 查询提速：appID + envName
	return &DbHelmAppComponentStore{collection: coll}, nil
}

// Add 添加一个组件引用
func (s *DbHelmAppComponentStore) Add(ctx context.Context, comp *HelmAppComponent) error {
	now := time.Now()
	comp.CreatedAt = now
	comp.UpdatedAt = now

	// 确保 Name 已生成
	comp.EnsureName()

	result, err := s.collection.InsertOne(ctx, comp)
	if err != nil {
		return errors.Wrap(err, "insert helm app component")
	}
	if id, ok := result.InsertedID.(bson.ObjectID); ok {
		comp.ID = id
	}
	return nil
}

// Get 根据 ID 获取组件引用
func (s *DbHelmAppComponentStore) Get(ctx context.Context, id bson.ObjectID) (*HelmAppComponent, error) {
	filter := bson.M{"_id": id}
	var comp HelmAppComponent
	err := s.collection.FindOne(ctx, filter).Decode(&comp)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrHelmAppComponentNotFound
		}
		return nil, errors.Wrap(err, "find helm app component")
	}
	return &comp, nil
}

// ListByAppAndEnv 根据应用 ID 和环境名称查询组件引用列表（按 priority 升序）
func (s *DbHelmAppComponentStore) ListByAppAndEnv(
	ctx context.Context, appID, envName string,
) ([]*HelmAppComponent, error) {
	filter := bson.M{"appID": appID, "envName": envName}
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: 1}, {Key: "createdAt", Value: 1}})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, errors.Wrap(err, "find helm app components")
	}
	defer cursor.Close(ctx)

	var comps []*HelmAppComponent
	if err = cursor.All(ctx, &comps); err != nil {
		return nil, errors.Wrap(err, "decode helm app components")
	}
	return comps, nil
}

// Update 更新组件引用
func (s *DbHelmAppComponentStore) Update(ctx context.Context, id bson.ObjectID, data *UpdateData) error {
	setFields := bson.M{
		"updatedAt": time.Now(),
	}
	if data.Properties != nil {
		setFields["properties"] = data.Properties
	}
	if data.Target != nil {
		setFields["target"] = *data.Target
	}
	if data.Priority != nil {
		setFields["priority"] = *data.Priority
	}

	update := bson.M{"$set": setFields}
	result, err := s.collection.UpdateByID(ctx, id, update)
	if err != nil {
		return errors.Wrap(err, "update helm app component")
	}
	if result.MatchedCount == 0 {
		return ErrHelmAppComponentNotFound
	}
	return nil
}

// Delete 删除组件引用（幂等操作）
func (s *DbHelmAppComponentStore) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return errors.Wrap(err, "delete helm app component")
	}
	return nil
}

// DeleteByApp 删除应用下所有组件引用
func (s *DbHelmAppComponentStore) DeleteByApp(ctx context.Context, appID string) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{"appID": appID})
	if err != nil {
		return errors.Wrap(err, "delete helm app components by app")
	}
	return nil
}
