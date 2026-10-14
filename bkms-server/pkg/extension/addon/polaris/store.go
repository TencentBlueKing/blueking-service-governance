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
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

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

// ConfigStore 北极星配置存储
type ConfigStore = store.GenericStore[PolarisConfig]

// ConfigMeta 北极星配置的存储元数据。
//
// 顶层 updatedAt 同时是乐观锁版本号（见 UpsertEnvStateIfUpdatedAtMatch），
// 只记录系统观测结果的环境状态写入与清理须带 store.KeepUpdatedAt()，不能推进它。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + name
var ConfigMeta = store.Meta[PolarisConfig]{
	Collection: "polaris_configs",
	NotFound:   ErrConfigNotFound,
	Duplicate:  ErrConfigNameExists,
	BeforeSave: []func(*PolarisConfig) error{fillConfigDefaults},
}

// fillConfigDefaults 补齐配置名称与空的服务标签
func fillConfigDefaults(config *PolarisConfig) error {
	if config.Name == "" {
		config.Name = config.GenerateName()
	}
	if config.ServiceLabels == nil {
		config.ServiceLabels = make(map[string]string)
	}
	return nil
}

// PolarisEnvStateUpdate 定义单个环境信息的可选更新字段。
type PolarisEnvStateUpdate struct {
	AppliedFields *RedeployRequiredFields
	LastError     *string
}

// ByApp 匹配应用下的全部北极星配置
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByName 按应用 ID 和配置名称匹配北极星配置
func ByName(appID, name string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("name", name))
}

// ListByEnv 获取应用在指定环境下可用的北极星配置列表
func ListByEnv(ctx context.Context, db *mongo.Database, appID, envName string) ([]*PolarisConfig, error) {
	configs, err := store.Of(db, ConfigMeta).List(ctx, ByApp(appID))
	if err != nil {
		return nil, errors.Wrapf(err, "list polaris configs of app %s", appID)
	}
	result := make([]*PolarisConfig, 0, len(configs))
	for i := range configs {
		// 检查配置是否在当前环境中可用
		if configs[i].IsAvailableInEnv(envName) {
			result = append(result, &configs[i])
		}
	}
	return result, nil
}

// UpdateConfig 根据应用 ID 和配置名称更新 updateData 中给出的字段；没有待更新的字段时直接返回，
// 配置不存在时返回 ErrConfigNotFound
func UpdateConfig(ctx context.Context, db *mongo.Database, appID, name string, updateData *ConfigUpdateData) error {
	fields := updateData.toUpdateFields()
	if len(fields) == 0 {
		return nil
	}
	err := store.Of(db, ConfigMeta).Update(ctx, fields, ByName(appID, name))
	return errors.Wrapf(err, "update polaris config %s of app %s", name, appID)
}

// toUpdateFields 将非空的待更新字段转换为 Update 使用的 map
func (d *ConfigUpdateData) toUpdateFields() map[string]any {
	if d == nil {
		return nil
	}
	fields := map[string]any{}
	if d.InstanceKey != nil {
		fields["instanceKey"] = *d.InstanceKey
	}
	if d.ServicePort != nil {
		fields["servicePort"] = *d.ServicePort
	}
	if d.Direct != nil {
		fields["direct"] = *d.Direct
	}
	if d.KeepNotReadyPod != nil {
		fields["keepNotReadyPod"] = *d.KeepNotReadyPod
	}
	if d.EnableHealthCheck != nil {
		fields["enableHealthCheck"] = *d.EnableHealthCheck
	}
	if d.ServiceLabels != nil {
		fields["serviceLabels"] = d.ServiceLabels
	}
	if d.PolarisToken != nil {
		fields["polarisToken"] = *d.PolarisToken
	}
	if d.Operator != nil {
		fields["operator"] = *d.Operator
	}
	if d.ScopeEnvNames != nil {
		fields["scopeEnvNames"] = d.ScopeEnvNames
	}
	if d.envWeights != nil {
		fields[envWeightsField] = d.envWeights
	}
	if d.envDynamicWeights != nil {
		fields[envDynamicWeightsField] = d.envDynamicWeights
	}
	return fields
}

// UpsertEnvState 幂等新增或更新指定环境的信息，不推进配置的顶层 updatedAt；配置不存在时返回 ErrConfigNotFound
func UpsertEnvState(
	ctx context.Context, db *mongo.Database, appID, name, envName string, update PolarisEnvStateUpdate,
) error {
	fields, err := envStateFields(envName, update)
	if err != nil {
		return err
	}
	err = store.Of(db, ConfigMeta).Update(ctx, fields, ByName(appID, name), store.KeepUpdatedAt())
	return errors.Wrapf(err, "upsert polaris config %s of app %s env %s state", name, appID, envName)
}

