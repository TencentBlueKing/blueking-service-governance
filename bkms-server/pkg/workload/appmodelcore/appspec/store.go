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

package appspec

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const appSpecCollectionName = "app_specs"

// AppSpecStore is the interface for app spec storage.
type AppSpecStore interface {
	// Create creates a new app spec.
	Create(ctx context.Context, spec *AppSpec) error

	// Get gets app spec by appID and envName.
	Get(ctx context.Context, appID, envName string) (*AppSpec, error)

	// ListEnvNamesByApp lists env names that have app spec configured, the default env("") is excluded.
	ListEnvNamesByApp(ctx context.Context, appID string) ([]string, error)

	// Upsert creates or replaces app spec.
	Upsert(ctx context.Context, spec *AppSpec) error

	// Patch updates some fields of app spec, ignoring nil fields in override.
	Patch(ctx context.Context, spec AppSpec) error

	// SetSections replaces only the specified top-level sections, leaving all other sections intact.
	SetSections(ctx context.Context, spec AppSpec, sectionIDs ...AppSpecSectionID) error

	// Delete deletes app spec by appID and envName.
	Delete(ctx context.Context, appID, envName string) error

	// DeleteByApp deletes all app specs by appID.
	DeleteByApp(ctx context.Context, appID string) error
}

var _ AppSpecStore = &AppSpecStoreMongo{}

// AppSpecStoreMongo is the MongoDB implementation of AppSpecStore.
type AppSpecStoreMongo struct {
	collection *mongo.Collection
}

// NewAppSpecStoreMongo creates a new AppSpecStore.
func NewAppSpecStoreMongo(client *mongo.Client, dbName string) (*AppSpecStoreMongo, error) {
	coll := client.Database(dbName).Collection(appSpecCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + envName
	return &AppSpecStoreMongo{collection: coll}, nil
}

// Create validates and inserts a new application specification.
func (s *AppSpecStoreMongo) Create(ctx context.Context, spec *AppSpec) error {
	scoped := cloneSpec(spec)
	if err := validate.Struct(scoped); err != nil {
		return errors.Wrap(err, "app spec validation failed")
	}
	_, err := s.collection.InsertOne(ctx, scoped)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Errorf("app spec for app %s env %q already exists", scoped.AppID, scoped.EnvName)
		}
		return err
	}
	return nil
}

// Get returns the application specification for an application and environment.
func (s *AppSpecStoreMongo) Get(ctx context.Context, appID, envName string) (*AppSpec, error) {
	spec := new(AppSpec)
	err := s.collection.FindOne(ctx, bson.M{"appID": appID, "envName": envName}).Decode(spec)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrAppSpecNotFound
		}
		return nil, err
	}
	return cloneSpec(spec), nil
}

// ListEnvNamesByApp returns non-default environment names with configured sections.
func (s *AppSpecStoreMongo) ListEnvNamesByApp(ctx context.Context, appID string) ([]string, error) {
	filter := bson.M{
		"appID":   appID,
		"envName": bson.M{"$ne": DefaultEnvName},
	}
	cursor, err := s.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "envName", Value: 1}}))
	if err != nil {
		return nil, errors.Wrap(err, "listing app specs for env names")
	}
	defer cursor.Close(ctx)

	envNames := make([]string, 0)
	for cursor.Next(ctx) {
		spec := new(AppSpec)
		if err := cursor.Decode(spec); err != nil {
			return nil, errors.Wrap(err, "decoding app spec")
		}
		if !spec.HasConfiguredSections() {
			continue
		}
		envNames = append(envNames, spec.EnvName)
	}
	if err := cursor.Err(); err != nil {
		return nil, errors.Wrap(err, "iterating app specs")
	}
	sort.Strings(envNames)
	return envNames, nil
}

// Upsert validates and replaces an application specification, creating it when absent.
func (s *AppSpecStoreMongo) Upsert(ctx context.Context, spec *AppSpec) error {
	scoped := cloneSpec(spec)
	if err := validate.Struct(scoped); err != nil {
		return errors.Wrap(err, "app spec validation failed")
	}

	filter := bson.M{"appID": scoped.AppID, "envName": scoped.EnvName}
	opts := options.Replace().SetUpsert(true)
	_, err := s.collection.ReplaceOne(ctx, filter, scoped, opts)
	return err
}

// Patch updates the non-nil fields of an application specification.
func (s *AppSpecStoreMongo) Patch(ctx context.Context, spec AppSpec) error {
	scoped := cloneSpec(&spec)
	if err := validate.Struct(scoped); err != nil {
		return errors.Wrap(err, "app spec validation failed")
	}

	appID, envName := scoped.AppID, scoped.EnvName

	// Build the $set document by appending non-nil fields from each section.
	set := bson.D{}
	for _, section := range registeredSections {
		section.appendPatch(&set, scoped)
	}
	if len(set) == 0 {
		return nil
	}

	update := bson.D{
		{Key: "$set", Value: set},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "appID", Value: appID},
			{Key: "envName", Value: envName},
		}},
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err := s.collection.UpdateOne(ctx, bson.M{"appID": appID, "envName": envName}, update, opts)
	return err
}

// SetSections replaces or removes the specified top-level application specification sections.
func (s *AppSpecStoreMongo) SetSections(ctx context.Context, spec AppSpec, sectionIDs ...AppSpecSectionID) error {
	scoped := cloneSpec(&spec)
	if err := validate.Struct(scoped); err != nil {
		return errors.Wrap(err, "app spec validation failed")
	}

	set := bson.D{}
	unset := bson.D{}
	for _, sectionID := range sectionIDs {
		section, ok := getSection(sectionID)
		if !ok {
			return errors.Errorf("unknown app spec section %q", sectionID)
		}
		section.appendWholeUpdate(&set, &unset, scoped)
	}
	if len(set) == 0 && len(unset) == 0 {
		return nil
	}

	update := bson.D{}
	upsert := len(set) > 0
	if len(set) > 0 {
		update = append(update, bson.E{Key: "$set", Value: set})
	}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}
	if upsert {
		update = append(update, bson.E{Key: "$setOnInsert", Value: bson.D{
			{Key: "appID", Value: scoped.AppID},
			{Key: "envName", Value: scoped.EnvName},
		}})
	}

	opts := options.UpdateOne().SetUpsert(upsert)
	_, err := s.collection.UpdateOne(ctx, bson.M{"appID": scoped.AppID, "envName": scoped.EnvName}, update, opts)
	return err
}

// Delete removes the application specification for an application and environment.
func (s *AppSpecStoreMongo) Delete(ctx context.Context, appID, envName string) error {
	ret, err := s.collection.DeleteOne(ctx, bson.M{"appID": appID, "envName": envName})
	if err != nil {
		return err
	}
	if ret.DeletedCount == 0 {
		return ErrAppSpecNotFound
	}
	return nil
}

// DeleteByApp deletes all app specs by appID.
func (s *AppSpecStoreMongo) DeleteByApp(ctx context.Context, appID string) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{"appID": appID})
	return err
}
