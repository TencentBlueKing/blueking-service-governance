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
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// The name of the MongoDB collection for storing GPA config data.
const collectionName = "gpa_configs"

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

// GPAConfigStore GPA 配置存储接口。配置在应用 + 环境维度唯一。
type GPAConfigStore interface {
	// Create 创建 GPA 配置
	Create(ctx context.Context, config *GPAConfig) error
	// Get 根据应用 ID 和环境名称获取 GPA 配置
	Get(ctx context.Context, appID, envName string) (*GPAConfig, error)
	// ListByApp 获取应用下的所有 GPA 配置（各环境一份）
	ListByApp(ctx context.Context, appID string) ([]*GPAConfig, error)
	// Update 根据应用 ID 和环境名称局部更新 GPA 配置
	Update(ctx context.Context, appID, envName string, updateData *ConfigUpdateData) error
	// Delete 根据应用 ID 和环境名称删除 GPA 配置
	Delete(ctx context.Context, appID, envName string) error
	// DeleteByApp 删除应用下的所有 GPA 配置（仅测试使用）
	DeleteByApp(ctx context.Context, appID string) error
}

var _ GPAConfigStore = &GPAConfigStoreMongo{}

// GPAConfigStoreMongo GPA 配置的 MongoDB 存储实现
type GPAConfigStoreMongo struct {
	collection *mongo.Collection
}

// NewGPAConfigStoreMongo 创建 GPA 配置存储
func NewGPAConfigStoreMongo(client *mongo.Client, dbName string) (GPAConfigStore, error) {
	coll := client.Database(dbName).Collection(collectionName)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + envName
	return &GPAConfigStoreMongo{collection: coll}, nil
}

// Create 创建 GPA 配置
func (s *GPAConfigStoreMongo) Create(ctx context.Context, config *GPAConfig) error {
	if config.Name == "" {
		config.Name = config.GenerateName()
	}
	// 新建即生效：强制 Enabled=true，避免零值 false 导致创建即关闭。
	config.Enabled = true
	if err := config.Validate(); err != nil {
		return err
	}

	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	if config.UpdatedAt.IsZero() {
		config.UpdatedAt = now
	}

	if _, err := s.collection.InsertOne(ctx, config); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrConfigEnvExists
		}
		return errors.Wrap(err, "insert gpa config")
	}
	return nil
}

// Get 根据应用 ID 和环境名称获取 GPA 配置
func (s *GPAConfigStoreMongo) Get(ctx context.Context, appID, envName string) (*GPAConfig, error) {
	var config GPAConfig
	err := s.collection.FindOne(ctx, bson.M{"appID": appID, "envName": envName}).Decode(&config)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrConfigNotFound
		}
		return nil, errors.Wrap(err, "find gpa config by env")
	}
	return &config, nil
}

// ListByApp 获取应用下的所有 GPA 配置
func (s *GPAConfigStoreMongo) ListByApp(ctx context.Context, appID string) ([]*GPAConfig, error) {
	cursor, err := s.collection.Find(ctx, bson.M{"appID": appID})
	if err != nil {
		return nil, errors.Wrap(err, "find gpa configs")
	}
	defer cursor.Close(ctx)

	var configs []*GPAConfig
	if err = cursor.All(ctx, &configs); err != nil {
		return nil, errors.Wrap(err, "decode gpa configs")
	}
	return configs, nil
}

// Update 根据应用 ID 和环境名称局部更新 GPA 配置
func (s *GPAConfigStoreMongo) Update(
	ctx context.Context,
	appID, envName string,
	updateData *ConfigUpdateData,
) error {
	if updateData == nil {
		return nil
	}

	// 先取出当前配置，应用变更后整体校验，避免出现 max < min 等非法中间态
	current, err := s.Get(ctx, appID, envName)
	if err != nil {
		return err
	}

	updateSet := bson.M{}
	if updateData.MinReplicas != nil {
		current.MinReplicas = *updateData.MinReplicas
		updateSet["minReplicas"] = *updateData.MinReplicas
	}
	if updateData.MaxReplicas != nil {
		current.MaxReplicas = *updateData.MaxReplicas
		updateSet["maxReplicas"] = *updateData.MaxReplicas
	}
	if updateData.Metrics != nil {
		current.Metrics = updateData.Metrics
		updateSet["metrics"] = updateData.Metrics
	}
	if updateData.TimeRanges != nil {
		current.TimeRanges = updateData.TimeRanges
		updateSet["timeRanges"] = updateData.TimeRanges
	}
	if updateData.ComputeByLimits != nil {
		current.ComputeByLimits = *updateData.ComputeByLimits
		updateSet["computeByLimits"] = *updateData.ComputeByLimits
	}
	if updateData.Enabled != nil {
		current.Enabled = *updateData.Enabled
		updateSet["enabled"] = *updateData.Enabled
	}

	if len(updateSet) == 0 {
		return nil
	}

	if err = current.Validate(); err != nil {
		return err
	}

	updateSet["updatedAt"] = time.Now()
	filter := bson.M{"appID": appID, "envName": envName}
	result, err := s.collection.UpdateOne(ctx, filter, bson.M{"$set": updateSet})
	if err != nil {
		return errors.Wrap(err, "update gpa config")
	}
	if result.MatchedCount == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// Delete 根据应用 ID 和环境名称删除 GPA 配置
func (s *GPAConfigStoreMongo) Delete(ctx context.Context, appID, envName string) error {
	filter := bson.M{"appID": appID, "envName": envName}
	result, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return errors.Wrap(err, "delete gpa config")
	}
	if result.DeletedCount == 0 {
		return ErrConfigNotFound
	}
	return nil
}

// DeleteByApp 删除应用下的所有 GPA 配置
func (s *GPAConfigStoreMongo) DeleteByApp(ctx context.Context, appID string) error {
	if _, err := s.collection.DeleteMany(ctx, bson.M{"appID": appID}); err != nil {
		return errors.Wrap(err, "delete gpa configs by app")
	}
	return nil
}
