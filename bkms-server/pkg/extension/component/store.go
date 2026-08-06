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

// Package component storage module for the component package, it helps with component data persistence.
package component

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// The name of the MongoDB collection for storing component defs.
const collectionName = "component_defs"

// ErrComponentDefNotFound is returned when a component-def is not found in the store.
var ErrComponentDefNotFound = errors.New("component def not found")

// ComponentDefStore defines the interface for component-def storage operations.
type ComponentDefStore interface {
	// Create creates a new component-def or updates if it already exists (based on name and version)
	Create(ctx context.Context, component *ComponentDef) error

	// Get gets an existing component-def by name and version
	Get(ctx context.Context, name, version string) (*ComponentDef, error)

	// List lists all component-defs with optional filters
	List(ctx context.Context, opts *ListOptions) ([]*ComponentDef, error)

	// Delete deletes a component-def by name and version
	Delete(ctx context.Context, name, version string) (int64, error)

	// UpdateInstanceCount 原子更新组件定义的实例计数。
	// name: 组件定义名称
	// field: 计数字段，FieldAppCompInstanceCount 或 FieldWorkspaceCompInstanceCount
	// delta: 增量，正数表示增加，负数表示减少
	UpdateInstanceCount(ctx context.Context, name string, field InstanceType, delta int) error
}

// ListOptions defines options for listing component-defs
type ListOptions struct {
	// ScopeWorkspaceID filters by workspace scope (returns global + workspace-specific)
	ScopeWorkspaceID string
	// ManagedByWorkspaceID filters by managed by workspace (returns global + workspace-specific)
	ManagedByWorkspaceID string
	// Keyword searches in name and displayName
	Keyword string
	// ExcludeInvisible filters out invisible component-defs when set to true
	ExcludeInvisible bool
}

var _ ComponentDefStore = &ComponentDefStoreMongo{}

// ComponentDefStoreMongo is the MongoDB implementation of ComponentDefStore.
type ComponentDefStoreMongo struct {
	// Collection is the MongoDB collection for storing component-def data.
	collection *mongo.Collection
}

// NewComponentDefStoreMongo creates a new ComponentDefStore, it accepts a MongoDB client and database name.
func NewComponentDefStoreMongo(client *mongo.Client, dbName string) (*ComponentDefStoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：name + version
	return &ComponentDefStoreMongo{collection: coll}, nil
}

// Create creates a new component-def or updates an existing one(based on name and version)
//
// - ctx: The context object for cancellation and timeout
// - compDef: The component-def to create or update
//
// Return an error if the operation fails.
func (s *ComponentDefStoreMongo) Create(ctx context.Context, compDef *ComponentDef) error {
	// Validate the component struct to ensure required fields are present
	if err := ValidateComponentDef(compDef); err != nil {
		return errors.Wrap(err, "component validation failed")
	}

	// Prepare the timestamp fields
	now := time.Now()
	if compDef.CreatedAt.IsZero() {
		compDef.CreatedAt = now
	}
	compDef.UpdatedAt = now

	filter := bson.M{"name": compDef.Name, "version": compDef.Version}
	setDoc, err := buildComponentDefSetDoc(compDef)
	if err != nil {
		return errors.Wrap(err, "prepare the creation document")
	}
	update := bson.M{
		"$set":         setDoc,
		"$setOnInsert": bson.M{"createdAt": compDef.CreatedAt},
	}
	opts := options.UpdateOne().SetUpsert(true)
	if _, err := s.collection.UpdateOne(ctx, filter, update, opts); err != nil {
		return errors.Wrap(err, "creating/updating component")
	}
	return nil
}

// Get gets a component-def by name and version
//
// - ctx: The context object for cancellation and timeout
// - name: The name of the component-def
// - version: The version of the component-def
//
// Return an error if the operation fails, ErrComponentDefNotFound if the component-def is not found.
func (s *ComponentDefStoreMongo) Get(ctx context.Context, name, version string) (*ComponentDef, error) {
	compDef := new(ComponentDef)
	filter := bson.M{"name": name, "version": version}

	err := s.collection.FindOne(ctx, filter).Decode(compDef)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// When no record can be found, return a new error
			return nil, ErrComponentDefNotFound
		}
		return nil, err
	}
	return compDef, nil
}

