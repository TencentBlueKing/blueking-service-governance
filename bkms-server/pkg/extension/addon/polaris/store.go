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

package polaris

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// The name of the MongoDB collection for storing polaris config data.
const collectionName = "polaris_configs"

// 环境维度子文档的根字段名，key 均为环境名。
const (
	envStatesField         = "envStates"
	envWeightsField        = "envWeights"
	envDynamicWeightsField = "envDynamicWeights"
)

// envSettingFields 登记全部「环境级配置」根字段：由用户按环境设置、共享同一套生命周期，
// 随环境离域或卸载一起清理。新增这类字段时在此登记，清理路径无需再改。
//
// envStates 不属于此列——它记录的是系统观测到的部署事实，清理时机与写入方都不同。
var envSettingFields = []string{envWeightsField, envDynamicWeightsField}

// 错误定义
var (
	// ErrConfigNotFound 北极星配置不存在
	ErrConfigNotFound = errors.New("polaris config not found")
	// ErrConfigNameExists 北极星配置名称已存在
	ErrConfigNameExists = errors.New("polaris config name already exists")
	// ErrOperatorEmpty 不允许将负责人清空
	ErrOperatorEmpty = errors.New("operator cannot be empty")
	// ErrOperatorNotManaged 仅平台创建的北极星配置允许修改负责人
	ErrOperatorNotManaged = errors.New("operator can only be updated for platform-created polaris services")
)

// PolarisConfigStore 北极星配置存储接口
type PolarisConfigStore interface {
	// Create 创建北极星配置
	Create(ctx context.Context, config *PolarisConfig) error
	// Get 根据应用 ID 和配置名称获取北极星配置
	Get(ctx context.Context, appID, name string) (*PolarisConfig, error)
	// ListByApp 获取应用下的所有北极星配置
	ListByApp(ctx context.Context, appID string) ([]*PolarisConfig, error)
	// ListByEnv 获取应用在指定环境下可用的北极星配置列表
	ListByEnv(ctx context.Context, appID, envName string) ([]*PolarisConfig, error)
	// Update 根据应用 ID 和配置名称更新北极星配置
	Update(ctx context.Context, appID, name string, updateData *ConfigUpdateData) error
	// UpsertEnvState 幂等新增或更新指定环境的信息
	UpsertEnvState(ctx context.Context, appID, name, envName string, update PolarisEnvStateUpdate) error
	// UpsertEnvStateIfUpdatedAtMatch 仅当配置的顶层 updatedAt 与期望值匹配时更新环境信息。
	// 返回值表示是否匹配到期望版本；版本不匹配时不会返回错误，也不会写入数据。
	UpsertEnvStateIfUpdatedAtMatch(
		ctx context.Context,
		appID, name, envName string,
		expectedUpdatedAt time.Time,
		update PolarisEnvStateUpdate,
	) (bool, error)
	// UpsertEnvWeight 幂等设置指定环境的单实例权重与动态权重开关。
	// dynamicWeight 为 nil 表示不改开关；两个字段在同一次更新中写入，不会出现中间态
	UpsertEnvWeight(ctx context.Context, appID, name, envName string, weight int32, dynamicWeight *bool) error
	// RemoveEnvStates 幂等移除多个指定环境的信息
	RemoveEnvStates(ctx context.Context, appID, name string, envNames []string) error
	// RemoveEnvSettings 幂等移除多个指定环境的全部环境级配置（见 envSettingFields），
	RemoveEnvSettings(ctx context.Context, appID, name string, envNames []string) error
	// Delete 根据应用 ID 和配置名称删除北极星配置
	Delete(ctx context.Context, appID, name string) error
	// DeleteByApp 删除应用下的所有北极星配置（仅测试使用）
	DeleteByApp(ctx context.Context, appID string) error
}

var _ PolarisConfigStore = &PolarisConfigStoreMongo{}

// PolarisEnvStateUpdate 定义单个环境信息的可选更新字段。
type PolarisEnvStateUpdate struct {
	AppliedFields *RedeployRequiredFields
	LastError     *string
}

// PolarisConfigStoreMongo 北极星配置的 MongoDB 存储实现
type PolarisConfigStoreMongo struct {
	collection *mongo.Collection
}

// NewPolarisConfigStoreMongo 创建北极星配置存储
func NewPolarisConfigStoreMongo(client *mongo.Client, dbName string) (PolarisConfigStore, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + name
	return &PolarisConfigStoreMongo{collection: coll}, nil
}

// Create 创建北极星配置
func (s *PolarisConfigStoreMongo) Create(ctx context.Context, config *PolarisConfig) error {
	now := time.Now()
	if config.Name == "" {
		config.Name = config.GenerateName()
	}
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	if config.UpdatedAt.IsZero() {
		config.UpdatedAt = now
	}
	if config.ServiceLabels == nil {
		config.ServiceLabels = make(map[string]string)
	}
	if config.EnvStates == nil {
		config.EnvStates = make(map[string]PolarisEnvState)
	}

	_, err := s.collection.InsertOne(ctx, config)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrConfigNameExists
		}
		return errors.Wrap(err, "insert polaris config")
	}
	return nil
}

