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

package workspace

import "github.com/pkg/errors"

// LinkedSystem 关联项目所属平台
type LinkedSystem string

const (
	// LinkedSystemBKCI 蓝盾
	LinkedSystemBKCI LinkedSystem = "bkci"
	// LinkedSystemBKMonitor 蓝鲸监控
	LinkedSystemBKMonitor LinkedSystem = "bkmonitor"
	// LinkedSystemBCS 容器服务
	LinkedSystemBCS LinkedSystem = "bcs"
)

// LinkedProjectOrigin 关联项目来源
type LinkedProjectOrigin string

const (
	// LinkedProjectCreated 随空间创建
	LinkedProjectCreated LinkedProjectOrigin = "created"
	// LinkedProjectBound 创建空间时绑定的已有项目
	LinkedProjectBound LinkedProjectOrigin = "bound"
)

// LinkedProjectState 关联项目在平台侧的启停状态
type LinkedProjectState string

const (
	// LinkedProjectReady 启用中
	LinkedProjectReady LinkedProjectState = "Ready"
	// LinkedProjectDisabled 已停用
	LinkedProjectDisabled LinkedProjectState = "Disabled"
	// LinkedProjectUnknown 查询平台失败，状态未知
	LinkedProjectUnknown LinkedProjectState = "Unknown"
)

// LinkedProjectBlockedReason 关联项目不能由 BKMS 启停的原因
type LinkedProjectBlockedReason string

const (
	// LinkedProjectUnsupported 平台不支持 BKMS 启停该项目
	LinkedProjectUnsupported LinkedProjectBlockedReason = "unsupported"
	// LinkedProjectNoPermission 当前用户没有该项目的管理权限
	LinkedProjectNoPermission LinkedProjectBlockedReason = "noPermission"
)

// LinkedProjectFailureReason 关联项目启停失败的原因分类，前端据此展示处理建议
type LinkedProjectFailureReason string

const (
	// LinkedProjectFailureNotFound 项目在平台已不存在
	LinkedProjectFailureNotFound LinkedProjectFailureReason = "notFound"
	// LinkedProjectFailurePermission 当前用户没有项目的管理权限
	LinkedProjectFailurePermission LinkedProjectFailureReason = "permission"
	// LinkedProjectFailureUnavailable 平台暂时无法响应
	LinkedProjectFailureUnavailable LinkedProjectFailureReason = "unavailable"
	// LinkedProjectFailureUnknown 无法识别的平台错误
	LinkedProjectFailureUnknown LinkedProjectFailureReason = "unknown"
)

// ErrLinkedProjectNotOperable 请求启停的关联项目不存在或不能由 BKMS 启停
var ErrLinkedProjectNotOperable = errors.New("linked project is not operable")

// LinkedProject 工作空间关联的平台项目
type LinkedProject struct {
	// System 所属平台
	System LinkedSystem
	// ProjectID 平台项目 ID：蓝盾、容器为项目 Code，监控为空间 ID
	ProjectID string
	// ProjectName 平台项目名称，查询平台失败时为 ProjectID
	ProjectName string
	// Origin 来源
	Origin LinkedProjectOrigin
	// State 平台侧启停状态
	State LinkedProjectState
	// BlockedReason 不能启停的原因，为空表示可以启停
	BlockedReason LinkedProjectBlockedReason

	// code 调用平台启停接口使用的项目标识，监控为容器项目 Code 而非空间 ID
	code string
}

// Operable 项目是否可以由 BKMS 启停
func (p LinkedProject) Operable() bool {
	return p.BlockedReason == ""
}

// LinkedProjectResult 单个关联项目的启停结果
type LinkedProjectResult struct {
	// System 所属平台
	System LinkedSystem
	// ProjectID 平台项目 ID
	ProjectID string
	// ProjectName 平台项目名称
	ProjectName string
	// Success 是否成功
	Success bool
	// Reason 失败原因分类
	Reason LinkedProjectFailureReason
	// Detail 平台返回的原始错误信息
	Detail string
	// RequestID 平台请求 ID
	RequestID string
}
