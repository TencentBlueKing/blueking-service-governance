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

package appcfg

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const defCollectionName = "app_config_file_defs"

// AppConfigFileDefStore 配置文件 def 的存储接口。
type AppConfigFileDefStore interface {
	Add(ctx context.Context, def AppConfigFileDef) (bson.ObjectID, error)
	GetByID(ctx context.Context, id bson.ObjectID) (*AppConfigFileDef, error)
	ListByApp(ctx context.Context, appID string, opts ...DefListOption) ([]AppConfigFileDef, error)
	Update(ctx context.Context, def AppConfigFileDef) (int64, error)
	DeleteByID(ctx context.Context, id bson.ObjectID) (int64, error)
	DeleteByApp(ctx context.Context, appID string) (int64, error)
}

// DefListOption def 列表查询选项。
type DefListOption interface {
	ApplyToDefOptions(*DefListOptions)
}

// DefListOptions def 列表查询的聚合选项。
type DefListOptions struct {
	filterConfigKind *ConfigKind
}

// DefFilterConfigKind 按 ConfigKind 过滤 def 记录。
type DefFilterConfigKind ConfigKind

// ApplyToDefOptions 应用 ConfigKind 过滤条件。
func (f DefFilterConfigKind) ApplyToDefOptions(opts *DefListOptions) {
	ck := ConfigKind(f)
	opts.filterConfigKind = &ck
}

var _ AppConfigFileDefStore = &AppConfigFileDefStoreMongo{}

// AppConfigFileDefStoreMongo AppConfigFileDefStore 的 MongoDB 实现。
type AppConfigFileDefStoreMongo struct {
	collection *mongo.Collection
}

// NewAppConfigFileDefStoreMongo 创建 MongoDB 驱动的 def store。
func NewAppConfigFileDefStoreMongo(client *mongo.Client, dbName string) (*AppConfigFileDefStoreMongo, error) {
	coll := client.Database(dbName).Collection(defCollectionName)
	return &AppConfigFileDefStoreMongo{collection: coll}, nil
}

// Add 插入一条 def 记录。
func (s *AppConfigFileDefStoreMongo) Add(ctx context.Context, def AppConfigFileDef) (bson.ObjectID, error) {
	now := time.Now()
	if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}

	ret, err := s.collection.InsertOne(ctx, def)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, errors.New("app config file def already exists")
		}
		return bson.NilObjectID, err
	}
	oid, ok := ret.InsertedID.(bson.ObjectID)
	if !ok {
		return bson.NilObjectID, errors.New("failed to get inserted ID")
	}
	return oid, nil
}

// GetByID 按主键查询 def 记录。
func (s *AppConfigFileDefStoreMongo) GetByID(ctx context.Context, id bson.ObjectID) (*AppConfigFileDef, error) {
	var obj AppConfigFileDef
	if err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&obj); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Errorf("app config file def %s not found", id.Hex())
		}
		return nil, err
	}
	return &obj, nil
}

// ListByApp 返回指定应用下的全部 def 记录。
func (s *AppConfigFileDefStoreMongo) ListByApp(
	ctx context.Context,
	appID string,
	opts ...DefListOption,
) ([]AppConfigFileDef, error) {
	listOpts := &DefListOptions{}
	for _, o := range opts {
		o.ApplyToDefOptions(listOpts)
	}

	filter := bson.M{"appID": appID}
	if listOpts.filterConfigKind != nil {
		filter["configKind"] = string(*listOpts.filterConfigKind)
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})
	cursor, err := s.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []AppConfigFileDef
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Update 更新 def 记录的可变字段。
func (s *AppConfigFileDefStoreMongo) Update(ctx context.Context, def AppConfigFileDef) (int64, error) {
	if def.ID == bson.NilObjectID {
		return 0, errors.New("def ID is required for update")
	}
	result, err := s.collection.UpdateOne(ctx, bson.M{"_id": def.ID}, bson.M{"$set": bson.M{
		"name":          def.Name,
		"mountDir":      def.MountDir,
		"envConfigMode": def.EnvConfigMode,
	}})
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

// DeleteByID 按 ID 删除 def 记录。
func (s *AppConfigFileDefStoreMongo) DeleteByID(ctx context.Context, id bson.ObjectID) (int64, error) {
	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// DeleteByApp 删除指定应用下的全部 def 记录。
func (s *AppConfigFileDefStoreMongo) DeleteByApp(ctx context.Context, appID string) (int64, error) {
	result, err := s.collection.DeleteMany(ctx, bson.M{"appID": appID})
	if err != nil {
		return 0, errors.Wrapf(err, "delete app config file defs for app [%s]", appID)
	}
	return result.DeletedCount, nil
}
