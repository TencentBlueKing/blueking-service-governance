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
	errInvalidHelmSpec = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_INVALID_HELM_SPEC",
		"invalid helm spec",
	)
	errHelmRepoCredentialsInvalid = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APP_HELM_REPO_CREDENTIALS_INVALID",
		"invalid helm repo credentials",
	)
	errAppAlreadyExists = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_ALREADY_EXISTS",
		"app already exists",
	)
	errInvalidBuildConfig = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_INVALID_BUILD_CONFIG",
		"invalid build config",
	)
	errAppHasActiveDeployments = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_HAS_ACTIVE_DEPLOYMENTS",
		"app has active deployments, please remove them first",
	)
	errVisibleEnvsNotFound = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_VISIBLE_ENVS_NOT_FOUND",
		"visible environments do not exist",
	)
	errVisibleEnvsFeatureNotAllowed = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_VISIBLE_ENVS_FEATURE_NOT_ALLOWED",
		"feature environments are not allowed in visible environments",
	)
	errVisibleEnvsMixedInvalid = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_VISIBLE_ENVS_INVALID",
		"invalid visible environment names",
	)
	errVisibleEnvHasDeployment = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_VISIBLE_ENV_HAS_DEPLOYMENT",
		"environment has a deployment and must remain visible",
	)
	errVisibleEnvHasFeatureChild = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_VISIBLE_ENV_HAS_FEATURE_CHILD",
		"environment is the parent of a feature environment and must remain visible",
	)
	errDeployOverviewUnsupportedType = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"CORE_APP_DEPLOY_OVERVIEW_UNSUPPORTED_TYPE",
		"deploy overview is only supported for trpc/taf apps",
	)
	errUnsupportedAppType = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_UNSUPPORTED_TYPE",
		"unsupported app type",
	)
	errInvalidBKMSAppParams = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_INVALID_BKMSAPP_PARAMS",
		"invalid bkmsApp parameters",
	)
	errStartupConfigUnsupportedType = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_STARTUP_CONFIG_UNSUPPORTED_TYPE",
		"app type does not support startup-config",
	)
	errInvalidStartupConfig = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_INVALID_STARTUP_CONFIG",
		"invalid startup config",
	)
	errBKMSAppMustUseStartupConfig = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_BKMSAPP_MUST_USE_STARTUP_CONFIG",
		"bkmsApp app must use startup-config",
	)
	errOnlyTAFSupportsTafSpec = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_ONLY_TAF_SUPPORTS_TAF_SPEC",
		"only TAF app supports taf spec",
	)
	errTafSpecRequired = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_TAF_SPEC_REQUIRED",
		"appModelSpec.tafSpec is required",
	)
	errOnlyTRPCSupportsTrpcSpec = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_ONLY_TRPC_SUPPORTS_TRPC_SPEC",
		"only tRPC app supports trpc spec",
	)
	errTrpcSpecRequired = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_TRPC_SPEC_REQUIRED",
		"appModelSpec.trpcSpec is required",
	)
	errInvalidConfigSpecRequest = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_INVALID_CONFIG_SPEC",
		"invalid config spec",
	)
	errReferenceComponentCannotUpdate = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"CORE_APP_REFERENCE_COMPONENT_CANNOT_UPDATE",
		"reference component cannot be updated",
	)
)
