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

// Package app storage module for the arrangement package, it helps with data persistence.
package app

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
)

// The name of the MongoDB collection for storing application data.
const collectionName = "applications"

// use a single instance of Validate, it caches struct info
var validate *validator.Validate

var ErrAppNotFound = errors.New("application not found")

// ListOpts defines the options for listing applications
type ListOpts struct {
	// WorkspaceID filters applications by workspace ID
	WorkspaceID string
	// AppName filters applications by name
	AppName string
	// AppType filters applications by type (e.g., "trpc", "helm")
	AppType string
}

// ApplicationStore is the interface for storing application data
type ApplicationStore interface {
	// ListApps lists applications based on the provided options
	ListApps(ctx context.Context, opts *ListOpts) ([]*Application, error)

	// CountByWorkspaceIDs returns application counts grouped by workspace ID.
	CountByWorkspaceIDs(ctx context.Context, workspaceIDs []string) (map[string]int, error)

	// CreateApp creates a new application
	CreateApp(ctx context.Context, app *Application) error

	// GetApp gets an application by id
	GetApp(ctx context.Context, id string) (*Application, error)

	// GetAppsByIDs gets applications by id list in one query.
	// The returned slice is in the same order as ids.
	// If any requested id is missing from storage, returns an error wrapping ErrAppNotFound.
	GetAppsByIDs(ctx context.Context, ids []string) ([]*Application, error)

	// GetAppByName gets an application by name
	GetAppByName(ctx context.Context, workspaceID, name string) (*Application, error)

	// DeleteAppByName delete an application by name
	DeleteAppByName(ctx context.Context, workspaceID, name string) error

	// UpdateDisplayName updates the display name of an application
	UpdateDisplayName(ctx context.Context, app *Application, displayName string) error

	// UpdateHelmSource updates the helm source of an application
	UpdateHelmSource(ctx context.Context, app *Application, helmSource *HelmSource) error
}

var _ ApplicationStore = &ApplicationStoreMongo{}

// ApplicationStoreMongo is the implementation of ApplicationStore
type ApplicationStoreMongo struct {
	// Collection is the MongoDB collection for storing arrangement data.
	collection *mongo.Collection
}

// NewApplicationStoreMongo creates a new ApplicationStore, it accepts a MongoDB client and database name.
func NewApplicationStoreMongo(client *mongo.Client, dbName string) (*ApplicationStoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：id
	// - 唯一：workspaceID + name
	return &ApplicationStoreMongo{collection: coll}, nil
}

// CreateApp creates a new application
//
// - ctx: The context object for cancellation and timeout
// - app: The application to create
//
// Return an error if the operation fails.
func (s *ApplicationStoreMongo) CreateApp(ctx context.Context, app *Application) error {
	// 入库前对敏感字段进行加密
	if err := s.handleSensitiveFields(app, crypto.AESEncrypt); err != nil {
		return errors.Wrap(err, "encrypt fields")
	}
	// Validate the application struct to ensure required fields are present
	validate = validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(app); err != nil {
		return errors.Wrap(err, "application validation failed")
	}

	_, err := s.collection.InsertOne(ctx, app)
	if err != nil {
		// Check if it's a duplicate key error
		if mongo.IsDuplicateKeyError(err) {
			dupKey := lo.Ternary(strings.Contains(err.Error(), "id_1 dup key"), "id", "name")
			return errors.Errorf("application with the same %s already exists", dupKey)
		}
		return err
	}
	return nil
}

// GetApp gets an application by ID
func (s *ApplicationStoreMongo) GetApp(ctx context.Context, id string) (*Application, error) {
	app := new(Application)
	err := s.collection.FindOne(ctx, bson.M{"id": id}).Decode(app)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// When no record can be found, return a new error
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	// 出库前对敏感字段进行解密
	if err = s.handleSensitiveFields(app, crypto.AESDecrypt); err != nil {
		return nil, errors.Wrap(err, "decrypt fields")
	}

	return app, nil
}

