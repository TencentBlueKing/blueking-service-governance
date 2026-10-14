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

package envvars

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// ScopedEnvVarCollectionName is the MongoDB collection storing scoped env vars.
const ScopedEnvVarCollectionName = "scoped_env_vars"

// ErrScopedEnvVarNotFound is returned when the target scoped env var does not exist.
var ErrScopedEnvVarNotFound = errors.New("scoped env var not found")

// ErrScopedEnvVarKeyConflict is returned when creating or updating a scoped env var would cause
// a duplicate key within the same scope. The unique index is on
// (workspaceID, scopeType, scopeValue, key), so currently only key collisions trigger this.
var ErrScopedEnvVarKeyConflict = errors.New("an env var with the same key already exists in this scope")

// ScopedEnvVarStore is the store of scoped env vars.
type ScopedEnvVarStore = store.GenericStore[ScopedEnvVar]

// ScopedEnvVarMeta describes how scoped env vars are stored. Values are encrypted before
// being written and decrypted after being read.
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + scopeType + scopeValue + key
var ScopedEnvVarMeta = store.Meta[ScopedEnvVar]{
	Collection: ScopedEnvVarCollectionName,
	NotFound:   ErrScopedEnvVarNotFound,
	Duplicate:  ErrScopedEnvVarKeyConflict,
	Encode:     []func(*ScopedEnvVar) error{encryptValue},
	Decode:     []func(*ScopedEnvVar) error{decryptValue},
}

// ScopedEnvVarUpdateData defines the updatable fields of a scoped env var.
type ScopedEnvVarUpdateData struct {
	Key string
	// Value is optional. Nil keeps the existing value; a non-nil pointer updates it, including to an empty string.
	Value       *string
	Description string
	// IsSensitive is optional. Nil means keeping the existing value unchanged.
	IsSensitive *bool
}

// InWorkspace matches env vars in the given workspace.
func InWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// InScopes matches env vars in any of the given scopes; no scope means no scope filtering.
func InScopes(scopes ...envvartypes.ScopedEnvVarScope) store.Opt {
	switch len(scopes) {
	case 0:
		return store.Combine()
	case 1:
		// 单个作用域写成等值条件，Upsert 插入时会从过滤条件中取得作用域字段
		return store.Filter(bson.M{"scopeType": scopes[0].ScopeType, "scopeValue": scopes[0].ScopeValue})
	default:
		return store.Where("$or", lo.Map(scopes, func(scope envvartypes.ScopedEnvVarScope, _ int) bson.M {
			return bson.M{"scopeType": scope.ScopeType, "scopeValue": scope.ScopeValue}
		}))
	}
}

// ByID matches the env var with the given ID. Operations by ID are limited to a single workspace.
func ByID(workspaceID string, id bson.ObjectID) store.Opt {
	return store.Combine(store.ByID(id), InWorkspace(workspaceID))
}

type scopedEnvVarListOptions struct {
	onlyBuiltin bool
	ordering    string
	scopes      []envvartypes.ScopedEnvVarScope
}

// ScopedEnvVarListOption customizes List behavior.
type ScopedEnvVarListOption func(*scopedEnvVarListOptions)

// WithOrdering sorts list results by the named field in ascending order.
// Supported values:
// - "key": environment variable key, used by default.
// - "created": creation time.
func WithOrdering(ordering string) ScopedEnvVarListOption {
	return func(opts *scopedEnvVarListOptions) {
		opts.ordering = ordering
	}
}

// WithScopes makes List return env vars in the given scopes only.
func WithScopes(scopes ...envvartypes.ScopedEnvVarScope) ScopedEnvVarListOption {
	copiedScopes := append([]envvartypes.ScopedEnvVarScope(nil), scopes...)
	return func(opts *scopedEnvVarListOptions) {
		opts.scopes = copiedScopes
	}
}

// WithOnlyBuiltin makes List return only built-in scoped env vars.
func WithOnlyBuiltin() ScopedEnvVarListOption {
	return func(opts *scopedEnvVarListOptions) {
		opts.onlyBuiltin = true
	}
}

