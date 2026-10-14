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

package autodeploy

import (
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"

	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrRecordNotFound build auto deploy record not found error
var ErrRecordNotFound = errors.New("build auto deploy record not found")

// RecordStore build auto deploy 记录存储
type RecordStore = store.GenericStore[Record]

// RecordMeta build auto deploy 记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 唯一：appID + buildID
// - 查询提速：appID + envName + trafficLaneName + createdAt(倒序)
// - 查询提速：appID + trafficLaneName + createdAt(倒序)
// - 条件唯一：appID + deployID（仅 deployID 非空）
var RecordMeta = store.Meta[Record]{
	Collection: "build_auto_deploy_records",
	NotFound:   ErrRecordNotFound,
}

// ByBuild 按应用与 buildID 匹配记录
func ByBuild(appID, buildID string) store.Opt {
	return store.Combine(store.Where("appID", appID), store.Where("buildID", buildID))
}

// ByDeploy 按应用与 deployID 匹配记录
func ByDeploy(appID, deployID string) store.Opt {
	return store.Combine(store.Where("appID", appID), store.Where("deployID", deployID))
}

// UpdateRecord 更新记录的状态字段（deployID、stage、status、message、endedAt），记录不存在时返回 ErrRecordNotFound
func UpdateRecord(ctx context.Context, db *mongo.Database, record *Record) error {
	err := store.Of(db, RecordMeta).Update(ctx, record,
		store.UpdateSelected("deployID", "stage", "status", "message", "endedAt"),
		store.ByID(record.ID),
	)
	return errors.Wrapf(err, "update build auto deploy record %s", record.ID.Hex())
}

// GetLatest 获取应用在指定环境、泳道下最新的一条记录，不存在时返回 ErrRecordNotFound
func GetLatest(ctx context.Context, db *mongo.Database, appID, envName, trafficLaneName string) (*Record, error) {
	record, err := store.Of(db, RecordMeta).Get(ctx,
		deploytypes.ByAppEnvLane(appID, envName, trafficLaneName), deploytypes.NewestFirst(),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "get latest build auto deploy record of app %s env %s", appID, envName)
	}
	return record, nil
}

// ListLatestByApps 返回一批应用在指定泳道下各环境最新的一条记录。
// 外层 key 为 appID，内层 key 为 envName；无记录的应用或环境不出现在结果中。
func ListLatestByApps(
	ctx context.Context, db *mongo.Database, appIDs []string, trafficLaneName string,
) (map[string]map[string]*Record, error) {
	return deploytypes.ListLatestByApps(ctx, store.Of(db, RecordMeta), appIDs, trafficLaneName,
		func(r *Record) (string, string) { return r.AppID, r.EnvName },
	)
}
