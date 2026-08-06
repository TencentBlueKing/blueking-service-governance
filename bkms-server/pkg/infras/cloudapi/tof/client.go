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

import "context"

// Client TOF API 客户端接口
//
// 抽象出接口以便 facade 依赖抽象而非具体实现：未注册具体实现时，
// factory 会 fallback 到 noopClient。
type Client interface {
	// GetStaffInfo 获取员工信息
	GetStaffInfo(ctx context.Context, username string) (*StaffInfo, error)

	// GetDeptInfo 获取指定部门信息
	GetDeptInfo(ctx context.Context, deptID string) (*DeptInfo, error)

	// GetParentDeptInfos 获取指定部门的所有父部门信息
	GetParentDeptInfos(ctx context.Context, deptID string) ([]DeptInfo, error)
}
