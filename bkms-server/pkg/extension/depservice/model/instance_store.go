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

package model

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
)

const serviceInstanceCollName = "depservice_instances"

const configValueKey = "value"

// ErrInstanceNameExists 同一 workspace / serviceName 下实例名已存在
var ErrInstanceNameExists = errors.New("service instance name already exists")

// use a single instance of Validate, it caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	validate.RegisterStructValidation(validateServiceInstance, ServiceInstance{})
	validate.RegisterStructValidation(validateSvcInstUpdateData, SvcInstUpdateData{})
}

func validateServiceInstance(sl validator.StructLevel) {
	inst, ok := sl.Current().Interface().(ServiceInstance)
	if !ok {
		return
	}
	if err := validateScope(inst.ScopeType, inst.ScopeValue); err != nil {
		sl.ReportError(inst.ScopeValue, "ScopeValue", "scopeValue", "invalid_scope", err.Error())
	}
}

func validateSvcInstUpdateData(sl validator.StructLevel) {
	data, ok := sl.Current().Interface().(SvcInstUpdateData)
	if !ok {
		return
	}
	if err := validateScope(data.ScopeType, data.ScopeValue); err != nil {
		sl.ReportError(data.ScopeValue, "ScopeValue", "scopeValue", "invalid_scope", err.Error())
	}
}

// SvcInstQueryOptions is the query options for service instance
type SvcInstQueryOptions struct {
	WorkspaceID string
	ServiceName string

	// EnvName 当非空时(通常与 EnvType 一起填充), 过滤出对该环境可见的实例:
	// - ScopeType=workspace 的实例
	// - ScopeType=envType 且 ScopeValue=EnvType 的实例
	// - ScopeType=env     且 ScopeValue=EnvName 的实例
	EnvName string
	EnvType string

	// Status 当非空时, 只返回该状态的实例。
	Status InstanceStatus

	// ScopeType 当非空时, 只返回该作用域类型的实例。
	ScopeType ScopeType
}

// SvcInstUpdateData is the update data for service instance
type SvcInstUpdateData struct {
	ScopeType   ScopeType      `bson:"scopeType"`
	ScopeValue  string         `bson:"scopeValue"`
	Config      map[string]any `bson:"config"`
	Credentials map[string]any `bson:"credentials"`
	Operator    string         `bson:"operator"`
}

// ServiceInstanceStore defines persistence operations for service instances.
type ServiceInstanceStore interface {
	// Create creates a new service instance
	//
	// - ctx: The context object for cancellation and timeout
	// - inst: The service instance to create
	//
	// Return the id of the created service instance and an error if the operation fails.
	Create(ctx context.Context, inst *ServiceInstance) (bson.ObjectID, error)
	// Get gets a service instance by id
	//
	// - ctx: The context object for cancellation and timeout
	// - id: The id of the service instance
	//
	// Return the service instance and an error if the operation fails.
	Get(ctx context.Context, id bson.ObjectID) (*ServiceInstance, error)
	// ListByIDs 按 ID 批量查询服务实例；找不到的 ID 不会出现在结果中。
	ListByIDs(ctx context.Context, ids []bson.ObjectID) ([]*ServiceInstance, error)
	// List lists service instances
	//
	// - ctx: The context object for cancellation and timeout
	// - opts: The query options
	//
	// Return the list of service instances and an error if any.
	List(ctx context.Context, opts *SvcInstQueryOptions) ([]*ServiceInstance, error)
	// Update updates a service instance
	//
	// - ctx: The context object for cancellation and timeout
	// - id: The id of the service instance
	// - updateData: The update data for the service instance
	//
	// Return an error if the operation fails.
	Update(ctx context.Context, id bson.ObjectID, updateData *SvcInstUpdateData) error
	// UpdateConfig 整量替换服务实例的 Config。
	//
	// - ctx: The context object for cancellation and timeout
	// - id: The id of the service instance
	// - config: The config of the service instance to update
	//
	// Return an error if the operation fails.
	UpdateConfig(ctx context.Context, id bson.ObjectID, config map[string]any) error
	// PatchConfig 将 patch 合并进现有 Config（同名 key 覆盖），再写回。
	// Config 在库中以 JSON blob 存储，因此实现为读改写，而非 Mongo 点路径原子更新。
	PatchConfig(ctx context.Context, id bson.ObjectID, patch map[string]any) error
	// UpdateCredentials 整量替换服务实例的 Credentials。
	//
	// - ctx: The context object for cancellation and timeout
	// - id: The id of the service instance
	// - credentials: The credentials of the service instance to update
	//
	// Return an error if the operation fails.
	UpdateCredentials(ctx context.Context, id bson.ObjectID, credentials map[string]any) error
	// PatchCredentials 将 patch 合并进现有 Credentials（同名 key 覆盖），再写回。
	// Credentials 加密后整段存储，因此实现为读改写。
	PatchCredentials(ctx context.Context, id bson.ObjectID, patch map[string]any) error
	// UpdateStatus updates the status of a service instance
	//
	// - ctx: The context object for cancellation and timeout
	// - id: The id of the service instance
	// - status: The status of the service instance to update
	// - message: The message of the service instance to update
	//
	// Return an error if the operation fails.
	UpdateStatus(ctx context.Context, id bson.ObjectID, status InstanceStatus, message string) error
	// Delete deletes a service instance
	//
	// - ctx: The context object for cancellation and timeout
	// - id: The id of the service instance to delete
	//
	// Return an error if the operation fails.
	Delete(ctx context.Context, id bson.ObjectID) error
	// DeleteAll deletes all service instances while preserving the collection and its indexes.
	// Attention: only used in unit test
	DeleteAll(ctx context.Context) error
}

