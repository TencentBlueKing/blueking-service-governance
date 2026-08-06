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
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
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

// ScopedEnvVarUpdateData defines the updatable fields of a scoped env var.
type ScopedEnvVarUpdateData struct {
	Key string
	// Value is optional. Nil keeps the existing value; a non-nil pointer updates it, including to an empty string.
	Value       *string
	Description string
	// IsSensitive is optional. Nil means keeping the existing value unchanged.
	IsSensitive *bool
}

type scopedEnvVarListOptions struct {
	onlyBuiltin bool
	ordering    string
	scopes      []envvartypes.ScopedEnvVarScope
}

// ScopedEnvVarListOption customizes ScopedEnvVarStore.List behavior.
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

// ScopedEnvVarStore defines the storage interface for scoped env vars.
type ScopedEnvVarStore interface {
	// List lists env vars in the given workspace.
	// Use WithScopes to limit the result to one or more scopes; when WithScopes
	// is omitted, all scopes are returned (i.e. no scope filtering is applied).
	List(
		ctx context.Context,
		workspaceID string,
		opts ...ScopedEnvVarListOption,
	) ([]ScopedEnvVar, error)
	// ListPublic lists public(scopeType workspace/envType) env vars in the target workspace.
	ListPublic(ctx context.Context, workspaceID string) ([]ScopedEnvVar, error)
	// Create creates a scoped env var.
	Create(ctx context.Context, envVar ScopedEnvVar) (bson.ObjectID, error)
	// CreateSimpleEnvScopeVar creates a non-builtin, non-sensitive env-scoped var.
	CreateSimpleEnvScopeVar(
		ctx context.Context,
		environment envmodel.Environment,
		key string,
		value string,
		description string,
	) (bson.ObjectID, error)
	// BatchUpsertByKey creates or updates env vars in the given scope by key.
	// Existing vars only have value and description updated.
	BatchUpsertByKey(
		ctx context.Context,
		workspaceID string,
		scope envvartypes.ScopedEnvVarScope,
		vars []ScopedEnvVar,
	) error

	// NOTE: Below methods are limited to operate on a single workspace.

	// GetByID gets a scoped env var by ID.
	GetByID(ctx context.Context, workspaceID string, id bson.ObjectID) (*ScopedEnvVar, error)
	// UpdateByID updates a scoped env var by ID.
	UpdateByID(ctx context.Context, workspaceID string, id bson.ObjectID, updateData ScopedEnvVarUpdateData) error
	// DeleteByID deletes a scoped env var by workspace and ID.
	DeleteByID(ctx context.Context, workspaceID string, id bson.ObjectID) error
	// DeleteByEnv deletes all scoped env vars defined in the target environment scope.
	DeleteByEnv(ctx context.Context, environment envmodel.Environment) error

	// DeleteAll deletes all scoped env vars while preserving the collection and its indexes.
	// Attention: only used in unit test
	DeleteAll(ctx context.Context) error
}

var _ ScopedEnvVarStore = (*ScopedEnvVarStoreMongo)(nil)

// ScopedEnvVarStoreMongo is the MongoDB implementation of ScopedEnvVarStore.
type ScopedEnvVarStoreMongo struct {
	collection *mongo.Collection
}

