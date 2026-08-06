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

package iam

import (
	"context"
	"strconv"
	"strings"

	"github.com/TencentBlueKing/bk-apigateway-sdks/core/bkapi"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/iam/types"
)

// 用户组相关常量
const (
	// userGroupMemberType 用户组成员类型，目前固定为 user
	userGroupMemberType = "user"
	// userGroupMembersListPage 用户组成员列表查询起始页
	userGroupMembersListPage = "1"
	// userGroupMembersListPageSize 用户组成员列表单页大小，按照 10000 评估，可视为全量查询
	userGroupMembersListPageSize = "10000"
)

// CreateUserGroups 在分级管理员下批量创建用户组。
func (c *BKIAMClient) CreateUserGroups(
	ctx context.Context,
	gradeManagerID int,
	groups ...types.UserGroupParam,
) ([]types.UserGroup, error) {
	const errPrefix = "create user groups error"

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "v2_management_grade_manager_create_groups",
		Method: "POST",
		Path:   "/api/v2/open/management/systems/{system_id}/grade_managers/{grade_manager_id}/groups/",
	}, bkapi.OptSetRequestPathParams(map[string]string{
		"grade_manager_id": strconv.Itoa(gradeManagerID),
		"system_id":        config.G.BkIAMSystemIDs.Bkms,
	}), bkapi.OptSetRequestBody(map[string]any{"groups": groups}))

	var resp types.CreateUserGroupsResp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return nil, errors.Wrap(err, errPrefix)
	}

	if resp.Code != 0 {
		return nil, errors.Errorf("%s: %s", errPrefix, resp.Message)
	}

	ids := resp.Data
	if len(ids) != len(groups) {
		return nil, errors.Errorf(
			"%s: response id count %d does not match group count %d",
			errPrefix, len(ids), len(groups),
		)
	}

	userGroups := make([]types.UserGroup, len(ids))
	for idx, id := range ids {
		userGroups[idx] = types.UserGroup{
			ID:          id,
			Name:        groups[idx].Name,
			Description: groups[idx].Description,
			Readonly:    groups[idx].Readonly,
		}
	}
	return userGroups, nil
}

// DeleteUserGroup 删除用户组。
func (c *BKIAMClient) DeleteUserGroup(ctx context.Context, userGroupID int) error {
	const errPrefix = "delete user group error"

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "v2_management_grade_manager_delete_group",
		Method: "DELETE",
		Path:   "/api/v2/open/management/systems/{system_id}/groups/{group_id}/",
	}, bkapi.OptSetRequestPathParams(map[string]string{
		"group_id":  strconv.Itoa(userGroupID),
		"system_id": config.G.BkIAMSystemIDs.Bkms,
	}))

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return errors.Wrap(err, errPrefix)
	}

	if resp.Code != 0 {
		return errors.Errorf("%s: %s", errPrefix, resp.Message)
	}

	return nil
}

// GrantUserGroupPolicies 给指定的用户组授权。
//
// 注意：IAM 网关授权接口要求每次请求只授权一个 system 的一组 actions。
// 因此对 authScopes 列表逐个发送请求；任意一次失败立即返回错误，不会回滚。
//
// 由此可能产生的副作用：当 authScopes 包含多个 scope 时，若中途某次请求失败，
// 之前已成功的 scope 不会被回滚，会留下"部分授权"的中间态。调用方在重试时
// 需自行保证幂等性——好在 IAM 授权接口对相同 (group, scope) 的重复授权本身是幂等的，
// 直接整体重试即可，不会因为部分 scope 已生效而产生重复授权或冲突。
func (c *BKIAMClient) GrantUserGroupPolicies(
	ctx context.Context,
	userGroupID int,
	authScopes []types.AuthorizationScope,
) error {
	const errPrefix = "grant user group policies error"

	for _, scope := range authScopes {
		op := c.NewOperation(
			bkapi.OperationConfig{
				Name:   "management_groups_policies",
				Method: "POST",
				Path:   "/api/v1/open/management/groups/{group_id}/policies/",
			},
			bkapi.OptSetRequestPathParams(map[string]string{
				"group_id": strconv.Itoa(userGroupID),
			}),
			bkapi.OptSetRequestBody(scope),
		)

		var resp types.Resp
		if err := c.executeOperation(ctx, op, &resp); err != nil {
			return errors.Wrap(err, errPrefix)
		}

		if resp.Code != 0 {
			return errors.Errorf("%s: %s", errPrefix, resp.Message)
		}
	}

	return nil
}

