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

package dbm

import (
	"context"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// ------------------------------------------ Redis 工单 API ------------------------------------------

// CreateRedis 提交创建 Redis 工单
func (c *ApiClient) CreateRedis(ctx context.Context, params *CreateRedisParams, username string) (int, error) {
	if params == nil {
		return 0, errors.New("create redis params are required")
	}

	var details any
	switch params.TicketType {
	case TicketTypeRedisClusterApply:
		details = &redisClusterApplyDetails{
			BkCloudID:              params.BkCloudID,
			DBAppAbbr:              params.DBAppAbbr,
			ClusterName:            params.ClusterName,
			ClusterAlias:           params.ClusterAlias,
			ClusterType:            params.ClusterType,
			DBVersion:              params.DBVersion,
			ProxyPort:              params.ProxyPort,
			ProxyPwd:               params.ProxyPwd,
			CityCode:               params.CityCode,
			DisasterToleranceLevel: params.DisasterToleranceLevel,
			ClusterShardNum:        params.ClusterShardNum,
			IPSource:               params.IPSource,
			ResourceSpec:           params.ResourceSpec,
		}
	case TicketTypeRedisInsApply:
		details = &redisInsApplyDetails{
			BkCloudID:              params.BkCloudID,
			DBAppAbbr:              params.DBAppAbbr,
			ClusterType:            params.ClusterType,
			DBVersion:              params.DBVersion,
			Port:                   params.Port,
			RedisPwd:               params.RedisPwd,
			CityCode:               params.CityCode,
			DisasterToleranceLevel: params.DisasterToleranceLevel,
			AppendApply:            params.AppendApply,
			IPSource:               params.IPSource,
			Infos:                  params.Infos,
			ResourceSpec:           params.ResourceSpec,
		}
	default:
		return 0, errors.Errorf("unsupported ticket type: %s", params.TicketType)
	}

	payload := &createTicketPayload{
		BkBizID:           params.BkBizID,
		TicketType:        params.TicketType,
		Details:           details,
		Remark:            params.Remark,
		IgnoreDuplication: params.IgnoreDuplication,
	}

	result, err := c.handleOperation(ctx, c.newCreateTicketOperation(payload, username))
	if err != nil {
		return 0, errors.Wrap(err, "create redis ticket")
	}
	return cast.ToInt(mapx.Get(result, "data.id", 0)), nil
}

// DisableRedis 提交禁用 Redis 工单
func (c *ApiClient) DisableRedis(ctx context.Context, params *DisableRedisParams, username string) (int, error) {
	if params == nil {
		return 0, errors.New("disable redis params are required")
	}

	payload := &disableTicketPayload{
		BkBizID:    params.BkBizID,
		TicketType: params.TicketType,
		Details: disableRedisDetails{
			ClusterID: params.ClusterID,
			Force:     params.Force,
		},
	}

	result, err := c.handleOperation(ctx, c.newCreateTicketOperation(payload, username))
	if err != nil {
		return 0, errors.Wrap(err, "disable redis ticket")
	}
	return cast.ToInt(mapx.Get(result, "data.id", 0)), nil
}

// DeleteRedis 提交删除 Redis 工单
//   - REDIS_DESTROY（集群删除）：单集群操作，details 仅含 cluster_id
//   - REDIS_INSTANCE_DESTROY（主从删除）：多集群操作，details 含 cluster_ids 列表
func (c *ApiClient) DeleteRedis(ctx context.Context, params *DeleteRedisParams, username string) (int, error) {
	if params == nil {
		return 0, errors.New("delete redis params are required")
	}

	var details any
	switch params.TicketType {
	case TicketTypeRedisInstanceDestroy:
		// 多集群操作，使用 cluster_ids 列表
		ids := params.ClusterIDs
		if len(ids) == 0 && params.ClusterID != 0 {
			ids = []int{params.ClusterID}
		}
		details = deleteRedisInstanceDetails{ClusterIDs: ids, Force: params.Force}
	case TicketTypeRedisDestroy:
		// 单集群操作，使用 cluster_id
		details = deleteRedisClusterDetails{ClusterID: params.ClusterID, Force: params.Force}
	default:
		return 0, errors.Errorf("unsupported delete ticket type: %s", params.TicketType)
	}

	payload := &deleteTicketPayload{
		BkBizID:    params.BkBizID,
		TicketType: params.TicketType,
		Details:    details,
	}

	result, err := c.handleOperation(ctx, c.newCreateTicketOperation(payload, username))
	if err != nil {
		return 0, errors.Wrap(err, "delete redis ticket")
	}
	return cast.ToInt(mapx.Get(result, "data.id", 0)), nil
}

// ------------------------------------------ Redis 集群查询 API ------------------------------------------

// FindClusterByName 按业务ID、集群名和集群类型查找集群
func (c *ApiClient) FindClusterByName(
	ctx context.Context,
	bkBizID int,
	clusterName string,
	clusterType ClusterType,
	username string,
) (*ClusterInfo, error) {
	clusters, err := c.filterClusters(ctx, &filterClustersOpts{
		BkBizID:     bkBizID,
		ClusterType: string(clusterType),
		Name:        clusterName,
	}, username)
	if err != nil {
		return nil, err
	}

	// name 为模糊查询，需精确比对集群名称
	for _, item := range clusters {
		if item.ClusterName == clusterName {
			return toClusterInfo(item), nil
		}
	}

	return nil, errors.Errorf("cluster not found: bk_biz_id=%d, name=%s", bkBizID, clusterName)
}

// GetClusterInfo 按集群ID获取集群详情
func (c *ApiClient) GetClusterInfo(ctx context.Context, clusterID int, username string) (*ClusterInfo, error) {
	clusters, err := c.filterClusters(ctx, &filterClustersOpts{
		ClusterIDs: []int{clusterID},
	}, username)
	if err != nil {
		return nil, err
	}

	for _, item := range clusters {
		if item.ID == clusterID {
			return toClusterInfo(item), nil
		}
	}

	return nil, errors.Errorf("cluster not found: id=%d", clusterID)
}

// filterClusters 调用 DBM 过滤集群接口
func (c *ApiClient) filterClusters(
	ctx context.Context, query *filterClustersOpts, username string,
) ([]filterClusterItem, error) {
	apiOperation := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "filter_clusters",
			Method: "GET",
			Path:   "/dbbase/filter_clusters/",
		},
	).SetQueryParams(query.toQueryParams()).SetHeaders(c.authHeaders(username))

	result, err := c.handleOperation(ctx, apiOperation)
	if err != nil {
		return nil, err
	}

	var items []filterClusterItem
	for _, d := range mapx.GetList(result, "data") {
		raw, ok := d.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, filterClusterItem{
			ID:                cast.ToInt(mapx.Get(raw, "id", 0)),
			Status:            mapx.GetStr(raw, "status"),
			ClusterName:       mapx.GetStr(raw, "cluster_name"),
			ClusterAccessPort: cast.ToInt(mapx.Get(raw, "cluster_access_port", 0)),
			MasterDomain:      mapx.GetStr(raw, "master_domain"),
			BkBizID:           cast.ToInt(mapx.Get(raw, "bk_biz_id", 0)),
			ClusterType:       ClusterType(mapx.GetStr(raw, "cluster_type")),
		})
	}

	return items, nil
}
