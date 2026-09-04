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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/dbutil"
)

// The name of the MongoDB collection for storing app config files.
const collectionName = "app_config_files"

// ListOrderByField defines the order choices when listing app config files.
type ListOrderByField string

const (
	// ListOrderByName 保留给上层兼容使用；
	// 由于 name 已迁移到 def 表，需在补齐 def 后于上层按原 name 排序语义处理。
	ListOrderByName ListOrderByField = "name"
)

// ErrAppConfigFileVersionConflict indicates the file was modified concurrently.
var ErrAppConfigFileVersionConflict = errors.New("app config file version conflict")

// AppConfigFileStore defines the interface for storing app config files.
type AppConfigFileStore interface {
	// Add adds an app config file to the store.
	Add(ctx context.Context, acf AppConfigFile) (bson.ObjectID, error)

	// GetByID gets the object by the primary key.
	GetByID(ctx context.Context, id bson.ObjectID) (*AppConfigFile, error)

	// IsOwnedByApp checks if the given ID belongs to the specified app.
	IsOwnedByApp(ctx context.Context, id bson.ObjectID, appID string) (bool, error)

	// List lists the app config files for the given app.
	List(ctx context.Context, appID string, listOpts ...AcfListOption) ([]AppConfigFile, error)

	// Update updates an existed AppConfigFile object, returns the number of modified documents and an error if any.
	Update(ctx context.Context, acf AppConfigFile) (int64, error)

	// UpdateIfVersionMatches updates an existing AppConfigFile only when the current version matches expectedVersion.
	UpdateIfVersionMatches(ctx context.Context, acf AppConfigFile, expectedVersion int64) (int64, error)

	// DeleteByID delete an app config file from the store by ID,
	// returns the number of deleted documents and an error if any.
	DeleteByID(ctx context.Context, appID string, id bson.ObjectID) (int64, error)

	// DeleteByApp deletes all app config files for the given app,
	// returns the number of deleted documents and an error if any.
	DeleteByApp(ctx context.Context, appID string) (int64, error)

	// IsReferencedByOther checks if the given app config file ID is referenced by other overlay files.
	IsReferencedByOther(ctx context.Context, id bson.ObjectID) (bool, error)

	// --- Def 关联查询 ---

	// GetByDefIDAndEnv 按 defID + envName 组合查询文件记录。
	GetByDefIDAndEnv(ctx context.Context, defID bson.ObjectID, envName string) (*AppConfigFile, error)

	// ListByDefID 返回指定 def 下的所有文件记录。
	ListByDefID(ctx context.Context, defID bson.ObjectID) ([]AppConfigFile, error)

	// DeleteByDefID 删除指定 def 下的所有文件记录。
	DeleteByDefID(ctx context.Context, defID bson.ObjectID) (int64, error)
}

// AppValuesConfigStore defines the interface for storing app values configurations.
type AppValuesConfigStore interface {
	// Get gets the app values configuration by workspaceID and app ID.
	// It returns a default configuration event if not exists.
	Get(ctx context.Context, appID string) (AppValuesConfig, error)

	// SetDefaultValuesFile sets the default values file ID for an app.
	SetDefaultValuesFile(ctx context.Context, appID string, defaultValuesFileID bson.ObjectID) error

	// DeleteByApp deletes the app values configuration by appID.
	DeleteByApp(ctx context.Context, appID string) error
}

var _ AppConfigFileStore = &AppConfigFileStoreMongo{}

// AppConfigFileStoreMongo is the MongoDB implementation of AppConfigFileStore.
type AppConfigFileStoreMongo struct {
	// Collection is the MongoDB collection for storing app config files data.
	collection *mongo.Collection
}

// NewAppConfigFileStoreMongo creates a new AppConfigFileStore, it accepts a MongoDB client and database name.
func NewAppConfigFileStoreMongo(client *mongo.Client, dbName string) (*AppConfigFileStoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + name
	// - 查询提速：baseAppConfigFileID
	return &AppConfigFileStoreMongo{collection: coll}, nil
}

// Add adds an app config file to the store.
//
// - acf: The app config file object to add
//
// Return the ID of the created object and an error if the operation fails.
func (s *AppConfigFileStoreMongo) Add(ctx context.Context, acf AppConfigFile) (bson.ObjectID, error) {
	// Set the CreatedAt and UpdatedAt fields
	now := time.Now()
	acf.CreatedAt = now
	acf.UpdatedAt = now

	ret, err := s.collection.InsertOne(ctx, acf)
	if err != nil {
		// Check if it's a duplicate key error
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, errors.New("app config file already exists")
		}
		return bson.NilObjectID, err
	}
	if oid, ok := ret.InsertedID.(bson.ObjectID); ok {
		return oid, nil
	}
	return bson.NilObjectID, errors.New("failed to get inserted ID")
}