// AddUserGroupMembers 添加用户组成员，其中 expiredAt 为过期时间戳。
func (c *BKIAMClient) AddUserGroupMembers(
	ctx context.Context,
	userGroupID int,
	members []string,
	expiredAt int,
) error {
	const errPrefix = "add user group members error"

	userMembers := make([]types.UserMemberParam, len(members))
	for idx, member := range members {
		userMembers[idx] = types.UserMemberParam{
			Type: userGroupMemberType,
			ID:   member,
		}
	}

	reqBody := types.AddUserGroupMembersReq{
		Members:   userMembers,
		ExpiredAt: expiredAt,
	}

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v2_management_add_group_members",
			Method: "POST",
			Path:   "/api/v2/open/management/systems/{system_id}/groups/{group_id}/members/",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"system_id": config.G.BkIAMSystemIDs.Bkms,
			"group_id":  strconv.Itoa(userGroupID),
		}),
		bkapi.OptSetRequestBody(reqBody),
	)

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return errors.Wrap(err, errPrefix)
	}

	if resp.Code != 0 {
		return errors.Errorf("%s: %s", errPrefix, resp.Message)
	}

	return nil
}

// DeleteUserGroupMembers 删除某个用户组的成员。
func (c *BKIAMClient) DeleteUserGroupMembers(
	ctx context.Context,
	userGroupID int,
	members []string,
) error {
	const errPrefix = "delete user group members error"

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v2_management_delete_group_members",
			Method: "DELETE",
			Path:   "/api/v2/open/management/systems/{system_id}/groups/{group_id}/members/",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"system_id": config.G.BkIAMSystemIDs.Bkms,
			"group_id":  strconv.Itoa(userGroupID),
		}),
		bkapi.OptAddRequestQueryParam("type", userGroupMemberType),
		bkapi.OptAddRequestQueryParam("ids", strings.Join(members, ",")),
	)

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return errors.Wrap(err, errPrefix)
	}

	if resp.Code != 0 {
		return errors.Errorf("%s: %s", errPrefix, resp.Message)
	}

	return nil
}

// ListUserGroupMembers 获取用户组成员。
func (c *BKIAMClient) ListUserGroupMembers(
	ctx context.Context,
	userGroupID int,
) ([]types.UserMember, error) {
	const errPrefix = "list user group members error"

	op := c.NewOperation(
		bkapi.OperationConfig{
			Name:   "v2_management_group_members",
			Method: "GET",
			Path:   "/api/v2/open/management/systems/{system_id}/groups/{group_id}/members/",
		},
		bkapi.OptSetRequestPathParams(map[string]string{
			"system_id": config.G.BkIAMSystemIDs.Bkms,
			"group_id":  strconv.Itoa(userGroupID),
		}),
		bkapi.OptSetRequestQueryParams(map[string]string{
			"page":      userGroupMembersListPage,
			"page_size": userGroupMembersListPageSize,
		}),
	)

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return nil, errors.Wrap(err, errPrefix)
	}

	if resp.Code != 0 {
		return nil, errors.Errorf("%s: %s", errPrefix, resp.Message)
	}

	data := types.UserMemberData{}
	if err := mapstructure.Decode(resp.Data, &data); err != nil {
		return nil, errors.Wrap(err, errPrefix)
	}

	return data.Results, nil
}
