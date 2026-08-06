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
)

// GetDefault gets the default app spec for the application.
// It will read the app model and construct a base spec from it for the first time.
//
// NOTE: the lazy initialization uses a check-then-write pattern without
// distributed locking. Two concurrent calls may both construct and Upsert the default spec
// from the same AppModel. Since Upsert is idempotent and both callers derive the spec from
// the same AppModel snapshot, the result is deterministic, but a redundant write may occur.
func GetDefault(
	ctx context.Context,
	store AppSpecStore,
	appModelStore appmodel.AppModelStore,
	appID string,
) (*AppSpec, error) {
	// If the default spec already exists, return it directly.
	spec, err := store.Get(ctx, appID, DefaultEnvName)
	if err == nil {
		return spec, nil
	}
	if !errors.Is(err, ErrAppSpecNotFound) {
		return nil, errors.Wrapf(err, "getting default app spec for appID %q", appID)
	}

	// The default spec does not exist, we need to construct it from app model and persist it for future use.
	appModel, err := appModelStore.GetAppModel(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting app model for appID %q", appID)
	}
	baseSpec := FromAppModel(appID, DefaultEnvName, appModel)
	if err = store.Upsert(ctx, baseSpec); err != nil {
		return nil, errors.Wrapf(err, "persisting default app spec for appID %q", appID)
	}
	return baseSpec, nil
}

// SetDefault stores the default app spec and syncs AppModel-backed fields.
//
// NOTE: The Upsert and AppModel sync are NOT atomic. If the Upsert succeeds but the AppModel
// sync fails, the two stores will be inconsistent. Callers should be prepared to retry on
// transient errors, or run a reconciliation pass to bring the two stores back in sync.
func SetDefault(
	ctx context.Context,
	store AppSpecStore,
	appModelStore appmodel.AppModelStore,
	appID string,
	spec *AppSpec,
) error {
	if spec == nil {
		return errors.New("app spec is nil")
	}

	scoped := cloneSpec(spec)
	scoped.AppID = appID
	scoped.EnvName = DefaultEnvName
	if err := validate.Struct(scoped); err != nil {
		return wrapValidationErr(err)
	}

	if err := store.Upsert(ctx, scoped); err != nil {
		return errors.Wrapf(err, "upserting default app spec for appID %q", appID)
	}
	if err := syncDefaultToAppModel(ctx, appModelStore, appID, scoped); err != nil {
		return errors.Wrapf(err, "syncing default app spec to app model for appID %q", appID)
	}
	return nil
}

// syncDefaultToAppModel syncs AppModel-backed fields in the default app spec to app model.
func syncDefaultToAppModel(
	ctx context.Context,
	appModelStore appmodel.AppModelStore,
	appID string,
	spec *AppSpec,
) error {
	appModel, err := appModelStore.GetAppModel(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "getting app model for appID %q", appID)
	}

	// Directly apply the spec object to app model, replace all existing AppModel-backed fields to
	// avoid complicated and error-prone partial updates.
	ApplyToAppModel(spec, appModel)
	if err := appModelStore.UpdateAppModel(ctx, appModel); err != nil {
		return errors.Wrapf(err, "updating app model for appID %q", appID)
	}
	return nil
}
