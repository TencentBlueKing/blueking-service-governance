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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const appSpecCollectionName = "app_specs"

// AppSpecStore is the generic store of AppSpec.
type AppSpecStore = store.GenericStore[AppSpec]

// AppSpecMeta 描述 AppSpec 的存储方式。写入与读取时都会经过 Clone，使空的 section 统一折叠为 nil。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + envName
var AppSpecMeta = store.Meta[AppSpec]{
	Collection: appSpecCollectionName,
	NotFound:   ErrAppSpecNotFound,
	Duplicate:  ErrAppSpecAlreadyExists,
	BeforeSave: []func(*AppSpec) error{Validate},
	Encode:     []func(*AppSpec) error{normalizeSpec},
	Decode:     []func(*AppSpec) error{normalizeSpec},
}

// ByApp matches all app specs of an application, including the default (envName="") entry.
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByAppEnv matches the app spec of an application in an environment, use DefaultEnvName for the default spec.
func ByAppEnv(appID, envName string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("envName", envName))
}

// ListEnvNamesByApp returns non-default environment names with configured sections, sorted by name.
func ListEnvNamesByApp(ctx context.Context, db *mongo.Database, appID string) ([]string, error) {
	specs, err := store.Of(db, AppSpecMeta).List(
		ctx,
		ByApp(appID),
		store.Where("envName", bson.M{"$ne": DefaultEnvName}),
		store.Asc("envName"),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "listing app specs of app %s", appID)
	}
	return lo.FilterMap(specs, func(spec AppSpec, _ int) (string, bool) {
		return spec.EnvName, spec.HasConfiguredSections()
	}), nil
}

// Patch updates the non-nil fields of an application specification, creating it when absent.
func Patch(ctx context.Context, db *mongo.Database, spec AppSpec) error {
	scoped := Clone(&spec)
	if err := Validate(scoped); err != nil {
		return err
	}

	// 各 section 以嵌套路径给出待更新的字段，例如 "resources.replicas"
	set := bson.D{}
	for _, section := range registeredSections {
		section.appendPatch(&set, scoped)
	}
	if len(set) == 0 {
		return nil
	}

	fields := make(map[string]any, len(set))
	for _, e := range set {
		fields[e.Key] = e.Value
	}
	if _, err := store.Of(db, AppSpecMeta).Upsert(ctx, fields, ByAppEnv(scoped.AppID, scoped.EnvName)); err != nil {
		return errors.Wrapf(err, "patching app spec for app %s env %q", scoped.AppID, scoped.EnvName)
	}
	return nil
}

// SetSections replaces or removes the specified top-level application specification sections,
// leaving all other sections intact. The document is created only when some section is set.
func SetSections(ctx context.Context, db *mongo.Database, spec AppSpec, sectionIDs ...AppSpecSectionID) error {
	scoped := Clone(&spec)
	if err := Validate(scoped); err != nil {
		return err
	}

	// 值为 nil 的 section 会被删除
	fields := make(map[string]any, len(sectionIDs))
	hasSet := false
	for _, sectionID := range sectionIDs {
		section, ok := getSection(sectionID)
		if !ok {
			return errors.Errorf("unknown app spec section %q", sectionID)
		}
		value := section.wholeValue(scoped)
		fields[sectionID.String()] = value
		hasSet = hasSet || value != nil
	}
	if len(fields) == 0 {
		return nil
	}

	s, filter := store.Of(db, AppSpecMeta), ByAppEnv(scoped.AppID, scoped.EnvName)
	var err error
	if hasSet {
		_, err = s.Upsert(ctx, fields, filter)
	} else {
		// 只删除 section 时不需要创建文档
		err = store.IgnoreNotFound(s.Update(ctx, fields, filter))
	}
	if err != nil {
		return errors.Wrapf(err, "setting sections %v for app %s env %q", sectionIDs, scoped.AppID, scoped.EnvName)
	}
	return nil
}

func normalizeSpec(spec *AppSpec) error {
	*spec = *Clone(spec)
	return nil
}
