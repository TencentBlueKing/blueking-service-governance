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

package gpa

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// 错误定义
var (
	// ErrConfigNotFound GPA 配置不存在
	ErrConfigNotFound = errors.New("gpa config not found")
	// ErrConfigEnvExists 该环境已存在 GPA 配置
	ErrConfigEnvExists = errors.New("gpa config for this environment already exists")
)

// Validate 校验 GPA 配置的合法性
//   - minReplicas >= 1
//   - maxReplicas >= minReplicas
//   - 指标模式与定时模式至少配置其一
//   - 指标至多 2 条，每条 resource 仅 cpu/memory 且不重复，averageUtilization 取值 1-100
//   - 每条定时规则 desiredReplicas >= 1，schedule 为合法的 5 段 Crontab 表达式
func (c *GPAConfig) Validate() error {
	if err := validate.Struct(c); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// ConfigStore GPA 配置存储，配置在应用 + 环境维度唯一
type ConfigStore = store.GenericStore[GPAConfig]

// ConfigMeta GPA 配置的存储元数据，写入完整文档前会整体校验配置。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + envName
var ConfigMeta = store.Meta[GPAConfig]{
	Collection: "gpa_configs",
	NotFound:   ErrConfigNotFound,
	Duplicate:  ErrConfigEnvExists,
	BeforeSave: []func(*GPAConfig) error{(*GPAConfig).Validate},
}

// ByApp 按应用 ID 匹配 GPA 配置
func ByApp(appID string) store.Opt {
	return store.Where("appID", appID)
}

// ByAppAndEnv 按应用 ID 和环境名称匹配 GPA 配置（至多一条）
func ByAppAndEnv(appID, envName string) store.Opt {
	return store.Combine(ByApp(appID), store.Where("envName", envName))
}

// CreateConfig 创建 GPA 配置：未指定名称时按规则生成；新建即生效，强制 Enabled=true，
// 避免零值 false 导致创建即关闭。同一应用 + 环境已有配置时返回 ErrConfigEnvExists
func CreateConfig(ctx context.Context, configs *ConfigStore, config *GPAConfig) error {
	if config.Name == "" {
		config.Name = config.GenerateName()
	}
	config.Enabled = true
	if err := configs.Create(ctx, config); err != nil {
		return errors.Wrapf(err, "create gpa config of app %s env %s", config.AppID, config.EnvName)
	}
	return nil
}

// UpdateConfig 根据应用 ID 和环境名称局部更新 GPA 配置。
// 先取出当前配置，应用变更后整体校验，避免出现 max < min 等非法中间态；配置不存在时返回 ErrConfigNotFound
func UpdateConfig(
	ctx context.Context,
	configs *ConfigStore,
	appID, envName string,
	updateData *ConfigUpdateData,
) error {
	if updateData == nil {
		return nil
	}
	current, err := configs.Get(ctx, ByAppAndEnv(appID, envName))
	if err != nil {
		return errors.Wrapf(err, "get gpa config of app %s env %s", appID, envName)
	}

	set := updateData.applyTo(current)
	if len(set) == 0 {
		return nil
	}
	if err = current.Validate(); err != nil {
		return err
	}
	if err = configs.Update(ctx, set, ByAppAndEnv(appID, envName)); err != nil {
		return errors.Wrapf(err, "update gpa config of app %s env %s", appID, envName)
	}
	return nil
}

// applyTo 将变更应用到 config 上，并返回对应的 $set 内容；没有任何变更时返回空文档
func (d *ConfigUpdateData) applyTo(config *GPAConfig) bson.M {
	set := bson.M{}
	if d.MinReplicas != nil {
		config.MinReplicas = *d.MinReplicas
		set["minReplicas"] = *d.MinReplicas
	}
	if d.MaxReplicas != nil {
		config.MaxReplicas = *d.MaxReplicas
		set["maxReplicas"] = *d.MaxReplicas
	}
	if d.Metrics != nil {
		config.Metrics = d.Metrics
		set["metrics"] = d.Metrics
	}
	if d.TimeRanges != nil {
		config.TimeRanges = d.TimeRanges
		set["timeRanges"] = d.TimeRanges
	}
	if d.ComputeByLimits != nil {
		config.ComputeByLimits = *d.ComputeByLimits
		set["computeByLimits"] = *d.ComputeByLimits
	}
	if d.Enabled != nil {
		config.Enabled = *d.Enabled
		set["enabled"] = *d.Enabled
	}
	return set
}
