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

package instancelog

import (
	"errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	appmodeldeploy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/appmodel"
)

// WrapManagerError maps instance log manager errors to bkms API errors.
func WrapManagerError(err error, appID, envName, instanceID string) error {
	switch {
	case errors.Is(err, appmodeldeploy.ErrDeployRecordNotFound):
		return bkerrs.Wrapf(
			err,
			bkerrs.ErrCodeNotFound,
			"deploy record not found for app %s env %s",
			appID,
			envName,
		)
	case errors.Is(err, ErrInstanceNotFound):
		return bkerrs.Wrapf(
			err,
			bkerrs.ErrCodeNotFound,
			"instance %s not found in app %s env %s",
			instanceID,
			appID,
			envName,
		)
	default:
		return bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "new instance log manager")
	}
}
