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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// ApmService encapsulates APM instance config business logic, including remote fetching,
// local persistence, environment binding, and environment variable management.
type ApmService struct {
	store ApmInstConfigStore

	scopedEnvVarStore envvars.ScopedEnvVarStore
}

// NewApmService creates a new ApmService instance.
func NewApmService(store ApmInstConfigStore, scopedEnvVarStore envvars.ScopedEnvVarStore) *ApmService {
	return &ApmService{
		store:             store,
		scopedEnvVarStore: scopedEnvVarStore,
	}
}

// Get retrieves an APM instance config record from local store by apmID.
// If not found locally, it fetches from bkmonitor remote API and persists to local store.
func (s *ApmService) Get(
	ctx context.Context,
	apmID int64,
	param CreateApmInstParams,
) (*ApmInstConfig, error) {
	// Try local store first
	apm, err := s.store.GetByApmID(ctx, apmID)
	if err == nil {
		return apm, nil
	}
	if !errors.Is(err, ErrApmInstConfigNotFound) {
		return nil, errors.Wrap(err, "get apm from store")
	}

	// Not found locally, fetch from remote
	client, err := bkmapi.New(param.Username)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	apmAppsData, err := client.ListApmApp(ctx, param.BkmProjectID)
	if err != nil {
		return nil, errors.Wrap(err, "list apm app from bkmonitor")
	}
	targetApmApp, ok := lo.Find(apmAppsData, func(item *bkmapi.ApmApp) bool {
		return item.ID == apmID
	})
	if !ok || targetApmApp == nil {
		return nil, errors.Errorf("apm %d not found in bkmonitor", apmID)
	}

	newApm := &ApmInstConfig{
		WorkspaceID: param.WorkspaceID,
		ApmID:       targetApmApp.ID,
		Name:        targetApmApp.AppName,
		Token:       targetApmApp.Token,
	}
	apmObjID, err := s.store.Create(ctx, newApm)
	if err != nil {
		return nil, errors.Wrap(err, "create apm in store")
	}
	newApm.ID = apmObjID

	return newApm, nil
}

// BindToEnv binds an APM record to the specified environment.
// It performs the following steps:
// 1. Unbind the environment from all existing APM associations
// 2. Bind the environment to the target APM
// 3. Write/update APM-related environment variables
func (s *ApmService) BindToEnv(
	ctx context.Context,
	apm *ApmInstConfig,
	envID bson.ObjectID,
	envName string,
) error {
	// 1. Clear old associations
	if err := s.store.UnbindEnvFromAll(ctx, envID); err != nil {
		metrics.BindEnvApmFailed(apm.WorkspaceID, envName, "unbind_old_failed")
		return errors.Wrap(err, "remove env from all apms")
	}

	// 2. Establish new association
	if err := s.store.BindEnv(ctx, apm.ID, envID, envName); err != nil {
		metrics.BindEnvApmFailed(apm.WorkspaceID, envName, "bindenv_failed")
		return errors.Wrap(err, "add env to apm")
	}

	// 3. Write/update APM environment variables
	apmApp := &bkmapi.ApmApp{
		ID:      apm.ApmID,
		Token:   apm.Token,
		AppName: apm.Name,
	}
	if err := s.upsertApmEnvVars(ctx, apm.WorkspaceID, envName, apmApp); err != nil {
		metrics.BindEnvApmFailed(apm.WorkspaceID, envName, "upsert_envvars_failed")
		return errors.Wrap(err, "upsert apm env vars")
	}

	return nil
}

func (s *ApmService) upsertApmEnvVars(
	ctx context.Context,
	workspaceID string,
	envName string,
	apmData *bkmapi.ApmApp,
) error {
	if apmData == nil {
		return errors.New("apm data is nil")
	}

	// APM 采集地址在配置为空时跳过下发，避免向业务应用写入空的 API 环境变量，导致业务侧 SDK 行为不可控
	envVars := make([]envvars.ScopedEnvVar, 0, 3)
	if config.G.BkMonitor.APMEndpoint != "" {
		envVars = append(envVars, envvars.ScopedEnvVar{
			Key:         bkmsenv.EnvVarNameApmGRPCAPI,
			Value:       config.G.BkMonitor.APMEndpoint,
			IsBuiltin:   true,
			IsSensitive: false,
			Description: "APM gRPC 采集地址",
		})
	}
	if config.G.BkMonitor.APMHttpEndpoint != "" {
		envVars = append(envVars, envvars.ScopedEnvVar{
			Key:         bkmsenv.EnvVarNameApmHTTPAPI,
			Value:       config.G.BkMonitor.APMHttpEndpoint,
			IsBuiltin:   true,
			IsSensitive: false,
			Description: "APM HTTP 采集地址",
		})
	}
	envVars = append(envVars, envvars.ScopedEnvVar{
		Key:         bkmsenv.EnvVarNameApmToken,
		Value:       apmData.Token,
		IsBuiltin:   true,
		IsSensitive: true,
		Description: "APM 采集 token",
	})

	return s.scopedEnvVarStore.BatchUpsertByKey(ctx, workspaceID, envvartypes.ScopeEnv(envName), envVars)
}

// CreateAndBindToEnv creates (or reuses) an APM app for the given environment and binds it.
// It performs the following steps:
// 1. Check if a local APM with the same name already exists
// 2. If not, call bkmonitor API to create the APM, then persist locally
// 3. Bind the APM to the environment and write environment variables
func (s *ApmService) CreateAndBindToEnv(
	ctx context.Context,
	envID bson.ObjectID,
	envName, bcsProjectCode string,
	param CreateApmInstParams,
) (*ApmInstConfig, error) {
	// 1. Check local store for existing APM with same name
	apm, err := s.store.GetByName(ctx, param.WorkspaceID, envName)
	if err != nil && !errors.Is(err, ErrApmInstConfigNotFound) {
		return nil, errors.Wrap(err, "get apm by name")
	}

	// 2. If not found locally, create via remote API and persist
	if apm == nil {
		client, cErr := bkmapi.New(param.Username)
		if cErr != nil {
			return nil, errors.Wrap(cErr, "new bkmonitor client")
		}
		apmData, cErr := client.GetOrCreate(
			ctx,
			param.BkmProjectID,
			bcsProjectCode,
			envName,
			envName,
			param.Username,
			param.WorkspaceID,
		)
		if cErr != nil {
			return nil, errors.Wrap(cErr, "create or get apm app")
		}
		apm, err = s.Get(ctx, apmData.ID, param)
		if err != nil {
			return nil, errors.Wrap(err, "get or create apm from remote")
		}
	}

	// 3. Bind to environment
	if err = s.BindToEnv(ctx, apm, envID, envName); err != nil {
		return nil, err
	}

	// 4. Re-read the latest record (BindToEnv has updated associatedEnvs) to ensure consistent return data
	apm, err = s.store.GetByApmID(ctx, apm.ApmID)
	if err != nil {
		return nil, errors.Wrap(err, "get apm after binding")
	}

	return apm, nil
}
