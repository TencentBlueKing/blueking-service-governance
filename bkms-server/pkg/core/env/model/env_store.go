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

package model

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// The name of the MongoDB collection for storing environment data.
const environmentCollectionName = "environments"

// ErrEnvNotFound represents the error of environment not found.
var ErrEnvNotFound = errors.New("environment not found")

// EnvironmentUpdateData represents the fields that can be updated for an environment.
type EnvironmentUpdateData struct {
	Type        *string
	DisplayName *string
	Description *string
	// ClusterID, ClusterType, Namespace 是集群相关信息
	ClusterID   *string
	ClusterType *string
	Namespace   *string
	// IsFederation 是否为联邦集群
	IsFederation *bool
}

// ToBSON converts EnvironmentUpdateData to bson.M for update
func (d *EnvironmentUpdateData) ToBSON() (bson.M, bool) {
	data := bson.M{}

	isEmpty := true

	setIfNotNil := func(field *string, bsonField string) {
		if field != nil {
			data[bsonField] = *field
			isEmpty = false
		}
	}

	setIfNotNil(d.Type, "type")
	setIfNotNil(d.DisplayName, "displayName")
	setIfNotNil(d.Description, "description")
	setIfNotNil(d.ClusterID, "cluster.clusterID")
	setIfNotNil(d.ClusterType, "cluster.clusterType")
	setIfNotNil(d.Namespace, "cluster.namespace")
	if d.IsFederation != nil {
		data["cluster.isFederation"] = *d.IsFederation
		isEmpty = false
	}

	return data, isEmpty
}

// EnvironmentStore stores environment data.
type EnvironmentStore interface {
	// General mutations and direct lookups
	// -------------------------------------------------------------------------

	// Create creates a new environment.
	// Return the environment id if the creation succeeds.
	Create(ctx context.Context, env *Environment) (bson.ObjectID, error)

	// Get gets an environment by environment id.
	Get(ctx context.Context, envID bson.ObjectID) (*Environment, error)
	// Update updates an existing environment.
	Update(ctx context.Context, envID bson.ObjectID, updateData *EnvironmentUpdateData) error
	// Delete deletes an environment by environment id.
	Delete(ctx context.Context, envID bson.ObjectID) error

	// AddApp adds an app to an environment.
	// 当应用部署到当前环境时, 会调用此方法
	AddApp(ctx context.Context, envID bson.ObjectID, appID string) error
	// RemoveApp deletes an app from an environment.
	// 当应用从当前环境删除时, 会调用此方法
	RemoveApp(ctx context.Context, envID bson.ObjectID, appID string) error

	// DeleteAll deletes all environments while preserving the collection and its indexes.
	// Attention: only used in unit test
	DeleteAll(ctx context.Context) error

	// -------------------------------------------------------------------------
	// Standard-environment-only queries
	// -------------------------------------------------------------------------

	// ListStdEnvs lists all standard environments in a workspace.
	ListStdEnvs(ctx context.Context, workspaceID string) ([]Environment, error)
	// GetStdEnvByName gets a standard environment by workspace id and name.
	GetStdEnvByName(ctx context.Context, workspaceID, name string) (*Environment, error)
	// CountByWorkspaceIDs returns environment counts grouped by workspace ID.
	CountByWorkspaceIDs(ctx context.Context, workspaceIDs []string) (map[string]int, error)
	// GetEnvTypeMap queries standard environments in a workspace and returns envName→envType mapping.
	GetEnvTypeMap(ctx context.Context, workspaceID string) (map[string]string, error)

	// -------------------------------------------------------------------------
	// App-aware queries (may include feature environments)
	// -------------------------------------------------------------------------

	// ListAppEnvs lists all environments available to an app.
	ListAppEnvs(ctx context.Context, workspaceID, appID string) ([]Environment, error)
	// ListAppEnvsByIDs lists specified environments available to an app.
	ListAppEnvsByIDs(ctx context.Context, workspaceID, appID string, envIDs []bson.ObjectID) ([]Environment, error)
	// ListAppEnvsByTypes lists environments with specified types available to an app.
	ListAppEnvsByTypes(ctx context.Context, workspaceID, appID string, envTypes []string) ([]Environment, error)
	// ListBatchAppEnvs lists all environments available to a batch of apps.
	//
	// The result consists of standard environments and feature environments that belong to the
	// given applications. No specific ordering is guaranteed.
	ListBatchAppEnvs(ctx context.Context, workspaceID string, appIDs []string) ([]Environment, error)
	// ListAppFeatEnvs lists all feature environments owned by an app.
	ListAppFeatEnvs(ctx context.Context, appID string) ([]Environment, error)
	// GetByName gets an environment by workspace id and name, allowing the caller to
	// read either a standard environment or a feature environment owned by appID.
	GetByName(ctx context.Context, workspaceID, appID, name string) (*Environment, error)
}

