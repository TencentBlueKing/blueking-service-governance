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

package helm

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	helmrelease "helm.sh/helm/v3/pkg/release"

	deploytypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/types"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/helm"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// ErrRecordNotFound 部署记录未找到。调用方据此区分「记录不可寻回」与 DB 瞬时故障
// 记录不存在、以及非法 ObjectID（同样不可能查到记录）都 wrap 本哨兵错误
var ErrRecordNotFound = errors.New("helm deploy record not found")

// ErrLatestDeployRecordNotFound 最新部署记录不存在
var ErrLatestDeployRecordNotFound = errors.New("latest deploy record not found")

// RecordStore Helm 应用部署记录存储
type RecordStore = store.GenericStore[Record]

// RecordMeta Helm 应用部署记录的存储元数据。
//
// 索引（由 golang-migrate 维护）：
// - 查询提速：appID + envName + trafficLaneName + createdAt(倒序)
// - 查询提速：appID + trafficLaneName + createdAt(倒序)
var RecordMeta = store.Meta[Record]{
	Collection: "helm_deploy_records",
	NotFound:   ErrRecordNotFound,
}

// imageTagHistoryStatuses 按镜像标签查询部署记录时关注的状态（经讨论确认该功能仅关注部分状态）
var imageTagHistoryStatuses = []helmrelease.Status{
	// 部署成功
	helm.StatusDeployed,
	// 部署中
	helm.StatusPendingInstall,
	helm.StatusPendingUpgrade,
	// 部署失败（含超时、中断）
	helm.StatusFailed,
	helm.StatusPollingTimeout,
	helm.StatusPollingBroken,
}

// activeStatuses 视为活跃部署（未卸载）的状态：已部署、部署中或卸载中
var activeStatuses = []helmrelease.Status{
	helmrelease.StatusDeployed,
	helmrelease.StatusPendingInstall,
	helmrelease.StatusPendingUpgrade,
	helmrelease.StatusPendingRollback,
	helmrelease.StatusUninstalling,
}

// KeywordMatches 按 Chart 版本、镜像标签或操作人模糊匹配（不区分大小写），keyword 为空白时不过滤
func KeywordMatches(keyword string) store.Opt {
	return store.ContainsIgnoreCase(keyword, "chartVersion", "imageTag", "operator")
}

// ByImageTag 匹配应用中使用指定镜像标签、且状态为部署成功、部署中或部署失败的记录
func ByImageTag(appID, imageTag string) store.Opt {
	return store.Combine(
		store.Where("appID", appID),
		store.Where("imageTag", imageTag),
		store.Where("status", bson.M{"$in": imageTagHistoryStatuses}),
	)
}

// GetRecord 获取应用的一条部署记录；记录不存在或 recordID 不是合法的 ObjectID 时返回 ErrRecordNotFound
func GetRecord(ctx context.Context, db *mongo.Database, appID, recordID string) (*Record, error) {
	objID, err := bson.ObjectIDFromHex(recordID)
	if err != nil {
		return nil, errors.Wrapf(ErrRecordNotFound, "invalid record id %s", recordID)
	}
	record, err := store.Of(db, RecordMeta).Get(ctx, store.ByID(objID), store.Where("appID", appID))
	if err != nil {
		return nil, errors.Wrapf(err, "get app %s deploy record %s", appID, recordID)
	}
	return record, nil
}

// UpdateRecord 更新部署记录的结果字段（revision、status、message、extras、endedAt），记录不存在时返回 ErrRecordNotFound
func UpdateRecord(ctx context.Context, db *mongo.Database, record *Record) error {
	err := store.Of(db, RecordMeta).Update(ctx, record,
		store.UpdateSelected("revision", "status", "message", "extras", "endedAt"),
		store.ByID(record.ID),
	)
	return errors.Wrapf(err, "update workspace %s app %s deploy record %s",
		record.WorkspaceID, record.AppID, record.ID.Hex())
}

// GetLatest 获取应用在指定环境、泳道下最新的一条部署记录；传入 statuses 时只在这些状态中查找。
// 不存在时返回 ErrLatestDeployRecordNotFound
func GetLatest(
	ctx context.Context, db *mongo.Database, appID, envName, trafficLaneName string, statuses ...helmrelease.Status,
) (*Record, error) {
	opts := []store.Opt{deploytypes.ByAppEnvLane(appID, envName, trafficLaneName), deploytypes.NewestFirst()}
	if len(statuses) > 0 {
		opts = append(opts, store.Where("status", bson.M{"$in": statuses}))
	}
	record, err := store.Of(db, RecordMeta).Get(ctx, opts...)
	if errors.Is(err, ErrRecordNotFound) {
		return nil, errors.Wrapf(
			ErrLatestDeployRecordNotFound,
			"app %s env %s lane %q",
			appID,
			envName,
			trafficLaneName,
		)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "get latest helm deploy record of app %s env %s", appID, envName)
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
		ctx, store.Of(db, RecordMeta), appID, string(helmrelease.StatusDeployed), "imageTag",
	)
}

// ListChartVersionDeployedEnvs 获取指定应用各 Chart 版本的已部署环境列表（去重，仅统计部署成功的记录，默认查询所有环境），
// 用于「Helm Chart 制品列表」展示已部署环境
func ListChartVersionDeployedEnvs(
	ctx context.Context, db *mongo.Database, appID string,
) ([]deploytypes.ChartVersionEnvPair, error) {
	return deploytypes.ListDeployedEnvPairs[deploytypes.ChartVersionEnvPair](
		ctx, store.Of(db, RecordMeta), appID, string(helmrelease.StatusDeployed), "chartVersion",
	)
}

// HasActiveDeployments 检查是否存在活跃的部署（已部署、部署中或卸载中）：
// 按环境 + 泳道分组获取每组最新记录，检查最新记录是否为活跃状态
func HasActiveDeployments(ctx context.Context, db *mongo.Database, appID string) (bool, error) {
	return deploytypes.HasActiveDeployment(ctx, store.Of(db, RecordMeta), appID, func(r *Record) bool {
		return lo.Contains(activeStatuses, r.Status)
	})
}
