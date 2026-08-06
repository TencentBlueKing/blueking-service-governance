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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// apmInstConfigCollectionName is the MongoDB collection name for storing APM instance config data.
const apmInstConfigCollectionName = "bkmonitor_apm_inst_config"

// ErrApmInstConfigNotFound is returned when an APM instance config record is not found.
var ErrApmInstConfigNotFound = errors.New("ApmInstConfig not found")

// ApmInstConfigStoreMongo implements ApmInstConfigStore interface with MongoDB.
var _ ApmInstConfigStore = &ApmInstConfigStoreMongo{}

// ApmInstConfigStore defines the storage interface for APM instance config data.
type ApmInstConfigStore interface {
	// Create creates a new APM instance config record.
	Create(ctx context.Context, apm *ApmInstConfig) (bson.ObjectID, error)

	// List returns APM instance config records by workspace ID.
	List(ctx context.Context, workspaceID string) ([]ApmInstConfig, error)

	// GetApmIDMap returns APM instance config records by workspace ID as a map keyed by ApmID.
	GetApmIDMap(ctx context.Context, workspaceID string) (map[int64]ApmInstConfig, error)

	// GetByApmID returns an APM instance config record by its APM ID.
	GetByApmID(ctx context.Context, apmID int64) (*ApmInstConfig, error)

	// GetByEnvID returns an APM instance config record by associated environment ID.
	GetByEnvID(ctx context.Context, envID bson.ObjectID) (*ApmInstConfig, error)

	// GetByName returns an APM instance config record by name and workspace ID.
	GetByName(ctx context.Context, workspaceID, name string) (*ApmInstConfig, error)

	// UnbindEnvFromAll unbinds the specified environment from all APM instance config records.
	UnbindEnvFromAll(ctx context.Context, envID bson.ObjectID) error

	// Update updates an APM instance config record.
	Update(ctx context.Context, id bson.ObjectID, updateData *ApmInstConfigUpdateData) error

	// UnbindEnv unbinds the specified environment from an APM instance config record.
	UnbindEnv(ctx context.Context, id, envID bson.ObjectID) error

	// BindEnv binds an environment to an APM instance config record.
	BindEnv(ctx context.Context, id, envID bson.ObjectID, envName string) error

	// Delete deletes an APM instance config record by ID.
	Delete(ctx context.Context, id bson.ObjectID) error
}

// ApmInstConfigStoreMongo implements ApmInstConfigStore interface with MongoDB.
type ApmInstConfigStoreMongo struct {
	collection *mongo.Collection
}

// NewApmInstConfigStoreMongo creates a new ApmInstConfigStore instance.
func NewApmInstConfigStoreMongo(client *mongo.Client, dbName string) (ApmInstConfigStore, error) {
	coll := client.Database(dbName).Collection(apmInstConfigCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：apmID
	return &ApmInstConfigStoreMongo{collection: coll}, nil
}

// Create creates a new APM instance config record.
func (s *ApmInstConfigStoreMongo) Create(ctx context.Context, apm *ApmInstConfig) (bson.ObjectID, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(apm); err != nil {
		return bson.NilObjectID, errors.Wrap(err, "ApmInstConfig validation failed")
	}

	if apm.CreatedAt.IsZero() {
		apm.CreatedAt = time.Now()
	}
	apm.UpdatedAt = apm.CreatedAt

	if apm.AssociatedEnvs == nil {
		apm.AssociatedEnvs = make([]EnvInfo, 0)
	}

	ret, err := s.collection.InsertOne(ctx, apm)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, errors.Errorf("ApmInstConfig with ID %d already exists", apm.ApmID)
		}
		return bson.NilObjectID, err
	}
	return ret.InsertedID.(bson.ObjectID), nil
}