// UpsertEnvStateIfUpdatedAtMatch 仅在配置顶层 updatedAt 仍与 expectedUpdatedAt 匹配时更新环境信息。
// 返回值表示是否匹配到期望版本；版本不匹配（通常表示任务处理的是旧配置）或配置不存在时返回 false 并跳过写入，不返回错误。
func UpsertEnvStateIfUpdatedAtMatch(
	ctx context.Context,
	db *mongo.Database,
	appID, name, envName string,
	expectedUpdatedAt time.Time,
	update PolarisEnvStateUpdate,
) (bool, error) {
	fields, err := envStateFields(envName, update)
	if err != nil {
		return false, err
	}
	err = store.Of(db, ConfigMeta).Update(ctx, fields,
		ByName(appID, name), store.Where("updatedAt", expectedUpdatedAt), store.KeepUpdatedAt(),
	)
	if errors.Is(err, ErrConfigNotFound) {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrapf(err, "conditionally upsert polaris config %s of app %s env %s state",
			name, appID, envName)
	}
	return true, nil
}

// envStateFields 生成单个环境信息的待更新字段：总是刷新该环境的 updatedAt，其余字段非 nil 时写入
func envStateFields(envName string, update PolarisEnvStateUpdate) (map[string]any, error) {
	fieldPrefix, err := envFieldPrefix(envStatesField, envName)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{fieldPrefix + ".updatedAt": time.Now()}
	if update.AppliedFields != nil {
		fields[fieldPrefix+".appliedFields"] = update.AppliedFields
	}
	if update.LastError != nil {
		fields[fieldPrefix+".lastError"] = *update.LastError
	}
	return fields, nil
}

// UpsertEnvWeight 幂等设置指定环境的单实例权重与动态权重开关，dynamicWeight 为 nil 表示不改开关。
// 两者在同一次更新中写入，保证集群侧的单次 Patch 与库侧记录一致；配置不存在时返回 ErrConfigNotFound
func UpsertEnvWeight(
	ctx context.Context,
	db *mongo.Database,
	appID, name, envName string,
	weight int32,
	dynamicWeight *bool,
) error {
	weightPath, err := envFieldPrefix(envWeightsField, envName)
	if err != nil {
		return err
	}
	fields := map[string]any{weightPath: weight}
	if dynamicWeight != nil {
		dynamicWeightPath, pathErr := envFieldPrefix(envDynamicWeightsField, envName)
		if pathErr != nil {
			return pathErr
		}
		fields[dynamicWeightPath] = *dynamicWeight
	}
	err = store.Of(db, ConfigMeta).Update(ctx, fields, ByName(appID, name))
	return errors.Wrapf(err, "upsert polaris config %s of app %s env %s weight", name, appID, envName)
}

// RemoveEnvStates 幂等移除多个指定环境的信息，配置不存在时静默成功
func RemoveEnvStates(ctx context.Context, db *mongo.Database, appID, name string, envNames []string) error {
	return removeEnvFields(ctx, db, appID, name, []string{envStatesField}, envNames)
}

// RemoveEnvSettings 幂等移除多个指定环境的全部环境级配置（见 envSettingFields），配置不存在时静默成功
func RemoveEnvSettings(ctx context.Context, db *mongo.Database, appID, name string, envNames []string) error {
	return removeEnvFields(ctx, db, appID, name, envSettingFields, envNames)
}

// removeEnvFields 从 roots 指向的各个环境维度子文档中删除多个环境条目，不推进配置的顶层 updatedAt。
// 所有路径在同一次更新中提交，避免部分字段清理成功、部分残留。
func removeEnvFields(ctx context.Context, db *mongo.Database, appID, name string, roots, envNames []string) error {
	if len(envNames) == 0 {
		return nil
	}

	fields := make(map[string]any, len(envNames)*len(roots))
	for _, root := range roots {
		for _, envName := range envNames {
			fieldPath, err := envFieldPrefix(root, envName)
			if err != nil {
				return err
			}
			// map 条目置为 nil 即从库中删除该条目
			fields[fieldPath] = nil
		}
	}
	err := store.Of(db, ConfigMeta).Update(ctx, fields, ByName(appID, name), store.KeepUpdatedAt())
	return errors.Wrapf(store.IgnoreNotFound(err), "remove polaris config %s of app %s env fields %v",
		name, appID, roots)
}

// envFieldPrefix 校验环境名并生成环境维度子文档的嵌套字段路径前缀
func envFieldPrefix(root, envName string) (string, error) {
	if envName == "" || strings.ContainsAny(envName, ".$") {
		return "", errors.Errorf("invalid env name %q", envName)
	}
	return root + "." + envName, nil
}
