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

package client

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
)

var _ = Describe("AuthClient", func() {
	var testServer *httptest.Server

	BeforeEach(func() {
		testServer = httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/user_token/validate":
					// 验证鉴权请求不携带 Authorization 头
					if authHeader := r.Header.Get("Authorization"); authHeader != "" {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"error": "auth request should not carry Authorization header"}`))
						return
					}
					accessToken := r.URL.Query().Get("access_token")
					switch accessToken {
					case "valid_token":
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"username": "blueking"}`))
					case "expired_token":
						w.WriteHeader(http.StatusUnauthorized)
					default:
						w.WriteHeader(http.StatusInternalServerError)
					}
				case "/user_token/token":
					// 验证鉴权请求不携带 Authorization 头
					if authHeader := r.Header.Get("Authorization"); authHeader != "" {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"error": "auth request should not carry Authorization header"}`))
						return
					}
					uidCookie, uidErr := r.Cookie("bk_uid")
					ticketCookie, ticketErr := r.Cookie("bk_ticket")
					// 缺少必要的 cookie
					if uidErr != nil || ticketErr != nil || uidCookie.Value == "" || ticketCookie.Value == "" {
						w.WriteHeader(http.StatusBadRequest)
						_, _ = w.Write([]byte(`{"message": "no user credentials found in cookie"}`))
						return
					}
					if ticketCookie.Value != "valid_ticket" || uidCookie.Value != "testuser" {
						w.WriteHeader(http.StatusUnauthorized)
						_, _ = w.Write([]byte(`{"message": "invalid credentials"}`))
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"access_token": "exchanged_token"}`))
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}),
		)

		config.G = &config.Config{
			BkmsBaseURL: testServer.URL,
			AccessToken: "some_existing_token", // 模拟已有 token，验证鉴权请求不会携带它
		}
	})

	AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
	})

	Describe("ValidateAccessToken", func() {
		It("should return username when token is valid", func() {
			authCli := New()
			username, err := authCli.ValidateAccessToken("valid_token")
			Expect(err).ToNot(HaveOccurred())
			Expect(username).To(Equal("blueking"))
		})

		It("should return error with status code when token is expired", func() {
			authCli := New()
			_, err := authCli.ValidateAccessToken("expired_token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("401"))
		})

		It("should return error with status code when server returns error", func() {
			authCli := New()
			_, err := authCli.ValidateAccessToken("error_token")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("500"))
		})

		It("should not carry Authorization header in auth requests", func() {
			authCli := New()
			username, err := authCli.ValidateAccessToken("valid_token")
			Expect(err).ToNot(HaveOccurred())
			Expect(username).To(Equal("blueking"))
		})
	})

	Describe("ExchangeBkTicketForToken", func() {
		It("should exchange valid bk_uid + bk_ticket for access_token", func() {
			authCli := New()
			token, err := authCli.ExchangeBkTicketForToken("testuser", "valid_ticket")
			Expect(err).ToNot(HaveOccurred())
			Expect(token).To(Equal("exchanged_token"))
		})

		It("should return error for invalid bk_ticket", func() {
			authCli := New()
			_, err := authCli.ExchangeBkTicketForToken("testuser", "invalid_ticket")
			Expect(err).To(HaveOccurred())
		})

		It("should return error when bk_uid is missing", func() {
			authCli := New()
			_, err := authCli.ExchangeBkTicketForToken("", "valid_ticket")
			Expect(err).To(HaveOccurred())
		})

		It("should not carry Authorization header in auth requests", func() {
			authCli := New()
			token, err := authCli.ExchangeBkTicketForToken("testuser", "valid_ticket")
			Expect(err).ToNot(HaveOccurred())
			Expect(token).To(Equal("exchanged_token"))
		})
	})
})