var _ ServiceInstanceStore = &ServiceInstanceStoreMongo{}

// ServiceInstanceStoreMongo implements ServiceInstanceStore interface with mongodb
type ServiceInstanceStoreMongo struct {
	collection *mongo.Collection
}

// NewServiceInstanceStoreMongo creates a new ServiceInstanceStoreMongo
func NewServiceInstanceStoreMongo(client *mongo.Client, dbName string) (ServiceInstanceStore, error) {
	coll := client.Database(dbName).Collection(serviceInstanceCollName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：workspaceID + serviceName + name
	return &ServiceInstanceStoreMongo{collection: coll}, nil
}

// Create creates a new service instance
//
// - ctx: The context object for cancellation and timeout
// - inst: The service instance to create
//
// Return the id of the created service instance and an error if the operation fails.
func (s *ServiceInstanceStoreMongo) Create(ctx context.Context, inst *ServiceInstance) (bson.ObjectID, error) {
	if err := validate.Struct(inst); err != nil {
		return bson.NilObjectID, errors.Wrap(err, "service instance validation failed")
	}

	if inst.CreatedAt.IsZero() {
		inst.CreatedAt = time.Now()
	}
	inst.UpdatedAt = inst.CreatedAt

	dbValue, err := serviceInstancePrepDBValue(inst)
	if err != nil {
		return bson.NilObjectID, errors.Wrap(err, "prep db value")
	}

	result, err := s.collection.InsertOne(ctx, dbValue)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.NilObjectID, ErrInstanceNameExists
		}
		return bson.NilObjectID, err
	}
	return result.InsertedID.(bson.ObjectID), nil
}

// Get gets a service instance by id
//
// - ctx: The context object for cancellation and timeout
// - id: The id of the service instance
//
// Return the service instance and an error if the operation fails.
func (s *ServiceInstanceStoreMongo) Get(ctx context.Context, id bson.ObjectID) (*ServiceInstance, error) {
	instance := new(ServiceInstance)

	err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(instance)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, NewNotFoundError(fmt.Sprintf("service instance(id:%s)", id))
		}
		return nil, err
	}

	dbValue, err := serviceInstanceFromDBValue(instance)
	if err != nil {
		return nil, errors.Wrap(err, "from db value")
	}

	return dbValue, nil
}

