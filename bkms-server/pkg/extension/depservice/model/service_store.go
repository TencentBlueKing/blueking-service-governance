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

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const serviceCollName = "depservice_services"

// ServiceStore defines the interface for service storage
// - name is the unique identifier of service
type ServiceStore interface {
	// Create creates a new service
	//
	// - ctx: The context object for cancellation and timeout
	// - service: The service to create
	//
	// Return an error if the operation fails.
	Create(ctx context.Context, service *Service) error
	// Upsert updates a service or creates a new service if it does not exist
	//
	// - ctx: The context object for cancellation and timeout
	// - service: The service to update
	//
	// Return an error if the operation fails.
	Upsert(ctx context.Context, service *Service) error
	// Get gets a service by name
	//
	// - ctx: The context object for cancellation and timeout
	// - name: The name of the service to get
	//
	// Return the service if found, otherwise return an error.
	Get(ctx context.Context, name string) (*Service, error)
	// List lists all services
	//
	// - ctx: The context object for cancellation and timeout
	//
	// Return the list of services and an error if any.
	List(ctx context.Context) ([]Service, error)
	// Delete deletes a service
	//
	// - ctx: The context object for cancellation and timeout
	// - name: The name of the service to delete
	//
	// Return an error if the operation fails.
	Delete(ctx context.Context, name string) error
	// DeleteAll deletes all services while preserving the collection and its indexes.
	// Attention: only used in unit test
	DeleteAll(ctx context.Context) error
}

var _ ServiceStore = &ServiceStoreMongo{}

// ServiceStoreMongo implements ServiceStore interface with mongodb
type ServiceStoreMongo struct {
	collection *mongo.Collection
}

// NewServiceStoreMongo creates a new ServiceStoreMongo instance
func NewServiceStoreMongo(client *mongo.Client, dbName string) (*ServiceStoreMongo, error) {
	coll := client.Database(dbName).Collection(serviceCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：name
	return &ServiceStoreMongo{collection: coll}, nil
}

// Create creates a new service
//
// - ctx: The context object for cancellation and timeout
// - service: The service to create
//
// Return an error if the operation fails.
func (s *ServiceStoreMongo) Create(ctx context.Context, service *Service) error {
	if err := validate.Struct(service); err != nil {
		return errors.Wrap(err, "service validation failed")
	}

	dbSvc, err := servicePrepDBValue(service)
	if err != nil {
		return errors.Wrap(err, "create service")
	}

	if _, err = s.collection.InsertOne(ctx, dbSvc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.New("service with the same name already exists")
		}
		return err
	}
	return nil
}

// Upsert updates a service or creates a new service if it does not exist
//
// - ctx: The context object for cancellation and timeout
// - service: The service to update
//
// Return an error if the operation fails.
func (s *ServiceStoreMongo) Upsert(ctx context.Context, service *Service) error {
	if err := validate.Struct(service); err != nil {
		return errors.Wrap(err, "service validation failed")
	}

	dbSvc, err := servicePrepDBValue(service)
	if err != nil {
		return errors.Wrap(err, "prep db value")
	}

	filter := bson.M{"name": service.Name}

	updateSet := bson.M{}
	if service.DisplayName != "" {
		updateSet["displayName"] = dbSvc.DisplayName
	}
	if service.Description != "" {
		updateSet["description"] = dbSvc.Description
	}
	if service.Category != "" {
		updateSet["category"] = dbSvc.Category
	}
	updateSet["plans"] = dbSvc.Plans

	update := bson.M{"$set": updateSet}
	opts := options.UpdateOne().SetUpsert(true)
	_, err = s.collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// Get gets a service by name
//
// - ctx: The context object for cancellation and timeout
// - name: The name of the service to get
//
// Return the service if found, otherwise return an error.
func (s *ServiceStoreMongo) Get(ctx context.Context, name string) (*Service, error) {
	dbSvc := new(Service)
	err := s.collection.FindOne(ctx, bson.M{"name": name}).Decode(dbSvc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, NewNotFoundError(fmt.Sprintf("service(name:%s)", name))
		}
		return nil, err
	}

	if svc, err := serviceFromDBValue(dbSvc); err != nil {
		return nil, errors.Wrap(err, "from db value")
	} else {
		return svc, nil
	}
}

// List lists all services
//
// - ctx: The context object for cancellation and timeout
//
// Return the list of services and an error if any.
func (s *ServiceStoreMongo) List(ctx context.Context) ([]Service, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // nolint

	services := make([]Service, 0)
	if err = cursor.All(ctx, &services); err != nil {
		return nil, err
	}
	return services, nil
}

// Delete deletes a service
//
// - ctx: The context object for cancellation and timeout
// - name: The name of the service to delete
//
// Return an error if the operation fails.
func (s *ServiceStoreMongo) Delete(ctx context.Context, name string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"name": name})
	return err
}

// DeleteAll deletes all services while preserving the collection and its indexes.
// Attention: only used in unit test
func (s *ServiceStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}

// servicePrepDBValue prepares the service to save in db.
// Storing a map[string]any directly in MongoDB converts it to bson.D,
// unmarshalling this back can be inconvenient, so we serialize it to a json string first.
// - marshals the plan config to map {"value": "plan's config json string"}
func servicePrepDBValue(svc *Service) (*Service, error) {
	dbValue := new(Service)
	if err := copier.CopyWithOption(dbValue, svc, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		return nil, err
	}

	for idx := 0; idx < len(dbValue.Plans); idx++ {
		if err := dbValue.Plans[idx].MarshalConfig(); err != nil {
			return nil, err
		}
	}

	return dbValue, nil
}

// serviceFromDBValue converts the service from db service value. it is the reverse of servicePrepDBValue.
// - unmarshals the plan config from map {"value": "plan's config json string"}
func serviceFromDBValue(dbValue *Service) (*Service, error) {
	svc := new(Service)
	if err := copier.CopyWithOption(svc, dbValue, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		return nil, err
	}

	for idx := 0; idx < len(svc.Plans); idx++ {
		if err := svc.Plans[idx].UnmarshalConfig(); err != nil {
			return nil, err
		}
	}

	return svc, nil
}