// List returns APM instance config records by workspace ID.
func (s *ApmInstConfigStoreMongo) List(ctx context.Context, workspaceID string) ([]ApmInstConfig, error) {
	filter := bson.M{"workspaceID": workspaceID}
	sort := bson.D{{Key: "createdAt", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx) // nolint

	apmList := make([]ApmInstConfig, 0)
	if err = cursor.All(ctx, &apmList); err != nil {
		return nil, err
	}

	return apmList, nil
}

// GetApmIDMap returns APM instance config records by workspace ID as a map keyed by ApmID.
func (s *ApmInstConfigStoreMongo) GetApmIDMap(
	ctx context.Context,
	workspaceID string,
) (map[int64]ApmInstConfig, error) {
	list, err := s.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	apmMap := lo.SliceToMap(list, func(item ApmInstConfig) (int64, ApmInstConfig) {
		return item.ApmID, item
	})

	return apmMap, nil
}

// GetByApmID returns an APM instance config record by its APM ID.
func (s *ApmInstConfigStoreMongo) GetByApmID(ctx context.Context, apmID int64) (*ApmInstConfig, error) {
	return s.findOne(ctx, bson.M{"apmID": apmID})
}

// GetByEnvID returns an APM instance config record by associated environment ID.
func (s *ApmInstConfigStoreMongo) GetByEnvID(ctx context.Context, envID bson.ObjectID) (*ApmInstConfig, error) {
	// Query for APM records whose associatedEnvs array contains the specified envID.
	filter := bson.M{
		"associatedEnvs": bson.M{
			"$elemMatch": bson.M{
				"envID": envID,
			},
		},
	}

	return s.findOne(ctx, filter)
}

// GetByName returns an APM instance config record by name and workspace ID.
func (s *ApmInstConfigStoreMongo) GetByName(
	ctx context.Context,
	workspaceID, name string,
) (*ApmInstConfig, error) {
	return s.findOne(ctx, bson.M{"workspaceID": workspaceID, "name": name})
}

// Update updates an APM instance config record.
func (s *ApmInstConfigStoreMongo) Update(
	ctx context.Context,
	id bson.ObjectID,
	updateData *ApmInstConfigUpdateData,
) error {
	if updateData == nil {
		return nil
	}

	filter := bson.M{"_id": id}
	update, isEmpty := updateData.ToBSON()

	if !isEmpty {
		update["updatedAt"] = time.Now()
		return s.updateOne(ctx, filter, bson.M{"$set": update})
	}

	return nil
}

// Delete deletes an APM instance config record by ID.
func (s *ApmInstConfigStoreMongo) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// BindEnv binds an environment to an APM instance config record.
func (s *ApmInstConfigStoreMongo) BindEnv(ctx context.Context, id, envID bson.ObjectID, envName string) error {
	filter := bson.M{"_id": id}

	envInfo := EnvInfo{
		EnvID:   envID,
		EnvName: envName,
	}

	// Use $addToSet to add the element while ensuring uniqueness.
	update := bson.M{
		"$addToSet": bson.M{
			"associatedEnvs": envInfo,
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	return s.updateOne(ctx, filter, update)
}

// UnbindEnv unbinds the specified environment from an APM instance config record.
func (s *ApmInstConfigStoreMongo) UnbindEnv(ctx context.Context, id, envID bson.ObjectID) error {
	filter := bson.M{"_id": id}

	// Use $pull to remove the element.
	update := bson.M{
		"$pull": bson.M{
			"associatedEnvs": bson.M{"envID": envID},
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	return s.updateOne(ctx, filter, update)
}

// UnbindEnvFromAll unbinds the specified environment from all APM instance config records.
func (s *ApmInstConfigStoreMongo) UnbindEnvFromAll(ctx context.Context, envID bson.ObjectID) error {
	filter := bson.M{
		"associatedEnvs": bson.M{
			"$elemMatch": bson.M{
				"envID": envID,
			},
		},
	}

	update := bson.M{
		"$pull": bson.M{
			"associatedEnvs": bson.M{"envID": envID},
		},
		"$set": bson.M{
			"updatedAt": time.Now(),
		},
	}

	_, err := s.collection.UpdateMany(ctx, filter, update)
	return err
}

// ToBSON converts ApmInstConfigUpdateData to bson.M for update operations.
func (d *ApmInstConfigUpdateData) ToBSON() (bson.M, bool) {
	data := bson.M{}
	isEmpty := true

	if d.WorkspaceID != nil {
		data["workspaceID"] = *d.WorkspaceID
		isEmpty = false
	}
	if d.ApmID != nil {
		data["apmID"] = *d.ApmID
		isEmpty = false
	}
	if d.Name != nil {
		data["name"] = *d.Name
		isEmpty = false
	}
	if d.Token != nil {
		data["token"] = *d.Token
		isEmpty = false
	}

	return data, isEmpty
}

// findOne returns a single APM instance config record matching the given filter.
func (s *ApmInstConfigStoreMongo) findOne(ctx context.Context, filter bson.M) (*ApmInstConfig, error) {
	apm := new(ApmInstConfig)
	if err := s.collection.FindOne(ctx, filter).Decode(apm); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrApmInstConfigNotFound
		}
		return nil, err
	}
	return apm, nil
}

// updateOne updates a single APM instance config record matching the given filter.
func (s *ApmInstConfigStoreMongo) updateOne(ctx context.Context, filter, update bson.M) error {
	opts := options.UpdateOne().SetUpsert(false)
	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return ErrApmInstConfigNotFound
	}
	return err
}