// ListByIDs 按 ID 批量查询服务实例
func (s *ServiceInstanceStoreMongo) ListByIDs(
	ctx context.Context,
	ids []bson.ObjectID,
) ([]*ServiceInstance, error) {
	ids = lo.Uniq(lo.Filter(ids, func(id bson.ObjectID, _ int) bool {
		return !id.IsZero()
	}))
	if len(ids) == 0 {
		return []*ServiceInstance{}, nil
	}

	cursor, err := s.collection.Find(ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // nolint

	insts := make([]*ServiceInstance, 0)
	if err = cursor.All(ctx, &insts); err != nil {
		return nil, err
	}

	result := make([]*ServiceInstance, 0, len(insts))
	for _, inst := range insts {
		dbValue, err := serviceInstanceFromDBValue(inst)
		if err != nil {
			return nil, errors.Wrap(err, "from db value")
		}
		result = append(result, dbValue)
	}
	return result, nil
}

// List lists service instances
//
// - ctx: The context object for cancellation and timeout
// - opts: The query options
//
// Return the list of service instances and an error if any.
func (s *ServiceInstanceStoreMongo) List(
	ctx context.Context,
	opts *SvcInstQueryOptions,
) ([]*ServiceInstance, error) {
	filter := bson.M{}

	if opts == nil {
		opts = &SvcInstQueryOptions{}
	}

	if opts.WorkspaceID != "" {
		filter["workspaceID"] = opts.WorkspaceID
	}
	if opts.ServiceName != "" {
		filter["serviceName"] = opts.ServiceName
	}
	if opts.Status != "" {
		filter["status"] = opts.Status
	}
	if opts.ScopeType != "" {
		filter["scopeType"] = opts.ScopeType
	}
	// 按环境维度过滤可见实例: workspace 全局可见、envType 命中、env 命中, 三者任一即可
	if opts.EnvName != "" || opts.EnvType != "" {
		scopeOr := []bson.M{
			{"scopeType": ScopeTypeWorkspace},
		}
		if opts.EnvType != "" {
			scopeOr = append(scopeOr, bson.M{
				"scopeType":  ScopeTypeEnvType,
				"scopeValue": opts.EnvType,
			})
		}
		if opts.EnvName != "" {
			scopeOr = append(scopeOr, bson.M{
				"scopeType":  ScopeTypeEnv,
				"scopeValue": opts.EnvName,
			})
		}
		filter["$or"] = scopeOr
	}

	sort := bson.D{{Key: "updatedAt", Value: -1}}
	findOptions := options.Find().SetSort(sort)

	cursor, err := s.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) // nolint

	insts := make([]*ServiceInstance, 0)
	if err = cursor.All(ctx, &insts); err != nil {
		return nil, err
	}

	result := make([]*ServiceInstance, 0, len(insts))
	for _, inst := range insts {
		dbValue, err := serviceInstanceFromDBValue(inst)
		if err != nil {
			return nil, errors.Wrap(err, "from db value")
		}
		result = append(result, dbValue)
	}

	return result, nil
}

// Update updates a service instance
//
// - ctx: The context object for cancellation and timeout
// - id: The id of the service instance
// - updateData: The update data for the service instance
//
// Return an error if the operation fails.
func (s *ServiceInstanceStoreMongo) Update(
	ctx context.Context,
	id bson.ObjectID,
	updateData *SvcInstUpdateData,
) error {
	if err := validate.Struct(updateData); err != nil {
		return errors.Wrap(err, "service instance update data validation failed")
	}

	inst := &ServiceInstance{Config: updateData.Config, Credentials: updateData.Credentials}
	dbValue, err := serviceInstancePrepDBValue(inst)
	if err != nil {
		return errors.Wrap(err, "prep db value")
	}

	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"scopeType":   updateData.ScopeType,
			"scopeValue":  updateData.ScopeValue,
			"config":      dbValue.Config,
			"credentials": dbValue.Credentials,
			"operator":    updateData.Operator,
			"updatedAt":   time.Now(),
		},
	}
	opts := options.UpdateOne().SetUpsert(false)

	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return NewNotFoundError(fmt.Sprintf("service instance(id:%s)", id))
	}
	return err
}

// UpdateConfig 整量替换服务实例的 Config。
func (s *ServiceInstanceStoreMongo) UpdateConfig(
	ctx context.Context,
	id bson.ObjectID,
	config map[string]any,
) error {
	inst := &ServiceInstance{Config: config}
	dbValue, err := serviceInstancePrepDBValue(inst)
	if err != nil {
		return errors.Wrap(err, "prep db value")
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"config": dbValue.Config, "updatedAt": time.Now()}}
	opts := options.UpdateOne().SetUpsert(false)

	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return NewNotFoundError(fmt.Sprintf("service instance(id:%s)", id))
	}
	return err
}

// PatchConfig 将 patch 合并进现有 Config 后写回。
func (s *ServiceInstanceStoreMongo) PatchConfig(
	ctx context.Context,
	id bson.ObjectID,
	patch map[string]any,
) error {
	if len(patch) == 0 {
		return nil
	}
	inst, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.UpdateConfig(ctx, id, lo.Assign(map[string]any{}, inst.Config, patch))
}

