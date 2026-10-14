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
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrConfigNotFound is returned when the app has no HostPort config document.
var ErrConfigNotFound = errors.New("hostport config not found")

// HostPortStore persists HostPortConfig documents.
type HostPortStore = store.GenericStore[HostPortConfig]

// HostPortConfigMeta is the storage metadata of HostPortConfig, one document per app keyed by appID.
// Reads normalize ports and never return a nil EnvStates map.
var HostPortConfigMeta = store.Meta[HostPortConfig]{
	Collection: "hostport_configs",
	NotFound:   ErrConfigNotFound,
	Decode:     []func(*HostPortConfig) error{normalizeConfig},
}

// ByApp matches the HostPort config of an app.
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ListPorts returns declared container ports; missing config yields an empty slice.
func ListPorts(ctx context.Context, db *mongo.Database, appID string) ([]int32, error) {
	config, err := store.Of(db, HostPortConfigMeta).Get(ctx, ByApp(appID))
	if errors.Is(err, ErrConfigNotFound) {
		return []int32{}, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get hostport config of app %s", appID)
	}
	return config.Ports, nil
}

// ReplacePorts replaces the declared container ports for an app, creating the config when absent,
// and returns the normalized ports that were stored.
func ReplacePorts(ctx context.Context, db *mongo.Database, appID string, ports []int32) ([]int32, error) {
	config := &HostPortConfig{AppID: appID, Ports: NormalizePorts(ports), EnvStates: map[string]HostPortEnvState{}}
	if _, err := store.Of(db, HostPortConfigMeta).Upsert(ctx, config,
		ByApp(appID), store.UpdateSelected("ports"),
	); err != nil {
		return nil, errors.Wrapf(err, "replace hostport ports of app %s", appID)
	}
	return config.Ports, nil
}

// UpsertEnvState records applied ports for an environment, creating the config when absent.
func UpsertEnvState(ctx context.Context, db *mongo.Database, appID, envName string, appliedPorts []int32) error {
	field, err := envStateField(envName)
	if err != nil {
		return err
	}
	config := &HostPortConfig{
		AppID: appID,
		Ports: []int32{},
		EnvStates: map[string]HostPortEnvState{
			envName: {AppliedPorts: NormalizePorts(appliedPorts), UpdatedAt: time.Now()},
		},
	}
	if _, err = store.Of(db, HostPortConfigMeta).Upsert(ctx, config,
		ByApp(appID), store.UpdateSelected(field),
	); err != nil {
		return errors.Wrapf(err, "upsert hostport env state of app %s, env %s", appID, envName)
	}
	return nil
}

// RemoveEnvState removes the env snapshot after uninstall; a missing config is ignored.
func RemoveEnvState(ctx context.Context, db *mongo.Database, appID, envName string) error {
	field, err := envStateField(envName)
	if err != nil {
		return err
	}
	err = store.Of(db, HostPortConfigMeta).Update(ctx, map[string]any{field: nil}, ByApp(appID))
	return errors.Wrapf(store.IgnoreNotFound(err), "remove hostport env state of app %s, env %s", appID, envName)
}

func normalizeConfig(config *HostPortConfig) error {
	config.Ports = NormalizePorts(config.Ports)
	if config.EnvStates == nil {
		config.EnvStates = map[string]HostPortEnvState{}
	}
	return nil
}

// envStateField returns the path of an env state; env names are used as map keys, so they must not contain
// characters that MongoDB treats as path separators or operators.
func envStateField(envName string) (string, error) {
	if envName == "" || strings.ContainsAny(envName, ".$") {
		return "", errors.Errorf("invalid env name %q", envName)
	}
	return "envStates." + envName, nil
}