var _ EnvironmentStore = &EnvironmentStoreMongo{}

// EnvironmentStoreMongo implements EnvironmentStore interface with mongodb
type EnvironmentStoreMongo struct {
	collection *mongo.Collection
}

// NewEnvironmentStoreMongo creates a new EnvironmentStoreMongo
func NewEnvironmentStoreMongo(client *mongo.Client, dbName string) (EnvironmentStore, error) {
	coll := client.Database(dbName).Collection(environmentCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID + name
	// - 查询提速：ownerAppID
	return &EnvironmentStoreMongo{collection: coll}, nil
}

// Create creates a new environment.
// Return the environment id if the creation succeeds.
func (s *EnvironmentStoreMongo) Create(ctx context.Context, env *Environment) (bson.ObjectID, error) {
	if env.CreatedAt.IsZero() {
		env.CreatedAt = time.Now()
	}
	env.UpdatedAt = env.CreatedAt
	if env.Kind == "" {
		env.Kind = EnvironmentKindStandard
	}

	if env.AppIDs == nil {
		env.AppIDs = make([]string, 0)
	}

	ret, err := s.collection.InsertOne(ctx, env)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, errors.Errorf("environment %s already exists", env.Name)
		}
		return bson.NilObjectID, err
	}
	return ret.InsertedID.(bson.ObjectID), nil
}

// -----------------------------------------------------------------------------
// Standard-environment-only queries
// -----------------------------------------------------------------------------
//
// NOTE: These queries match kind == "standard" only. Before rolling out feature
// environments, existing documents must be backfilled with kind=standard so
// historical records (which omit kind) remain visible.

// ListStdEnvs lists all standard environments in a workspace.
func (s *EnvironmentStoreMongo) ListStdEnvs(ctx context.Context, workspaceID string) ([]Environment, error) {
	filter := bson.M{
		"workspaceID": workspaceID,
		"kind":        EnvironmentKindStandard,
	}

	return s.listByFilter(ctx, filter)
}

// GetStdEnvByName gets a standard environment by workspace id and name.
func (s *EnvironmentStoreMongo) GetStdEnvByName(
	ctx context.Context,
	workspaceID, name string,
) (*Environment, error) {
	return s.findOne(ctx, bson.M{
		"workspaceID": workspaceID,
		"name":        name,
		"kind":        EnvironmentKindStandard,
	})
}

type envCountByWorkspace struct {
	WorkspaceID string `bson:"_id"`
	Count       int    `bson:"count"`
}

