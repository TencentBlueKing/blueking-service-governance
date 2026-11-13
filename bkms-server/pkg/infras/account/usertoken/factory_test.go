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

package usertoken

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
)

var _ = Describe("NewTokenClient", func() {
	app := config.BkAppConfig{Code: "app-code", Secret: "app-secret"}
	account := config.AccountConfig{
		AuthBaseURL: "http://apigw.example.com",
		SSMBaseURL:  "http://bkssm.example.com",
	}

	It("uses SSM when the auth backend is bk_token and SSM is configured", func() {
		account := account
		account.BackendType = config.AuthBackendBkToken
		client := NewTokenClient(context.Background(), account, app)
		Expect(client).To(BeAssignableToTypeOf(&SSMTokenClient{}))
		Expect(client.CredentialName()).To(Equal(CredentialBkToken))
		Expect(client.RequiresUsername()).To(BeFalse())
	})

	It("falls back to the API gateway when SSM is not configured", func() {
		account := account
		account.BackendType = config.AuthBackendBkToken
		account.SSMBaseURL = ""
		client := NewTokenClient(context.Background(), account, app)
		Expect(client).To(BeAssignableToTypeOf(&APIGatewayTokenClient{}))
		Expect(client.CredentialName()).To(Equal(CredentialBkTicket))
		Expect(client.RequiresUsername()).To(BeTrue())
	})

	It("uses the API gateway when the auth backend is bk_ticket", func() {
		account := account
		account.BackendType = config.AuthBackendBkTicket
		client := NewTokenClient(context.Background(), account, app)
		Expect(client).To(BeAssignableToTypeOf(&APIGatewayTokenClient{}))
		Expect(client.CredentialName()).To(Equal(CredentialBkTicket))
	})
})
