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

package handler_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/handler"
)

var _ = Describe("LinkedProjectsEnableFailedCode", func() {
	failed := func(reasons ...workspace.LinkedProjectFailureReason) []workspace.LinkedProjectResult {
		results := make([]workspace.LinkedProjectResult, 0, len(reasons))
		for _, reason := range reasons {
			results = append(results, workspace.LinkedProjectResult{Reason: reason})
		}
		return results
	}

	DescribeTable("keeps caller-fixable failures as invalid request",
		func(reason workspace.LinkedProjectFailureReason) {
			code := handler.LinkedProjectsEnableFailedCode(failed(reason))
			Expect(code).To(Equal(bkerrs.ErrCodeInvalidRequest))
		},
		Entry("project not found", workspace.LinkedProjectFailureNotFound),
		Entry("no permission", workspace.LinkedProjectFailurePermission),
	)

	DescribeTable("maps platform failures to internal server error",
		func(reason workspace.LinkedProjectFailureReason) {
			code := handler.LinkedProjectsEnableFailedCode(failed(reason))
			Expect(code).To(Equal(bkerrs.ErrCodeInternalServerError))
		},
		Entry("platform unavailable", workspace.LinkedProjectFailureUnavailable),
		Entry("unrecognized platform error", workspace.LinkedProjectFailureUnknown),
	)

	It("returns internal server error when any failure is a platform fault", func() {
		code := handler.LinkedProjectsEnableFailedCode(failed(
			workspace.LinkedProjectFailurePermission,
			workspace.LinkedProjectFailureUnavailable,
		))
		Expect(code).To(Equal(bkerrs.ErrCodeInternalServerError))
	})
})
