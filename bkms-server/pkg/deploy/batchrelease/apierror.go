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

var (
	errInvalidBatchTarget = bkerrs.Define(
		bkerrs.ErrCodeInvalidArgument,
		"DEPLOY_RELEASE_STRATEGY_INVALID_BATCH_TARGET",
		"invalid batch target in release strategy",
	)
	errReleaseStrategyNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound,
		"DEPLOY_RELEASE_STRATEGY_NOT_FOUND",
		"release strategy not found",
	)
	errRolloutNotFound = bkerrs.Define(
		bkerrs.ErrCodeNotFound,
		"DEPLOY_BATCH_RELEASE_ROLLOUT_NOT_FOUND",
		"rollout resource not found in cluster",
	)
	errRolloutNotInstalled = bkerrs.Define(
		bkerrs.ErrCodeNotFound,
		"DEPLOY_BATCH_RELEASE_ROLLOUT_NOT_INSTALLED",
		"rollout component is not installed in target cluster",
	)
	errStrategyNameConflict = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"DEPLOY_RELEASE_STRATEGY_NAME_CONFLICT",
		"release strategy name already exists",
	)
	errBuiltinStrategyReadOnly = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"DEPLOY_RELEASE_STRATEGY_BUILTIN_READ_ONLY",
		"builtin release strategy cannot be modified or deleted",
	)
	errBatchReleaseInProgress = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"DEPLOY_BATCH_RELEASE_IN_PROGRESS",
		"batch release is in progress",
	)
	errWorkloadNotSupported = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"DEPLOY_BATCH_RELEASE_WORKLOAD_NOT_SUPPORTED",
		"current workload does not support batch release",
	)
	errActionNotAllowed = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"DEPLOY_BATCH_RELEASE_ACTION_NOT_ALLOWED",
		"batch release action is not allowed in current state",
	)
	errNoBatchReleaseInProgress = bkerrs.Define(
		bkerrs.ErrCodeInvalidRequest,
		"DEPLOY_BATCH_RELEASE_NOT_IN_PROGRESS",
		"no batch release is in progress",
	)
	errReleaseSwitching = bkerrs.Define(
		bkerrs.ErrCodeAborted,
		"DEPLOY_BATCH_RELEASE_SWITCHING",
		"batch release is switching to the latest deploy",
	)
	errRolloutChanged = bkerrs.Define(
		bkerrs.ErrCodeAborted,
		"DEPLOY_BATCH_RELEASE_ROLLOUT_CHANGED",
		"batch release state has changed",
	)
)

// APIError 将发布策略与分批发布相关的错误转换为 API 错误：
// 4xx 领域错误映射到对应已定义的详情码（保留原错误作为 cause 供 details 排查）；
// 其余错误（含 Rollout CR 读写失败）以 INTERNAL_SERVER_ERROR 包装返回。
func APIError(err error, msg string) error {
	switch {
	case errors.Is(err, releasestrategy.ErrInvalidBatchTarget):
		return errInvalidBatchTarget.WithCause(err)
	case errors.Is(err, releasestrategy.ErrStrategyNotFound):
		return errReleaseStrategyNotFound.WithCause(err)
	case errors.Is(err, ErrRolloutNotFound):
		return errRolloutNotFound.WithCause(err)
	case errors.Is(err, ErrRolloutNotInstalled):
		return errRolloutNotInstalled.WithCause(err)
	case errors.Is(err, releasestrategy.ErrStrategyNameConflict):
		return errStrategyNameConflict.WithCause(err)
	case errors.Is(err, releasestrategy.ErrBuiltinStrategyReadOnly):
		return errBuiltinStrategyReadOnly.WithCause(err)
	case errors.Is(err, ErrBatchReleaseInProgress):
		return errBatchReleaseInProgress.WithCause(err)
	case errors.Is(err, ErrWorkloadNotSupported):
		return errWorkloadNotSupported.WithCause(err)
	case errors.Is(err, ErrActionNotAllowed):
		return errActionNotAllowed.WithCause(err)
	case errors.Is(err, ErrNoBatchReleaseInProgress):
		return errNoBatchReleaseInProgress.WithCause(err)
	case errors.Is(err, ErrReleaseSwitching):
		return errReleaseSwitching.WithCause(err)
	case errors.Is(err, ErrRolloutChanged):
		return errRolloutChanged.WithCause(err)
	default:
		return errors.Wrap(err, msg)
	}
}
