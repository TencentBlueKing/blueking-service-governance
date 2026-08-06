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

package snapshot

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GenerateRepoKey", func() {
	It("should produce deterministic output for same input", func() {
		key1 := GenerateRepoKey("registry.example.com/myapp", "user1", "pass1")
		key2 := GenerateRepoKey("registry.example.com/myapp", "user1", "pass1")
		Expect(key1).To(Equal(key2))
	})

	It("should produce different keys for different credentials", func() {
		key1 := GenerateRepoKey("registry.example.com/myapp", "user1", "pass1")
		key2 := GenerateRepoKey("registry.example.com/myapp", "user2", "pass2")
		Expect(key1).NotTo(Equal(key2))
	})

	It("should produce different keys for different registry addresses", func() {
		key1 := GenerateRepoKey("registry1.example.com/myapp", "user1", "pass1")
		key2 := GenerateRepoKey("registry2.example.com/myapp", "user1", "pass1")
		Expect(key1).NotTo(Equal(key2))
	})

	It("should produce output of 64 hex characters (full SHA256)", func() {
		key := GenerateRepoKey("registry.example.com/myapp", "user1", "pass1")
		Expect(key).To(HaveLen(64))
		Expect(key).To(MatchRegexp("^[0-9a-f]{64}$"))
	})

	It("should produce different keys when same address but different passwords", func() {
		key1 := GenerateRepoKey("registry.example.com/myapp", "user1", "passA")
		key2 := GenerateRepoKey("registry.example.com/myapp", "user1", "passB")
		Expect(key1).NotTo(Equal(key2))
	})
})
