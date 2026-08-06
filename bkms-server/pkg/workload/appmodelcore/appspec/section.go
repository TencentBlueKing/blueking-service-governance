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

// GetDefaultSection gets a single section from the default AppSpec.
func GetDefaultSection[T any](
	ctx context.Context,
	store AppSpecStore,
	appModelStore appmodel.AppModelStore,
	appID string,
	section SectionHandle[T],
) (*T, error) {
	spec, err := GetDefault(ctx, store, appModelStore, appID)
	if err != nil {
		return nil, err
	}
	return section.getCloned(spec), nil
}

// GetEnvSection gets a single section from the raw env AppSpec override.
func GetEnvSection[T any](
	ctx context.Context,
	store AppSpecStore,
	appID, envName string,
	section SectionHandle[T],
) (*T, error) {
	spec, err := GetEnv(ctx, store, appID, envName)
	if err != nil {
		return nil, err
	}
	return section.getCloned(spec), nil
}

// GetEnvEffectiveSection gets a single section from the effective env AppSpec.
func GetEnvEffectiveSection[T any](
	ctx context.Context,
	store AppSpecStore,
	appModelStore appmodel.AppModelStore,
	appID, envName string,
	section SectionHandle[T],
) (*T, error) {
	spec, err := GetEnvEffective(ctx, store, appModelStore, appID, envName)
	if err != nil {
		return nil, err
	}
	return section.getCloned(spec), nil
}

// SetDefaultSection writes one section on the default AppSpec and syncs only that section to AppModel.
func SetDefaultSection[T any](
	ctx context.Context,
	store AppSpecStore,
	appModelStore appmodel.AppModelStore,
	appID string,
	section SectionHandle[T],
	input *T,
	mode SectionWriteMode,
) error {
	// Ensure the default document is materialized before section-scoped updates, otherwise
	// a replace/patch write would create a partial default spec and lose untouched defaults.
	if _, err := GetDefault(ctx, store, appModelStore, appID); err != nil {
		return errors.Wrapf(err, "getting default app spec for appID %q", appID)
	}

	return writeDefaultSection(ctx, store, appModelStore, appID, section, input, mode)
}

// SetEnvSection writes one section on an env AppSpec override.
func SetEnvSection[T any](
	ctx context.Context,
	store AppSpecStore,
	appID, envName string,
	section SectionHandle[T],
	input *T,
	mode SectionWriteMode,
) error {
	if envName == "" {
		return errors.New("envName must not be empty, use SetDefaultSection for default spec")
	}
	return writeEnvSection(ctx, store, appID, envName, section, input, mode)
}

// writeDefaultSection writes one section on the default AppSpec and syncs only that section to AppModel.
//
// NOTE: The store write and AppModel sync are NOT atomic. If the store write succeeds but the
// AppModel sync fails, the two stores will be inconsistent. See SetDefault for details.
func writeDefaultSection[T any](
	ctx context.Context,
	store AppSpecStore,
	appModelStore appmodel.AppModelStore,
	appID string,
	section SectionHandle[T],
	input *T,
	mode SectionWriteMode,
) error {
	target := &AppSpec{AppID: appID, EnvName: DefaultEnvName}
	section.setCloned(target, input)
	if err := validate.Struct(target); err != nil {
		return wrapValidationErr(err)
	}

	switch mode {
	case SectionWriteModeReplace:
		if err := store.SetSections(ctx, *target, section.id()); err != nil {
			return errors.Wrapf(err, "setting section %s for default app spec appID %q", section.id(), appID)
		}

		// sync to AppModel
		if err := replaceDefaultSectionToAppModel(ctx, appModelStore, appID, section, input); err != nil {
			return errors.Wrapf(err, "syncing section %s to app model appID %q", section.id(), appID)
		}
	case SectionWriteModePatch:
		if section.getCloned(target) == nil {
			return nil
		}
		if err := store.Patch(ctx, *target); err != nil {
			return errors.Wrapf(err, "patching section %s for default app spec appID %q", section.id(), appID)
		}

		// sync to AppModel
		updatedValue, err := GetDefaultSection(ctx, store, appModelStore, appID, section)
		if err != nil {
			return errors.Wrapf(err, "getting patched section %s for default app spec appID %q", section.id(), appID)
		}
		if err := replaceDefaultSectionToAppModel(ctx, appModelStore, appID, section, updatedValue); err != nil {
			return errors.Wrapf(err, "patching section %s to app model appID %q", section.id(), appID)
		}
	default:
		return errors.Errorf("unsupported section write mode %q", mode)
	}
	return nil
}

// writeEnvSection writes one section on an env AppSpec override.
func writeEnvSection[T any](
	ctx context.Context,
	store AppSpecStore,
	appID, envName string,
	section SectionHandle[T],
	input *T,
	mode SectionWriteMode,
) error {
	target := &AppSpec{AppID: appID, EnvName: envName}
	section.setCloned(target, input)
	if err := validate.Struct(target); err != nil {
		return wrapValidationErr(err)
	}

	switch mode {
	case SectionWriteModeReplace:
		if err := store.SetSections(ctx, *target, section.id()); err != nil {
			return errors.Wrapf(err, "setting section %s for app %s env %s", section.id(), appID, envName)
		}
	case SectionWriteModePatch:
		if section.getCloned(target) == nil {
			return nil
		}
		if err := store.Patch(ctx, *target); err != nil {
			return errors.Wrapf(err, "patching section %s for app %s env %s", section.id(), appID, envName)
		}
	default:
		return errors.Errorf("unsupported section write mode %q", mode)
	}
	return nil
}

// replaceDefaultSectionToAppModel replaces one section on AppModel with the given value.
func replaceDefaultSectionToAppModel[T any](
	ctx context.Context,
	appModelStore appmodel.AppModelStore,
	appID string,
	section SectionHandle[T],
	value *T,
) error {
	appModel, err := appModelStore.GetAppModel(ctx, appID)
	if err != nil {
		return errors.Wrapf(err, "getting app model for appID %q", appID)
	}

	if section.driver.ApplyToAppModel != nil {
		section.driver.ApplyToAppModel(section.driver.Clone(value), appModel)
	}
	if err := appModelStore.UpdateAppModel(ctx, appModel); err != nil {
		return errors.Wrapf(err, "updating app model for appID %q", appID)
	}
	return nil
}
