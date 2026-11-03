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

package bkuser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Client with httptest server", func() {
	It("gets a user in the given tenant using app authorization header", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal(http.MethodGet))
			Expect(r.URL.Path).To(Equal("/api/bk-user/test/api/v3/open/tenant/users/alice/"))
			Expect(r.Header.Get("X-Bk-Tenant-Id")).To(Equal("default"))
			var authHeader map[string]string
			Expect(json.Unmarshal([]byte(r.Header.Get("X-Bkapi-Authorization")), &authHeader)).To(Succeed())
			Expect(authHeader).To(Equal(map[string]string{"bk_app_code": "bkms", "bk_app_secret": "secret"}))
			_, _ = w.Write([]byte(`{
				"data": {
					"tenant_id": "default",
					"bk_username": "alice",
					"login_name": "alice_login",
					"display_name": "alice(display)",
					"time_zone": "Asia/Shanghai",
					"language": "zh-cn",
					"status": "enabled"
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test", "bkms", "secret")
		Expect(err).NotTo(HaveOccurred())

		user, err := client.GetUser(context.Background(), "default", "alice")
		Expect(err).NotTo(HaveOccurred())
		Expect(user).To(Equal(&User{
			TenantID:    "default",
			BkUsername:  "alice",
			LoginName:   "alice_login",
			DisplayName: "alice(display)",
			TimeZone:    "Asia/Shanghai",
			Language:    "zh-cn",
			Status:      "enabled",
		}))
	})

	It("returns an error when the user payload has no tenant id", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{
				"data": {
					"tenant_id": "",
					"bk_username": "alice",
					"status": "enabled"
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test", "bkms", "secret")
		Expect(err).NotTo(HaveOccurred())

		_, err = client.GetUser(context.Background(), "default", "alice")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("tenant_id"))
	})

	It("returns ErrUserNotFound when the user is not in the given tenant", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": "user not found"}}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test", "bkms", "secret")
		Expect(err).NotTo(HaveOccurred())

		_, err = client.GetUser(context.Background(), "tenant-a", "alice")
		Expect(err).To(MatchError(ErrUserNotFound))
	})

	DescribeTable("does not treat a 404 without the bk-user NOT_FOUND code as ErrUserNotFound",
		func(header map[string]string, body string) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, value := range header {
					w.Header().Set(key, value)
				}
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()

			client, err := NewClient(server.URL, "test", "bkms", "secret")
			Expect(err).NotTo(HaveOccurred())

			_, err = client.GetUser(context.Background(), "tenant-a", "alice")
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(ErrUserNotFound))
		},
		Entry("gateway resource not found", map[string]string{
			"X-Bkapi-Error-Code":    "1640401",
			"X-Bkapi-Error-Message": "API not found",
		}, `{"code": 1640401, "result": false, "message": "API not found"}`),
		Entry("other business code", nil, `{"error": {"code": "INVALID_ARGUMENT", "message": "bad request"}}`),
		Entry("no error body", nil, `{}`),
	)

	It("returns the upstream error details when bk-user rejects the request", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{
				"error": {
					"code": "NO_PERMISSION",
					"message": "forbidden"
				}
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test", "bkms", "secret")
		Expect(err).NotTo(HaveOccurred())

		_, err = client.GetUser(context.Background(), "default", "alice")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("NO_PERMISSION"))
		Expect(err.Error()).To(ContainSubstring("forbidden"))
	})
})
