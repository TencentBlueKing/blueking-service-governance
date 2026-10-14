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

package deploytypes

import (
	"context"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// 以下查询条件与查询函数由三种部署记录（一键构建部署、Helm 部署、AppModel 部署）共用，
// 它们都有 appID、envName、trafficLaneName 与 createdAt 字段

// ByAppEnvLane 匹配应用在指定环境、泳道下的记录
func ByAppEnvLane(appID, envName, trafficLaneName string) store.Opt {
	return store.Combine(
		store.Where("appID", appID),
		store.Where("envName", envName),
		store.Where("trafficLaneName", trafficLaneName),
	)
}

// NewestFirst 按创建时间倒序；创建时间相同时按 _id 倒序，保证取最新一条与翻页的结果稳定
func NewestFirst() store.Opt {
	return store.Combine(store.Desc("createdAt"), store.Desc("_id"))
}

// ListLatestByApps 返回一批应用在指定泳道下各环境最新的一条记录。
// 外层 key 为 appID，内层 key 为 envName；无记录的应用或环境不出现在结果中。
// 用于批量聚合部署状态，往返次数与应用数、环境数无关
func ListLatestByApps[T any](
	ctx context.Context,
	s *store.GenericStore[T],
	appIDs []string,
	trafficLaneName string,
	keyOf func(*T) (appID, envName string),
) (map[string]map[string]*T, error) {
	out := make(map[string]map[string]*T, len(appIDs))
	if len(appIDs) == 0 {
		return out, nil
	}
	// 排序以（appID, trafficLaneName, createdAt）开头，命中「appID + trafficLaneName + createdAt(倒序)」索引
	records, err := s.ListLatestBy(ctx, []string{"appID", "envName"},
		store.Where("appID", bson.M{"$in": lo.Uniq(appIDs)}),
		store.Where("trafficLaneName", trafficLaneName),
		store.Asc("appID"), store.Asc("trafficLaneName"), store.Desc("createdAt"),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "list latest records of %d apps in lane %q", len(appIDs), trafficLaneName)
	}
	for i := range records {
		appID, envName := keyOf(&records[i])
		if out[appID] == nil {
			out[appID] = make(map[string]*T)
		}
		out[appID][envName] = &records[i]
	}
	return out, nil
}

// HasActiveDeployment 判断应用是否存在活跃的部署：按（环境, 泳道）分组取各自最新的一条记录，
// 任意一条满足 isActive 即为活跃。
//
// 记录只读出了 status 字段，isActive 只能依据 status 判断，读取其它字段只会得到零值
func HasActiveDeployment[T any](
	ctx context.Context, s *store.GenericStore[T], appID string, isActive func(*T) bool,
) (bool, error) {
	// 排序以（appID, envName, trafficLaneName, createdAt）开头，命中「appID + envName + trafficLaneName + createdAt(倒序)」索引
	records, err := s.ListLatestBy(ctx, []string{"envName", "trafficLaneName"},
		store.Where("appID", appID),
		store.Asc("envName"), store.Asc("trafficLaneName"), store.Desc("createdAt"),
		store.SelectFields("status"),
	)
	if err != nil {
		return false, errors.Wrapf(err, "list latest records of app %s", appID)
	}
	return lo.ContainsBy(records, func(r T) bool { return isActive(&r) }), nil
}

// ListDeployedEnvPairs 返回应用中状态为 deployedStatus 的记录里 (field, envName) 的去重组合，解码为 R
// （如 ImageTagEnvPair、ChartVersionEnvPair），默认查询所有环境
func ListDeployedEnvPairs[R, T any](
	ctx context.Context, s *store.GenericStore[T], appID, deployedStatus, field string,
) ([]R, error) {
	pairs, err := store.Aggregate[R](ctx, s, mongo.Pipeline{
		// 按 appID 和 status 过滤，只保留成功部署的记录
		{{Key: "$match", Value: bson.M{"appID": appID, "status": deployedStatus}}},
		// 按 (field, envName) 复合键分组去重
		{{Key: "$group", Value: bson.M{"_id": bson.M{field: "$" + field, "envName": "$envName"}}}},
		// 将分组 _id 中的字段提升为顶层字段，输出扁平结构
		{{Key: "$project", Value: bson.M{field: "$_id." + field, "envName": "$_id.envName", "_id": 0}}},
	})
	return pairs, errors.Wrapf(err, "list distinct %s and envs deployed for app %s", field, appID)
}
