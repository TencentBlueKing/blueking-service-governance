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

// Package app storage module for the arrangement package, it helps with data persistence.
package app

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// The name of the MongoDB collection for storing application data.
const collectionName = "applications"

// use a single instance of Validate, it caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

var (
	ErrAppNotFound = errors.New("application not found")
	// ErrAppAlreadyExists 应用 id 或（同一工作空间内的）名称与已有应用冲突
	ErrAppAlreadyExists = errors.New("application already exists")
)

// ApplicationStore 应用的 GenericStore
type ApplicationStore = store.GenericStore[Application]

// AppMeta 应用的存储元信息：入库前校验必填字段，Helm 仓库密码加密存储。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：id
// - 唯一：workspaceID + name
var AppMeta = store.Meta[Application]{
	Collection: collectionName,
	NotFound:   ErrAppNotFound,
	Duplicate:  ErrAppAlreadyExists,
	BeforeSave: []func(*Application) error{validateApp},
	Encode:     []func(*Application) error{encryptHelmRepoPassword},
	Decode:     []func(*Application) error{decryptHelmRepoPassword},
}

// ByID 按应用 id 匹配。应用 id 是业务主键，与 MongoDB 自动生成的 _id 无关，不能使用 store.ByID
func ByID(id string) store.Opt { return store.Where("id", id) }

// InWorkspace 匹配工作空间内的应用
func InWorkspace(workspaceID string) store.Opt { return store.Where("workspaceID", workspaceID) }

// ByName 按工作空间与应用名称匹配
func ByName(workspaceID, name string) store.Opt {
	return store.Combine(InWorkspace(workspaceID), store.Where("name", name))
}

// OfType 按应用类型匹配，如 AppTypeTRPC、AppTypeHelm。
// type=trpc/taf 同时匹配 framework 对应的 type=bkmsApp 应用
func OfType(appType string) store.Opt {
	query := appruntime.ListTypeQuery(appType)
	if !query.Expand {
		return store.Where("type", query.ExactType)
	}
	return store.Where("$or", bson.A{
		bson.M{"type": query.LegacyType},
		bson.M{"type": appruntime.AppTypeBKMSApp, "framework": query.Framework},
	})
}

// CreateApp 创建应用。与已有应用冲突时返回的错误满足 errors.Is(err, ErrAppAlreadyExists)，
// 错误信息中指明冲突的是 id 还是名称
func CreateApp(ctx context.Context, db *mongo.Database, app *Application) error {
	err := store.Of(db, AppMeta).Create(ctx, app)
	if errors.Is(err, ErrAppAlreadyExists) {
		// 集合上有 id、workspaceID + name 两个唯一索引，只能从驱动返回的错误信息中区分
		dupKey := lo.Ternary(strings.Contains(err.Error(), "id_1 dup key"), "id", "name")
		return errors.Wrapf(err, "application with the same %s already exists", dupKey)
	}
	return errors.Wrapf(err, "create application %s", app.ID)
}

// GetAppsByIDs gets applications by id list in one query.
// The returned slice is in the same order as ids.
// If any requested id is missing from storage, returns an error wrapping ErrAppNotFound.
func GetAppsByIDs(ctx context.Context, db *mongo.Database, ids []string) ([]*Application, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	apps, err := store.Of(db, AppMeta).List(ctx, store.Where("id", bson.M{"$in": lo.Uniq(ids)}))
	if err != nil {
		return nil, errors.Wrap(err, "list applications by ids")
	}
	byID := lo.SliceToMap(apps, func(a Application) (string, *Application) { return a.ID, &a })

	out := make([]*Application, 0, len(ids))
	for _, id := range ids {
		a, ok := byID[id]
		if !ok {
			return nil, errors.Wrapf(ErrAppNotFound, "get app %s", id)
		}
		out = append(out, a)
	}
	return out, nil
}

// RemoveVisibleEnvName pulls envName from visibleEnvNames of all apps in the workspace.
func RemoveVisibleEnvName(ctx context.Context, db *mongo.Database, workspaceID, envName string) error {
	_, err := store.Of(db, AppMeta).ModifyMany(
		ctx,
		bson.M{"$pull": bson.M{"visibleEnvNames": envName}},
		InWorkspace(workspaceID),
		store.Where("visibleEnvNames", envName),
	)
	return errors.Wrapf(err, "remove visible env %s from workspace %s apps", envName, workspaceID)
}

type appCountByWorkspace struct {
	WorkspaceID string `bson:"_id"`
	Count       int    `bson:"count"`
}

// CountByWorkspaceIDs returns application counts grouped by workspace ID.
func CountByWorkspaceIDs(ctx context.Context, db *mongo.Database, workspaceIDs []string) (map[string]int, error) {
	if len(workspaceIDs) == 0 {
		return map[string]int{}, nil
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"workspaceID": bson.M{"$in": lo.Uniq(workspaceIDs)}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$workspaceID",
			"count": bson.M{"$sum": 1},
		}}},
	}
	results, err := store.Aggregate[appCountByWorkspace](ctx, store.Of(db, AppMeta), pipeline)
	if err != nil {
		return nil, errors.Wrap(err, "aggregate application counts by workspace IDs")
	}

	counts := make(map[string]int, len(results))
	for _, item := range results {
		counts[item.WorkspaceID] = item.Count
	}
	return counts, nil
}

func validateApp(app *Application) error {
	return errors.Wrap(validate.Struct(app), "application validation failed")
}

func encryptHelmRepoPassword(app *Application) error {
	return transformHelmRepoPassword(app, crypto.AESEncrypt)
}

func decryptHelmRepoPassword(app *Application) error {
	return transformHelmRepoPassword(app, crypto.AESDecrypt)
}

// transformHelmRepoPassword encrypts or decrypts the helm repo password, the only sensitive field in Application
func transformHelmRepoPassword(app *Application, fn func(key, data string) (string, error)) (err error) {
	if app.HelmSpec == nil || app.HelmSpec.HelmSource == nil || app.HelmSpec.HelmSource.HelmRepoConfig == nil {
		return nil
	}
	repoConfig := app.HelmSpec.HelmSource.HelmRepoConfig
	repoConfig.Password, err = crypto.OnDemand(config.G.Encrypt.Secret, fn, repoConfig.Password)
	return errors.Wrap(err, "transform helm repo password")
}