// CountByWorkspaceIDs returns environment counts grouped by workspace ID.
func (s *EnvironmentStoreMongo) CountByWorkspaceIDs(
	ctx context.Context,
	workspaceIDs []string,
) (map[string]int, error) {
	if len(workspaceIDs) == 0 {
		return map[string]int{}, nil
	}

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"workspaceID": bson.M{"$in": lo.Uniq(workspaceIDs)},
			"kind":        EnvironmentKindStandard,
		}}},
		{{"$group", bson.M{
			"_id":   "$workspaceID",
			"count": bson.M{"$sum": 1},
		}}},
	}
	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "aggregate environment counts by workspace IDs")
	}
	defer cursor.Close(ctx)

	var results []envCountByWorkspace
	if err := cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "decode environment counts by workspace IDs")
	}

	counts := make(map[string]int, len(results))
	for _, item := range results {
		counts[item.WorkspaceID] = item.Count
	}
	return counts, nil
}

// GetEnvTypeMap 查询指定空间下的标准环境列表，返回 envName→envType 映射
func (s *EnvironmentStoreMongo) GetEnvTypeMap(
	ctx context.Context, workspaceID string,
) (map[string]string, error) {
	envs, err := s.ListStdEnvs(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	envTypeMap := make(map[string]string, len(envs))
	for _, env := range envs {
		envTypeMap[env.Name] = env.Type
	}
	return envTypeMap, nil
}

// -----------------------------------------------------------------------------
// Queries that may include feature environments
// -----------------------------------------------------------------------------

// ListAppEnvs lists all environments available to an app.
func (s *EnvironmentStoreMongo) ListAppEnvs(ctx context.Context, workspaceID, appID string) ([]Environment, error) {
	return s.ListBatchAppEnvs(ctx, workspaceID, []string{appID})
}

// ListAppEnvsByIDs lists specified environments available to an app.
func (s *EnvironmentStoreMongo) ListAppEnvsByIDs(
	ctx context.Context,
	workspaceID, appID string,
	envIDs []bson.ObjectID,
) ([]Environment, error) {
	if len(envIDs) == 0 {
		return []Environment{}, nil
	}

	filter := bson.M{
		"workspaceID": workspaceID,
		"_id":         bson.M{"$in": lo.Uniq(envIDs)},
		"$or": bson.A{
			bson.M{
				"kind":   EnvironmentKindStandard,
				"appIDs": appID,
			},
			bson.M{
				"kind":       EnvironmentKindFeature,
				"ownerAppID": appID,
			},
		},
	}

	return s.listByFilter(ctx, filter)
}

// ListAppEnvsByTypes lists environments with specified types available to an app.
func (s *EnvironmentStoreMongo) ListAppEnvsByTypes(
	ctx context.Context,
	workspaceID, appID string,
	envTypes []string,
) ([]Environment, error) {
	if len(envTypes) == 0 {
		return []Environment{}, nil
	}

	filter := bson.M{
		"workspaceID": workspaceID,
		"type":        bson.M{"$in": lo.Uniq(envTypes)},
		"$or": bson.A{
			bson.M{
				"kind":   EnvironmentKindStandard,
				"appIDs": appID,
			},
			bson.M{
				"kind":       EnvironmentKindFeature,
				"ownerAppID": appID,
			},
		},
	}

	return s.listByFilter(ctx, filter)
}

// ListBatchAppEnvs lists all environments available to a batch of apps.
func (s *EnvironmentStoreMongo) ListBatchAppEnvs(
	ctx context.Context,
	workspaceID string,
	appIDs []string,
) ([]Environment, error) {
	if len(appIDs) == 0 {
		return []Environment{}, nil
	}

	filter := bson.M{
		"workspaceID": workspaceID,
		"$or": bson.A{
			bson.M{"kind": EnvironmentKindStandard},
			bson.M{
				"kind":       EnvironmentKindFeature,
				"ownerAppID": bson.M{"$in": lo.Uniq(appIDs)},
			},
		},
	}

	return s.listByFilter(ctx, filter)
}

// ListAppFeatEnvs lists all feature environments owned by an app.
func (s *EnvironmentStoreMongo) ListAppFeatEnvs(ctx context.Context, appID string) ([]Environment, error) {
	return s.listByFilter(ctx, bson.M{
		"kind":       EnvironmentKindFeature,
		"ownerAppID": appID,
	})
}

// Get gets an environment by environment id.
func (s *EnvironmentStoreMongo) Get(ctx context.Context, envID bson.ObjectID) (*Environment, error) {
	env, err := s.findOne(ctx, bson.M{"_id": envID})
	if err != nil {
		return nil, err
	}

	env.Status = getEnvStatusByCluster(env.Cluster)
	return env, nil
}

// GetByName gets an environment by workspace id and name.
func (s *EnvironmentStoreMongo) GetByName(
	ctx context.Context, workspaceID, appID, name string,
) (*Environment, error) {
	filter := bson.M{
		"workspaceID": workspaceID,
		"name":        name,
	}
	filter["$or"] = bson.A{
		bson.M{"kind": EnvironmentKindStandard},
		bson.M{
			"kind":       EnvironmentKindFeature,
			"ownerAppID": appID,
		},
	}
	return s.findOne(ctx, filter)
}

// Update updates an existing environment.
func (s *EnvironmentStoreMongo) Update(
	ctx context.Context,
	envID bson.ObjectID,
	updateData *EnvironmentUpdateData,
) error {
	filter := bson.M{"_id": envID}

	update, isEmpty := updateData.ToBSON()

	if !isEmpty {
		update["updatedAt"] = time.Now()
		return s.collUpdateOne(ctx, filter, bson.M{"$set": update})
	}
	return nil
}

// Delete deletes an environment by environment id.
func (s *EnvironmentStoreMongo) Delete(ctx context.Context, envID bson.ObjectID) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": envID})
	return err
}

