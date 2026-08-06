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

package appspec

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"
	resourcessection "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec/sections/resources"
)

// GetEnv gets app spec for a specific environment.
func GetEnv(ctx context.Context, store AppSpecStore, appID, envName string) (*AppSpec, error) {
	spec, err := store.Get(ctx, appID, envName)
	if err != nil {
		if errors.Is(err, ErrAppSpecNotFound) {
			return nil, err
		}
		return nil, errors.Wrapf(err, "getting app spec for app %s env %s", appID, envName)
	}
	return spec, nil
}

// GetEnvEffective gets the effective app spec for a specific environment.
func GetEnvEffective(
	ctx context.Context,
	store AppSpecStore,
	appModelStore appmodel.AppModelStore,
	appID, envName string,
) (*AppSpec, error) {
	defaultSpec, err := GetDefault(ctx, store, appModelStore, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting default app spec for app %s", appID)
	}

	envSpec, err := GetEnv(ctx, store, appID, envName)
	if err != nil {
		if errors.Is(err, ErrAppSpecNotFound) {
			effective := cloneSpec(defaultSpec)
			effective.EnvName = envName
			return effective, nil
		}
		return nil, errors.Wrapf(err, "getting env app spec for app %s env %s", appID, envName)
	}
	return Merge(defaultSpec, envSpec), nil
}

// SetEnv stores app spec for a specific environment.
func SetEnv(ctx context.Context, store AppSpecStore, spec *AppSpec) error {
	if spec == nil {
		return errors.New("app spec is nil")
	}
	if spec.EnvName == "" {
		return errors.New("envName must not be empty, use SetDefault for default spec")
	}

	if err := validate.Struct(spec); err != nil {
		return wrapValidationErr(err)
	}

	if err := store.Upsert(ctx, spec); err != nil {
		return errors.Wrapf(err, "upserting app spec for app %s env %s", spec.AppID, spec.EnvName)
	}
	return nil
}

// DeleteEnv deletes app spec for a specific environment.
func DeleteEnv(ctx context.Context, store AppSpecStore, appID, envName string) error {
	if err := store.Delete(ctx, appID, envName); err != nil {
		if errors.Is(err, ErrAppSpecNotFound) {
			return nil
		}
		return errors.Wrapf(err, "deleting app spec for app %s env %s", appID, envName)
	}
	return nil
}

// Misc functions

// SetReplicas sets only replicas for a specific environment.
func SetReplicas(ctx context.Context, store AppSpecStore, appID, envName string, replicas int32) error {
	if err := resourcessection.ValidateReplicas(&replicas); err != nil {
		return wrapValidationErr(err)
	}
	if err := SetEnvSection(
		ctx,
		store,
		appID,
		envName,
		ResourcesSection,
		&ResourcesSpec{Replicas: &replicas},
		SectionWriteModePatch,
	); err != nil {
		return errors.Wrapf(err, "setting replicas for app %s env %s", appID, envName)
	}
	return nil
}
