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

package polaris

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	polarisprovider "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/polaris"
)

var (
	// ErrConfigNotFound 北极星配置不存在
	ErrConfigNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound, "EXTENSION_POLARIS_CONFIG_NOT_FOUND", "polaris config not found",
	)
	// ErrConfigNameExists 北极星配置名称已存在
	ErrConfigNameExists = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest, "EXTENSION_POLARIS_CONFIG_NAME_EXISTS", "polaris config name already exists",
	)
	// ErrOperatorEmpty 不允许将负责人清空
	ErrOperatorEmpty = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest, "EXTENSION_POLARIS_OPERATOR_EMPTY", "operator cannot be empty",
	)
	// ErrNotManaged 部分字段仅平台创建的北极星服务允许修改
	ErrNotManaged = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_POLARIS_NOT_MANAGED",
		"field can only be set for platform-created polaris services",
	)
	// ErrImportedTokenUnauthorized 引入已有服务时用户提供的 Token 无效或没有写权限
	ErrImportedTokenUnauthorized = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"EXTENSION_POLARIS_TOKEN_UNAUTHORIZED",
		"polaris token is invalid or has no write permission",
	)
	// ErrImportedServiceNotFound 引入已有服务时用户指定的北极星服务不存在
	ErrImportedServiceNotFound = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest, "EXTENSION_POLARIS_SERVICE_NOT_FOUND", "polaris service not found",
	)
)

// classifyImportedServiceErr 为引入已有服务的北极星调用定码：服务名与 Token 都由用户提供，
// 鉴权失败和服务不存在属于调用方输入问题；平台创建的服务使用平台 Token，不经过这里
func classifyImportedServiceErr(err error) error {
	switch {
	case errors.Is(err, polarisprovider.ErrUnauthorized):
		return ErrImportedTokenUnauthorized.WithCause(err)
	case errors.Is(err, polarisprovider.ErrServiceNotFound):
		return ErrImportedServiceNotFound.WithCause(err)
	default:
		return err
	}
}
