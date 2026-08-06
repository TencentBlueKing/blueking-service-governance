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

// Package usergroup 提供蓝鲸监控告警组管理能力
package usergroup

import (
	"context"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// saveUserGroupDefaultTimezone 保存告警组时的默认时区。
const saveUserGroupDefaultTimezone = "Asia/Shanghai"

// ErrUserGroupNotInWorkspace 表示告警组不属于当前 workspace 对应的监控空间。
var ErrUserGroupNotInWorkspace = errors.New("user group does not belong to workspace")

// Service 提供告警组管理能力，不影响既有 APM 告警组同步链路。
type Service struct {
	newClient func(operator string) (bkmapi.MonitorClient, error)
}

// SaveParams 保存告警组参数。
type SaveParams struct {
	ID           int64
	Name         string                 `json:"name" binding:"required" validate:"required"`
	Timezone     string                 `json:"timezone"`
	NeedDuty     bool                   `json:"needDuty"`
	Channels     []string               `json:"channels" binding:"required,min=1" validate:"min=1"`
	Desc         string                 `json:"desc"`
	AlertNotice  []bkmapi.AlertNotice   `json:"alertNotice" binding:"required,min=1" validate:"min=1"`
	ActionNotice []bkmapi.ActionNotice  `json:"actionNotice" binding:"required,min=1" validate:"min=1"`
	DutyArranges []bkmapi.DutyArrange   `json:"dutyArranges"`
	DutyRules    []int64                `json:"dutyRules"`
	DutyNotice   *bkmapi.DutyNotice     `json:"dutyNotice"`
	MentionList  []bkmapi.UserGroupUser `json:"mentionList"`
	MentionType  int64                  `json:"mentionType"`
	Path         string                 `json:"path"`
	Operator     string                 `validate:"required"`
}

// New 创建告警组管理服务。
func New() *Service {
	return &Service{newClient: bkmapi.NewMonitorClient}
}

// List 列出 workspace 对应监控空间下的全部告警组。
func (s *Service) List(ctx context.Context, ws *workspace.Workspace, operator string) ([]*bkmapi.UserGroup, error) {
	bkBizID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	groups, err := client.SearchUserGroups(ctx, &bkmapi.SearchUserGroupsReq{
		BkBizIDs: []int64{bkBizID},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "search user groups, bk_biz_id=%d", bkBizID)
	}
	return groups, nil
}

// Get 获取告警组详情，并校验其属于 workspace 对应监控空间。
func (s *Service) Get(
	ctx context.Context,
	ws *workspace.Workspace,
	groupID int64,
	operator string,
) (*bkmapi.UserGroupDetail, error) {
	bkBizID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return s.getWithClient(ctx, client, groupID, bkBizID)
}

func (s *Service) getWithClient(
	ctx context.Context,
	client bkmapi.MonitorClient,
	groupID, bkBizID int64,
) (*bkmapi.UserGroupDetail, error) {
	detail, err := client.SearchUserGroupDetail(ctx, &bkmapi.SearchUserGroupDetailReq{ID: groupID})
	if err != nil {
		return nil, errors.Wrapf(err, "search user group detail, id=%d", groupID)
	}
	if detail == nil || detail.BkBizID != bkBizID {
		return nil, errors.Wrapf(ErrUserGroupNotInWorkspace, "group_id=%d, expected_bk_biz_id=%d", groupID, bkBizID)
	}
	return detail, nil
}

// Save 创建或更新告警组。更新时会先校验告警组归属。
func (s *Service) Save(
	ctx context.Context,
	ws *workspace.Workspace,
	params *SaveParams,
) (*bkmapi.UserGroupDetail, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	bkBizID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(params.Operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}

	if params.ID > 0 {
		if _, err = s.getWithClient(ctx, client, params.ID, bkBizID); err != nil {
			return nil, errors.Wrap(err, "validate user group ownership before save")
		}
	}

	timezone := params.Timezone
	if timezone == "" {
		timezone = saveUserGroupDefaultTimezone
	}
	var requestID *int64
	if params.ID > 0 {
		requestID = &params.ID
	}

	detail, err := client.SaveUserGroup(ctx, &bkmapi.SaveUserGroupReq{
		ID:           requestID,
		BkBizID:      bkBizID,
		Name:         params.Name,
		Timezone:     timezone,
		NeedDuty:     params.NeedDuty,
		Channels:     params.Channels,
		Desc:         params.Desc,
		AlertNotice:  params.AlertNotice,
		ActionNotice: params.ActionNotice,
		Operator:     params.Operator,
		DutyArranges: params.DutyArranges,
		DutyRules:    params.DutyRules,
		DutyNotice:   params.DutyNotice,
		MentionList:  params.MentionList,
		MentionType:  params.MentionType,
		Path:         params.Path,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "save user group, bk_biz_id=%d, id=%d", bkBizID, params.ID)
	}
	return detail, nil
}

// Delete 删除告警组。删除前会先校验告警组归属。
func (s *Service) Delete(
	ctx context.Context,
	ws *workspace.Workspace,
	groupID int64,
	operator string,
) error {
	bkBizID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return errors.Wrap(err, "resolve bkmonitor space id")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return errors.Wrap(err, "new bkmonitor client")
	}
	if _, err = s.getWithClient(ctx, client, groupID, bkBizID); err != nil {
		return errors.Wrap(err, "validate user group ownership before delete")
	}
	if err = client.DeleteUserGroup(ctx, &bkmapi.DeleteUserGroupReq{
		IDs:      []int64{groupID},
		BkBizIDs: []int64{bkBizID},
		Operator: operator,
	}); err != nil {
		return errors.Wrapf(err, "delete user group, bk_biz_id=%d, id=%d", bkBizID, groupID)
	}
	return nil
}
