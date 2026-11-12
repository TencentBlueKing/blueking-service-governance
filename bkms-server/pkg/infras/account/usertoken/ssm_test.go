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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ TokenClient = (*SSMTokenClient)(nil)

var _ = Describe("SSMTokenClient with httptest server", func() {
	readJSONBody := func(r *http.Request) map[string]string {
		Expect(r.Method).To(Equal(http.MethodPost))
		Expect(r.Header.Get("X-Bk-App-Code")).To(Equal("app-code"))
		Expect(r.Header.Get("X-Bk-App-Secret")).To(Equal("app-secret"))
		Expect(r.Header.Get("Content-Type")).To(ContainSubstring("application/json"))
		body, err := io.ReadAll(r.Body)
		Expect(err).NotTo(HaveOccurred())
		payload := map[string]string{}
		Expect(json.Unmarshal(body, &payload)).To(Succeed())
		return payload
	}

	It("requests token, refreshes token and resolves user info", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/auth/access-tokens":
				Expect(readJSONBody(r)).To(Equal(map[string]string{
					"grant_type":  "authorization_code",
					"id_provider": "bk_login",
					"bk_token":    "token",
				}))
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"access_token":"access-1",` +
					`"refresh_token":"refresh-1","expires_in":43200,` +
					`"identity":{"user_type":"bkuser","username":"admin"}}}`))
			case "/api/v1/auth/access-tokens/verify":
				Expect(readJSONBody(r)).To(Equal(map[string]string{"access_token": "access-1"}))
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"bk_app_code":"app-code",` +
					`"expires_in":43006,"grant_type":"authorization_code",` +
					`"identity":{"user_type":"bkuser","username":"admin"}}}`))
			case "/api/v1/auth/access-tokens/refresh":
				Expect(readJSONBody(r)).To(Equal(map[string]string{"refresh_token": "refresh-1"}))
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"access_token":"access-2",` +
					`"refresh_token":"refresh-1","expires_in":43200,` +
					`"identity":{"user_type":"bkuser","username":"admin"}}}`))
			default:
				Fail("unexpected request path: " + r.URL.Path)
			}
		}))
		defer server.Close()

		client := NewSSMTokenClient(server.URL, "app-code", "app-secret")

		accessToken, err := client.GetToken(
			context.Background(), "", map[string]string{CredentialBkToken: "token"}, "prod", true,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(accessToken).To(Equal(&AccessToken{
			AccessToken:  "access-1",
			RefreshToken: "refresh-1",
			ExpiresIn:    43200,
		}))

		username, err := client.GetUserInfo(context.Background(), accessToken.AccessToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(username).To(Equal("admin"))

		refreshedToken, err := client.RefreshToken(context.Background(), accessToken.RefreshToken, "prod", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(refreshedToken).To(Equal(&AccessToken{
			AccessToken:  "access-2",
			RefreshToken: "refresh-1",
			ExpiresIn:    43200,
		}))
	})

	It("returns error when the access token is invalid", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"code":1901403,"message":"access_token invalid","data":{}}`))
		}))
		defer server.Close()

		client := NewSSMTokenClient(server.URL, "app-code", "app-secret")
		_, err := client.GetUserInfo(context.Background(), "invalid")
		Expect(err).To(MatchError(ContainSubstring("1901403")))
	})

	It("reports the status code when the service fails with a non-JSON body", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`<html>502 Bad Gateway</html>`))
		}))
		defer server.Close()

		client := NewSSMTokenClient(server.URL, "app-code", "app-secret")
		_, err := client.GetUserInfo(context.Background(), "access-1")
		Expect(err).To(MatchError(ContainSubstring("status 502")))
		Expect(err).To(MatchError(ContainSubstring("parsing response result")))
	})
})
