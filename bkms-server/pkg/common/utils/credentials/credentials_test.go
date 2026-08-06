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

package credentials

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ValidateOptionalUserPass", func() {
	It("should pass when username and password are both empty", func() {
		Expect(ValidateOptionalUserPass("", "")).To(Succeed())
	})

	It("should pass when username and password both have values", func() {
		Expect(ValidateOptionalUserPass("alice", "secret")).To(Succeed())
	})

	It("should reject partial credentials", func() {
		Expect(ValidateOptionalUserPass("alice", "")).To(MatchError(ErrInvalidUserPass))
		Expect(ValidateOptionalUserPass("", "secret")).To(MatchError(ErrInvalidUserPass))
	})

	It("should reject whitespace credentials", func() {
		Expect(ValidateOptionalUserPass("  ", "  ")).To(MatchError(ErrInvalidUserPass))
		Expect(ValidateOptionalUserPass("alice", "  ")).To(MatchError(ErrInvalidUserPass))
	})
})

var _ = Describe("HasUserPass", func() {
	It("should only report credentials with non-whitespace username and password", func() {
		Expect(HasUserPass("alice", "secret")).To(BeTrue())
		Expect(HasUserPass("alice", "")).To(BeFalse())
		Expect(HasUserPass("  ", "  ")).To(BeFalse())
	})
})
