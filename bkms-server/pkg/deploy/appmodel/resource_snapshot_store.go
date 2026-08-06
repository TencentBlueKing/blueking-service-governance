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

package appmodel

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ResourceSnapshotStore 部署资源清单快照存储（每快照一行）
type ResourceSnapshotStore interface {
	// DeleteByDeployRecord 删除该 deploy 记录下的全部快照行
	DeleteByDeployRecord(ctx context.Context, appID, deployID string) error
	// CreateResources 批量插入快照行。
	CreateResources(ctx context.Context, snapshots []ResourceSnapshot) error
	// ListByDeployRecord 列出某次部署下的全部快照。
	// 以下场景均返回 ErrResourceSnapshotNotFound：
	//   - 查不到任何记录
	//   - deployID 非合法 hex（语义上等价于「找不到」）
	ListByDeployRecord(ctx context.Context, appID, deployID string) ([]ResourceSnapshot, error)
	// ListMetaByDeployRecord 分页列出快照元数据（不加载 manifest）。
	// 无记录或 deployID 非合法 hex 时均返回 (nil, 0, nil)。
	ListMetaByDeployRecord(
		ctx context.Context, appID, deployID string, page, pageSize int64,
	) ([]ResourceSnapshot, int64, error)
	// GetByID 按 (appID, deployID, snapshotID) 读取完整快照文档
	// 以下场景均返回 ErrResourceSnapshotRowNotFound：
	//   - appID / deployID / snapshotID 任一不匹配
	//   - deployID / snapshotID 非合法 hex（语义上等价于「找不到」）
	GetByID(ctx context.Context, appID, deployID, snapshotID string) (*ResourceSnapshot, error)
}

var _ ResourceSnapshotStore = (*ResourceSnapshotStoreMongo)(nil)

// ResourceSnapshotStoreMongo ResourceSnapshotStore 的 Mongo 实现。
//
// 设计说明：
//   - 数据模型：一次部署下的每个 K8s 资源占一行文档，而不是把资源嵌进同一文档的数组。
//     单条 manifest 上限约 5MB，若合并为单文档易触及 MongoDB 16MB 文档上限，且大数组不利于按条分页与 projection。
type ResourceSnapshotStoreMongo struct {
	collection *mongo.Collection
}

// NewResourceSnapshotStoreMongo 创建 ResourceSnapshotStoreMongo
func NewResourceSnapshotStoreMongo(client *mongo.Client, dbName string) (*ResourceSnapshotStoreMongo, error) {
	coll := client.Database(dbName).Collection(resourceSnapshotCollectionName)
	// 索引（由 golang-migrate 维护）：
	// - 查询提速：appID + deployRecordId
	return &ResourceSnapshotStoreMongo{collection: coll}, nil
}

// DeleteByDeployRecord 删除某次部署下的全部快照行
func (s *ResourceSnapshotStoreMongo) DeleteByDeployRecord(
	ctx context.Context, appID, deployID string,
) error {
	if appID == "" {
		return errors.New("appID is required")
	}
	deployObjID, err := bson.ObjectIDFromHex(deployID)
	if err != nil {
		return errors.New("invalid deployID")
	}
	filter := bson.M{"deployRecordId": deployObjID, "appID": appID}
	if _, err := s.collection.DeleteMany(ctx, filter); err != nil {
		return errors.Wrapf(err, "delete resource snapshots app %s record %s", appID, deployID)
	}
	return nil
}

// CreateResources 批量插入快照行
func (s *ResourceSnapshotStoreMongo) CreateResources(ctx context.Context, snapshots []ResourceSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	if _, err := s.collection.InsertMany(ctx, snapshots); err != nil {
		return errors.Wrapf(err, "insert resource snapshots")
	}

	return nil
}

// ListByDeployRecord 列出某次部署的快照。
// 非法 hex 等价于「找不到」，与 GetByID 保持一致的对外语义。
func (s *ResourceSnapshotStoreMongo) ListByDeployRecord(
	ctx context.Context, appID, deployID string,
) ([]ResourceSnapshot, error) {
	oid, err := bson.ObjectIDFromHex(deployID)
	if err != nil {
		return nil, ErrResourceSnapshotNotFound
	}
	opts := options.Find().
		SetSort(bson.D{
			{Key: "kind", Value: 1},
			{Key: "name", Value: 1},
		})
	cursor, err := s.collection.Find(ctx, bson.M{"deployRecordId": oid, "appID": appID}, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "find resource snapshots app %s record %s", appID, deployID)
	}
	defer cursor.Close(ctx)
	var out []ResourceSnapshot
	if err = cursor.All(ctx, &out); err != nil {
		return nil, errors.Wrapf(err, "decode resource snapshots app %s record %s", appID, deployID)
	}
	if len(out) == 0 {
		return nil, ErrResourceSnapshotNotFound
	}
	return out, nil
}

// ListMetaByDeployRecord 分页列出快照元数据（不加载 manifest）
// 非法 hex 等价于「无记录」，按现有约定返回 (nil, 0, nil)
func (s *ResourceSnapshotStoreMongo) ListMetaByDeployRecord(
	ctx context.Context, appID, deployID string, page, pageSize int64,
) ([]ResourceSnapshot, int64, error) {
	oid, err := bson.ObjectIDFromHex(deployID)
	if err != nil {
		return nil, 0, nil
	}
	filter := bson.M{"deployRecordId": oid, "appID": appID}
	total, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "count resource snapshot meta app %s record %s", appID, deployID)
	}
	if total == 0 {
		return nil, 0, nil
	}
	opts := options.Find().
		SetSort(bson.D{
			{Key: "kind", Value: 1},
			{Key: "name", Value: 1},
		}).
		SetProjection(bson.M{"manifest": 0}).
		SetSkip((page - 1) * pageSize).
		SetLimit(pageSize)
	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "find resource snapshot meta app %s record %s", appID, deployID)
	}
	defer cursor.Close(ctx)
	var out []ResourceSnapshot
	if err = cursor.All(ctx, &out); err != nil {
		return nil, 0, errors.Wrapf(err, "decode resource snapshot meta app %s record %s", appID, deployID)
	}
	return out, total, nil
}

// GetByID 读取单条快照
func (s *ResourceSnapshotStoreMongo) GetByID(
	ctx context.Context, appID, deployID, snapshotID string,
) (*ResourceSnapshot, error) {
	if appID == "" {
		return nil, errors.New("appID is required")
	}
	// 非法 hex 等价于「找不到」：调用方拿到的就是一个不可能存在的资源 id，对外语义就是 404。
	deployObjID, err := bson.ObjectIDFromHex(deployID)
	if err != nil {
		return nil, ErrResourceSnapshotRowNotFound
	}
	snapshotObjID, err := bson.ObjectIDFromHex(snapshotID)
	if err != nil {
		return nil, ErrResourceSnapshotRowNotFound
	}
	var doc ResourceSnapshot
	filter := bson.M{"_id": snapshotObjID, "appID": appID, "deployRecordId": deployObjID}
	if err = s.collection.FindOne(ctx, filter).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrResourceSnapshotRowNotFound
		}
		return nil, errors.Wrapf(
			err, "find resource snapshot app %s record %s id %s",
			appID, deployID, snapshotID,
		)
	}
	return &doc, nil
}
