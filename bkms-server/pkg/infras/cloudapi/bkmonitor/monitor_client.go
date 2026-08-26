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

// Package bkmonitor api client，如：蓝鲸监控的 apm、蓝鲸监控的 metadata
package bkmonitor

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

// MonitorGatewayClient 使用新版 bk-monitor 网关访问新增监控能力。
type MonitorGatewayClient struct {
	*ApiClient
}

// SearchUserGroups 查询告警组列表。
func (c *MonitorGatewayClient) SearchUserGroups(
	ctx context.Context,
	req *SearchUserGroupsReq,
) ([]*UserGroup, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "search_user_groups",
			Method: http.MethodPost,
			Path:   "/app/user_group/search/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return nil, errors.Wrapf(err, "search user groups failed, bk_biz_ids: %v", req.BkBizIDs)
	}

	result := make([]*UserGroup, 0)
	if err = mapstructure.Decode(mapx.GetList(resp, "data"), &result); err != nil {
		return nil, errors.Wrap(err, "decode user groups failed")
	}

	return result, nil
}

// SearchUserGroupDetail 查询告警组详情。
func (c *MonitorGatewayClient) SearchUserGroupDetail(
	ctx context.Context,
	req *SearchUserGroupDetailReq,
) (*UserGroupDetail, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "search_user_group_detail",
			Method: http.MethodPost,
			Path:   "/app/user_group/detail/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return nil, errors.Wrapf(err, "search user group detail failed, id: %d", req.ID)
	}

	result := new(UserGroupDetail)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrapf(err, "decode user group detail failed, id: %d", req.ID)
	}

	return result, nil
}

// SaveUserGroup 保存（创建/更新）告警组。
func (c *MonitorGatewayClient) SaveUserGroup(ctx context.Context, req *SaveUserGroupReq) (*UserGroupDetail, error) {
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "save_user_group",
			Method: http.MethodPost,
			Path:   "/app/user_group/save/",
		},
		bkapi.OptSetRequestBody(req),
		bkapi.OptSetRequestHeader(headerBkapiUserName, req.Operator),
	))
	if err != nil {
		return nil, errors.Wrapf(err, "save user group failed, bk_biz_id: %d, name: %s", req.BkBizID, req.Name)
	}

	result := new(UserGroupDetail)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrapf(
			err,
			"decode save user group response failed, bk_biz_id: %d, name: %s",
			req.BkBizID,
			req.Name,
		)
	}

	return result, nil
}

// CreateApmApp 创建 APM 应用。
func (c *MonitorGatewayClient) CreateApmApp(
	ctx context.Context,
	bkBizID int64,
	bcsProjectCode, envName, description, operator, workspaceID string,
) (*ApmApp, error) {
	req := NewDefaultCreateApmAppReq(bkBizID, bcsProjectCode, envName, description, operator)
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if _, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "apm_create_application",
			Method: http.MethodPost,
			Path:   "/app/apm/create_application/",
		},
		bkapi.OptSetRequestBody(req),
		bkapi.OptSetRequestHeader(headerBkapiUserName, req.Operator),
	)); err != nil {
		if strings.Contains(err.Error(), "已被创建") {
			metrics.CreateEnvApmFailed(workspaceID, envName, "already_exists")
			return nil, ErrApmAppDuplicate
		}
		metrics.CreateEnvApmFailed(workspaceID, envName, "create_failed")
		return nil, errors.Wrapf(err, "create apm app failed, bk_biz_id: %d, app_name: %s", req.BkBizID, req.AppName)
	}

	result, err := c.GetApmApp(ctx, req.BkBizID, 0, req.AppName)
	if err != nil {
		metrics.CreateEnvApmFailed(workspaceID, envName, "query_after_create_failed")
		return nil, errors.Wrap(err, "get apm app after creation failed")
	}

	return result, nil
}

// GetApmApp 获取 APM 应用详情。
func (c *MonitorGatewayClient) GetApmApp(
	ctx context.Context,
	bkBizID, apmAppID int64,
	envName string,
) (*ApmApp, error) {
	req := NewGetApmAppReq(bkBizID, apmAppID, envName)
	if err := req.Validate(); err != nil {
		return nil, err
	}

	params := map[string]string{
		"bk_biz_id": cast.ToString(req.BkBizID),
	}
	if req.AppName != "" {
		params["app_name"] = req.AppName
	} else {
		params["application_id"] = cast.ToString(req.ApmAppID)
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "detail_apm_application",
			Method: http.MethodGet,
			Path:   "/app/apm/detail_apm_application/",
		},
	).SetQueryParams(params))
	if err != nil {
		return nil, errors.Wrapf(ErrApmAppNotFound, "get apm app failed: %v", err)
	}

	result := new(ApmApp)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrapf(err, "decode apm app failed, envName: %s", req.AppName)
	}

	return result, nil
}

