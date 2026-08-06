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

package tof

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

// ParentDeptNotFound 指定类型的父部门不存在
var ParentDeptNotFound = errors.New("parent dept not found")

// GetUserOrganization 获取用户组织信息
func GetUserOrganization(ctx context.Context, username string) (*Organization, error) {
	client, err := New()
	if err != nil {
		return nil, err
	}
	info, err := client.GetStaffInfo(ctx, username)
	if err != nil {
		return nil, err
	}
	// 无部门信息（如 noop 实现返回空）时视为无组织信息，返回空组织
	// 下游 CreateProject 对空组织信息天然容忍
	if info.DeptID == "" {
		return nil, nil
	}

	bgInfo, err := getParentDeptInfoByDeptIDAndType(ctx, info.DeptID, info.DeptName, OstBG)
	if err != nil {
		return nil, err
	}
	// 注：目前中心信息非必需，可以先不调 API 获取
	// 如果要获取中心，需要考虑该层级以上的员工使用什么默认值？
	org := Organization{
		BgID:      bgInfo.ID,
		BgName:    bgInfo.Name,
		DeptID:    info.DeptID,
		DeptName:  info.DeptName,
		GroupID:   info.GroupID,
		GroupName: info.GroupName,
	}
	return &org, nil
}

// getParentDeptInfoByDeptIDAndType 通过指定的部门 ID 获取指定类型的父部门
func getParentDeptInfoByDeptIDAndType(
	ctx context.Context, deptID, deptName string, ost orgStructureType,
) (*DeptInfo, error) {
	client, err := New()
	if err != nil {
		return nil, err
	}
	if deptID == "" {
		return nil, ParentDeptNotFound
	}

	// 高层级部门特殊处理
	if deptID == "0" || deptID == "1" {
		return &DeptInfo{ID: deptID, Name: deptName}, nil
	}

	parentDeptInfos, err := client.GetParentDeptInfos(ctx, deptID)
	if err != nil {
		return nil, err
	}
	for _, deptInfo := range parentDeptInfos {
		if orgStructureType(cast.ToInt(deptInfo.TypeID)) == ost {
			return &deptInfo, nil
		}
	}

	log.Warnf(
		ctx, "current organization <%s:%s> lost parent dept with type %d, try get self info",
		deptID, deptName, ost,
	)
	curDeptInfo, err := client.GetDeptInfo(ctx, deptID)
	if err != nil {
		return nil, err
	}
	if orgStructureType(cast.ToInt(curDeptInfo.TypeID)) == ost {
		return curDeptInfo, nil
	}
	return nil, ParentDeptNotFound
}
