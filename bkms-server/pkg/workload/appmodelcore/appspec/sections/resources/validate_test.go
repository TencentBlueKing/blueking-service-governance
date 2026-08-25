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

package resources

import (
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

var _ = Describe("validation", func() {
	var validate *validator.Validate

	BeforeEach(func() {
		validate = validator.New(validator.WithRequiredStructEnabled())
		RegisterValidation(validate)
	})

	It("allows a lone CPU limit so overlays can change only that field", func() {
		err := validate.Struct(Spec{CPULimits: lo.ToPtr("2")})
		Expect(err).NotTo(HaveOccurred())
	})

	It("allows a lone CPU request", func() {
		err := validate.Struct(Spec{CPURequests: lo.ToPtr("500m")})
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects request larger than limit when both are present", func() {
		err := validate.Struct(Spec{
			CPURequests: lo.ToPtr("1500m"),
			CPULimits:   lo.ToPtr("1200m"),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("resource_request_lte_limit"))
	})
})