// Get 根据应用 ID 和配置名称获取北极星配置
func (s *PolarisConfigStoreMongo) Get(ctx context.Context, appID, name string) (*PolarisConfig, error) {
	var config PolarisConfig
	err := s.collection.FindOne(ctx, bson.M{"appID": appID, "name": name}).Decode(&config)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrConfigNotFound
		}
		return nil, errors.Wrap(err, "find polaris config by name")
	}
	return &config, nil
}

// ListByApp 获取应用下的所有北极星配置
func (s *PolarisConfigStoreMongo) ListByApp(ctx context.Context, appID string) ([]*PolarisConfig, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"appID": appID})
	if err != nil {
		return nil, errors.Wrap(err, "find polaris configs")
	}
	defer cursor.Close(ctx)

	var configs []*PolarisConfig
	if err = cursor.All(ctx, &configs); err != nil {
		return nil, errors.Wrap(err, "decode polaris configs")
	}

	return configs, nil
}

// Update 根据应用 ID 和配置名称更新北极星配置
func (s *PolarisConfigStoreMongo) Update(
	ctx context.Context,
	appID, name string,
	updateData *ConfigUpdateData,
) error {
	if updateData == nil {
		return nil
	}

	updateSet := bson.M{}
	needUpdate := false

	if updateData.InstanceKey != nil {
		updateSet["instanceKey"] = *updateData.InstanceKey
		needUpdate = true
	}
	if updateData.ServicePort != nil {
		updateSet["servicePort"] = *updateData.ServicePort
		needUpdate = true
	}
	if updateData.Direct != nil {
		updateSet["direct"] = *updateData.Direct
		needUpdate = true
	}
	if updateData.KeepNotReadyPod != nil {
		updateSet["keepNotReadyPod"] = *updateData.KeepNotReadyPod
		needUpdate = true
	}
	if updateData.EnableHealthCheck != nil {
		updateSet["enableHealthCheck"] = *updateData.EnableHealthCheck
		needUpdate = true
	}
	if updateData.EnableWeightFactor != nil {
		updateSet["enableWeightFactor"] = *updateData.EnableWeightFactor
		needUpdate = true
	}
	if updateData.ServiceLabels != nil {
		updateSet["serviceLabels"] = updateData.ServiceLabels
		needUpdate = true
	}
	if updateData.PolarisToken != nil {
		updateSet["polarisToken"] = *updateData.PolarisToken
		needUpdate = true
	}
	if updateData.Operator != nil {
		updateSet["operator"] = *updateData.Operator
		needUpdate = true
	}
	if updateData.ScopeEnvNames != nil {
		updateSet["scopeEnvNames"] = updateData.ScopeEnvNames
		needUpdate = true
	}
	if updateData.envWeights != nil {
		updateSet[envWeightsField] = updateData.envWeights
		needUpdate = true
	}
	if updateData.envDynamicWeights != nil {
		updateSet[envDynamicWeightsField] = updateData.envDynamicWeights
		needUpdate = true
	}
	if !needUpdate {
		return nil
	}

	updateSet["updatedAt"] = time.Now()

	filter := bson.M{"appID": appID, "name": name}
	result, err := s.collection.UpdateOne(ctx, filter, bson.M{"$set": updateSet})
	if err != nil {
		return errors.Wrap(err, "update polaris config")
	}
	if result.MatchedCount == 0 {
		return ErrConfigNotFound
	}

	return nil
}

// Delete 根据应用 ID 和配置名称删除北极星配置
func (s *PolarisConfigStoreMongo) Delete(ctx context.Context, appID, name string) error {
	filter := bson.M{"appID": appID, "name": name}
	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "delete polaris config")
	}
	if result.DeletedCount == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// UpsertEnvState 幂等新增或更新指定环境的信息。
