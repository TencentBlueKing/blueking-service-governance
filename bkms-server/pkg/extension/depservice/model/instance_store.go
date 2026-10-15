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

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const serviceInstanceCollName = "depservice_instances"

const configValueKey = "value"

var (
	// ErrInstanceNameExists 同一 workspace / serviceName 下实例名已存在
	ErrInstanceNameExists = errors.New("service instance name already exists")
	// ErrInstanceNotFound 服务实例不存在，AsNotFoundError 同样将其视为未找到
	ErrInstanceNotFound = errors.New("service instance not found")
)

// use a single instance of Validate, it caches struct info
var validate = validator.New(validator.WithRequiredStructEnabled())

func init() {
	validate.RegisterStructValidation(validateServiceInstanceScope, ServiceInstance{})
}

func validateServiceInstanceScope(sl validator.StructLevel) {
	inst, ok := sl.Current().Interface().(ServiceInstance)
	if !ok {
		return
	}
	if err := validateScope(inst.ScopeType, inst.ScopeValue); err != nil {
		sl.ReportError(inst.ScopeValue, "ScopeValue", "scopeValue", "invalid_scope", err.Error())
	}
}

// ServiceInstanceStore 依赖服务实例存储
type ServiceInstanceStore = store.GenericStore[ServiceInstance]

// ServiceInstanceMeta 依赖服务实例的存储元数据。
//
// MongoDB 中直接存储 map[string]any 会被解码为 bson.D，读回后不便使用，因此由 Encode / Decode 钩子转换：
// - config 在库中保存为 {"value": "<config 的 json 串>"}
// - credentials 在库中保存为 {"value": "<credentials 的 json 串加密后的结果>"}
//
// 索引（由 golang-migrate 维护）：
// - 唯一：workspaceID + serviceName + name
var ServiceInstanceMeta = store.Meta[ServiceInstance]{
	Collection: serviceInstanceCollName,
	NotFound:   ErrInstanceNotFound,
	Duplicate:  ErrInstanceNameExists,
	BeforeSave: []func(*ServiceInstance) error{validateServiceInstance},
	Encode:     []func(*ServiceInstance) error{encodeInstanceSecrets},
	Decode:     []func(*ServiceInstance) error{decodeInstanceSecrets},
}

func validateServiceInstance(inst *ServiceInstance) error {
	return errors.Wrap(validate.Struct(inst), "service instance validation failed")
}

// encodeInstanceSecrets 也会在只填写了部分字段的对象上执行（Update），此时未写入的字段的编码结果会被丢弃
func encodeInstanceSecrets(inst *ServiceInstance) error {
	cfg, err := json.Marshal(inst.Config)
	if err != nil {
		return errors.Wrap(err, "marshal config")
	}
	creds, err := json.Marshal(inst.Credentials)
	if err != nil {
		return errors.Wrap(err, "marshal credentials")
	}
	encryptedCreds, err := crypto.AESEncrypt(config.G.Encrypt.Secret, string(creds))
	if err != nil {
		return errors.Wrap(err, "encrypt credentials")
	}
	inst.Config = map[string]any{configValueKey: string(cfg)}
	inst.Credentials = map[string]any{configValueKey: encryptedCreds}
	return nil
}

