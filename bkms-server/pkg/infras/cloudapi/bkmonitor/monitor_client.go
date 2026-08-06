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
	"net/http"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
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
