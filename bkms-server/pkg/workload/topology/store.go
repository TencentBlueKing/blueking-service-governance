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

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrVersionConflict 乐观锁版本冲突错误，表示写入时 DataVersion 已被其他刷新任务更新
var ErrVersionConflict = errors.New("version conflict: data has been updated by another refresh")

// ResourceSnapshotStore 资源拓扑快照存储
type ResourceSnapshotStore = store.GenericStore[ResourceSnapshot]

// ResourceSnapshotMeta 资源拓扑快照的存储元数据，每个 appID + envName + trafficLaneName 作用域一条记录。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + envName + trafficLaneName（SaveWithVersion 的乐观锁依赖它）
// - 查询提速：appID
var ResourceSnapshotMeta = store.Meta[ResourceSnapshot]{Collection: "topology_resource_snapshots"}

// ByScope 按作用域（应用 + 环境 + 泳道）匹配
func ByScope(appID, envName, trafficLaneName string) store.Opt {
	return store.Filter(bson.M{"appID": appID, "envName": envName, "trafficLaneName": trafficLaneName})
}

// FindSnapshot 获取指定作用域的资源快照，不存在时返回 nil, nil：尚未刷新过的作用域是正常状态，
// 调用方据此返回空拓扑或跳过，而不是当作错误处理
func FindSnapshot(
	ctx context.Context, snapshots *ResourceSnapshotStore, appID, envName, trafficLaneName string,
) (*ResourceSnapshot, error) {
	snapshot, err := snapshots.Get(ctx, ByScope(appID, envName, trafficLaneName))
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get resource snapshot for %s/%s/%s", appID, envName, trafficLaneName)
	}
	return snapshot, nil
}

// SaveWithVersion 带乐观锁地创建或更新资源快照：库中记录的 dataVersion <= expectedVersion 时写入 snapshot 的全部字段，
// 记录不存在时插入；若版本已被其他刷新任务推进（dataVersion > expectedVersion），返回 ErrVersionConflict。
//
// 只需一次原子的 upsert：过滤条件带上版本约束，版本已被推进时匹配不到文档，MongoDB 转而尝试插入，
// 插入因作用域唯一索引冲突而失败，据此判定为版本冲突。snapshot 带 _id 时（如回写读出的旧快照）同理
func SaveWithVersion(
	ctx context.Context, snapshots *ResourceSnapshotStore, snapshot *ResourceSnapshot, expectedVersion int64,
) error {
	_, err := snapshots.Upsert(
		ctx,
		snapshot,
		ByScope(snapshot.AppID, snapshot.EnvName, snapshot.TrafficLaneName),
		store.Where("dataVersion", bson.M{"$lte": expectedVersion}),
	)
	if errors.Is(err, store.ErrDuplicate) {
		return ErrVersionConflict
	}
	if err != nil {
		return errors.Wrapf(
			err, "save resource snapshot %s/%s/%s with version %d",
			snapshot.AppID, snapshot.EnvName, snapshot.TrafficLaneName, expectedVersion,
		)
	}
	return nil
}

// UpdateRefreshStatus 仅更新刷新状态和警告摘要，保留已有的 resources 和 relations；记录不存在时不做任何操作
func UpdateRefreshStatus(
	ctx context.Context,
	snapshots *ResourceSnapshotStore,
	appID, envName, trafficLaneName, status, warningSummary string,
) error {
	err := snapshots.Update(
		ctx,
		map[string]any{"refreshStatus": status, "warningSummary": warningSummary},
		ByScope(appID, envName, trafficLaneName),
	)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return errors.Wrapf(err, "update refresh status for %s/%s/%s", appID, envName, trafficLaneName)
	}
	return nil
}
