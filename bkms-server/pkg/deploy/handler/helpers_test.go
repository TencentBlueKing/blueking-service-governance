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

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("wrapDeployErr", func() {
	It("maps shared namespace unsupported errors to invalid request with a detail code", func() {
		cause := errors.Wrap(bkmsenv.ErrSharedNamespaceUnsupported, "components [redis] cannot be used in env feat-1")

		err := wrapDeployErr(cause, "pre-check deployment")

		var bkErr *bkerrs.Error
		Expect(errors.As(err, &bkErr)).To(BeTrue())
		Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
		details := bkErr.Details()
		Expect(details).To(HaveLen(1))
		Expect(details.AsMaps()[0]["code"]).To(Equal(bkerrs.ErrDetailCodeSharedNamespaceUnsupported))
		Expect(err).To(MatchError(ContainSubstring("components [redis] cannot be used in env feat-1")))
	})

	It("keeps other errors as internal server errors", func() {
		err := wrapDeployErr(errors.New("cluster unavailable"), "execute deploy steps")

		var bkErr *bkerrs.Error
		Expect(errors.As(err, &bkErr)).To(BeTrue())
		Expect(bkErr.Code()).To(Equal(bkerrs.ErrCodeInternalServerError))
		Expect(bkErr.Details()).To(BeEmpty())
	})
})
