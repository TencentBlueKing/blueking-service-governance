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

// Package apierrs 把 BSCP client 的错误映射为 API 错误的规则，供 BSCP 拦截中间件、全局错误规则
// 以及会调用 BSCP 的 handler 共用
package apierrs

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
)

const disabledMsg = i18n.Msg("bscp is disabled in current environment")

var (
	// Disabled 当前环境关闭了 BSCP 对接，含义与具体接口无关，注册为全局规则
	Disabled = bkerrs.WhenIsFunc(bscp.ErrDisabled, WrapDisabled)

	// NoPermission 把 bscp.ErrNoPermission 映射为无权限错误，前端据详情码 BSCP_NO_PERMISSION 提示用户申请业务权限。
	// 不注册为全局规则：client 把 BSCP 返回的 401 / 403 都视为无权限，其中也包括平台应用凭证失效、未获网关授权等
	// 用户无法自行解决的情况，只有用户浏览、选择自己的 BSCP 资源时，提示申请权限才大概率成立
	NoPermission = bkerrs.WhenIsFunc(bscp.ErrNoPermission, wrapNoPermission)
)

// WrapDisabled 包装为当前环境关闭了 BSCP 对接的错误，拦截中间件与 Disabled 规则共用
func WrapDisabled(err error) *bkerrs.Error {
	return bkerrs.Wrap(err, bkerrs.ErrCodeNotImplemented, disabledMsg).SetDetails(
		bkerrs.NewDetail(
			bkerrs.ErrDetailCodeBSCPDisabled,
			disabledMsg,
			bkerrs.WithSystem(bkerrs.SystemName),
			bkerrs.WithModule(bscp.ModuleName),
		),
	)
}

func wrapNoPermission(err error) *bkerrs.Error {
	return bkerrs.Wrap(err, bkerrs.ErrCodeNoPermission, i18n.Msg("no permission to access bscp resources")).SetDetails(
		bkerrs.NewDetail(
			bkerrs.ErrDetailCodeBSCPNoPermission,
			"bscp service permission denied",
			bkerrs.WithSystem(bkerrs.SystemName),
			bkerrs.WithModule(bscp.ModuleName),
		),
	)
}
