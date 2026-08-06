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

package topology

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ResourceSnapshotsCollection MongoDB Collection 名称
const ResourceSnapshotsCollection = "topology_resource_snapshots"

// initialDataVersion 首次插入时的初始数据版本号
const initialDataVersion = 0

// ErrVersionConflict 乐观锁版本冲突错误，表示写入时 DataVersion 已被其他刷新任务更新
var ErrVersionConflict = errors.New("version conflict: data has been updated by another refresh")

// ResourceSnapshotStore 资源范围存储接口
type ResourceSnapshotStore interface {
	// UpsertWithVersion 带乐观锁的创建或更新资源范围（基于 appID + envName + trafficLaneName 唯一索引）
	// 若记录不存在则自动创建；若存在且 dataVersion <= expectedVersion 则更新
	// 若版本已被其他任务推进（dataVersion > expectedVersion），返回 ErrVersionConflict
	UpsertWithVersion(ctx context.Context, snapshot *ResourceSnapshot, expectedVersion int64) error
	// UpdateStatus 仅更新刷新状态和警告摘要，保留已有的 resources 和 extensionRelations 数据
	UpdateStatus(ctx context.Context, appID, envName, trafficLaneName, status, warningSummary string) error
	// Get 获取指定作用域的资源范围
	Get(ctx context.Context, appID, envName, trafficLaneName string) (*ResourceSnapshot, error)
	// Delete 删除指定作用域的资源范围
	Delete(ctx context.Context, appID, envName, trafficLaneName string) error
	// DeleteAll 删除所有记录并保留集合及索引（仅用于单元测试）
	DeleteAll(ctx context.Context) error
}

// 编译期接口实现检查
var _ ResourceSnapshotStore = &ResourceSnapshotStoreMongo{}

// ResourceSnapshotStoreMongo 基于 MongoDB 的 ResourceSnapshotStore 实现
type ResourceSnapshotStoreMongo struct {
	collection *mongo.Collection
}

// NewResourceSnapshotStoreMongo 创建 ResourceSnapshotStoreMongo 实例
func NewResourceSnapshotStoreMongo(client *mongo.Client, dbName string) (*ResourceSnapshotStoreMongo, error) {
	coll := client.Database(dbName).Collection(ResourceSnapshotsCollection)
	// 索引（由 golang-migrate 维护）：
	// - 唯一：appID + envName + trafficLaneName
	// - 查询提速：appID
	return &ResourceSnapshotStoreMongo{collection: coll}, nil
}