// GetByID gets the object by the primary key.
//
// - id: The ID of the app config file
//
// Return the app config file object and an error if any.
func (s *AppConfigFileStoreMongo) GetByID(ctx context.Context, id bson.ObjectID) (*AppConfigFile, error) {
	var obj AppConfigFile

	filter := bson.M{"_id": id}
	err := s.collection.FindOne(ctx, filter).Decode(&obj)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// When no record can be found, return a new error
			return nil, errors.Errorf("app config file %s not found", id.Hex())
		}
		return nil, err
	}
	return &obj, nil
}

// IsOwnedByApp checks if the given ID belongs to the specified app name.
//
// - id: The ID of the app config file
// - appID: The ID of the application
//
// Return whether the ID belongs to the application and an error if any.
func (s *AppConfigFileStoreMongo) IsOwnedByApp(
	ctx context.Context, id bson.ObjectID, appID string,
) (bool, error) {
	var result bson.M
	filter := bson.M{"_id": id, "appID": appID}
	// Set the projection to only return the ID field for efficiency
	opts := options.FindOne().SetProjection(bson.M{"_id": 1})

	err := s.collection.FindOne(ctx, filter, opts).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// AcfListOption defines an option for listing app config files, options can change the behavior of List method.
type AcfListOption interface {
	ApplyToOptions(*ListOptions)
}

// ListOptions holds the options for listing app config files.
type ListOptions struct {
	// The field to order the results by. Store 层当前仅支持空值（按 createdAt）。
	orderBy ListOrderByField
	// The type of app config file to filter, default to no filtering
	filterType AppConfigFileType
	// The environment name to filter, default to no filtering
	// Use EnvNameFilterAppLevel to filter only app-level config files
	filterEnvName *string
}

// AcfFilterType usage: .List(..., AcfFilterType("normal"))
type AcfFilterType string

// ApplyToOptions applies the option to the given options.
func (ft AcfFilterType) ApplyToOptions(opts *ListOptions) {
	opts.filterType = AppConfigFileType(ft)
}

// AcfFilterEnvName usage: .List(..., AcfFilterEnvName("prod")) or .List(..., AcfFilterEnvName(EnvNameFilterAppLevel))
type AcfFilterEnvName string

// ApplyToOptions applies the option to the given options.
func (e AcfFilterEnvName) ApplyToOptions(opts *ListOptions) {
	s := string(e)
	opts.filterEnvName = &s
}

// AcfOrderBy usage: .List(..., AcfOrderBy(ListOrderByName))
type AcfOrderBy string

// ApplyToOptions applies the option to the given options.
func (o AcfOrderBy) ApplyToOptions(opts *ListOptions) {
	opts.orderBy = ListOrderByField(o)
}

// List lists the app config files of the given app.
//
// - appID: The ID of the application
// - orderBy: 排序字段；为空时按 createdAt 升序。
//
// Return a slice of app config files and an error if any.
func (s *AppConfigFileStoreMongo) List(
	ctx context.Context,
	appID string,
	listOpts ...AcfListOption,
) ([]AppConfigFile, error) {
	listOptsObj := &ListOptions{}
	for _, opt := range listOpts {
		opt.ApplyToOptions(listOptsObj)
	}

	filter := bson.M{"appID": appID}
	if listOptsObj.filterType != "" {
		filter["type"] = listOptsObj.filterType
	}
	// Apply envName filter if specified
	if listOptsObj.filterEnvName != nil {
		filter["envName"] = *listOptsObj.filterEnvName
	}

	// name 已迁移到 def 表。为避免把“按 name 排序”错误降级成其他排序，
	// store 层仅保留默认 createdAt 排序；兼容原 name 排序语义由上层补齐 def 后完成。
	opts := options.Find()
	if listOptsObj.orderBy == "" {
		opts.SetSort(bson.D{{Key: "createdAt", Value: 1}})
	} else {
		return nil, errors.New("unsupported orderBy field")
	}

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []AppConfigFile
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Update updates an existed AppConfigFile object, returns the number of modified documents and an error if any.
//
// - acf: The app config file data to update
//
// Return the number of modified documents and an error if any.
func (s *AppConfigFileStoreMongo) Update(ctx context.Context, acf AppConfigFile) (int64, error) {
	return s.updateFieldsWithVersionCheck(ctx, acf, nil)
}

// UpdateIfVersionMatches updates an existing AppConfigFile only when the current version matches expectedVersion.
func (s *AppConfigFileStoreMongo) UpdateIfVersionMatches(
	ctx context.Context,
	acf AppConfigFile,
	expectedVersion int64,
) (int64, error) {
	return s.updateFieldsWithVersionCheck(ctx, acf, &expectedVersion)
}

func (s *AppConfigFileStoreMongo) updateFieldsWithVersionCheck(
	ctx context.Context,
	acf AppConfigFile,
	curVersion *int64,
) (int64, error) {
	// When the given object does not have an ID, return an error
	if acf.ID == bson.NilObjectID {
		return 0, errors.New("object's ID is required for update")
	}

	// Create the update operation with $set
	acf.UpdatedAt = time.Now()
	updateDoc, err := dbutil.ToBsonWithoutID(acf)
	if err != nil {
		return 0, errors.Wrap(err, "convert object to BSON")
	}
	update := bson.M{"$set": updateDoc}
	filter := bson.M{"_id": acf.ID}
	if curVersion != nil {
		filter["currentVersion"] = *curVersion
	}

	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if curVersion != nil && result.MatchedCount == 0 {
		return 0, ErrAppConfigFileVersionConflict
	}
	return result.ModifiedCount, nil
}

// IsReferencedByOther checks if the given app config file ID is referenced by other overlay files.
//
// - id: The ID of the app config file to check
//
// Return whether the file is referenced by other files and an error if any.
func (s *AppConfigFileStoreMongo) IsReferencedByOther(ctx context.Context, id bson.ObjectID) (bool, error) {
	var result bson.M
	filter := bson.M{"baseAppConfigFileID": id}
	opts := options.FindOne().SetProjection(bson.M{"_id": 1})

	err := s.collection.FindOne(ctx, filter, opts).Decode(&result)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

// DeleteByID delete an app config file from the store by ID,
// returns the number of deleted documents and an error if any.
//
//   - appID: The ID of the application
//   - id: The ID of the app config file to delete, if the ID does not belong to the given
//     application, the deletion will not be performed.
//
// Return the number of deleted documents and an error if any.
func (s *AppConfigFileStoreMongo) DeleteByID(
	ctx context.Context, appID string, id bson.ObjectID,
) (int64, error) {
	isRefed, err := s.IsReferencedByOther(ctx, id)
	if err != nil {
		return 0, errors.Wrap(err, "checking if app config file is referenced")
	}
	if isRefed {
		return 0, ErrAppConfigFileReferenced
	}

	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": id, "appID": appID})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// DeleteByApp deletes all app config files for the given app,
// returns the number of deleted documents and an error if any.
//
//   - appID: The ID of the application
//
// Return the number of deleted documents and an error if any.
func (s *AppConfigFileStoreMongo) DeleteByApp(ctx context.Context, appID string) (int64, error) {
	filter := bson.M{"appID": appID}
	result, err := s.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, errors.Wrapf(err, "delete app config files for app [%s]", appID)
	}
	return result.DeletedCount, nil
}

// GetByDefIDAndEnv 按 defID + envName 组合查询文件记录。
func (s *AppConfigFileStoreMongo) GetByDefIDAndEnv(
	ctx context.Context, defID bson.ObjectID, envName string,
) (*AppConfigFile, error) {
	var obj AppConfigFile
	filter := bson.M{"defID": defID, "envName": envName}
	if err := s.collection.FindOne(ctx, filter).Decode(&obj); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.Errorf("app config file for def %s env %q not found", defID.Hex(), envName)
		}
		return nil, err
	}
	return &obj, nil
}

// ListByDefID 返回指定 def 下的所有文件记录。
func (s *AppConfigFileStoreMongo) ListByDefID(ctx context.Context, defID bson.ObjectID) ([]AppConfigFile, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"defID": defID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []AppConfigFile
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteByDefID 删除指定 def 下的所有文件记录。
func (s *AppConfigFileStoreMongo) DeleteByDefID(ctx context.Context, defID bson.ObjectID) (int64, error) {
	result, err := s.collection.DeleteMany(ctx, bson.M{"defID": defID})
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}
