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

// noopClient 是 TOF Client 的默认空实现
//
// 在未配置可用 TOF 服务时使用，所有查询均返回空值。上层 facade
// GetUserOrganization 会据此返回空组织，CreateProject 对空组织信息天然容忍。
type noopClient struct{}

// 编译期确认 noopClient 实现 Client 接口
var _ Client = noopClient{}

// newNoopClient 创建 noopClient
func newNoopClient() Client {
	return noopClient{}
}

// GetStaffInfo 空实现，返回空员工信息
func (noopClient) GetStaffInfo(_ context.Context, _ string) (*StaffInfo, error) {
	return &StaffInfo{}, nil
}

// GetDeptInfo 空实现，返回空部门信息
func (noopClient) GetDeptInfo(_ context.Context, _ string) (*DeptInfo, error) {
	return &DeptInfo{}, nil
}

// GetParentDeptInfos 空实现，返回空父部门列表
func (noopClient) GetParentDeptInfos(_ context.Context, _ string) ([]DeptInfo, error) {
	return nil, nil
}
