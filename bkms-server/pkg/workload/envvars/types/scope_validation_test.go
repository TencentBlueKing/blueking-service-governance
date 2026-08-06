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

package types

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env"
)

var _ = Describe("ParseScopedEnvVarScope", func() {
	It("accepts staging env type as a valid scope", func() {
		scope, err := ParseScopedEnvVarScope(string(ScopeTypeEnvType), string(bkmsenv.TypeStaging))
		Expect(err).NotTo(HaveOccurred())
		Expect(scope).To(Equal(ScopeEnvType(string(bkmsenv.TypeStaging))))
	})

	It("lists staging in validation error for invalid env type", func() {
		_, err := ParseScopedEnvVarScope(string(ScopeTypeEnvType), "invalid")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("staging"))
	})
})