// UpsertWithVersion 带乐观锁的创建或更新资源范围
// 采用两步策略：先确保记录存在（首次插入），再带版本条件更新
// 若版本已被其他任务推进（dataVersion > expectedVersion），返回 ErrVersionConflict
func (s *ResourceSnapshotStoreMongo) UpsertWithVersion(
	ctx context.Context, snapshot *ResourceSnapshot, expectedVersion int64,
) error {
	now := time.Now()
	snapshot.UpdatedAt = now
	snapshotKey := bson.M{
		"appID":           snapshot.AppID,
		"envName":         snapshot.EnvName,
		"trafficLaneName": snapshot.TrafficLaneName,
	}

	// 步骤一：确保记录存在（若不存在则插入初始记录，若已存在则为幂等 no-op）
	// $setOnInsert 仅在插入时生效，不会覆盖已有记录的任何字段
	ensureExist := bson.M{
		"$setOnInsert": bson.M{
			"appID":           snapshot.AppID,
			"envName":         snapshot.EnvName,
			"trafficLaneName": snapshot.TrafficLaneName,
			"dataVersion":     int64(initialDataVersion),
			"createdAt":       now,
			"updatedAt":       now,
		},
	}
	ensureOpts := options.UpdateOne().SetUpsert(true)
	if _, err := s.collection.UpdateOne(ctx, snapshotKey, ensureExist, ensureOpts); err != nil {
		// duplicate key error 说明重复插入，记录已存在，可安全忽略
		if !mongo.IsDuplicateKeyError(err) {
			return errors.Wrapf(
				err, "ensure resource snapshot exists for %s/%s/%s",

				snapshot.AppID, snapshot.EnvName, snapshot.TrafficLaneName,
			)
		}
	}

	// 步骤二：带版本条件的更新（乐观锁）
	// filter 要求 dataVersion <= expectedVersion，确保不会覆盖更新的数据
	versionedFilter := bson.M{
		"appID":           snapshot.AppID,
		"envName":         snapshot.EnvName,
		"trafficLaneName": snapshot.TrafficLaneName,
		"dataVersion":     bson.M{"$lte": expectedVersion},
	}
	update := bson.M{
		"$set": bson.M{
			"clusterID":      snapshot.ClusterID,
			"namespace":      snapshot.Namespace,
			"releaseName":    snapshot.ReleaseName,
			"dataVersion":    snapshot.DataVersion,
			"refreshStatus":  snapshot.RefreshStatus,
			"refreshedAt":    snapshot.RefreshedAt,
			"warningSummary": snapshot.WarningSummary,
			"resources":      snapshot.Resources,
			"relations":      snapshot.Relations,
			"updatedAt":      now,
		},
	}

	result, err := s.collection.UpdateOne(ctx, versionedFilter, update)
	if err != nil {
		return errors.Wrapf(
			err, "upsert with version for %s/%s/%s",
			snapshot.AppID, snapshot.EnvName, snapshot.TrafficLaneName,
		)
	}

	// 没有匹配到任何文档，说明已存在更高版本，当前待更新内容已过时 -> 跳过
	if result.MatchedCount == 0 {
		return ErrVersionConflict
	}
	return nil
}

// Get 获取指定作用域的资源范围
func (s *ResourceSnapshotStoreMongo) Get(
	ctx context.Context,
	appID, envName, trafficLaneName string,
) (*ResourceSnapshot, error) {
	filter := bson.M{
		"appID":           appID,
		"envName":         envName,
		"trafficLaneName": trafficLaneName,
	}

	var snapshot ResourceSnapshot
	err := s.collection.FindOne(ctx, filter).Decode(&snapshot)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get resource snapshot for %s/%s/%s", appID, envName, trafficLaneName)
	}
	return &snapshot, nil
}

// Delete 删除指定作用域的资源范围
func (s *ResourceSnapshotStoreMongo) Delete(ctx context.Context, appID, envName, trafficLaneName string) error {
	filter := bson.M{
		"appID":           appID,
		"envName":         envName,
		"trafficLaneName": trafficLaneName,
	}

	_, err := s.collection.DeleteOne(ctx, filter)
	if err != nil {
		return errors.Wrapf(err, "delete resource snapshot for %s/%s/%s", appID, envName, trafficLaneName)
	}
	return nil
}

// UpdateStatus 仅更新刷新状态和警告摘要，保留已有的 resources 和 extensionRelations 数据
// 若记录不存在则不做任何操作（不自动创建）
func (s *ResourceSnapshotStoreMongo) UpdateStatus(
	ctx context.Context, appID, envName, trafficLaneName, status, warningSummary string,
) error {
	now := time.Now()

	filter := bson.M{
		"appID":           appID,
		"envName":         envName,
		"trafficLaneName": trafficLaneName,
	}

	update := bson.M{
		"$set": bson.M{
			"refreshStatus":  status,
			"warningSummary": warningSummary,
			"updatedAt":      now,
		},
	}

	_, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.Wrapf(err, "update status for %s/%s/%s", appID, envName, trafficLaneName)
	}
	return nil
}

// DeleteAll 删除所有记录并保留集合及索引（仅用于单元测试）
func (s *ResourceSnapshotStoreMongo) DeleteAll(ctx context.Context) error {
	_, err := s.collection.DeleteMany(ctx, bson.M{})
	return err
}
