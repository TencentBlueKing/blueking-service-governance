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
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ TokenClient = (*APIGatewayTokenClient)(nil)

var _ = Describe("APIGatewayTokenClient with httptest server", func() {
	It("requests token, refreshes token and resolves user info", func() {
		// Set up a test HTTP server to mock the API Gateway responses
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/auth_api/token/":
				Expect(r.Method).To(Equal(http.MethodPost))
				Expect(r.Header.Get("X-Bk-App-Code")).To(Equal("app-code"))
				Expect(r.Header.Get("X-Bk-App-Secret")).To(Equal("app-secret"))

				Expect(r.Header.Get("Content-Type")).To(ContainSubstring("application/json"))
				body, err := io.ReadAll(r.Body)
				Expect(err).NotTo(HaveOccurred())

				payload := map[string]string{}
				Expect(json.Unmarshal(body, &payload)).To(Succeed())
				Expect(payload).To(HaveKeyWithValue("app_code", "app-code"))
				Expect(payload).To(HaveKeyWithValue("app_secret", "app-secret"))
				Expect(payload).To(HaveKeyWithValue("env_name", "test"))
				Expect(payload).To(HaveKeyWithValue("grant_type", "authorization_code"))
				Expect(payload).To(HaveKeyWithValue("rtx", "blueking"))
				Expect(payload).To(HaveKeyWithValue("bk_ticket", "ticket"))
				Expect(payload).To(HaveKeyWithValue("need_new_token", "1"))

				_, _ = w.Write(
					[]byte(
						`{"code":0,"message":"ok","result":true,"data":{"access_token":"access-1","refresh_token":"refresh-1","expires_in":7200}}`,
					),
				)
			case "/auth_api/check_token/":
				Expect(r.Method).To(Equal(http.MethodGet))
				Expect(r.URL.Query().Get("access_token")).To(Equal("access-1"))
				Expect(r.Header.Get("X-Bk-App-Code")).To(Equal("app-code"))
				Expect(r.Header.Get("X-Bk-App-Secret")).To(Equal("app-secret"))

				_, _ = w.Write(
					[]byte(
						`{"code":"0","message":"ok","result":true,"data":{"id_providers":{"rtx":{"username":"blueking"}}}}`,
					),
				)
			case "/auth_api/refresh_token/":
				Expect(r.Method).To(Equal(http.MethodGet))
				Expect(r.URL.Query()).To(Equal(url.Values{
					"app_code":       []string{"app-code"},
					"app_secret":     []string{"app-secret"},
					"env_name":       []string{"test"},
					"grant_type":     []string{"refresh_token"},
					"refresh_token":  []string{"refresh-1"},
					"need_new_token": []string{"0"},
				}))

				_, _ = w.Write(
					[]byte(
						`{"code":0,"message":"ok","result":true,"data":{"access_token":"access-2","refresh_token":"refresh-2","expires_in":3600}}`,
					),
				)
			default:
				Fail("unexpected request path: " + r.URL.Path)
			}
		}))
		defer server.Close()

		client := NewAPIGatewayTokenClient(server.URL, "app-code", "app-secret")

		accessToken, err := client.GetToken(
			context.Background(),
			"blueking",
			map[string]string{"bk_ticket": "ticket"},
			"test",
			true,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(accessToken).To(Equal(&AccessToken{
			AccessToken:  "access-1",
			RefreshToken: "refresh-1",
			ExpiresIn:    7200,
		}))

		username, err := client.GetUserInfo(context.Background(), accessToken.AccessToken)
		Expect(err).NotTo(HaveOccurred())
		Expect(username).To(Equal("blueking"))

		refreshedToken, err := client.RefreshToken(context.Background(), accessToken.RefreshToken, "test", false)
		Expect(err).NotTo(HaveOccurred())
		Expect(refreshedToken).To(Equal(&AccessToken{
			AccessToken:  "access-2",
			RefreshToken: "refresh-2",
			ExpiresIn:    3600,
		}))
	})
})
