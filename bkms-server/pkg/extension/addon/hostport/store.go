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

package hostport

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const collectionName = "hostport_configs"

var (
	// ErrConfigNotFound is returned when the app has no HostPort config document.
	ErrConfigNotFound = errors.New("hostport config not found")
)

// HostPortStore persists HostPortConfig documents.
type HostPortStore interface {
	Get(ctx context.Context, appID string) (*HostPortConfig, error)
	ListPorts(ctx context.Context, appID string) ([]int32, error)
	ReplacePorts(ctx context.Context, appID string, ports []int32) (*HostPortConfig, error)
	UpsertEnvState(ctx context.Context, appID, envName string, appliedPorts []int32) error
	RemoveEnvState(ctx context.Context, appID, envName string) error
	DeleteByApp(ctx context.Context, appID string) error
}

var _ HostPortStore = &HostPortStoreMongo{}

// HostPortStoreMongo is the MongoDB implementation of HostPortStore.
type HostPortStoreMongo struct {
	collection *mongo.Collection
}

// NewHostPortStoreMongo creates a HostPortStore backed by MongoDB.
func NewHostPortStoreMongo(client *mongo.Client, dbName string) (HostPortStore, error) {
	coll := client.Database(dbName).Collection(collectionName)
	return &HostPortStoreMongo{collection: coll}, nil
}

// Get returns the HostPort config for an app.
func (s *HostPortStoreMongo) Get(ctx context.Context, appID string) (*HostPortConfig, error) {
	var config HostPortConfig
	err := s.collection.FindOne(ctx, bson.M{"appID": appID}).Decode(&config)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrConfigNotFound
		}
		return nil, errors.Wrap(err, "find hostport config")
	}
	normalizeConfig(&config)
	return &config, nil
}

// ListPorts returns declared container ports; missing config yields an empty slice.
func (s *HostPortStoreMongo) ListPorts(ctx context.Context, appID string) ([]int32, error) {
	config, err := s.Get(ctx, appID)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return []int32{}, nil
		}
		return nil, err
	}
	return config.Ports, nil
}

// ReplacePorts replaces the declared container ports for an app (upsert).
// Ports are normalized (unique, sorted). An empty slice deletes the config document.
func (s *HostPortStoreMongo) ReplacePorts(ctx context.Context, appID string, ports []int32) (*HostPortConfig, error) {
	ports = NormalizePorts(ports)
	if len(ports) == 0 {
		if err := s.DeleteByApp(ctx, appID); err != nil {
			return nil, err
		}
		return &HostPortConfig{
			AppID:     appID,
			Ports:     []int32{},
			EnvStates: map[string]HostPortEnvState{},
		}, nil
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"ports":     ports,
			"updatedAt": now,
		},
		"$setOnInsert": bson.M{
			"appID":     appID,
			"envStates": bson.M{},
			"createdAt": now,
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var config HostPortConfig
	if err := s.collection.FindOneAndUpdate(ctx, bson.M{"appID": appID}, update, opts).Decode(&config); err != nil {
		return nil, errors.Wrap(err, "replace hostport mappings")
	}
	normalizeConfig(&config)
	return &config, nil
}

// UpsertEnvState records applied ports for an environment in one upsert round-trip.
func (s *HostPortStoreMongo) UpsertEnvState(
	ctx context.Context,
	appID, envName string,
	appliedPorts []int32,
) error {
	fieldPrefix, err := envFieldPrefix("envStates", envName)
	if err != nil {
		return err
	}

	now := time.Now()
	appliedPorts = NormalizePorts(appliedPorts)
	update := bson.M{
		"$set": bson.M{
			fieldPrefix + ".appliedPorts": appliedPorts,
			fieldPrefix + ".updatedAt":    now,
			"updatedAt":                   now,
		},
		"$setOnInsert": bson.M{
			"appID":     appID,
			"ports":     []int32{},
			"createdAt": now,
		},
	}
	opts := options.UpdateOne().SetUpsert(true)
	_, err = s.collection.UpdateOne(ctx, bson.M{"appID": appID}, update, opts)
	if err != nil {
		return errors.Wrap(err, "upsert hostport env state")
	}
	return nil
}

// RemoveEnvState removes the env snapshot after uninstall.
func (s *HostPortStoreMongo) RemoveEnvState(ctx context.Context, appID, envName string) error {
	fieldPrefix, err := envFieldPrefix("envStates", envName)
	if err != nil {
		return err
	}
	_, err = s.collection.UpdateOne(
		ctx,
		bson.M{"appID": appID},
		bson.M{
			"$unset": bson.M{fieldPrefix: ""},
			"$set":   bson.M{"updatedAt": time.Now()},
		},
	)
	if err != nil {
		return errors.Wrap(err, "remove hostport env state")
	}
	return nil
}

// DeleteByApp deletes the HostPort config for an app.
func (s *HostPortStoreMongo) DeleteByApp(ctx context.Context, appID string) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"appID": appID})
	if err != nil {
		return errors.Wrap(err, "delete hostport config by app")
	}
	return nil
}

func normalizeConfig(config *HostPortConfig) {
	config.Ports = NormalizePorts(config.Ports)
	if config.EnvStates == nil {
		config.EnvStates = map[string]HostPortEnvState{}
	}
}

func envFieldPrefix(root, envName string) (string, error) {
	if envName == "" || strings.ContainsAny(envName, ".$") {
		return "", errors.Errorf("invalid env name %q", envName)
	}
	return root + "." + envName, nil
}
