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

package appcfgfiledef

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"

var (
	errInvalidConfigSpec = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_INVALID_CONFIG_SPEC",
		"invalid config file spec",
	)
	errInvalidConfigKind = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_INVALID_CONFIG_KIND",
		"invalid config kind for application",
	)
	errInvalidBaseFile = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_INVALID_BASE_FILE",
		"invalid base app config file",
	)
	errInvalidBSCPConfig = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_INVALID_BSCP_CONFIG",
		"invalid BSCP config",
	)
	errInvalidQuery = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_INVALID_QUERY",
		"invalid query parameters",
	)
	errDefaultEnvCannotBeReset = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_DEFAULT_ENV_CANNOT_RESET",
		"default env cannot be reset",
	)
	errResetRequiresIndependentMode = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_RESET_REQUIRES_INDEPENDENT_MODE",
		"reset requires independent env config mode",
	)
	errInvalidDefID = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_DEF_INVALID_ID",
		"invalid config file def ID",
	)
	errDefNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound,
		"CORE_APPCFG_DEF_NOT_FOUND",
		"config file def not found",
	)
)