// List lists env vars in the given workspace.
// Use WithScopes to limit the result to one or more scopes; when WithScopes
// is omitted, all scopes are returned (i.e. no scope filtering is applied).
func List(
	ctx context.Context,
	db *mongo.Database,
	workspaceID string,
	opts ...ScopedEnvVarListOption,
) ([]ScopedEnvVar, error) {
	listOpts := scopedEnvVarListOptions{ordering: "key"}
	for _, opt := range opts {
		opt(&listOpts)
	}

	// Q: 为什么这里的 filter 设计只允许查询内置或非内置，而不允许查询全部？
	//
	// A: ScopedEnvVar 中 isBuiltin 为 true 的比较特殊，它们只有在系统需要后期动态写入一些内置变量时
	// 才会被用到，比如 APM_TOKEN，且它们会被认为是和那些动态的（无需写入数据库）的内置变量同属“内置”
	// 这个分类，而非某个 Scoped 环境级变量。因此，单次查询 ScopedEnvVar 列表数据，不会同时需要两类数据。
	return store.Of(db, ScopedEnvVarMeta).List(
		ctx,
		InWorkspace(workspaceID),
		store.Where("isBuiltin", listOpts.onlyBuiltin),
		InScopes(listOpts.scopes...),
		scopedEnvVarSort(listOpts.ordering),
	)
}

// ListPublic lists workspace-level and env-type-level env vars in the target workspace.
func ListPublic(ctx context.Context, db *mongo.Database, workspaceID string) ([]ScopedEnvVar, error) {
	result, err := store.Of(db, ScopedEnvVarMeta).List(
		ctx,
		InWorkspace(workspaceID),
		store.Where("scopeType", bson.M{
			"$in": []envvartypes.ScopeType{envvartypes.ScopeTypeWorkspace, envvartypes.ScopeTypeEnvType},
		}),
		store.Where("isBuiltin", false),
	)
	if err != nil {
		return nil, err
	}

	SortPublicScopedEnvVars(result)
	return result, nil
}

// SortPublicScopedEnvVars sorts public scoped env vars by scope type and env type order.
func SortPublicScopedEnvVars(result []ScopedEnvVar) {
	// 提供一个稳定的默认排序，workspace 作用域在前，envType 作用域按核心环境类型顺序排序。
	publicScopeTypeOrder := []envvartypes.ScopeType{envvartypes.ScopeTypeWorkspace, envvartypes.ScopeTypeEnvType}
	publicEnvTypeOrder := lo.Map(bkmsenv.PublicTypeOrder(), func(t bkmsenv.Type, _ int) string {
		return string(t)
	})

	slices.SortFunc(result, func(a, b ScopedEnvVar) int {
		if c := cmp.Compare(
			slices.Index(publicScopeTypeOrder, a.ScopeType),
			slices.Index(publicScopeTypeOrder, b.ScopeType),
		); c != 0 {
			return c
		}
		if c := cmp.Compare(
			slices.Index(publicEnvTypeOrder, a.ScopeValue),
			slices.Index(publicEnvTypeOrder, b.ScopeValue),
		); c != 0 {
			return c
		}
		return cmp.Compare(a.Key, b.Key)
	})
}

func scopedEnvVarSort(ordering string) store.Opt {
	switch ordering {
	case "created":
		return store.Asc("createdAt")
	case "key":
		fallthrough
	default:
		return store.Asc("key")
	}
}

// CopyEnvVars copies only variables directly defined in the source environment.
// Public and built-in variables keep their existing resolution rules. The copies
// are new records, so later edits on either side stay independent. A failed copy
// may leave partial inserts; the caller must clean up through environment delete hooks.
func CopyEnvVars(ctx context.Context, db *mongo.Database, source, target envmodel.Environment) error {
	if source.WorkspaceID != target.WorkspaceID || source.Name == target.Name {
		return errors.New("env var copy requires distinct environments in the same workspace")
	}
	vars, err := List(ctx, db, source.WorkspaceID, WithScopes(envvartypes.ScopeEnv(source.Name)))
	if err != nil {
		return errors.Wrapf(err, "read custom variables from environment %s", source.Name)
	}
	if len(vars) == 0 {
		return nil
	}
	// Insert new records so a concurrently created key causes a conflict instead
	// of receiving a source secret while retaining a non-sensitive target flag.
	copies := lo.Map(vars, func(item ScopedEnvVar, _ int) *ScopedEnvVar {
		return &ScopedEnvVar{
			WorkspaceID: target.WorkspaceID,
			ScopeType:   envvartypes.ScopeTypeEnv,
			ScopeValue:  target.Name,
			Key:         item.Key,
			Value:       item.Value,
			Description: item.Description,
			IsSensitive: item.IsSensitive,
		}
	})
	if err = store.Of(db, ScopedEnvVarMeta).CreateMany(ctx, copies); err != nil {
		return errors.Wrapf(err, "copy custom variables to environment %s", target.Name)
	}
	return nil
}

// FeatureEnvVarCopier implements bkmsenv.FeatureEnvVarStore on top of the database.
type FeatureEnvVarCopier struct {
	db *mongo.Database
}