// NewScopedEnvVarStoreMongo creates a new ScopedEnvVarStore backed by MongoDB.
func NewScopedEnvVarStoreMongo(client *mongo.Client, dbName string) (ScopedEnvVarStore, error) {
	coll := client.Database(dbName).Collection(ScopedEnvVarCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID + scopeType + scopeValue + key
	return &ScopedEnvVarStoreMongo{collection: coll}, nil
}

// List lists env vars in the given workspace.
// When WithScopes is omitted, all scopes are returned.
func (s *ScopedEnvVarStoreMongo) List(
	ctx context.Context,
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
	filter := bson.M{
		"workspaceID": workspaceID,
		"isBuiltin":   listOpts.onlyBuiltin,
	}

	// Build the scope filter according to the provided scopes.
	switch len(listOpts.scopes) {
	case 0:
	case 1:
		filter["scopeType"] = listOpts.scopes[0].ScopeType
		filter["scopeValue"] = listOpts.scopes[0].ScopeValue
	default:
		scopeFilters := make([]bson.M, 0, len(listOpts.scopes))
		for _, scope := range listOpts.scopes {
			scopeFilters = append(scopeFilters, bson.M{
				"scopeType":  scope.ScopeType,
				"scopeValue": scope.ScopeValue,
			})
		}
		filter["$or"] = scopeFilters
	}

	cursor, err := s.collection.Find(ctx, filter, options.Find().SetSort(scopedEnvVarSort(listOpts.ordering)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []ScopedEnvVar
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return s.decryptEnvVarList(result)
}

// ListPublic lists workspace-level and env-type-level env vars in the target workspace.
func (s *ScopedEnvVarStoreMongo) ListPublic(ctx context.Context, workspaceID string) ([]ScopedEnvVar, error) {
	filter := bson.M{
		"workspaceID": workspaceID,
		"scopeType": bson.M{
			"$in": []envvartypes.ScopeType{envvartypes.ScopeTypeWorkspace, envvartypes.ScopeTypeEnvType},
		},
		"isBuiltin": false,
	}
	cursor, err := s.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var result []ScopedEnvVar
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	result, err = s.decryptEnvVarList(result)
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

func scopedEnvVarSort(ordering string) bson.D {
	switch ordering {
	case "created":
		return bson.D{{Key: "createdAt", Value: 1}}
	case "key":
		fallthrough
	default:
		return bson.D{{Key: "key", Value: 1}}
	}
}

// GetByID gets a scoped env var by workspaceID and ID.
func (s *ScopedEnvVarStoreMongo) GetByID(
	ctx context.Context, workspaceID string, id bson.ObjectID,
) (*ScopedEnvVar, error) {
	var result ScopedEnvVar
	if err := s.collection.FindOne(ctx, bson.M{"_id": id, "workspaceID": workspaceID}).Decode(&result); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrScopedEnvVarNotFound
		}
		return nil, err
	}

	decryptedValue, err := s.decryptEnvVar(result.Value)
	if err != nil {
		return nil, errors.Wrapf(err, "decrypt variable %s", result.Key)
	}
	result.Value = decryptedValue
	return &result, nil
}

// Create creates a scoped env var.
func (s *ScopedEnvVarStoreMongo) Create(ctx context.Context, envVar ScopedEnvVar) (bson.ObjectID, error) {
	now := time.Now()
	envVar.CreatedAt = now
	envVar.UpdatedAt = now

	// 加密变量值
	encryptedValue, err := s.encryptEnvVar(envVar.Value)
	if err != nil {
		return bson.NilObjectID, errors.Wrapf(err, "encrypt variable %s", envVar.Key)
	}
	envVar.Value = encryptedValue

	ret, err := s.collection.InsertOne(ctx, envVar)
	if err != nil {
		// The unique index is on (workspaceID, scopeType, scopeValue, key),
		// so a duplicate key error means a record with the same key already
		// exists in this scope.
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, ErrScopedEnvVarKeyConflict
		}
		return bson.NilObjectID, err
	}

	oid, ok := ret.InsertedID.(bson.ObjectID)
	if !ok {
		return bson.NilObjectID, errors.New("failed to get inserted ID")
	}
	return oid, nil
}

// CreateSimpleEnvScopeVar creates a non-builtin, non-sensitive env-scoped var.
func (s *ScopedEnvVarStoreMongo) CreateSimpleEnvScopeVar(
	ctx context.Context,
	environment envmodel.Environment,
	key string,
	value string,
	description string,
) (bson.ObjectID, error) {
	return s.Create(ctx, ScopedEnvVar{
		WorkspaceID: environment.WorkspaceID,
		ScopeType:   envvartypes.ScopeTypeEnv,
		ScopeValue:  environment.Name,
		Key:         key,
		Value:       value,
		Description: description,
		IsBuiltin:   false,
		IsSensitive: false,
	})
}

// BatchUpsertByKey creates or updates env vars in the given scope by key.
func (s *ScopedEnvVarStoreMongo) BatchUpsertByKey(
	ctx context.Context,
	workspaceID string,
	scope envvartypes.ScopedEnvVarScope,
	vars []ScopedEnvVar,
) error {
	if len(vars) == 0 {
		return nil
	}

	now := time.Now()
	models := make([]mongo.WriteModel, 0, len(vars))
	for _, item := range vars {
		if item.Key == "" {
			return errors.New("scoped env var key is required for upsert")
		}

		encryptedValue, err := s.encryptEnvVar(item.Value)
		if err != nil {
			return errors.Wrapf(err, "encrypt variable %s", item.Key)
		}

		filter := bson.M{
			"workspaceID": workspaceID,
			"scopeType":   scope.ScopeType,
			"scopeValue":  scope.ScopeValue,
			"key":         item.Key,
		}
		update := bson.M{
			"$set": bson.M{
				"value":       encryptedValue,
				"description": item.Description,
				"updatedAt":   now,
			},
			"$setOnInsert": bson.M{
				"workspaceID": workspaceID,
				"scopeType":   scope.ScopeType,
				"scopeValue":  scope.ScopeValue,
				"key":         item.Key,
				"isBuiltin":   item.IsBuiltin,
				"isSensitive": item.IsSensitive,
				"createdAt":   now,
			},
		}
		models = append(models, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update).SetUpsert(true))
	}

	if _, err := s.collection.BulkWrite(ctx, models); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrScopedEnvVarKeyConflict
		}
		return errors.Wrap(err, "bulk upsert scoped env vars")
	}
	return nil
}

