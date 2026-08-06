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

package auth

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auth context", func() {
	Describe("WithUser", func() {
		It("should store the authenticated user in context", func() {
			ctx := WithUser(context.Background(), User{
				ID:   "alice",
				Cred: UserCredential{AccessToken: "token"},
			})

			user, err := GetUser(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(user.ID).To(Equal("alice"))
			Expect(user.Cred.AccessToken).To(Equal("token"))
		})
	})

	Describe("WithMaintenanceUser", func() {
		It("should store the maintenance user in context", func() {
			ctx := WithMaintenanceUser(context.Background())

			user, err := GetUser(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(user.ID).To(Equal(MaintenanceUserID))
			Expect(user.Cred.IsEmpty()).To(BeTrue())
		})
	})
})