// NewFeatureEnvVarCopier creates a FeatureEnvVarCopier.
func NewFeatureEnvVarCopier(db *mongo.Database) *FeatureEnvVarCopier {
	return &FeatureEnvVarCopier{db: db}
}

// CopyEnvVars copies custom variables into another environment, including sensitive values.
func (c *FeatureEnvVarCopier) CopyEnvVars(ctx context.Context, source, target envmodel.Environment) error {
	return CopyEnvVars(ctx, c.db, source, target)
}

// CreateSimpleEnvScopeVar creates a non-builtin, non-sensitive env-scoped var.
func CreateSimpleEnvScopeVar(
	ctx context.Context,
	db *mongo.Database,
	environment envmodel.Environment,
	key string,
	value string,
	description string,
) (bson.ObjectID, error) {
	envVar := &ScopedEnvVar{
		WorkspaceID: environment.WorkspaceID,
		ScopeType:   envvartypes.ScopeTypeEnv,
		ScopeValue:  environment.Name,
		Key:         key,
		Value:       value,
		Description: description,
		IsBuiltin:   false,
		IsSensitive: false,
	}
	if err := store.Of(db, ScopedEnvVarMeta).Create(ctx, envVar); err != nil {
		return bson.NilObjectID, err
	}
	return envVar.ID, nil
}

// BatchUpsertByKey creates or updates env vars in the given scope by key.
// Existing vars only have value and description updated.
func BatchUpsertByKey(
	ctx context.Context,
	db *mongo.Database,
	workspaceID string,
	scope envvartypes.ScopedEnvVarScope,
	vars []ScopedEnvVar,
) error {
	objs := make([]*ScopedEnvVar, 0, len(vars))
	for _, item := range vars {
		if item.Key == "" {
			return errors.New("scoped env var key is required for upsert")
		}
		// 在副本上写入：变量按 workspace + scope + key 匹配，调用方传入的 ID 与时间不参与写入
		obj := item
		obj.ID = bson.NilObjectID
		obj.WorkspaceID, obj.ScopeType, obj.ScopeValue = workspaceID, scope.ScopeType, scope.ScopeValue
		obj.CreatedAt, obj.UpdatedAt = time.Time{}, time.Time{}
		objs = append(objs, &obj)
	}

	_, err := store.Of(db, ScopedEnvVarMeta).UpsertMany(
		ctx,
		objs,
		func(v *ScopedEnvVar) store.Opt { return store.Where("key", v.Key) },
		InWorkspace(workspaceID),
		InScopes(scope),
		store.UpdateSelected("value", "description"),
	)
	if err != nil {
		return errors.Wrap(err, "bulk upsert scoped env vars")
	}
	return nil
}

// UpdateByID updates a scoped env var by workspaceID and ID.
func UpdateByID(
	ctx context.Context,
	db *mongo.Database,
	workspaceID string,
	id bson.ObjectID,
	updateData ScopedEnvVarUpdateData,
) error {
	if id == bson.NilObjectID {
		return errors.New("scoped env var ID is required for update")
	}

	fields := map[string]any{
		"key":         updateData.Key,
		"description": updateData.Description,
	}
	if updateData.Value != nil {
		fields["value"] = *updateData.Value
	}
	if updateData.IsSensitive != nil {
		fields["isSensitive"] = *updateData.IsSensitive
	}
	return store.Of(db, ScopedEnvVarMeta).Update(ctx, fields, ByID(workspaceID, id))
}

// DeleteByEnv deletes all scoped env vars defined in the target environment scope.
func DeleteByEnv(ctx context.Context, db *mongo.Database, environment envmodel.Environment) error {
	_, err := store.Of(db, ScopedEnvVarMeta).DeleteMany(
		ctx,
		InWorkspace(environment.WorkspaceID),
		InScopes(envvartypes.ScopeEnv(environment.Name)),
	)
	return err
}

// encryptValue encrypts the variable value before it is written. Unlike crypto.OnDemand,
// an empty value is encrypted as well, so every stored value is ciphertext.
func encryptValue(v *ScopedEnvVar) (err error) {
	v.Value, err = crypto.AESEncrypt(config.G.Encrypt.Secret, v.Value)
	return errors.Wrapf(err, "encrypt variable %s", v.Key)
}

// decryptValue decrypts the variable value after it is read
func decryptValue(v *ScopedEnvVar) (err error) {
	v.Value, err = crypto.AESDecrypt(config.G.Encrypt.Secret, v.Value)
	return errors.Wrapf(err, "decrypt variable %s", v.Key)
}
