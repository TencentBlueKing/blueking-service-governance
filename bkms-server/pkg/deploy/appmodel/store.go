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
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrDeployRecordNotFound 部署记录不存在
var ErrDeployRecordNotFound = errors.New("app model deploy record not found")

// RecordStore 应用部署记录存储
type RecordStore = store.GenericStore[Record]

// RecordMeta 应用部署记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 查询提速：appID + envName + trafficLaneName + createdAt(倒序)
// - 查询提速：appID + trafficLaneName + createdAt(倒序)
var RecordMeta = store.Meta[Record]{
	Collection: "app_model_deploy_records",
	NotFound:   ErrDeployRecordNotFound,
}

// imageTagHistoryStatuses 按镜像标签查询部署记录时关注的状态（经讨论确认该功能仅关注部分状态）
var imageTagHistoryStatuses = []Status{
	// 部署成功
	StatusDeployed,
	// 部署中
	StatusDeploying,
	// 部署失败（含超时、中断）
	StatusFailed,
	StatusPollingTimeout,
	StatusPollingBroken,
}

// activeStatuses 视为活跃部署（未卸载）的状态：已部署、部署中、卸载中或分批发布已回滚
var activeStatuses = []Status{
	StatusDeployed,
	StatusDeploying,
	StatusUninstalling,
	// 回滚后工作负载仍在运行（Pod 为旧版本）
	StatusRolledBack,
}

// KeywordMatches 按镜像标签或更新人模糊匹配（不区分大小写），keyword 为空白时不过滤
func KeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "imageTag", "updater")
}

// ByImageTag 匹配应用中使用指定镜像标签、且状态为部署成功、部署中或部署失败的记录
func ByImageTag(appID, imageTag string) store.Opt {
	return store.Combine(
		store.Where("appID", appID),
		store.Where("imageTag", imageTag),
		store.Where("status", bson.M{"$in": imageTagHistoryStatuses}),
	)
}

// GetRecord 获取应用的一条部署记录，记录不存在时返回 ErrDeployRecordNotFound
func GetRecord(ctx context.Context, db *mongo.Database, appID, id string) (*Record, error) {
	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid record id %s", id)
	}
	record, err := store.Of(db, RecordMeta).Get(ctx, store.ByID(objID), store.Where("appID", appID))
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s deploy record %s", appID, id)
	}
	return record, nil
}

// UpdateRecord 更新部署记录的可变字段（replicas、deletionCost、message、status、extras、endedAt、updater），
// 记录不存在时返回 ErrDeployRecordNotFound。不写 batchProgress，分批进度由 UpdateBatchProgress 定向更新
func UpdateRecord(ctx context.Context, db *mongo.Database, record *Record) error {
	err := store.Of(db, RecordMeta).Update(ctx, record,
		store.UpdateSelected("replicas", "deletionCost", "message", "status", "extras", "endedAt", "updater"),
		store.ByID(record.ID),
	)
	return errors.Wrapf(err, "update app model deploy record %s", record.ID.Hex())
}

// UpdateBatchProgress 只更新分批发布进度，供轮询任务写入，记录不存在时返回 ErrDeployRecordNotFound。
// 与 UpdateRecord 分开：非分批记录的该字段为 nil，放进 UpdateRecord 的字段中会被 $unset；
// 扩缩容、下架等先读再写的路径也会用旧快照覆盖轮询刚写入的进度。
func UpdateBatchProgress(ctx context.Context, db *mongo.Database, id string, progress *BatchProgress) error {
	if progress == nil {
		return errors.New("batch progress is nil")
	}
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return errors.Wrapf(err, "parse deploy record id %s", id)
	}
	err = store.Of(db, RecordMeta).Update(ctx, &Record{BatchProgress: progress},
		store.UpdateSelected("batchProgress"),
		store.ByID(oid),
	)
	return errors.Wrapf(err, "update batch progress of deploy record %s", id)
}

// GetLatest 获取应用在指定环境、泳道下最新的一条部署记录；传入 statuses 时只在这些状态中查找。
// 不存在时返回 ErrDeployRecordNotFound
func GetLatest(
	ctx context.Context, db *mongo.Database, appID, envName, trafficLaneName string, statuses ...Status,
) (*Record, error) {
	opts := []store.Opt{deploytypes.ByAppEnvLane(appID, envName, trafficLaneName), deploytypes.NewestFirst()}
	if len(statuses) > 0 {
		opts = append(opts, store.Where("status", bson.M{"$in": statuses}))
	}
	record, err := store.Of(db, RecordMeta).Get(ctx, opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "get latest app model deploy record of app %s env %s", appID, envName)
	}
	return record, nil
}

// ListLatestByApps 返回一批应用在指定泳道下各环境最新的一条部署记录。
// 外层 key 为 appID，内层 key 为 envName；无记录的应用或环境不出现在结果中。
func ListLatestByApps(
	ctx context.Context, db *mongo.Database, appIDs []string, trafficLaneName string,
) (map[string]map[string]*Record, error) {
	return deploytypes.ListLatestByApps(ctx, store.Of(db, RecordMeta), appIDs, trafficLaneName,
		func(r *Record) (string, string) { return r.AppID, r.EnvName },
	)
}

// ListImageTagDeployedEnvs 获取指定应用各镜像标签的已部署环境列表（去重，仅统计部署成功的记录，默认查询所有环境）
func ListImageTagDeployedEnvs(
	ctx context.Context,
	db *mongo.Database,
	appID string,
) ([]deploytypes.ImageTagEnvPair, error) {
	return deploytypes.ListDeployedEnvPairs[deploytypes.ImageTagEnvPair](
		ctx, store.Of(db, RecordMeta), appID, string(StatusDeployed), "imageTag",
	)
}

// HasActiveDeployments 检查是否存在活跃的部署（已部署、部署中、卸载中或分批发布已回滚）：
// 按环境 + 泳道分组获取每组最新记录，检查最新记录是否为活跃状态
func HasActiveDeployments(ctx context.Context, db *mongo.Database, appID string) (bool, error) {
	return deploytypes.HasActiveDeployment(ctx, store.Of(db, RecordMeta), appID, func(r *Record) bool {
		return lo.Contains(activeStatuses, r.Status)
	})
}
