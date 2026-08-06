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

// 分级管理员相关默认查询参数
const (
	// gradeManagerListPage 分级管理员列表查询起始页
	gradeManagerListPage = "1"
	// gradeManagerListPageSize 分级管理员列表查询单页大小，按照 15000 个 workspace 评估，可视为全量查询
	gradeManagerListPageSize = "15000"
)

// CreateGradeManager 创建分级管理员，返回分级管理员 ID。
//
// 当目标 name 已存在（IAM 网关返回 ConflictCode）时，自动改用 GetGradeManagerByName
// 查询并返回已存在的分级管理员 ID，保证幂等。
func (c *BKIAMClient) CreateGradeManager(
	ctx context.Context,
	name string,
	description string,
	members []string,
	authScopes []types.AuthorizationScope,
) (*int, error) {
	const errPrefix = "create grade manager error"

	reqBody := types.CreateGradeManagerReq{
		System:              config.G.BkIAMSystemIDs.Bkms,
		Name:                name,
		Description:         description,
		Members:             members,
		AuthorizationScopes: authScopes,
		SubjectScopes:       []types.SubjectScope{{Type: "*", ID: "*"}},
	}

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "management_grade_managers",
		Method: "POST",
		Path:   "/api/v1/open/management/grade_managers/",
	}, bkapi.OptSetRequestBody(reqBody))

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return nil, errors.Wrap(err, errPrefix)
	}

	if resp.Code == 0 {
		manager := types.GradeManager{}
		if err := mapstructure.Decode(resp.Data, &manager); err != nil {
			return nil, errors.Wrap(err, errPrefix)
		}
		return &manager.ID, nil
	}

	if resp.Code == ConflictCode {
		return c.GetGradeManagerByName(ctx, name)
	}

	return nil, errors.Errorf("%s: %s", errPrefix, resp.Message)
}

// GetGradeManagerByName 根据 name 查询分级管理员 ID。
func (c *BKIAMClient) GetGradeManagerByName(ctx context.Context, name string) (*int, error) {
	const errPrefix = "get grade manager error"

	params := map[string]string{
		"system":    config.G.BkIAMSystemIDs.Bkms,
		"page":      gradeManagerListPage,
		"page_size": gradeManagerListPageSize,
	}

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "management_grade_managers_list",
		Method: "GET",
		Path:   "/api/v1/open/management/grade_managers/",
	}, bkapi.OptSetRequestQueryParams(params))

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return nil, errors.Wrap(err, errPrefix)
	}

	if resp.Code != 0 {
		return nil, errors.Errorf("%s: %s", errPrefix, resp.Message)
	}

	data := types.GradeManagerData{}
	if err := mapstructure.Decode(resp.Data, &data); err != nil {
		return nil, errors.Wrap(err, errPrefix)
	}

	for _, d := range data.Results {
		if d.Name == name {
			return &d.ID, nil
		}
	}

	return nil, errors.Errorf("%s: %s not found", errPrefix, name)
}

// UpdateGradeManager 更新 id 为 gradeManagerID 的分级管理员信息。
func (c *BKIAMClient) UpdateGradeManager(
	ctx context.Context,
	gradeManagerID int,
	name string,
	description string,
	authScopes []types.AuthorizationScope,
) error {
	const errPrefix = "update grade manager error"

	reqBody := types.UpdateGradeManagerReq{
		System:              config.G.BkIAMSystemIDs.Bkms,
		Name:                name,
		Description:         description,
		AuthorizationScopes: authScopes,
		SubjectScopes:       []types.SubjectScope{{Type: "*", ID: "*"}},
	}

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "management_grade_managers_update",
		Method: "PUT",
		Path:   "/api/v1/open/management/grade_managers/{grade_manager_id}/",
	}, bkapi.OptSetRequestPathParams(map[string]string{
		"grade_manager_id": strconv.Itoa(gradeManagerID),
	}), bkapi.OptSetRequestBody(reqBody))

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return errors.Wrap(err, errPrefix)
	}

	// 当 IAM 网关返回 ConflictCode 时，通常意味着新 name 已被其他分级管理员占用。
	// bkms-server 在该层将 Conflict 视为更新成功（幂等语义），不向上抛错；
	// 重名约束由调用方在更高层（业务侧）保证，避免重复名校验逻辑下沉到 L1 客户端。
	if resp.Code == 0 || resp.Code == ConflictCode {
		return nil
	}

	return errors.Errorf("%s: %s", errPrefix, resp.Message)
}

// DeleteGradeManager 删除 id 为 gradeManagerID 的分级管理员。
func (c *BKIAMClient) DeleteGradeManager(ctx context.Context, gradeManagerID int) error {
	const errPrefix = "delete grade manager error"

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "v2_management_delete_grade_manager",
		Method: "DELETE",
		Path:   "/api/v2/open/management/systems/{system_id}/grade_managers/{grade_manager_id}/",
	}, bkapi.OptSetRequestPathParams(map[string]string{
		"grade_manager_id": strconv.Itoa(gradeManagerID),
		"system_id":        config.G.BkIAMSystemIDs.Bkms,
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

// AddGradeManagerMembers 将 members 添加到 id 为 gradeManagerID 的分级管理员成员中。
func (c *BKIAMClient) AddGradeManagerMembers(
	ctx context.Context,
	gradeManagerID int,
	members []string,
) error {
	const errPrefix = "add grade manager members error"

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "management_add_grade_manager_members",
		Method: "POST",
		Path:   "/api/v1/open/management/grade_managers/{grade_manager_id}/members/",
	}, bkapi.OptSetRequestPathParams(map[string]string{
		"grade_manager_id": strconv.Itoa(gradeManagerID),
	}), bkapi.OptSetRequestBody(map[string][]string{
		"members": members,
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

// DeleteGradeManagerMembers 从 id 为 gradeManagerID 的分级管理员中删除指定 members。
func (c *BKIAMClient) DeleteGradeManagerMembers(
	ctx context.Context,
	gradeManagerID int,
	members []string,
) error {
	const errPrefix = "delete grade manager members error"

	op := c.NewOperation(bkapi.OperationConfig{
		Name:   "management_delete_grade_manager_members",
		Method: "DELETE",
		Path:   "/api/v1/open/management/grade_managers/{grade_manager_id}/members/",
	}, bkapi.OptSetRequestPathParams(map[string]string{
		"grade_manager_id": strconv.Itoa(gradeManagerID),
	}), bkapi.OptAddRequestQueryParam("members", strings.Join(members, ",")))

	var resp types.Resp
	if err := c.executeOperation(ctx, op, &resp); err != nil {
		return errors.Wrap(err, errPrefix)
	}

	if resp.Code != 0 {
		return errors.Errorf("%s: %s", errPrefix, resp.Message)
	}

	return nil
}