// UpdateByID updates a scoped env var by workspaceID and ID.
func (s *ScopedEnvVarStoreMongo) UpdateByID(
	ctx context.Context, workspaceID string, id bson.ObjectID, updateData ScopedEnvVarUpdateData,
) error {
	if id == bson.NilObjectID {
		return errors.New("scoped env var ID is required for update")
	}

	updateSet := bson.M{
		"key":         updateData.Key,
		"description": updateData.Description,
		"updatedAt":   time.Now(),
	}
	if updateData.Value != nil {
		encryptedValue, err := s.encryptEnvVar(*updateData.Value)
		if err != nil {
			return errors.Wrapf(err, "encrypt variable %s", updateData.Key)
		}
		updateSet["value"] = encryptedValue
	}
	if updateData.IsSensitive != nil {
		updateSet["isSensitive"] = *updateData.IsSensitive
	}

	opts := options.UpdateOne().SetUpsert(false)
	result, err := s.collection.UpdateOne(
		ctx,
		bson.M{"_id": id, "workspaceID": workspaceID},
		bson.M{"$set": updateSet},
		opts,
	)
	if err != nil {
		// The unique index is on (workspaceID, scopeType, scopeValue, key),
		// so a duplicate key error means a record with the same key already
		// exists in this scope.
		if mongo.IsDuplicateKeyError(err) {
			return ErrScopedEnvVarKeyConflict
		}
		return err
	}
	if result.MatchedCount == 0 {
		return ErrScopedEnvVarNotFound
	}
	return nil
}

// DeleteByID deletes a scoped env var by workspaceID and ID.
func (s *ScopedEnvVarStoreMongo) DeleteByID(ctx context.Context, workspaceID string, id bson.ObjectID) error {
	if id == bson.NilObjectID {
		return errors.New("scoped env var ID is required for delete")
	}

	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": id, "workspaceID": workspaceID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrScopedEnvVarNotFound
	}
	return nil
}

// DeleteByEnv deletes all scoped env vars defined in the target environment scope.
func (s *ScopedEnvVarStoreMongo) DeleteByEnv(ctx context.Context, environment envmodel.Environment) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{
		"workspaceID": environment.WorkspaceID,
		"scopeType":   envvartypes.ScopeTypeEnv,
		"scopeValue":  environment.Name,
	})
	return err
}

// DeleteAll deletes all scoped env vars while preserving the collection and its indexes.
// Attention: only used in unit test.
func (s *ScopedEnvVarStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}

// encryptEnvVar encrypts a variable value through a crypto function
func (s *ScopedEnvVarStoreMongo) encryptEnvVar(v string) (string, error) {
	if data, err := crypto.AESEncrypt(config.G.Encrypt.Secret, v); err != nil {
		return "", err
	} else {
		return data, nil
	}
}

// decryptEnvVar decrypts a variable value through a crypto function
func (s *ScopedEnvVarStoreMongo) decryptEnvVar(v string) (string, error) {
	if data, err := crypto.AESDecrypt(config.G.Encrypt.Secret, v); err != nil {
		return "", err
	} else {
		return data, nil
	}
}

func (s *ScopedEnvVarStoreMongo) decryptEnvVarList(vars []ScopedEnvVar) ([]ScopedEnvVar, error) {
	// 解密环境变量值
	for i, v := range vars {
		decryptedValue, decryptErr := s.decryptEnvVar(v.Value)
		if decryptErr != nil {
			return nil, errors.Wrapf(decryptErr, "decrypt variable %s", v.Key)
		}
		vars[i].Value = decryptedValue
	}
	return vars, nil
}
