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

package account

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/usertoken"
)

var _ = Describe("APIHandler.CreateToken", func() {
	var tokenClient *usertoken.MockTokenClient

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		tokenClient = usertoken.NewMockTokenClient(GinkgoT())
	})

	doRequest := func(credential string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		h := NewHandler(Config{AuthEnvName: "prod", TokenCredential: credential}, tokenClient)
		r := gin.New()
		r.GET("/user_token/token", h.CreateToken)

		req := httptest.NewRequest(http.MethodGet, "/user_token/token", nil)
		for _, c := range cookies {
			req.AddCookie(c)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	It("issues token with bk_token cookie and no username", func() {
		tokenClient.EXPECT().
			GetToken(mock.Anything, "", map[string]string{"bk_token": "token"}, "prod", false).
			Return(&usertoken.AccessToken{AccessToken: "access", RefreshToken: "refresh", ExpiresIn: 100}, nil)

		w := doRequest(usertoken.CredentialBkToken, &http.Cookie{Name: "bk_token", Value: "token"})

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.String()).To(MatchJSON(`{"access_token":"access","refresh_token":"refresh","expires_in":100}`))
	})

	It("issues token with bk_ticket cookie and username", func() {
		tokenClient.EXPECT().
			GetToken(mock.Anything, "admin", map[string]string{"bk_ticket": "ticket"}, "prod", false).
			Return(&usertoken.AccessToken{AccessToken: "access"}, nil)

		w := doRequest(
			usertoken.CredentialBkTicket,
			&http.Cookie{Name: "bk_uid", Value: "admin"},
			&http.Cookie{Name: "bk_ticket", Value: "ticket"},
		)

		Expect(w.Code).To(Equal(http.StatusOK))
	})

	It("rejects bk_ticket credential without username", func() {
		w := doRequest(usertoken.CredentialBkTicket, &http.Cookie{Name: "bk_ticket", Value: "ticket"})

		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(w.Body.String()).To(MatchJSON(
			`{"code":"1000","message":"no user credentials found in cookie, please login first."}`,
		))
	})

	It("rejects request without the configured credential", func() {
		w := doRequest(usertoken.CredentialBkToken, &http.Cookie{Name: "bk_ticket", Value: "ticket"})

		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})
})