// GetOrCreate 创建或获取 APM 应用。
func (c *MonitorGatewayClient) GetOrCreate(
	ctx context.Context,
	bkBizID int64,
	bcsProjectCode, envName, description, operator, workspaceID string,
) (*ApmApp, error) {
	if result, err := c.GetApmApp(ctx, bkBizID, 0, envName); err == nil {
		return result, nil
	}

	return c.CreateApmApp(ctx, bkBizID, bcsProjectCode, envName, description, operator, workspaceID)
}

// ListApmApp 列出 APM 应用。
func (c *MonitorGatewayClient) ListApmApp(ctx context.Context, bkBizID int64) ([]*ApmApp, error) {
	req := NewListApmAppReq(bkBizID)
	if err := Validate(req); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "apm_service_list",
			Method: http.MethodPost,
			Path:   "/app/apm/service/service_list/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return nil, errors.Wrapf(err, "list apm app failed, bk_biz_id: %d", req.BkBizID)
	}

	items := mapx.GetList(resp, "data")
	if len(items) == 0 {
		items = mapx.GetList(resp, "data.list")
	}

	result := make([]*ApmApp, 0)
	if err = mapstructure.Decode(items, &result); err != nil {
		return nil, errors.Wrapf(err, "read apm app list failed, bk_biz_id: %d", req.BkBizID)
	}

	return result, nil
}

// GetMetadataSpaceDetail 获取空间详情。
func (c *MonitorGatewayClient) GetMetadataSpaceDetail(ctx context.Context, bcsProjectCode string) (*Space, error) {
	targetSpaceUID := fmt.Sprintf(SpaceUIDFormat, bcsProjectCode)
	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "metadata_list_spaces_by_user",
			Method: http.MethodGet,
			Path:   "/user/metadata/list_spaces/by_user/",
		},
	))
	if err != nil {
		return nil, errors.Wrapf(ErrSpaceNotFound, "get metadata space detail failed: %v", err)
	}

	items := mapx.GetList(resp, "data.list")
	if len(items) == 0 {
		items = mapx.GetList(resp, "data")
	}

	spaces := make([]*Space, 0)
	if err = mapstructure.Decode(items, &spaces); err != nil {
		return nil, errors.Wrapf(err, "get metadata space detail failed: %v", err)
	}

	for _, space := range spaces {
		if space == nil {
			continue
		}
		if space.SpaceUid != targetSpaceUID && space.SpaceID != bcsProjectCode {
			continue
		}

		if space.ID > 0 {
			space.ID = -space.ID
		}
		return space, nil
	}

	return nil, ErrSpaceNotFound
}

// TimeSeriesUnifyQuery 统一时序数据查询。
func (c *MonitorGatewayClient) TimeSeriesUnifyQuery(
	ctx context.Context, req *TimeSeriesUnifyQueryReq,
) (*TimeSeriesUnifyQueryResp, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "time_series_unify_query",
			Method: http.MethodPost,
			Path:   "/app/data_query/time_series_unify_query/",
		},
		bkapi.OptSetRequestBody(req),
	))
	if err != nil {
		return nil, errors.Wrap(err, "time series unify query failed")
	}

	result := new(TimeSeriesUnifyQueryResp)
	if err = mapstructure.Decode(resp["data"], result); err != nil {
		return nil, errors.Wrap(err, "decode time series unify query response failed")
	}

	return result, nil
}

// DeleteUserGroup 删除告警组。
func (c *MonitorGatewayClient) DeleteUserGroup(ctx context.Context, req *DeleteUserGroupReq) error {
	if err := Validate(req); err != nil {
		return err
	}

	_, err := c.handleOperation(ctx, c.NewOperation(
		bkapi.OperationConfig{
			Name:   "delete_user_group",
			Method: http.MethodPost,
			Path:   "/app/user_group/delete/",
		},
		bkapi.OptSetRequestBody(req),
		bkapi.OptSetRequestHeader(headerBkapiUserName, req.Operator),
	))
	if err != nil {
		return errors.Wrapf(err, "delete user groups failed, bk_biz_ids: %v, ids: %v", req.BkBizIDs, req.IDs)
	}
	return nil
}
