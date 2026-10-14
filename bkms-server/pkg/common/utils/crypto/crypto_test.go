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

package crypto_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
)

var _ = Describe("Test crypto tools", func() {
	DescribeTable("", func(
		name string,
		encFunc func(key, data string) (string, error),
		decFunc func(key, data string) (string, error),
		keySize int,
	) {
		plaintext := "hello world"
		key, err := crypto.GenerateKey(keySize)
		Expect(err).NotTo(HaveOccurred())

		// 第一次加密
		ciphertext1, err := encFunc(key, plaintext)
		Expect(err).NotTo(HaveOccurred())

		// 第二次加密
		ciphertext2, err := encFunc(key, plaintext)
		Expect(err).NotTo(HaveOccurred())

		// 确保两次加密后的结果不一致
		Expect(ciphertext1).NotTo(Equal(ciphertext2))

		// 解密第一次加密的结果
		decrypted1, err := decFunc(key, ciphertext1)
		Expect(err).NotTo(HaveOccurred())
		Expect(plaintext).To(Equal(decrypted1))

		// 解密第二次加密的结果
		decrypted2, err := decFunc(key, ciphertext2)
		Expect(err).NotTo(HaveOccurred())
		Expect(plaintext).To(Equal(decrypted2))
	},
		Entry("case AES 24 bytes", "AES", crypto.AESEncrypt, crypto.AESDecrypt, 24),
		Entry("case AES 32 bytes", "AES", crypto.AESEncrypt, crypto.AESDecrypt, 32),
	)

	Describe("OnDemand", func() {
		var key string

		BeforeEach(func() {
			var err error
			key, err = crypto.GenerateKey(32)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should round trip non-empty data", func() {
			ciphertext, err := crypto.OnDemand(key, crypto.AESEncrypt, "secret")
			Expect(err).NotTo(HaveOccurred())
			Expect(ciphertext).NotTo(Equal("secret"))

			plaintext, err := crypto.OnDemand(key, crypto.AESDecrypt, ciphertext)
			Expect(err).NotTo(HaveOccurred())
			Expect(plaintext).To(Equal("secret"))
		})

		It("should leave empty data untouched", func() {
			Expect(crypto.OnDemand(key, crypto.AESEncrypt, "")).To(BeEmpty())
			Expect(crypto.OnDemand(key, crypto.AESDecrypt, "")).To(BeEmpty())
		})

		It("should propagate errors from fn", func() {
			_, err := crypto.OnDemand(key, crypto.AESDecrypt, "not-hex")
			Expect(err).To(MatchError(ContainSubstring("hex decode string")))
		})
	})
})
