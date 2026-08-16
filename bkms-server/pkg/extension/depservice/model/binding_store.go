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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// ErrBindingNameExists 应用内同服务下绑定名已存在
var ErrBindingNameExists = errors.New("service binding name already exists")

// ServiceBindingStore defines persistence operations for service bindings.
type ServiceBindingStore interface {
	Create(ctx context.Context, binding *ServiceBinding) (bson.ObjectID, error)
	Get(ctx context.Context, appID, serviceName, name string) (*ServiceBinding, error)
	List(ctx context.Context, opts *BindingQueryOptions) ([]*ServiceBinding, error)
	Update(ctx context.Context, appID, serviceName, name string, data *ServiceBindingUpdateData) error
	Delete(ctx context.Context, appID, serviceName, name string) error
	// DeleteAll deletes all bindings while preserving the collection and its indexes.
	// Attention: only used in unit test
	DeleteAll(ctx context.Context) error
}

var _ ServiceBindingStore = &ServiceBindingStoreMongo{}

// ServiceBindingStoreMongo implements ServiceBindingStore with MongoDB.
type ServiceBindingStoreMongo struct {
	collection *mongo.Collection
}

// NewServiceBindingStoreMongo creates a ServiceBindingStoreMongo.
func NewServiceBindingStoreMongo(client *mongo.Client, dbName string) (ServiceBindingStore, error) {
	coll := client.Database(dbName).Collection(serviceBindingCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + serviceName + name
	// - instanceIDs：按实例反查引用
	return &ServiceBindingStoreMongo{collection: coll}, nil
}

// Create creates a new service binding.
func (s *ServiceBindingStoreMongo) Create(ctx context.Context, binding *ServiceBinding) (bson.ObjectID, error) {
	if err := validate.Struct(binding); err != nil {
		return bson.NilObjectID, errors.Wrap(err, "service binding validation failed")
	}

	binding.SyncInstanceIDs()
	if binding.CreatedAt.IsZero() {
		binding.CreatedAt = time.Now()
	}
	binding.UpdatedAt = binding.CreatedAt

	result, err := s.collection.InsertOne(ctx, binding)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, ErrBindingNameExists
		}
		return bson.NilObjectID, err
	}
	return result.InsertedID.(bson.ObjectID), nil
}

// Get gets a binding by app, service and name.
func (s *ServiceBindingStoreMongo) Get(
	ctx context.Context,
	appID, serviceName, name string,
) (*ServiceBinding, error) {
	binding := new(ServiceBinding)
	err := s.collection.FindOne(ctx, bson.M{
		"appID":       appID,
		"serviceName": serviceName,
		"name":        name,
	}).Decode(binding)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, NewNotFoundError(fmt.Sprintf(
				"service binding(app=%s, service=%s, name=%s)", appID, serviceName, name,
			))
		}
		return nil, err
	}
	return binding, nil
}

// List lists bindings by query options.
func (s *ServiceBindingStoreMongo) List(
	ctx context.Context,
	opts *BindingQueryOptions,
) ([]*ServiceBinding, error) {
	if opts == nil {
		opts = &BindingQueryOptions{}
	}
	filter := bson.M{}
	if opts.AppID != "" {
		filter["appID"] = opts.AppID
	}
	if opts.WorkspaceID != "" {
		filter["workspaceID"] = opts.WorkspaceID
	}
	if opts.ServiceName != "" {
		filter["serviceName"] = opts.ServiceName
	}
	if !opts.InstanceID.IsZero() {
		filter["instanceIDs"] = opts.InstanceID
	}

	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	bindings := make([]*ServiceBinding, 0)
	if err = cursor.All(ctx, &bindings); err != nil {
		return nil, err
	}
	return bindings, nil
}

// Update replaces env mapping, env vars and description of a binding.
func (s *ServiceBindingStoreMongo) Update(
	ctx context.Context,
	appID, serviceName, name string,
	data *ServiceBindingUpdateData,
) error {
	if data == nil {
		return nil
	}

	tmp := &ServiceBinding{
		EnvInstanceMap: data.EnvInstanceMap,
		EnvVars:        data.EnvVars,
	}
	tmp.SyncInstanceIDs()

	ret, err := s.collection.UpdateOne(ctx, bson.M{
		"appID":       appID,
		"serviceName": serviceName,
		"name":        name,
	}, bson.M{"$set": bson.M{
		"envInstanceMap": tmp.EnvInstanceMap,
		"envVars":        tmp.EnvVars,
		"instanceIDs":    tmp.InstanceIDs,
		"description":    data.Description,
		"updatedAt":      time.Now(),
	}})
	if err != nil {
		return err
	}
	if ret.MatchedCount == 0 {
		return NewNotFoundError(fmt.Sprintf(
			"service binding(app=%s, service=%s, name=%s)", appID, serviceName, name,
		))
	}
	return nil
}

// Delete deletes a binding by app, service and name.
func (s *ServiceBindingStoreMongo) Delete(ctx context.Context, appID, serviceName, name string) error {
	ret, err := s.collection.DeleteOne(ctx, bson.M{
		"appID":       appID,
		"serviceName": serviceName,
		"name":        name,
	})
	if err != nil {
		return err
	}
	if ret.DeletedCount == 0 {
		return NewNotFoundError(fmt.Sprintf(
			"service binding(app=%s, service=%s, name=%s)", appID, serviceName, name,
		))
	}
	return nil
}

// DeleteAll deletes all bindings. Attention: only used in unit test.
func (s *ServiceBindingStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}