// AddApp adds an app to an environment.
// 当应用部署到当前环境时, 会调用此方法
func (s *EnvironmentStoreMongo) AddApp(ctx context.Context, envID bson.ObjectID, appID string) error {
	filter := bson.M{"_id": envID}

	// 添加操作：使用 $addToSet 来添加元素并保证唯一性
	update := bson.M{
		"$addToSet": bson.M{
			"appIDs": appID,
		},
	}

	return s.collUpdateOne(ctx, filter, update)
}

// RemoveApp removes an app from an environment.
// 当应用从当前环境删除时, 会调用此方法
func (s *EnvironmentStoreMongo) RemoveApp(ctx context.Context, envID bson.ObjectID, appID string) error {
	filter := bson.M{"_id": envID}

	// 删除操作：使用 $pull 来删除元素
	update := bson.M{
		"$pull": bson.M{
			"appIDs": appID,
		},
	}
	return s.collUpdateOne(ctx, filter, update)
}

// DeleteAll deletes all environments while preserving the collection and its indexes.
// Attention: only used in unit test
func (s *EnvironmentStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}

func (s *EnvironmentStoreMongo) listByFilter(ctx context.Context, filter bson.M) ([]Environment, error) {
	sort := bson.D{{Key: "createdAt", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx) // nolint

	envs := make([]Environment, 0)
	if err = cursor.All(ctx, &envs); err != nil {
		return nil, err
	}

	return lo.Map(envs, func(env Environment, _ int) Environment {
		env.Status = getEnvStatusByCluster(env.Cluster)
		return env
	}), nil
}

// findOne gets an environment with filter
func (s *EnvironmentStoreMongo) findOne(ctx context.Context, filter bson.M) (*Environment, error) {
	env := new(Environment)
	if err := s.collection.FindOne(ctx, filter).Decode(env); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrEnvNotFound
		}
		return nil, err
	}
	return env, nil
}

func (s *EnvironmentStoreMongo) collUpdateOne(ctx context.Context, filter, update bson.M) error {
	opts := options.UpdateOne().SetUpsert(false)
	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return ErrEnvNotFound
	}
	return err
}

func getEnvStatusByCluster(cluster BizCluster) EnvStatus {
	if cluster.ClusterID != "" && cluster.Namespace != "" {
		return EnvStatusReady
	}
	return EnvStatusNotReady
}
