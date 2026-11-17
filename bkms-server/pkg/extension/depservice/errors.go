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

package depservice

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

// ErrInvalidArgument 参数/业务校验失败
var ErrInvalidArgument = bkerrs.Define(
	bkerrs.ErrCodeInvalidRequest,
	"EXTENSION_DEPSERVICE_INVALID_ARGUMENT",
	"invalid dependency service argument",
)

// ErrUpdateNotSupported provider 不支持更新服务实例
var ErrUpdateNotSupported = errors.New("service instance update is not supported")

var (
	errInstanceInUse = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_INSTANCE_IN_USE",
		"service instance is used by bindings, cannot delete",
	)
	errInstanceProvisioning = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_INSTANCE_PROVISIONING",
		"service instance is provisioning, cannot delete",
	)
	errInstanceDeleting = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_INSTANCE_DELETING",
		"service instance is deleting, cannot delete",
	)
	errInstanceStatusNotDeletable = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_INSTANCE_STATUS_NOT_DELETABLE",
		"service instance status does not allow delete",
	)
	errBindingEnvNameEmpty = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_BINDING_ENV_NAME_EMPTY",
		"env name in envInstanceMap must not be empty",
	)
	errBindingInstanceIDEmpty = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_BINDING_INSTANCE_ID_EMPTY",
		"instance id for env must not be empty",
	)
	// ErrBindingEnvNotFound 绑定中引用的环境不存在
	ErrBindingEnvNotFound = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_BINDING_ENV_NOT_FOUND",
		"environment not found",
	)
	errBindingInstanceNotFound = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_BINDING_INSTANCE_NOT_FOUND",
		"service instance not found",
	)
	errBindingInstanceWorkspaceMismatch = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_BINDING_INSTANCE_WORKSPACE_MISMATCH",
		"instance does not belong to workspace",
	)
	errBindingInstanceServiceMismatch = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_BINDING_INSTANCE_SERVICE_MISMATCH",
		"instance service type mismatch",
	)
	errBindingInstanceEnvUnavailable = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_DEPSERVICE_BINDING_INSTANCE_ENV_UNAVAILABLE",
		"instance is not available in environment",
	)
)
