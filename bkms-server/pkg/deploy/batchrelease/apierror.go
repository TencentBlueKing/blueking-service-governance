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

package batchrelease

import (
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
)

// APIError 将发布策略与分批发布相关的错误转换为 API 错误，msg 作为错误信息传给 bkerrs.Wrap。
// 只映射现有 ErrCode，不附加详情码；集群返回的信息保留在错误链中。
//
//   - 策略名称已存在、批次目标值不合法 → INVALID_ARGUMENT
//   - 策略不存在、Rollout CR 不存在、集群未安装 Rollout → NOT_FOUND
//   - 分批发布进行中、主工作负载不支持、当前状态不满足、没有进行中的分批发布 → INVALID_REQUEST
//   - 其他错误（含 Rollout CR 读写失败）→ INTERNAL_SERVER_ERROR
func APIError(err error, msg string) error {
	code := bkerrs.ErrCodeInternalServerError
	switch {
	case errors.Is(err, releasestrategy.ErrStrategyNameConflict),
		errors.Is(err, releasestrategy.ErrInvalidBatchTarget):
		code = bkerrs.ErrCodeInvalidArgument
	case errors.Is(err, releasestrategy.ErrStrategyNotFound),
		errors.Is(err, ErrRolloutNotFound),
		errors.Is(err, ErrRolloutNotInstalled):
		code = bkerrs.ErrCodeNotFound
	case errors.Is(err, ErrBatchReleaseInProgress),
		errors.Is(err, ErrWorkloadNotSupported),
		errors.Is(err, ErrActionNotAllowed),
		errors.Is(err, ErrNoBatchReleaseInProgress):
		code = bkerrs.ErrCodeInvalidRequest
	}
	return bkerrs.Wrap(err, code, msg)
}