func decodeInstanceSecrets(inst *ServiceInstance) error {
	cfg := make(map[string]any)
	if err := json.Unmarshal([]byte(cast.ToString(inst.Config[configValueKey])), &cfg); err != nil {
		return errors.Wrap(err, "unmarshal config")
	}
	creds, err := crypto.AESDecrypt(config.G.Encrypt.Secret, cast.ToString(inst.Credentials[configValueKey]))
	if err != nil {
		return errors.Wrap(err, "decrypt credentials")
	}
	credsMap := make(map[string]any)
	if err = json.Unmarshal([]byte(creds), &credsMap); err != nil {
		return errors.Wrap(err, "unmarshal credentials")
	}
	inst.Config, inst.Credentials = cfg, credsMap
	return nil
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

// Cond 将查询选项转换为查询条件，为空的选项不参与过滤
func (o *SvcInstQueryOptions) Cond() store.Opt {
	filter := bson.M{}
	if o == nil {
		return store.Filter(filter)
	}
	if o.WorkspaceID != "" {
		filter["workspaceID"] = o.WorkspaceID
	}
	if o.ServiceName != "" {
		filter["serviceName"] = o.ServiceName
	}
	if o.Status != "" {
		filter["status"] = o.Status
	}
	if o.ScopeType != "" {
		filter["scopeType"] = o.ScopeType
	}
	// 按环境维度过滤可见实例: workspace 全局可见、envType 命中、env 命中, 三者任一即可
	if o.EnvName != "" || o.EnvType != "" {
		scopeOr := []bson.M{
			{"scopeType": ScopeTypeWorkspace},
		}
		if o.EnvType != "" {
			scopeOr = append(scopeOr, bson.M{
				"scopeType":  ScopeTypeEnvType,
				"scopeValue": o.EnvType,
			})
		}
		if o.EnvName != "" {
			scopeOr = append(scopeOr, bson.M{
				"scopeType":  ScopeTypeEnv,
				"scopeValue": o.EnvName,
			})
		}
		filter["$or"] = scopeOr
	}
	return store.Filter(filter)
}

// InstancesByIDs 按 ID 批量匹配服务实例，找不到的 ID 不会出现在结果中
func InstancesByIDs(ids []bson.ObjectID) store.Opt {
	// $in 不接受 null，lo.Uniq 总是返回非 nil 的切片
	return store.ByID(bson.M{"$in": lo.Uniq(ids)})
}

// RecentlyUpdatedFirst 按更新时间倒序排列服务实例
func RecentlyUpdatedFirst() store.Opt {
	return store.Combine(store.Desc("updatedAt"), store.Desc("_id"))
}

// UpdateInstanceStatus 更新服务实例的状态与状态详情
func UpdateInstanceStatus(
	ctx context.Context,
	db *mongo.Database,
	id bson.ObjectID,
	status InstanceStatus,
	message string,
) error {
	return store.Of(db, ServiceInstanceMeta).Update(
		ctx,
		map[string]any{"status": status, "message": message},
		store.ByID(id),
	)
}

// PatchInstanceConfig 将 patch 合并进现有 Config（同名 key 覆盖），再写回。
// Config 在库中以 JSON blob 存储，因此实现为读改写，而非 Mongo 点路径原子更新。
func PatchInstanceConfig(ctx context.Context, db *mongo.Database, id bson.ObjectID, patch map[string]any) error {
	return patchInstanceField(ctx, db, id, "config", patch, func(inst *ServiceInstance) map[string]any {
		return inst.Config
	})
}

// PatchInstanceCredentials 将 patch 合并进现有 Credentials（同名 key 覆盖），再写回。
// Credentials 加密后整段存储，因此实现为读改写。
func PatchInstanceCredentials(ctx context.Context, db *mongo.Database, id bson.ObjectID, patch map[string]any) error {
	return patchInstanceField(ctx, db, id, "credentials", patch, func(inst *ServiceInstance) map[string]any {
		return inst.Credentials
	})
}

func patchInstanceField(
	ctx context.Context,
	db *mongo.Database,
	id bson.ObjectID,
	field string,
	patch map[string]any,
	current func(*ServiceInstance) map[string]any,
) error {
	if len(patch) == 0 {
		return nil
	}
	insts := store.Of(db, ServiceInstanceMeta)
	inst, err := insts.Get(ctx, store.ByID(id))
	if err != nil {
		return errors.Wrapf(err, "get service instance %s", id.Hex())
	}
	merged := lo.Assign(map[string]any{}, current(inst), patch)
	return errors.Wrapf(
		insts.Update(ctx, map[string]any{field: merged}, store.ByID(id)),
		"update %s of service instance %s", field, id.Hex(),
	)
}
