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

// Package bkmonitor 提供蓝鲸监控相关功能
package bkmonitor

import (
	"fmt"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// BuildUserGroupName 构造环境对应的告警组名称
func BuildUserGroupName(envName string) string {
	return fmt.Sprintf(userGroupNameTmpl, envName)
}

// ApplyMemberDiff 对当前 users 列表应用增量对比：
//   - toRemove 中 Type==user 的成员被删除；
//   - toAdd 中不存在的成员作为 Type=user 追加到末尾；
//   - Type==group 默认成员保持不变；
//   - 冲突时 toRemove 优先。
//
// 返回变更后的列表和是否发生变更。
func ApplyMemberDiff(
	users []bkmapi.UserGroupUser,
	toAdd []string,
	toRemove []string,
) ([]bkmapi.UserGroupUser, bool) {
	removeSet := make(map[string]bool, len(toRemove))
	for _, id := range toRemove {
		if id == "" {
			continue
		}
		removeSet[id] = true
	}

	newUsers := make([]bkmapi.UserGroupUser, 0, len(users)+len(toAdd))
	existingUserIDs := make(map[string]bool, len(users))
	changed := false
	for _, u := range users {
		// 仅删除 type=user 成员，type=group 默认成员保留
		if u.Type == userGroupUserTypeUser {
			if _, shouldRemove := removeSet[u.ID]; shouldRemove {
				changed = true
				continue
			}
			existingUserIDs[u.ID] = true
		}
		newUsers = append(newUsers, u)
	}

	for _, id := range toAdd {
		if id == "" {
			continue
		}
		// toRemove 优先
		if _, shouldRemove := removeSet[id]; shouldRemove {
			continue
		}
		if _, exists := existingUserIDs[id]; exists {
			continue
		}
		changed = true
		existingUserIDs[id] = true
		newUsers = append(newUsers, bkmapi.UserGroupUser{
			ID:          id,
			DisplayName: id,
			Type:        userGroupUserTypeUser,
		})
	}

	return newUsers, changed
}

// buildSaveUserGroupReq 基于 detail 构造 SaveUserGroup
func buildSaveUserGroupReq(
	detail *bkmapi.UserGroupDetail,
	newUsers []bkmapi.UserGroupUser,
	operator string,
) *bkmapi.SaveUserGroupReq {
	newArranges := make([]bkmapi.DutyArrange, len(detail.DutyArranges))
	copy(newArranges, detail.DutyArranges)
	newArranges[0].Users = newUsers

	timezone := detail.Timezone
	if timezone == "" {
		timezone = saveUserGroupDefaultTimezone
	}

	dutyNotice := detail.DutyNotice
	if dutyNotice == nil {
		dutyNotice = &bkmapi.DutyNotice{}
	}

	mentionList := detail.MentionList
	if mentionList == nil {
		mentionList = []bkmapi.UserGroupUser{}
	}

	return &bkmapi.SaveUserGroupReq{
		ID:           &detail.ID,
		BkBizID:      detail.BkBizID,
		Name:         detail.Name,
		Desc:         detail.Desc,
		DutyArranges: newArranges,
		DutyRules:    detail.DutyRules,
		AlertNotice:  detail.AlertNotice,
		ActionNotice: detail.ActionNotice,
		DutyNotice:   dutyNotice,
		NeedDuty:     detail.NeedDuty,
		Channels:     detail.Channels,
		Timezone:     timezone,
		Operator:     operator,
		Path:         detail.Path,
		MentionList:  mentionList,
		MentionType:  detail.MentionType,
	}
}
