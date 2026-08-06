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

package serializer

import (
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	bkmusergroup "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/usergroup"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators" // register global validators
)

// UserGroupURIInput 告警组路径参数。
type UserGroupURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
	GroupID     int64  `uri:"groupID" binding:"required,gt=0"`
}

// UserGroupWorkspaceURIInput 工作空间级告警组路径参数。
type UserGroupWorkspaceURIInput struct {
	WorkspaceID string `uri:"workspaceID" binding:"required,uri_slug"`
}

// SaveUserGroupBody 告警组创建/更新请求体。
type SaveUserGroupBody struct {
	Name         string                 `json:"name" binding:"required"`
	Timezone     string                 `json:"timezone"`
	NeedDuty     bool                   `json:"needDuty"`
	Channels     []string               `json:"channels" binding:"required,min=1"`
	Desc         string                 `json:"desc"`
	AlertNotice  []bkmapi.AlertNotice   `json:"alertNotice" binding:"required,min=1"`
	ActionNotice []bkmapi.ActionNotice  `json:"actionNotice" binding:"required,min=1"`
	DutyArranges []bkmapi.DutyArrange   `json:"dutyArranges"`
	DutyRules    []int64                `json:"dutyRules"`
	DutyNotice   *bkmapi.DutyNotice     `json:"dutyNotice"`
	MentionList  []bkmapi.UserGroupUser `json:"mentionList"`
	MentionType  int64                  `json:"mentionType"`
	Path         string                 `json:"path"`
}

// NewSaveParams converts the request body into service-layer save params.
func NewSaveParams(body SaveUserGroupBody, operator string) bkmusergroup.SaveParams {
	return bkmusergroup.SaveParams{
		Name:         body.Name,
		Timezone:     body.Timezone,
		NeedDuty:     body.NeedDuty,
		Channels:     body.Channels,
		Desc:         body.Desc,
		AlertNotice:  body.AlertNotice,
		ActionNotice: body.ActionNotice,
		DutyArranges: body.DutyArranges,
		DutyRules:    body.DutyRules,
		DutyNotice:   body.DutyNotice,
		MentionList:  body.MentionList,
		MentionType:  body.MentionType,
		Path:         body.Path,
		Operator:     operator,
	}
}

// ListUserGroupsOutput 告警组列表输出。
type ListUserGroupsOutput struct {
	Count   int64               `json:"count,string"`
	Results []*bkmapi.UserGroup `json:"results"`
}

// ListUserGroupsResp 告警组列表响应。
type ListUserGroupsResp struct {
	Data *ListUserGroupsOutput `json:"data"`
}

// GetUserGroupResp 告警组详情响应。
type GetUserGroupResp struct {
	Data *bkmapi.UserGroupDetail `json:"data"`
}

// SaveUserGroupResp 告警组创建/更新响应。
type SaveUserGroupResp struct {
	Data *bkmapi.UserGroupDetail `json:"data"`
}

// EmptyOutput is the JSON response for APIs that return no data.
type EmptyOutput struct{}
