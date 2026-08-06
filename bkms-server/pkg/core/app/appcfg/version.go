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

const versionCollectionName = "app_config_file_versions"

// AppConfigFileVersionStore defines version record storage behavior
type AppConfigFileVersionStore interface {
	Add(ctx context.Context, version AppConfigFileVersion) (bson.ObjectID, error)
	BatchGetByAppAndIDs(ctx context.Context, appID string, ids []bson.ObjectID) ([]AppConfigFileVersion, error)
	GetByFileAndVersion(
		ctx context.Context,
		appConfigFileID bson.ObjectID,
		version int64,
	) (*AppConfigFileVersion, error)
	List(ctx context.Context, opts AppConfigFileVersionListOptions) ([]AppConfigFileVersion, int64, error)
	SoftDeleteByID(ctx context.Context, id bson.ObjectID, deleter string) (int64, error)
	DeleteByFileID(ctx context.Context, appConfigFileID bson.ObjectID) (int64, error)
}

// AppConfigFileVersionListOptions holds version list filters
type AppConfigFileVersionListOptions struct {
	AppID           string
	AppConfigFileID *bson.ObjectID
	EnvName         *string
	Name            *string
	Version         *int64
	Creator         *string
	Description     *string
	Page            int64
	PageSize        int64
	IncludeDeleted  bool
}

// AppConfigFileVersionStoreMongo is the MongoDB implementation of AppConfigFileVersionStore
type AppConfigFileVersionStoreMongo struct {
	collection *mongo.Collection
}

var _ AppConfigFileVersionStore = &AppConfigFileVersionStoreMongo{}

// NewAppConfigFileVersionStoreMongo creates a new version store
func NewAppConfigFileVersionStoreMongo(client *mongo.Client, dbName string) (*AppConfigFileVersionStoreMongo, error) {
	coll := client.Database(dbName).Collection(versionCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appConfigFileID + version
	// - 查询提速：appConfigFileID + isDeleted + createdAt(倒序)
	// - 查询提速：appID + envName + isDeleted + createdAt(倒序)
	// - 查询提速：appConfigFileID + isDeleted + creator + createdAt(倒序)
	return &AppConfigFileVersionStoreMongo{collection: coll}, nil
}

// Add inserts a version record
func (s *AppConfigFileVersionStoreMongo) Add(
	ctx context.Context,
	version AppConfigFileVersion,
) (bson.ObjectID, error) {
	if version.CreatedAt.IsZero() {
		version.CreatedAt = time.Now()
	}
	ret, err := s.collection.InsertOne(ctx, version)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, errors.New("app config file version already exists")
		}
		return bson.NilObjectID, err
	}
	oid, ok := ret.InsertedID.(bson.ObjectID)
	if !ok {
		return bson.NilObjectID, errors.New("failed to get inserted ID")
	}
	return oid, nil
}

// BatchGetByAppAndIDs loads versions by primary keys and ensures they belong to the specified app
func (s *AppConfigFileVersionStoreMongo) BatchGetByAppAndIDs(
	ctx context.Context,
	appID string,
	ids []bson.ObjectID,
) ([]AppConfigFileVersion, error) {
	if len(ids) == 0 {
		return []AppConfigFileVersion{}, nil
	}
	cursor, err := s.collection.Find(ctx, bson.M{"appID": appID, "_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []AppConfigFileVersion
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetByFileAndVersion loads a version by file id and version number
func (s *AppConfigFileVersionStoreMongo) GetByFileAndVersion(
	ctx context.Context, appConfigFileID bson.ObjectID, version int64,
) (*AppConfigFileVersion, error) {
	var obj AppConfigFileVersion
	filter := bson.M{
		"appConfigFileID": appConfigFileID,
		"version":         version,
		"isDeleted":       bson.M{"$ne": true},
	}
	if err := s.collection.FindOne(ctx, filter).Decode(&obj); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Errorf("app config file version %s/%d not found", appConfigFileID.Hex(), version)
		}
		return nil, err
	}
	return &obj, nil
}

// List returns versions and total count
func (s *AppConfigFileVersionStoreMongo) List(
	ctx context.Context, opts AppConfigFileVersionListOptions,
) ([]AppConfigFileVersion, int64, error) {
	filter := bson.M{"appID": opts.AppID}
	if !opts.IncludeDeleted {
		filter["isDeleted"] = bson.M{"$ne": true}
	}
	if opts.AppConfigFileID != nil {
		filter["appConfigFileID"] = *opts.AppConfigFileID
	}
	if opts.EnvName != nil {
		filter["envName"] = *opts.EnvName
	}
	if opts.Name != nil {
		filter["name"] = *opts.Name
	}
	if opts.Version != nil {
		filter["version"] = *opts.Version
	}
	if opts.Creator != nil {
		filter["creator"] = *opts.Creator
	}
	if opts.Description != nil && *opts.Description != "" {
		filter["description"] = bson.M{"$regex": *opts.Description, "$options": "i"}
	}

	total, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	sortBy := bson.D{
		{Key: "createdAt", Value: -1},
		{Key: "_id", Value: -1}, // 增加 _id 作为二级排序
	}
	findOpts := options.Find().SetSort(sortBy)
	if opts.Page > 0 && opts.PageSize > 0 {
		findOpts.SetSkip((opts.Page - 1) * opts.PageSize)
		findOpts.SetLimit(opts.PageSize)
	}
	cursor, err := s.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var result []AppConfigFileVersion
	if err = cursor.All(ctx, &result); err != nil {
		return nil, 0, err
	}
	return result, total, nil
}

// SoftDeleteByID marks a version as deleted
func (s *AppConfigFileVersionStoreMongo) SoftDeleteByID(
	ctx context.Context,
	id bson.ObjectID,
	deleter string,
) (int64, error) {
	now := time.Now()
	result, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"isDeleted": true,
		"deleter":   deleter,
		"deletedAt": now,
	}})
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

// DeleteByFileID hard deletes all versions belonging to a config file.
// Used when deleting a config file so its version history is removed together.
func (s *AppConfigFileVersionStoreMongo) DeleteByFileID(
	ctx context.Context,
	appConfigFileID bson.ObjectID,
) (int64, error) {
	result, err := s.collection.DeleteMany(ctx, bson.M{"appConfigFileID": appConfigFileID})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}
