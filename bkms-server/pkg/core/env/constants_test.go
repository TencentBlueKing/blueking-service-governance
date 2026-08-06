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

package env_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
)

var _ = Describe("Environment type helpers", func() {
	It("recognizes staging as a valid env type", func() {
		Expect(bkmsenv.IsValidEnvType(string(bkmsenv.TypeStaging))).To(BeTrue())
	})

	It("PublicTypeOrder returns the expected ordered list", func() {
		expected := []bkmsenv.Type{
			bkmsenv.TypeDevelopment,
			bkmsenv.TypeTest,
			bkmsenv.TypeStaging,
			bkmsenv.TypeProduction,
		}
		Expect(bkmsenv.PublicTypeOrder()).To(Equal(expected))
	})

	It("PublicTypeOrder returns a defensive copy", func() {
		publicOrder := bkmsenv.PublicTypeOrder()
		publicOrder[0] = bkmsenv.TypeProduction

		Expect(bkmsenv.PublicTypeOrder()[0]).To(Equal(bkmsenv.TypeDevelopment))
	})
})