// Delete deletes a component-def by name and version
//
// - ctx: The context object for cancellation and timeout
// - name: The name of the component-def
// - version: The version of the component-def
//
// Return the number of deleted documents and an error if any.
func (s *ComponentDefStoreMongo) Delete(ctx context.Context, name, version string) (int64, error) {
	filter := bson.M{"name": name, "version": version}

	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// List lists all component-defs with optional filters
//
// - ctx: The context object for cancellation and timeout
// - opts: Optional filter options
//
// Return a list of component-defs and an error if any.
func (s *ComponentDefStoreMongo) List(ctx context.Context, opts *ListOptions) ([]*ComponentDef, error) {
	filter := bson.M{}

	if opts != nil {
		// Filter out invisible component-defs
		if opts.ExcludeInvisible {
			filter["invisible"] = bson.M{"$ne": true}
		}

		// Filter by workspace scope: return global + workspace-specific components
		if opts.ScopeWorkspaceID != "" {
			filter["$or"] = []bson.M{
				{"scopeType": ScopeTypeGlobal},
				{"scopeWorkspaceIDs": opts.ScopeWorkspaceID},
			}
		}

		if opts.ManagedByWorkspaceID != "" {
			filter["managedByWorkspaceIDs"] = opts.ManagedByWorkspaceID
		}

		// Search by keyword in name and displayName
		if opts.Keyword != "" {
			keywordFilter := bson.M{
				"$or": []bson.M{
					{"name": bson.M{"$regex": opts.Keyword, "$options": "i"}},
					{"displayName": bson.M{"$regex": opts.Keyword, "$options": "i"}},
				},
			}
			// Merge keyword filter with existing filter
			if existingOr, ok := filter["$or"]; ok {
				filter["$and"] = []bson.M{
					{"$or": existingOr},
					keywordFilter,
				}
				delete(filter, "$or")
			} else {
				filter["$or"] = keywordFilter["$or"]
			}
		}
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "find component-defs")
	}
	defer cursor.Close(ctx)

	var compDefs []*ComponentDef
	if err = cursor.All(ctx, &compDefs); err != nil {
		return nil, errors.Wrap(err, "decode component-defs")
	}

	return compDefs, nil
}

// UpdateInstanceCount 原子更新组件定义的实例计数。
// 使用 MongoDB $inc 操作符实现原子递增/递减，适用于并发场景。
//
// - ctx: The context object for cancellation and timeout
// - name: 组件定义名称（使用默认版本 DefaultComponentDefVersion）
// - field: 计数字段，FieldAppCompInstanceCount 或 FieldWorkspaceCompInstanceCount
// - delta: 增量，正数表示增加，负数表示减少
func (s *ComponentDefStoreMongo) UpdateInstanceCount(
	ctx context.Context, name string, instanceType InstanceType, delta int,
) error {
	var fieldName string
	switch instanceType {
	case FieldAppCompInstance:
		fieldName = "appCompInstanceCount"
	case FieldWorkspaceCompInstance:
		fieldName = "workspaceCompInstanceCount"
	default:
		return errors.Errorf("invalid instance type: %s", instanceType)
	}
	filter := bson.M{"name": name, "version": DefaultComponentDefVersion}
	update := bson.M{"$inc": bson.M{fieldName: delta}}
	_, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrapf(err, "incr ref count for component def(%s) field(%s) delta(%d)", name, instanceType, delta)
	}
	return nil
}

// Build the bson document for a component-def to be used in an update operation.
func buildComponentDefSetDoc(compDef *ComponentDef) (bson.M, error) {
	data, err := bson.Marshal(compDef)
	if err != nil {
		return nil, errors.Wrap(err, "marshal component-def for update")
	}

	setDoc := bson.M{}
	if err := bson.Unmarshal(data, &setDoc); err != nil {
		return nil, errors.Wrap(err, "unmarshal component-def for update")
	}
	// Patchers 和 Specs 允许任一侧为空，但数据库中始终以数组存储，避免 nil 被编码为 null。
	if compDef.Patchers == nil {
		setDoc["patchers"] = bson.A{}
	}
	if compDef.Specs == nil {
		setDoc["specs"] = bson.A{}
	}

	// "createdAt" field should be excluded from the $set document
	delete(setDoc, "createdAt")
	// 实例计数字段由 UpdateInstanceCount 维护，不应被 Create/Update 覆盖
	delete(setDoc, "appCompInstanceCount")
	delete(setDoc, "workspaceCompInstanceCount")
	return setDoc, nil
}