// UpdateCredentials 整量替换服务实例的 Credentials。
func (s *ServiceInstanceStoreMongo) UpdateCredentials(
	ctx context.Context,
	id bson.ObjectID,
	credentials map[string]any,
) error {
	inst := &ServiceInstance{Credentials: credentials}
	dbValue, err := serviceInstancePrepDBValue(inst)
	if err != nil {
		return errors.Wrap(err, "prep db value")
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"credentials": dbValue.Credentials, "updatedAt": time.Now()}}
	opts := options.UpdateOne().SetUpsert(false)

	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return NewNotFoundError(fmt.Sprintf("service instance(id:%s)", id))
	}
	return err
}

// PatchCredentials 将 patch 合并进现有 Credentials 后写回。
func (s *ServiceInstanceStoreMongo) PatchCredentials(
	ctx context.Context,
	id bson.ObjectID,
	patch map[string]any,
) error {
	if len(patch) == 0 {
		return nil
	}
	inst, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.UpdateCredentials(ctx, id, lo.Assign(map[string]any{}, inst.Credentials, patch))
}

// UpdateStatus updates the status of a service instance
//
// - ctx: The context object for cancellation and timeout
// - id: The id of the service instance
// - status: The status of the service instance to update
// - message: The message of the service instance to update
//
// Return an error if the operation fails.
func (s *ServiceInstanceStoreMongo) UpdateStatus(
	ctx context.Context,
	id bson.ObjectID,
	status InstanceStatus,
	message string,
) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"status": status, "message": message, "updatedAt": time.Now()}}
	opts := options.UpdateOne().SetUpsert(false)

	ret, err := s.collection.UpdateOne(ctx, filter, update, opts)
	if ret != nil && ret.MatchedCount == 0 {
		return NewNotFoundError(fmt.Sprintf("service instance(id:%s)", id))
	}
	return err
}

// Delete deletes a service instance
//
// - ctx: The context object for cancellation and timeout
// - id: The id of the service instance to delete
//
// Return an error if the operation fails.
func (s *ServiceInstanceStoreMongo) Delete(ctx context.Context, id bson.ObjectID) error {
	_, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteAll deletes all service instances while preserving the collection and its indexes.
// Attention: only used in unit test
func (s *ServiceInstanceStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}

// serviceInstancePrepDBValue prepares the service instance to save in db
// Storing a map[string]any directly in MongoDB converts it to bson.D,
// unmarshalling this back can be inconvenient, so we serialize it to a json string first.
// - marshals the config to map {"value": "config json string"} to save in db
// - marshals and encrypts the credentials to map {"value": "credentials json string"} to save in db
func serviceInstancePrepDBValue(instance *ServiceInstance) (*ServiceInstance, error) {
	dbValue := new(ServiceInstance)
	if err := copier.CopyWithOption(dbValue, instance, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		return nil, err
	}

	if c, err := json.Marshal(instance.Config); err != nil {
		return nil, err
	} else {
		dbValue.Config = map[string]any{configValueKey: string(c)}
	}

	creds, err := json.Marshal(instance.Credentials)
	if err != nil {
		return nil, err
	}

	encryptCreds, err := crypto.AESEncrypt(config.G.Encrypt.Secret, string(creds))
	if err != nil {
		return nil, err
	}
	dbValue.Credentials = map[string]any{configValueKey: encryptCreds}

	return dbValue, nil
}

// serviceInstanceFromDBValue converts the service instance from db service instance value. it is the reverse of
// serviceInstancePrepDBValue.
// - unmarshals the config from map {"value": "config json string"} to map[string]any
// - decrypts and unmarshals the credentials from map {"value": "credentials json string"} to map[string]any
func serviceInstanceFromDBValue(dbValue *ServiceInstance) (*ServiceInstance, error) {
	instance := new(ServiceInstance)
	if err := copier.CopyWithOption(instance, dbValue, copier.Option{IgnoreEmpty: true, DeepCopy: true}); err != nil {
		return nil, err
	}
	c := make(map[string]any)

	if err := json.Unmarshal([]byte(cast.ToString(instance.Config[configValueKey])), &c); err != nil {
		return nil, err
	} else {
		instance.Config = c
	}

	creds, err := crypto.AESDecrypt(config.G.Encrypt.Secret, cast.ToString(instance.Credentials[configValueKey]))
	if err != nil {
		return nil, err
	}
	credsMap := make(map[string]any)
	if err = json.Unmarshal([]byte(creds), &credsMap); err != nil {
		return nil, err
	}
	instance.Credentials = credsMap

	return instance, nil
}
