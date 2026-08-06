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

package clusteraddon

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// clusterAddonDefCollectionName 集群 Addon 定义集合名
const clusterAddonDefCollectionName = "cluster_addon_defs"

// ErrClusterAddonDefNotFound Addon 定义未找到
var ErrClusterAddonDefNotFound = errors.New("cluster addon def not found")

// ClusterAddonDefStore 集群 Addon 定义存储接口
type ClusterAddonDefStore interface {
	// Create 创建或更新 Addon 定义（基于 name 进行 upsert）
	Create(ctx context.Context, addonDef *ClusterAddonDef) error

	// Get 通过组件名称获取 Addon 定义
	Get(ctx context.Context, name string) (*ClusterAddonDef, error)

	// List 列出所有 Addon 定义
	List(ctx context.Context) ([]*ClusterAddonDef, error)

	// Delete 删除 Addon 定义
	Delete(ctx context.Context, name string) (int64, error)
}

var _ ClusterAddonDefStore = &ClusterAddonDefStoreMongo{}

// ClusterAddonDefStoreMongo 集群 Addon 定义 MongoDB 存储
type ClusterAddonDefStoreMongo struct {
	collection *mongo.Collection
}

// NewClusterAddonDefStoreMongo 创建 ClusterAddonDefStoreMongo 实例
func NewClusterAddonDefStoreMongo(client *mongo.Client, dbName string) (*ClusterAddonDefStoreMongo, error) {
	coll := client.Database(dbName).Collection(clusterAddonDefCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：name
	return &ClusterAddonDefStoreMongo{collection: coll}, nil
}

// Create 创建或更新 Addon 定义（基于 name 进行 upsert）
func (s *ClusterAddonDefStoreMongo) Create(ctx context.Context, addonDef *ClusterAddonDef) error {
	now := time.Now()
	if addonDef.CreatedAt.IsZero() {
		addonDef.CreatedAt = now
	}
	addonDef.UpdatedAt = now

	filter := bson.M{"name": addonDef.Name}
	setDoc, err := buildAddonDefSetDoc(addonDef)
	if err != nil {
		return errors.Wrap(err, "prepare the upsert document")
	}
	update := bson.M{
		"$set":         setDoc,
		"$setOnInsert": bson.M{"createdAt": addonDef.CreatedAt},
	}
	opts := options.UpdateOne().SetUpsert(true)
	if _, err := s.collection.UpdateOne(ctx, filter, update, opts); err != nil {
		return errors.Wrapf(err, "upsert cluster addon def %s", addonDef.Name)
	}
	return nil
}

// Get 通过组件名称获取 Addon 定义
func (s *ClusterAddonDefStoreMongo) Get(ctx context.Context, name string) (*ClusterAddonDef, error) {
	addonDef := new(ClusterAddonDef)
	filter := bson.M{"name": name}

	err := s.collection.FindOne(ctx, filter).Decode(addonDef)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrClusterAddonDefNotFound
		}
		return nil, errors.Wrapf(err, "get cluster addon def %s", name)
	}
	return addonDef, nil
}

// List 列出所有 Addon 定义
func (s *ClusterAddonDefStoreMongo) List(ctx context.Context) ([]*ClusterAddonDef, error) {
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := s.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, errors.Wrap(err, "list cluster addon defs")
	}
	defer cursor.Close(ctx)

	var addonDefs []*ClusterAddonDef
	if err = cursor.All(ctx, &addonDefs); err != nil {
		return nil, errors.Wrap(err, "decode cluster addon defs")
	}
	return addonDefs, nil
}

// Delete 删除 Addon 定义
func (s *ClusterAddonDefStoreMongo) Delete(ctx context.Context, name string) (int64, error) {
	filter := bson.M{"name": name}
	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return 0, errors.Wrapf(err, "delete cluster addon def %s", name)
	}
	return result.DeletedCount, nil
}

// buildAddonDefSetDoc 通过 bson marshal 生成 $set 文档，
// 避免新增字段时遗漏手动维护。
func buildAddonDefSetDoc(addonDef *ClusterAddonDef) (bson.M, error) {
	data, err := bson.Marshal(addonDef)
	if err != nil {
		return nil, errors.Wrap(err, "marshal addon def for update")
	}

	setDoc := bson.M{}
	if err = bson.Unmarshal(data, &setDoc); err != nil {
		return nil, errors.Wrap(err, "unmarshal addon def for update")
	}

	delete(setDoc, "createdAt")
	delete(setDoc, "_id")
	return setDoc, nil
}
