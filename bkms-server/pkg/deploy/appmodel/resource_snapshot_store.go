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

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ResourceSnapshotStore 部署资源清单快照存储（每快照一行）
type ResourceSnapshotStore = store.GenericStore[ResourceSnapshot]

// ResourceSnapshotMeta 部署资源清单快照的存储元数据。
//
// 设计说明：
//   - 数据模型：一次部署下的每个 K8s 资源占一行文档，而不是把资源嵌进同一文档的数组。
//     单条 manifest 上限约 5MB，若合并为单文档易触及 MongoDB 16MB 文档上限，且大数组不利于按条分页与 projection。
//
// 索引（由 golang-migrate 维护）：
// - 查询提速：appID + deployRecordId
var ResourceSnapshotMeta = store.Meta[ResourceSnapshot]{
	Collection: "app_model_resource_snapshots",
	NotFound:   ErrResourceSnapshotRowNotFound,
}

// ByDeployRecord 按应用 ID 与部署记录 ID 匹配资源清单快照
func ByDeployRecord(appID string, deployOID bson.ObjectID) store.Opt {
	return store.Filter(bson.M{"appID": appID, "deployRecordId": deployOID})
}

// InKindNameOrder 按资源类型、名称排序，最后按 _id 保证分页时顺序稳定
func InKindNameOrder() store.Opt {
	return store.Combine(store.Asc("kind"), store.Asc("name"), store.Asc("_id"))
}

// ListSnapshotMeta 分页列出某次部署的快照元数据（不加载 manifest）。
// deployID 非合法 hex 等价于「无记录」，返回空页
func ListSnapshotMeta(
	ctx context.Context, snapshots *ResourceSnapshotStore, appID, deployID string, page, pageSize int64,
) (*store.Page[ResourceSnapshot], error) {
	deployOID, err := bson.ObjectIDFromHex(deployID)
	if err != nil {
		//nolint:nilerr // 非法 hex 不可能对应任何部署记录，按「无记录」返回空页
		return &store.Page[ResourceSnapshot]{Items: []ResourceSnapshot{}}, nil
	}
	result, err := snapshots.ListPage(
		ctx, page, pageSize, ByDeployRecord(appID, deployOID), InKindNameOrder(), store.OmitFields("manifest"),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list resource snapshot meta app %s record %s", appID, deployID)
	}
	return result, nil
}

// GetSnapshot 按 (appID, deployID, snapshotID) 读取完整快照文档。
// 以下场景均返回 ErrResourceSnapshotRowNotFound：
//   - appID / deployID / snapshotID 任一不匹配
//   - deployID / snapshotID 非合法 hex（语义上等价于「找不到」）
func GetSnapshot(
	ctx context.Context, snapshots *ResourceSnapshotStore, appID, deployID, snapshotID string,
) (*ResourceSnapshot, error) {
	// 非法 hex 等价于「找不到」：调用方拿到的就是一个不可能存在的资源 id，对外语义就是 404。
	deployOID, err := bson.ObjectIDFromHex(deployID)
	if err != nil {
		return nil, ErrResourceSnapshotRowNotFound
	}
	snapshotOID, err := bson.ObjectIDFromHex(snapshotID)
	if err != nil {
		return nil, ErrResourceSnapshotRowNotFound
	}
	snap, err := snapshots.Get(ctx, ByDeployRecord(appID, deployOID), store.ByID(snapshotOID))
	if err != nil {
		return nil, errors.Wrapf(err, "get resource snapshot app %s record %s id %s", appID, deployID, snapshotID)
	}
	return snap, nil
}
