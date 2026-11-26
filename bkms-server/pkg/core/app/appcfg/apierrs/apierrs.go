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

// Package apierrs 把 appcfg 领域错误映射为 API 错误的规则，供配置文件相关的 handler 共用
package apierrs

import (
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app/appcfg"
	bscpapierrs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp/apierrs"
)

// BSCPConfigRules 校验用户所选的 BSCP 配置引用（appcfg.ResolveBSCPConfig）时由用户输入或权限导致的错误。
// BSCP 未启用由全局规则处理；调用 BSCP 失败等其余错误不命中，由错误出口按内部错误处理
var BSCPConfigRules = bkerrs.Rules(
	bscpapierrs.NoPermission,
	bkerrs.WhenIs(
		appcfg.ErrBSCPConfigRequired,
		bkerrs.ErrCodeInvalidArgument,
		i18n.Msg("bscp config is required when content source type is bscp"),
	),
	bkerrs.WhenIs(
		appcfg.ErrBSCPNoFullyReleasedVersion,
		bkerrs.ErrCodeInvalidRequest,
		i18n.Msg("the bscp service has no fully released version"),
	),
	bkerrs.WhenIs(
		appcfg.ErrBSCPContentInvalidYAML,
		bkerrs.ErrCodeInvalidArgument,
		i18n.Msg("bscp config content is not valid YAML"),
	),
)
