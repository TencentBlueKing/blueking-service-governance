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

package handler

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"

var (
	errInvalidAppConfigFileID = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_INVALID_FILE_ID",
		"invalid app config file ID",
	)
	errInvalidBaseAppConfigFile = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_INVALID_BASE_FILE",
		"invalid base app config file",
	)
	errInvalidBSCPConfig = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_INVALID_BSCP_CONFIG",
		"invalid BSCP config",
	)
	errAttachFrameworkEnvOverlay = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_ATTACH_FRAMEWORK_ENV_OVERLAY_FAILED",
		"failed to attach framework env overlay",
	)
	errPlainEnvInstanceDeleteNotAllowed = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_PLAIN_ENV_INSTANCE_DELETE_NOT_ALLOWED",
		"plain env instance cannot be deleted via legacy API",
	)
	errInvalidFileContent = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_INVALID_FILE_CONTENT",
		"invalid config file content",
	)
	errInvalidFileType = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_INVALID_FILE_TYPE",
		"invalid config file type",
	)
	errInitFileEditorFailed = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_INIT_FILE_EDITOR_FAILED",
		"failed to initialize config file editor",
	)
	errSetFileContentFailed = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_SET_FILE_CONTENT_FAILED",
		"failed to set config file content",
	)
	errInvalidVersionID = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_INVALID_VERSION_ID",
		"invalid config file version ID",
	)
	errVersionNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound,
		"CORE_APPCFG_VERSION_NOT_FOUND",
		"app config file version not found",
	)
	errCompareVersionsDifferentFiles = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_COMPARE_VERSIONS_DIFFERENT_FILES",
		"compared versions belong to different files",
	)
	errRollbackTypeMismatch = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_ROLLBACK_TYPE_MISMATCH",
		"current file type does not match target version type",
	)
	errActiveVersionCannotBeDeleted = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APPCFG_ACTIVE_VERSION_CANNOT_BE_DELETED",
		"current active version cannot be deleted",
	)
)