// GetAppsByIDs gets applications by id list in one query.
func (s *ApplicationStoreMongo) GetAppsByIDs(ctx context.Context, ids []string) ([]*Application, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	uniqueIDs := lo.Uniq(ids)
	cursor, err := s.collection.Find(ctx, bson.M{"id": bson.M{"$in": uniqueIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	byID := make(map[string]*Application, len(uniqueIDs))
	for cursor.Next(ctx) {
		app := new(Application)
		if err = cursor.Decode(app); err != nil {
			return nil, err
		}
		if err = s.handleSensitiveFields(app, crypto.AESDecrypt); err != nil {
			return nil, errors.Wrap(err, "decrypt fields")
		}
		byID[app.ID] = app
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	out := make([]*Application, 0, len(ids))
	for _, id := range ids {
		a, ok := byID[id]
		if !ok {
			return nil, errors.Wrapf(ErrAppNotFound, "get app %s", id)
		}
		out = append(out, a)
	}
	return out, nil
}

// GetAppByName gets an application by name
// Deprecated: use GetApp instead (only appID, no workspaceID + appName)
func (s *ApplicationStoreMongo) GetAppByName(ctx context.Context, workspaceID, name string) (*Application, error) {
	app := new(Application)
	err := s.collection.FindOne(ctx, bson.M{"name": name, "workspaceID": workspaceID}).
		Decode(app)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// When no record can be found, return a new error
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	// 出库前对敏感字段进行解密
	if err = s.handleSensitiveFields(app, crypto.AESDecrypt); err != nil {
		return nil, errors.Wrap(err, "decrypt fields")
	}

	return app, nil
}

// DeleteAppByName delete an application by name
func (s *ApplicationStoreMongo) DeleteAppByName(ctx context.Context, workspaceID, name string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"workspaceID": workspaceID, "name": name})
	return err
}

// UpdateDisplayName updates the display name of an application
func (s *ApplicationStoreMongo) UpdateDisplayName(
	ctx context.Context,
	app *Application,
	displayName string,
) error {
	filter := bson.M{"id": app.ID}
	update := bson.M{"$set": bson.M{"displayName": displayName}}
	return s.collUpdateOne(ctx, filter, update)
}

// UpdateHelmSource updates the helm source of an application
func (s *ApplicationStoreMongo) UpdateHelmSource(
	ctx context.Context,
	app *Application,
	helmSource *HelmSource,
) error {
	newApp := Application{
		Name:        app.Name,
		WorkspaceID: app.WorkspaceID,
		HelmSpec:    &HelmSpec{HelmSource: helmSource},
	}
	// 入库前对敏感字段进行加密
	if err := s.handleSensitiveFields(&newApp, crypto.AESEncrypt); err != nil {
		return errors.Wrap(err, "encrypt fields")
	}

	filter := bson.M{"name": app.Name, "workspaceID": app.WorkspaceID}
	update := bson.M{
		"$set": bson.M{"helmSpec.helmSource": newApp.HelmSpec.HelmSource},
	}
	return s.collUpdateOne(ctx, filter, update)
}

// ListApps lists applications based on the provided options
//
// - opts: The options for filtering applications. If nil, returns all applications.
//   - WorkspaceID: If non-empty, filter applications by workspace ID
//   - AppName: If non-empty, filter applications by name
//   - AppType: If non-empty, filter applications by type
func (s *ApplicationStoreMongo) ListApps(ctx context.Context, opts *ListOpts) ([]*Application, error) {
	// Build filter based on options
	filter := bson.M{}
	if opts != nil {
		if opts.WorkspaceID != "" {
			filter["workspaceID"] = opts.WorkspaceID
		}
		if opts.AppName != "" {
			filter["name"] = opts.AppName
		}
		if opts.AppType != "" {
			filter["type"] = opts.AppType
		}
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var apps []*Application
	for cursor.Next(ctx) {
		app := new(Application)
		if err = cursor.Decode(app); err != nil {
			return nil, err
		}
		// 出库前对敏感字段进行解密
		if err = s.handleSensitiveFields(app, crypto.AESDecrypt); err != nil {
			return nil, errors.Wrap(err, "decrypt fields")
		}

		apps = append(apps, app)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return apps, nil
}

type appCountByWorkspace struct {
	WorkspaceID string `bson:"_id"`
	Count       int    `bson:"count"`
}

// CountByWorkspaceIDs returns application counts grouped by workspace ID.
func (s *ApplicationStoreMongo) CountByWorkspaceIDs(
	ctx context.Context,
	workspaceIDs []string,
) (map[string]int, error) {
	if len(workspaceIDs) == 0 {
		return map[string]int{}, nil
	}

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"workspaceID": bson.M{"$in": lo.Uniq(workspaceIDs)}}}},
		{{"$group", bson.M{
			"_id":   "$workspaceID",
			"count": bson.M{"$sum": 1},
		}}},
	}
	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "aggregate application counts by workspace IDs")
	}
	defer cursor.Close(ctx)

	var results []appCountByWorkspace
	if err := cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "decode application counts by workspace IDs")
	}

	counts := make(map[string]int, len(results))
	for _, item := range results {
		counts[item.WorkspaceID] = item.Count
	}
	return counts, nil
}

// handleSensitiveFields encrypts or decrypts sensitive fields in the Application
// - handleFunc is a function that encrypts or decrypts a field
func (s *ApplicationStoreMongo) handleSensitiveFields(
	app *Application, handleFunc func(key, data string) (string, error),
) error {
	if app.HelmSpec == nil || app.HelmSpec.HelmSource == nil {
		return nil
	}

	if app.HelmSpec.HelmSource.HelmRepoConfig == nil {
		return nil
	}

	rawPassword := app.HelmSpec.HelmSource.HelmRepoConfig.Password
	if rawPassword == "" {
		return nil
	}

	password, err := handleFunc(config.G.Encrypt.Secret, rawPassword)
	if err != nil {
		return err
	}

	app.HelmSpec.HelmSource.HelmRepoConfig.Password = password

	return nil
}

func (s *ApplicationStoreMongo) collUpdateOne(ctx context.Context, filter, update bson.M) error {
	opts := options.UpdateOne().SetUpsert(false)
	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return ErrAppNotFound
	}
	return err
}
