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

package credential

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	svccfg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/crypto"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// setUpCredStore 初始化加密 secret，并通过 fx 构造凭证 Store
func setUpCredStore() *Store {
	secret, err := crypto.GenerateKey(32)
	Expect(err).NotTo(HaveOccurred())
	svccfg.G = &svccfg.Config{Encrypt: svccfg.EncryptConfig{Secret: secret}}

	return testutil.NewStore(GinkgoT(), Meta)
}

var _ = Describe("HelmRepoCredential store", func() {
	var (
		ctx       context.Context
		credStore *Store
	)

	BeforeEach(func() {
		ctx = context.Background()
		credStore = setUpCredStore()
	})

	newCred := func(prefix, password string) *HelmRepoCredential {
		return &HelmRepoCredential{
			WorkspaceID:  prefix + stringx.Random(8),
			CredentialID: helmRepoCredentialID,
			Username:     "admin",
			Password:     password,
		}
	}

	It("should return ErrHelmRepoCredentialNotFound for non-existent workspace", func() {
		_, err := credStore.Get(ctx, ByWorkspace("non-existent-ws-"+stringx.Random(8)))
		Expect(errors.Is(err, ErrHelmRepoCredentialNotFound)).To(BeTrue())
	})

	It("should return a duplicate error when creating credential for the same workspace twice", func() {
		cred := newCred("ws-dup-test-", "password")
		Expect(credStore.Create(ctx, cred)).To(Succeed())

		dup := *cred
		err := credStore.Create(ctx, &dup)
		Expect(errors.Is(err, store.ErrDuplicate)).To(BeTrue())
	})

	rawPassword := func(workspaceID string) string {
		raw, err := credStore.Collection().FindOne(ctx, bson.M{"workspaceID": workspaceID}).Raw()
		Expect(err).NotTo(HaveOccurred())
		return raw.Lookup("password").StringValue()
	}

	// insertRaw 绕过 Store 直接写入原始文档，用于验证读取路径的解密
	insertRaw := func(prefix, password string) string {
		workspaceID := prefix + stringx.Random(8)
		_, err := credStore.Collection().InsertOne(ctx, bson.M{
			"workspaceID": workspaceID, "credentialID": helmRepoCredentialID, "username": "admin", "password": password,
		})
		Expect(err).NotTo(HaveOccurred())
		return workspaceID
	}

	It("should create and retrieve credential with empty password", func() {
		cred := newCred("ws-empty-pwd-", "")
		Expect(credStore.Create(ctx, cred)).To(Succeed())
		Expect(rawPassword(cred.WorkspaceID)).To(BeEmpty())

		got, err := credStore.Get(ctx, ByWorkspace(cred.WorkspaceID))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Username).To(Equal("admin"))
		Expect(got.Password).To(BeEmpty())
	})

	It("should store encrypted password in DB without touching the caller's object", func() {
		plainPassword := "my-secret-password"
		cred := newCred("ws-encrypt-test-", plainPassword)
		Expect(credStore.Create(ctx, cred)).To(Succeed())
		Expect(cred.Password).To(Equal(plainPassword))

		// 直接读取库中原始文档，密码应为密文
		stored := rawPassword(cred.WorkspaceID)
		Expect(stored).NotTo(Equal(plainPassword))
		decrypted, err := crypto.AESDecrypt(svccfg.G.Encrypt.Secret, stored)
		Expect(err).NotTo(HaveOccurred())
		Expect(decrypted).To(Equal(plainPassword))
	})

	It("should decrypt the stored ciphertext on Get and List", func() {
		ciphertext, err := crypto.AESEncrypt(svccfg.G.Encrypt.Secret, "my-secret-password")
		Expect(err).NotTo(HaveOccurred())
		workspaceID := insertRaw("ws-decrypt-test-", ciphertext)

		got, err := credStore.Get(ctx, ByWorkspace(workspaceID))
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Password).To(Equal("my-secret-password"))

		list, err := credStore.List(ctx, ByWorkspace(workspaceID))
		Expect(err).NotTo(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Password).To(Equal("my-secret-password"))
	})

	It("should fail to read a password that is not valid ciphertext", func() {
		workspaceID := insertRaw("ws-bad-cipher-", "plain-password")

		_, err := credStore.Get(ctx, ByWorkspace(workspaceID))
		Expect(err).To(MatchError(ContainSubstring("decrypt password")))

		_, err = credStore.List(ctx, ByWorkspace(workspaceID))
		Expect(err).To(MatchError(ContainSubstring("decrypt password")))
	})
})
