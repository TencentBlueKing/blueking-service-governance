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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var (
	// ErrApmInstConfigNotFound is returned when an APM instance config record is not found.
	ErrApmInstConfigNotFound = errors.New("ApmInstConfig not found")
	// ErrApmInstConfigExists is returned when an APM instance config with the same apmID already exists.
	ErrApmInstConfigExists = errors.New("ApmInstConfig already exists")
)

// ApmInstConfigStore APM 实例配置存储
type ApmInstConfigStore = store.GenericStore[ApmInstConfig]

// ApmInstConfigMeta APM 实例配置的存储元数据，写入完整文档前会校验必填字段。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：apmID
var ApmInstConfigMeta = store.Meta[ApmInstConfig]{
	Collection: "bkmonitor_apm_inst_config",
	NotFound:   ErrApmInstConfigNotFound,
	Duplicate:  ErrApmInstConfigExists,
	BeforeSave: []func(*ApmInstConfig) error{validateApmInstConfig},
}

func validateApmInstConfig(apm *ApmInstConfig) error {
	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(apm); err != nil {
		return errors.Wrap(err, "ApmInstConfig validation failed")
	}
	// 存为空数组而不是 null，$addToSet / $pull 不能作用于 null
	if apm.AssociatedEnvs == nil {
		apm.AssociatedEnvs = []EnvInfo{}
	}
	return nil
}

// ApmInWorkspace 按所属工作空间匹配
func ApmInWorkspace(workspaceID string) store.Opt {
	return store.Where("workspaceID", workspaceID)
}

// ApmByApmID 按蓝鲸监控中的 APM ID 匹配
func ApmByApmID(apmID int64) store.Opt {
	return store.Where("apmID", apmID)
}

// ApmByName 按工作空间与 APM 名称匹配
func ApmByName(workspaceID, name string) store.Opt {
	return store.Combine(ApmInWorkspace(workspaceID), store.Where("name", name))
}

// ApmBoundToEnv 匹配关联了指定环境的 APM
func ApmBoundToEnv(envID bson.ObjectID) store.Opt {
	return store.Where("associatedEnvs.envID", envID)
}

// GetApmIDMap 返回工作空间下的全部 APM 实例配置，key 为 APM ID
func GetApmIDMap(ctx context.Context, db *mongo.Database, workspaceID string) (map[int64]ApmInstConfig, error) {
	list, err := store.Of(db, ApmInstConfigMeta).List(ctx, ApmInWorkspace(workspaceID))
	if err != nil {
		return nil, errors.Wrapf(err, "list apm inst configs of workspace %s", workspaceID)
	}
	return lo.SliceToMap(list, func(item ApmInstConfig) (int64, ApmInstConfig) { return item.ApmID, item }), nil
}

// BindEnv 将环境关联到指定的 APM 实例配置，已关联时不重复添加；APM 不存在时返回 ErrApmInstConfigNotFound
func BindEnv(ctx context.Context, db *mongo.Database, id, envID bson.ObjectID, envName string) error {
	_, err := store.Of(db, ApmInstConfigMeta).Modify(ctx,
		bson.M{"$addToSet": bson.M{"associatedEnvs": EnvInfo{EnvID: envID, EnvName: envName}}},
		store.ByID(id),
	)
	return errors.Wrapf(err, "bind env %s to apm %s", envID.Hex(), id.Hex())
}

// UnbindEnvFromAll 解除环境与所有 APM 实例配置的关联。
// 正常情况下一个环境只关联一个 APM，这里作用于全部匹配的记录，以便清理历史脏数据
func UnbindEnvFromAll(ctx context.Context, db *mongo.Database, envID bson.ObjectID) error {
	_, err := store.Of(db, ApmInstConfigMeta).ModifyMany(ctx,
		bson.M{"$pull": bson.M{"associatedEnvs": bson.M{"envID": envID}}},
		ApmBoundToEnv(envID),
	)
	return errors.Wrapf(err, "unbind env %s from all apms", envID.Hex())
}
