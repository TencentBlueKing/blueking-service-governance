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

package admin

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "plat_admin_role_bindings"

var (
	// ErrRoleBindingNotFound indicates the target platform administrator role binding does not exist.
	ErrRoleBindingNotFound = errors.New("platform administrator role binding not found")
	// ErrRoleBindingAlreadyExists indicates the target platform administrator role binding already exists.
	ErrRoleBindingAlreadyExists = errors.New("platform administrator role binding already exists")
	// ErrPermissionDenied indicates the current user is not a platform administrator.
	ErrPermissionDenied = errors.New("platform administrator permission denied")
)

// Store persists platform administrator records.
type Store interface {
	List(ctx context.Context, opts *ListOptions) ([]RoleBinding, error)
	Get(ctx context.Context, username string) (*RoleBinding, error)
	CreateMany(ctx context.Context, roleBindings []*RoleBinding) error
	Delete(ctx context.Context, username string) error
	// TODO: Add Update when platform role changes are supported.
}

var _ Store = (*StoreMongo)(nil)

// StoreMongo is the MongoDB implementation of Store.
type StoreMongo struct {
	collection *mongo.Collection
}

// NewStoreMongo creates a platform administrator store backed by MongoDB.
func NewStoreMongo(client *mongo.Client, dbName string) (*StoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：username
	return &StoreMongo{collection: coll}, nil
}

// List returns all platform administrator role bindings filtered by the given options.
func (s *StoreMongo) List(ctx context.Context, opts *ListOptions) ([]RoleBinding, error) {
	filter := bson.M{}
	if opts != nil && opts.Keyword != "" {
		keyword := regexp.QuoteMeta(opts.Keyword)
		filter["username"] = bson.M{"$regex": keyword, "$options": "i"}
	}

	findOpts := options.Find().SetSort(bson.D{{"username", 1}})
	cursor, err := s.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, errors.Wrap(err, "find platform administrators")
	}
	defer cursor.Close(ctx)

	var roleBindings []RoleBinding
	if err = cursor.All(ctx, &roleBindings); err != nil {
		return nil, errors.Wrap(err, "decode platform administrator role bindings")
	}
	return roleBindings, nil
}

// Get returns a platform administrator role binding by username.
func (s *StoreMongo) Get(ctx context.Context, username string) (*RoleBinding, error) {
	var roleBinding RoleBinding
	if err := s.collection.FindOne(ctx, bson.M{"username": username}).Decode(&roleBinding); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrRoleBindingNotFound
		}
		return nil, errors.Wrap(err, "find platform administrator role binding")
	}
	return &roleBinding, nil
}

// CreateMany inserts multiple platform administrator role bindings in one batch.
// Existing usernames are skipped silently to keep the operation idempotent.
func (s *StoreMongo) CreateMany(ctx context.Context, roleBindings []*RoleBinding) error {
	if len(roleBindings) == 0 {
		return nil
	}

	now := time.Now()
	models := make([]mongo.WriteModel, 0, len(roleBindings))
	seen := make(map[string]struct{}, len(roleBindings))
	for _, roleBinding := range roleBindings {
		if roleBinding == nil {
			continue
		}
		if _, ok := seen[roleBinding.Username]; ok {
			continue
		}
		seen[roleBinding.Username] = struct{}{}
		if roleBinding.CreatedAt.IsZero() {
			roleBinding.CreatedAt = now
		}
		if roleBinding.UpdatedAt.IsZero() {
			roleBinding.UpdatedAt = roleBinding.CreatedAt
		}
		if roleBinding.Updater == "" {
			roleBinding.Updater = roleBinding.Creator
		}
		models = append(models, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"username": roleBinding.Username}).
			SetUpdate(bson.M{"$setOnInsert": roleBinding}).
			SetUpsert(true))
	}
	if len(models) == 0 {
		return nil
	}

	if _, err := s.collection.BulkWrite(ctx, models, options.BulkWrite().SetOrdered(false)); err != nil {
		return errors.Wrap(err, "bulk create platform administrator role bindings")
	}
	return nil
}

// Delete removes a platform administrator role binding by username.
// Using the silent deletion design, if the role binding does not exist, the deletion will be performed by default.
func (s *StoreMongo) Delete(ctx context.Context, username string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"username": username})
	if err != nil {
		return errors.Wrap(err, "delete platform administrator role binding")
	}
	return nil
}
