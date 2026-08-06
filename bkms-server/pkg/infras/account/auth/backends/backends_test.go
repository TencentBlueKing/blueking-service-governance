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

package backends

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type userInfoBackend interface {
	GetUserInfo(context.Context, string) (*UserInfo, error)
}

type userCredentialBackend interface {
	GetUserCredential(*http.Request) string
}

var _ = Describe("Auth backends", func() {
	DescribeTable("从请求中获取用户票据",
		func(backend userCredentialBackend, header, cookieName string) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.AddCookie(&http.Cookie{Name: cookieName, Value: "cookie-credential"})
			Expect(backend.GetUserCredential(request)).To(Equal("cookie-credential"))

			request.Header.Set(header, "header-credential")
			Expect(backend.GetUserCredential(request)).To(Equal("header-credential"))
		},

		Entry("bk_ticket", NewBkTicketAuthBackend(""), "X-User-Bk-Ticket", "bk_ticket"),
		Entry("bk_token", NewBkTokenAuthBackend(""), "X-User-Bk-Token", "bk_token"),
	)

	It("使用 bk_ticket 获取用户信息", func() {
		server := newUserInfoServer(
			"/user/get_info/", "bk_ticket", "ticket", `{"ret":0,"data":{"username":"blueking"}}`,
		)
		defer server.Close()

		user, err := NewBkTicketAuthBackend(server.URL).GetUserInfo(context.Background(), "ticket")

		Expect(err).NotTo(HaveOccurred())
		Expect(user).To(Equal(&UserInfo{ID: "blueking"}))
	})

	It("使用 bk_token 获取用户信息", func() {
		server := newUserInfoServer(
			"/accounts/get_user/", "bk_token", "token", `{"code":0,"data":{"username":"blueking"}}`,
		)
		defer server.Close()

		user, err := NewBkTokenAuthBackend(server.URL).GetUserInfo(context.Background(), "token")

		Expect(err).NotTo(HaveOccurred())
		Expect(user).To(Equal(&UserInfo{ID: "blueking"}))
	})

	DescribeTable("返回用户信息接口错误",
		func(response, expectedError string, buildBackend func(string) userInfoBackend) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(response))
			}))
			defer server.Close()

			_, err := buildBackend(server.URL).GetUserInfo(context.Background(), "credential")

			Expect(err).To(MatchError(ContainSubstring(expectedError)))
		},
		Entry("bk_ticket 返回错误码", `{"ret":1,"msg":"invalid ticket"}`, "invalid ticket",
			func(url string) userInfoBackend {
				return NewBkTicketAuthBackend(url)
			}),
		Entry("bk_token 返回错误码", `{"code":1,"message":"invalid token"}`, "invalid token",
			func(url string) userInfoBackend {
				return NewBkTokenAuthBackend(url)
			}),
		Entry("响应缺少用户名", `{"code":0,"data":{}}`, "username not found",
			func(url string) userInfoBackend {
				return NewBkTokenAuthBackend(url)
			}),
	)
})

// 拉起用于单元测试的 HTTP Server，验证请求路径和查询参数，并返回指定响应
func newUserInfoServer(path, queryKey, expectedCredential, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		Expect(request.URL.Path).To(Equal(path))
		Expect(request.URL.Query().Get(queryKey)).To(Equal(expectedCredential))
		_, _ = w.Write([]byte(response))
	}))
}
