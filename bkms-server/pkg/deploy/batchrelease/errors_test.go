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

package batchrelease_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/batchrelease"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/deploy/releasestrategy"
)

var _ = Describe("domain errors", func() {
	DescribeTable("carry their API error codes through wrapping",
		func(cause error, code bkerrs.ErrCode) {
			err := pkgerrors.Wrap(cause, "context")

			apiErr := bkerrs.FromError(err)
			Expect(apiErr).NotTo(BeNil())
			Expect(apiErr.Code()).To(Equal(code))
			Expect(apiErr.DetailCode()).NotTo(BeEmpty())
			Expect(errors.Is(err, cause)).To(BeTrue())
		},
		Entry("strategy name conflict", releasestrategy.ErrStrategyNameConflict, bkerrs.ErrCodeInvalidRequest),
		Entry("builtin strategy read only", releasestrategy.ErrBuiltinStrategyReadOnly, bkerrs.ErrCodeInvalidRequest),
		Entry("invalid batch target", releasestrategy.ErrInvalidBatchTarget, bkerrs.ErrCodeInvalidArgument),
		Entry("strategy not found", releasestrategy.ErrStrategyNotFound, bkerrs.ErrCodeNotFound),
		Entry("rollout not found", batchrelease.ErrRolloutNotFound, bkerrs.ErrCodeNotFound),
		Entry("rollout not installed", batchrelease.ErrRolloutNotInstalled, bkerrs.ErrCodeNotFound),
		Entry("batch release in progress", batchrelease.ErrBatchReleaseInProgress, bkerrs.ErrCodeInvalidRequest),
		Entry("workload not supported", batchrelease.ErrWorkloadNotSupported, bkerrs.ErrCodeInvalidRequest),
		Entry("action not allowed", batchrelease.ErrActionNotAllowed, bkerrs.ErrCodeInvalidRequest),
		Entry("no batch release in progress", batchrelease.ErrNoBatchReleaseInProgress, bkerrs.ErrCodeInvalidRequest),
		Entry("release switching", batchrelease.ErrReleaseSwitching, bkerrs.ErrCodeAborted),
		Entry("rollout changed", batchrelease.ErrRolloutChanged, bkerrs.ErrCodeAborted),
	)

	It("leaves cluster errors uncoded so they surface as internal errors", func() {
		err := pkgerrors.Wrap(errors.New("admission webhook denied the request"), "apply rollout")
		Expect(bkerrs.FromError(err)).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("admission webhook denied the request"))
	})
})