func (s *PolarisConfigStoreMongo) UpsertEnvState(
	ctx context.Context,
	appID, name, envName string,
	update PolarisEnvStateUpdate,
) error {
	fieldPrefix, err := envFieldPrefix(envStatesField, envName)
	if err != nil {
		return err
	}
	setFields := bson.M{fieldPrefix + ".updatedAt": time.Now()}
	if update.AppliedFields != nil {
		setFields[fieldPrefix+".appliedFields"] = update.AppliedFields
	}
	if update.LastError != nil {
		setFields[fieldPrefix+".lastError"] = *update.LastError
	}

	result, err := s.collection.UpdateOne(ctx,
		bson.M{"appID": appID, "name": name},
		bson.M{"$set": setFields},
	)
	if err != nil {
		return errors.Wrap(err, "upsert polaris env state")
	}
	if result.MatchedCount == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// UpsertEnvStateIfUpdatedAtMatch 仅在配置顶层 updatedAt 仍与 expectedUpdatedAt 匹配时更新环境信息。
// 版本不匹配通常表示任务处理的是旧配置，此时返回 false 并跳过写入。
func (s *PolarisConfigStoreMongo) UpsertEnvStateIfUpdatedAtMatch(
	ctx context.Context,
	appID, name, envName string,
	expectedUpdatedAt time.Time,
	update PolarisEnvStateUpdate,
) (bool, error) {
	fieldPrefix, err := envFieldPrefix(envStatesField, envName)
	if err != nil {
		return false, err
	}
	setFields := bson.M{fieldPrefix + ".updatedAt": time.Now()}
	if update.AppliedFields != nil {
		setFields[fieldPrefix+".appliedFields"] = update.AppliedFields
	}
	if update.LastError != nil {
		setFields[fieldPrefix+".lastError"] = *update.LastError
	}

	result, err := s.collection.UpdateOne(ctx,
		bson.M{"appID": appID, "name": name, "updatedAt": expectedUpdatedAt},
		bson.M{"$set": setFields},
	)
	if err != nil {
		return false, errors.Wrap(err, "conditionally upsert polaris env state")
	}
	return result.MatchedCount > 0, nil
}

// UpsertEnvWeight 幂等设置指定环境的单实例权重与动态权重开关。
// 两者在同一次更新中写入，保证集群侧的单次 Patch 与库侧记录一致。
func (s *PolarisConfigStoreMongo) UpsertEnvWeight(
	ctx context.Context,
	appID, name, envName string,
	weight int32,
	dynamicWeight *bool,
) error {
	weightPath, err := envFieldPrefix(envWeightsField, envName)
	if err != nil {
		return err
	}
	setFields := bson.M{
		weightPath:  weight,
		"updatedAt": time.Now(),
	}
	if dynamicWeight != nil {
		dynamicWeightPath, pathErr := envFieldPrefix(envDynamicWeightsField, envName)
		if pathErr != nil {
			return pathErr
		}
		setFields[dynamicWeightPath] = *dynamicWeight
	}

	result, err := s.collection.UpdateOne(ctx,
		bson.M{"appID": appID, "name": name},
		bson.M{"$set": setFields},
	)
	if err != nil {
		return errors.Wrap(err, "upsert polaris env weight")
	}
	if result.MatchedCount == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// RemoveEnvStates 幂等移除多个指定环境的信息。
func (s *PolarisConfigStoreMongo) RemoveEnvStates(
	ctx context.Context, appID, name string, envNames []string,
) error {
	return s.removeEnvFields(ctx, appID, name, []string{envStatesField}, envNames)
}

// RemoveEnvSettings 幂等移除多个指定环境的全部环境级配置。
func (s *PolarisConfigStoreMongo) RemoveEnvSettings(
	ctx context.Context, appID, name string, envNames []string,
) error {
	return s.removeEnvFields(ctx, appID, name, envSettingFields, envNames)
}

// removeEnvFields 从 roots 指向的各个环境维度子文档中 unset 多个环境条目。
// 所有路径在同一次更新中提交，避免部分字段清理成功、部分残留。
func (s *PolarisConfigStoreMongo) removeEnvFields(
	ctx context.Context, appID, name string, roots, envNames []string,
) error {
	if len(envNames) == 0 {
		return nil
	}

	unsetFields := make(bson.M, len(envNames)*len(roots))
	for _, root := range roots {
		for _, envName := range envNames {
			fieldPath, err := envFieldPrefix(root, envName)
			if err != nil {
				return err
			}
			unsetFields[fieldPath] = ""
		}
	}
	_, err := s.collection.UpdateOne(ctx,
		bson.M{"appID": appID, "name": name},
		bson.M{"$unset": unsetFields},
	)
	if err != nil {
		return errors.Wrapf(err, "remove polaris env fields %v", roots)
	}
	return nil
}

// envFieldPrefix 校验环境名并生成环境维度子文档的嵌套字段路径前缀
func envFieldPrefix(root, envName string) (string, error) {
	if envName == "" || strings.ContainsAny(envName, ".$") {
		return "", errors.Errorf("invalid env name %q", envName)
	}
	return root + "." + envName, nil
}

// DeleteByApp 删除应用下的所有北极星配置
func (s *PolarisConfigStoreMongo) DeleteByApp(ctx context.Context, appID string) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{"appID": appID})
	if err != nil {
		return errors.Wrap(err, "delete polaris configs by app")
	}
	return nil
}

// ListByEnv 获取应用在指定环境下可用的北极星配置列表
func (s *PolarisConfigStoreMongo) ListByEnv(ctx context.Context, appID, envName string) ([]*PolarisConfig, error) {
	configs, err := s.ListByApp(ctx, appID)
	if err != nil {
		return nil, err
	}

	result := make([]*PolarisConfig, 0, len(configs))
	for _, config := range configs {
		// 检查配置是否在当前环境中可用
		if config.IsAvailableInEnv(envName) {
			result = append(result, config)
		}
	}

	return result, nil
}
