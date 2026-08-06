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

package appmodel

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// AppEnvVarService manages app-defined workload env vars stored in AppModel.
type AppEnvVarService struct {
	appModelStore AppModelStore
}

// AppEnvVarUpdateData defines the updatable fields of an app-defined env var.
type AppEnvVarUpdateData struct {
	Key string
	// Value is optional. Nil keeps the existing value; a non-nil pointer updates it, including to an empty string.
	Value       *string
	Description string
	// IsSensitive is optional. Nil keeps the existing value unchanged.
	IsSensitive *bool
}

// NewAppEnvVarService creates an app env var service.
func NewAppEnvVarService(appModelStore AppModelStore) *AppEnvVarService {
	return &AppEnvVarService{appModelStore: appModelStore}
}

// List lists app-defined env vars in workload.envVars.
func (s *AppEnvVarService) List(ctx context.Context, appID string) ([]Variable, error) {
	return s.appModelStore.ListAppDefinedEnvVars(ctx, appID)
}

// Create creates a new app-defined env var.
func (s *AppEnvVarService) Create(ctx context.Context, appID string, envVar Variable) (*Variable, error) {
	if err := envvartypes.ValidateEnvVarKey(envVar.Key); err != nil {
		return nil, ErrInvalidEnvVarKey
	}
	now := time.Now()
	envVar.CreatedAt = now
	envVar.UpdatedAt = now
	if err := s.appModelStore.AddAppDefinedEnvVar(ctx, appID, envVar); err != nil {
		return nil, err
	}
	return &envVar, nil
}

// Update updates an existing app-defined env var and supports renaming the key.
func (s *AppEnvVarService) Update(
	ctx context.Context,
	appID string,
	key string,
	updateData AppEnvVarUpdateData,
) (*Variable, *Variable, error) {
	envVars, err := s.appModelStore.ListAppDefinedEnvVars(ctx, appID)
	if err != nil {
		return nil, nil, err
	}

	oldIdx := findEnvVarIndex(envVars, key)
	if oldIdx < 0 {
		return nil, nil, ErrEnvVarNotFound
	}

	if updateData.Key != key && findEnvVarIndex(envVars, updateData.Key) >= 0 {
		return nil, nil, ErrEnvVarKeyExists
	}
	if err := envvartypes.ValidateEnvVarKey(updateData.Key); err != nil {
		return nil, nil, ErrInvalidEnvVarKey
	}

	oldEnvVar := envVars[oldIdx]
	updated := Variable{
		Key:         updateData.Key,
		Value:       oldEnvVar.Value,
		Description: updateData.Description,
		IsSensitive: oldEnvVar.IsSensitive,
	}
	if updateData.Value != nil {
		updated.Value = *updateData.Value
	}
	if updateData.IsSensitive != nil {
		updated.IsSensitive = *updateData.IsSensitive
	}

	// Sensitive env vars cannot be downgraded to non-sensitive; they can only be
	// deleted and recreated.
	if oldEnvVar.IsSensitive && !updated.IsSensitive {
		return nil, nil, ErrEnvVarSensitivityImmutable
	}

	updated.CreatedAt = oldEnvVar.CreatedAt
	updated.UpdatedAt = time.Now()

	if err := s.appModelStore.UpdateAppDefinedEnvVar(ctx, appID, key, updated); err != nil {
		return nil, nil, err
	}
	return &oldEnvVar, &updated, nil
}

// Delete deletes an existing app-defined env var by key.
func (s *AppEnvVarService) Delete(ctx context.Context, appID, key string) (*Variable, error) {
	envVars, err := s.appModelStore.ListAppDefinedEnvVars(ctx, appID)
	if err != nil {
		return nil, err
	}

	idx := findEnvVarIndex(envVars, key)
	if idx < 0 {
		return nil, ErrEnvVarNotFound
	}

	removedEnvVar := envVars[idx]
	if err := s.appModelStore.RemoveAppDefinedEnvVar(ctx, appID, key); err != nil {
		return nil, err
	}
	return &removedEnvVar, nil
}

// BatchUpsert 按 key 批量创建或更新应用自定义环境变量。
// 已存在的 key 会原位更新，未出现在本次请求中的旧变量会保留，新 key 按请求顺序追加。
// 敏感变量不能从敏感降级为非敏感。
func (s *AppEnvVarService) BatchUpsert(ctx context.Context, appID string, vars []Variable) error {
	if len(vars) == 0 {
		return nil
	}

	existingEnvVars, err := s.appModelStore.ListAppDefinedEnvVars(ctx, appID)
	if err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(vars))
	existingByKey := make(map[string]Variable, len(existingEnvVars))
	for _, item := range existingEnvVars {
		existingByKey[item.Key] = item
	}

	now := time.Now()
	for _, item := range vars {
		if err := envvartypes.ValidateEnvVarKey(item.Key); err != nil {
			return ErrInvalidEnvVarKey
		}
		if _, ok := seen[item.Key]; ok {
			return fmt.Errorf("duplicate env var key in batch: %s", item.Key)
		}
		seen[item.Key] = struct{}{}

		merged := Variable{
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
			IsSensitive: item.IsSensitive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if existing, ok := existingByKey[item.Key]; ok {
			// 覆盖已有变量时，沿用原始 CreatedAt 和敏感性语义，让批量导入的
			// 行为等价于“就地更新”。
			if existing.IsSensitive && !item.IsSensitive {
				return ErrEnvVarSensitivityImmutable
			}
			merged.IsSensitive = existing.IsSensitive || item.IsSensitive
			merged.CreatedAt = existing.CreatedAt
			if err := s.appModelStore.UpdateAppDefinedEnvVar(ctx, appID, item.Key, merged); err != nil {
				return err
			}
			continue
		}

		// 全新的 key 按请求里的出现顺序逐条追加，避免批量导入退化成整份
		// AppModel Replace，同时保持导出顺序稳定可预测。
		if err := s.appModelStore.AddAppDefinedEnvVar(ctx, appID, merged); err != nil {
			return err
		}
	}
	return nil
}

func findEnvVarIndex(envVars []Variable, key string) int {
	_, idx, ok := lo.FindIndexOf(envVars, func(envVar Variable) bool {
		return envVar.Key == key
	})
	if !ok {
		return -1
	}
	return idx
}
