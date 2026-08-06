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

package networking

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// The name of the MongoDB collection for storing networking services.
const collectionName = "networking_services"

// ErrServiceNotFound is returned when a service is not found in the store.
var ErrServiceNotFound = errors.New("service not found")

// use a single instance of Validate, it caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

// SvcUpdateData is the data for updating the networking service.
type SvcUpdateData struct {
	Selector           map[string]string
	Ports              []ServicePort
	TrafficLaneEnabled *bool
}

// ServiceStore is the interface for the networking service storage.
type ServiceStore interface {
	// Create creates a new networking service.
	Create(ctx context.Context, svc *Service) error

	// ListByApp gets the networking services by appID.
	ListByApp(ctx context.Context, appID string) ([]Service, error)

	// GroupByAppID fetches all services for the given appIDs from storage
	// and groups them by appID. Only appIDs with at least one service are returned.
	GroupByAppID(ctx context.Context, appIDs []string) (map[string][]Service, error)

	// Get gets the networking service.
	Get(ctx context.Context, appID, name string) (*Service, error)

	// Update updates the networking service.
	// Note: 不支持置空 selector 和 ports, 空值会被忽略更新
	Update(ctx context.Context, appID, name string, updateData *SvcUpdateData) error

	// Delete deletes the networking service.
	Delete(ctx context.Context, appID, name string) error

	// DeleteAll deletes all networking services while preserving the collection and its indexes.
	// Attention: only used in unit test
	DeleteAll(ctx context.Context) error
}

var _ ServiceStore = &ServiceStoreMongo{}

// ServiceStoreMongo is the MongoDB implementation of ServiceStore.
type ServiceStoreMongo struct {
	// Collection is the MongoDB collection for storing component-def data.
	collection *mongo.Collection
}

// NewServiceStoreMongo creates a new ServiceStore, it accepts a MongoDB client and database name.
func NewServiceStoreMongo(client *mongo.Client, dbName string) (*ServiceStoreMongo, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + name
	return &ServiceStoreMongo{collection: coll}, nil
}

// Create creates a new networking service.
func (s *ServiceStoreMongo) Create(ctx context.Context, svc *Service) error {
	if err := validate.Struct(svc); err != nil {
		return errors.Wrap(err, "service validation failed")
	}

	if svc.CreatedAt.IsZero() {
		svc.CreatedAt = time.Now()
	}
	svc.UpdatedAt = svc.CreatedAt

	_, err := s.collection.InsertOne(ctx, svc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return errors.Errorf("service %s already exists", svc.Name)
		}
		return err
	}
	return nil
}

// ListByApp gets the networking services by appID.
func (s *ServiceStoreMongo) ListByApp(ctx context.Context, appID string) ([]Service, error) {
	filter := bson.M{"appID": appID}

	sort := bson.D{{Key: "createdAt", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx) // nolint

	var services []Service
	if err = cursor.All(ctx, &services); err != nil {
		return nil, err
	}
	return services, nil
}

// GroupByAppID fetches all services for the given appIDs from storage
// and groups them by appID. Only appIDs with at least one service are returned.
func (s *ServiceStoreMongo) GroupByAppID(ctx context.Context, appIDs []string) (map[string][]Service, error) {
	if len(appIDs) == 0 {
		return make(map[string][]Service), nil
	}

	filter := bson.M{"appID": bson.M{"$in": appIDs}}

	sort := bson.D{{Key: "createdAt", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx) // nolint

	var services []Service
	if err = cursor.All(ctx, &services); err != nil {
		return nil, err
	}

	grouped := make(map[string][]Service)
	for _, svc := range services {
		grouped[svc.AppID] = append(grouped[svc.AppID], svc)
	}
	return grouped, nil
}

// Get gets the networking service.
func (s *ServiceStoreMongo) Get(ctx context.Context, appID, name string) (*Service, error) {
	filter := bson.M{"appID": appID, "name": name}

	svc := new(Service)
	err := s.collection.FindOne(ctx, filter).Decode(svc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrServiceNotFound
		}
		return nil, err
	}
	return svc, nil
}

// Update updates the networking service.
// Note: 不支持置空 selector 和 ports, 空值会被忽略更新
func (s *ServiceStoreMongo) Update(ctx context.Context, appID, name string, updateData *SvcUpdateData) error {
	filter := bson.M{"appID": appID, "name": name}

	update := bson.M{}
	if len(updateData.Selector) > 0 {
		update["selector"] = updateData.Selector
	}
	if len(updateData.Ports) > 0 {
		update["ports"] = updateData.Ports
	}
	if updateData.TrafficLaneEnabled != nil {
		update["trafficLaneEnabled"] = *updateData.TrafficLaneEnabled
	}

	update["updatedAt"] = time.Now()

	opts := options.UpdateOne().SetUpsert(false)
	ret, err := s.collection.UpdateOne(ctx, filter, bson.M{"$set": update}, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return errors.New("no matched service found")
	}
	return err
}

// Delete deletes the networking service.
func (s *ServiceStoreMongo) Delete(ctx context.Context, appID, name string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"appID": appID, "name": name})
	return err
}

// DeleteAll deletes all networking services while preserving the collection and its indexes.
// Attention: only used in unit test
func (s *ServiceStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}
