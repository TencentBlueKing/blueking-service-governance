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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth/backends"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/usertoken"
)

var _ = Describe("User authentication middleware", func() {
	var authBackend *MockAuthBackend
	var request *http.Request

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		authBackend = NewMockAuthBackend(GinkgoT())
		authBackend.EXPECT().GetLoginUrl().Return("http://login.example.com/plain/").Maybe()
		request = httptest.NewRequest(http.MethodGet, "/", nil)
	})

	Context("用户票据认证", func() {
		It("票据为空时保留未认证结果", func() {
			authBackend.EXPECT().GetUserCredential(request).Return("")

			result := authenticate(
				context.Background(),
				request,
				authBackend,
				BackendBkTicket,
				usertoken.NewMockTokenClient(GinkgoT()),
				false,
			)
			anonymous, ok := result.RequestUser.(AnonymousUser)
			Expect(ok).To(BeTrue())

			Expect(result.RequestUser.IsAuthenticated()).To(BeFalse())
			Expect(result.LoginURL).To(Equal("http://login.example.com/plain/"))
			Expect(result.ErrorMsg).To(ContainSubstring("no token or access token"))
			Expect(anonymous.Credential().IsEmpty()).To(BeTrue())
		})

		It("票据无效时保留票据信息和认证错误", func() {
			authBackend.EXPECT().GetUserCredential(request).Return("invalid-ticket")
			authBackend.EXPECT().GetUserInfo(mock.Anything, "invalid-ticket").
				Return(nil, errors.New("invalid ticket"))

			result := authenticate(
				context.Background(),
				request,
				authBackend,
				BackendBkTicket,
				usertoken.NewMockTokenClient(GinkgoT()),
				false,
			)
			anonymous, ok := result.RequestUser.(AnonymousUser)
			Expect(ok).To(BeTrue())

			Expect(result.RequestUser.IsAuthenticated()).To(BeFalse())
			Expect(result.ErrorMsg).To(ContainSubstring("invalid ticket"))
			Expect(anonymous.Credential().BkTicket).To(Equal("invalid-ticket"))
		})

		It("票据有效时写入用户与兼容元数据", func() {
			authBackend.EXPECT().GetUserCredential(request).Return("valid-ticket")
			authBackend.EXPECT().GetUserInfo(mock.Anything, "valid-ticket").
				Return(&backends.UserInfo{ID: "blueking"}, nil)

			result := authenticate(
				context.Background(),
				request,
				authBackend,
				BackendBkTicket,
				usertoken.NewMockTokenClient(GinkgoT()),
				false,
			)
			ctx := injectResult(context.Background(), result)
			user, ok := result.RequestUser.(User)
			Expect(ok).To(BeTrue())

			By("Check the authenticated user in the result")
			Expect(result.RequestUser.IsAuthenticated()).To(BeTrue())
			Expect(user.Credential().BkTicket).To(Equal("valid-ticket"))
			user, err := GetUser(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(user).To(Equal(User{ID: "blueking", Cred: UserCredential{BkTicket: "valid-ticket"}}))

			By("Get result back from the context")
			stored, ok := GetResult(ctx)
			Expect(ok).To(BeTrue())
			storedUser, ok := stored.RequestUser.(User)
			Expect(ok).To(BeTrue())
			Expect(storedUser.ID).To(Equal("blueking"))
		})
	})

	It("使用 Bearer access token 完成认证", func() {
		request.Header.Set("Authorization", "Bearer access-token")
		authBackend.EXPECT().GetUserCredential(request).Return("")
		tokenClient := usertoken.NewMockTokenClient(GinkgoT())
		tokenClient.EXPECT().GetUserInfo(mock.Anything, "access-token").Run(func(ctx context.Context, token string) {
			Expect(token).To(Equal("access-token"))
		}).Return("blueking", nil)

		result := authenticate(context.Background(), request, authBackend, BackendBkTicket, tokenClient, false)
		user, ok := result.RequestUser.(User)
		Expect(ok).To(BeTrue())

		Expect(result.RequestUser.IsAuthenticated()).To(BeTrue())
		Expect(user.Credential().AccessToken).To(Equal("access-token"))
	})

	It("按中间件模式决定是否中止未认证请求", func() {
		authBackend.EXPECT().GetUserCredential(mock.Anything).Return("").Twice()

		optionalRecorder := serveWithMiddleware(
			middleware(authBackend, BackendBkTicket, usertoken.NewMockTokenClient(GinkgoT()), false, false),
		)
		requiredRecorder := serveWithMiddleware(
			middleware(authBackend, BackendBkTicket, usertoken.NewMockTokenClient(GinkgoT()), false, true),
		)

		Expect(optionalRecorder.Code).To(Equal(http.StatusNoContent))
		Expect(requiredRecorder.Code).To(Equal(http.StatusUnauthorized))
	})

	Context("Set user through request headers", func() {
		BeforeEach(func() {
			request.Header.Set(BKAuthKey, `{"userId":"api-test-user"}`)
			request.Header.Set(BKCredentialKey, `{"bkTicket":"xxx"}`)
		})

		It("uses the user from headers when the switch is enabled", func() {
			result := authenticate(
				context.Background(),
				request,
				authBackend,
				BackendBkTicket,
				usertoken.NewMockTokenClient(GinkgoT()),
				true,
			)

			user, ok := result.RequestUser.(User)
			Expect(ok).To(BeTrue())
			Expect(user).To(Equal(User{ID: "api-test-user", Cred: UserCredential{BkTicket: "xxx"}}))
		})

		It("ignores the headers when the switch is disabled", func() {
			authBackend.EXPECT().GetUserCredential(request).Return("")

			result := authenticate(
				context.Background(),
				request,
				authBackend,
				BackendBkTicket,
				usertoken.NewMockTokenClient(GinkgoT()),
				false,
			)

			Expect(result.RequestUser.IsAuthenticated()).To(BeFalse())
		})

		It("uses normal authentication when the header identity is invalid", func() {
			request.Header.Set(BKAuthKey, `{"user-id":"api-test-user"}`)
			authBackend.EXPECT().GetUserCredential(request).Return("")

			result := authenticate(
				context.Background(),
				request,
				authBackend,
				BackendBkTicket,
				usertoken.NewMockTokenClient(GinkgoT()),
				true,
			)

			Expect(result.RequestUser.IsAuthenticated()).To(BeFalse())
		})

		It("preserves the original header parsing error", func() {
			request.Header.Set(BKAuthKey, "{")

			_, err := getUserFromHeaders(request)

			var syntaxError *json.SyntaxError
			Expect(errors.As(err, &syntaxError)).To(BeTrue())
		})
	})

	DescribeTable("选择用户认证后端",
		func(cfg Config, expectedType string) {
			_, backendType := getBackend(cfg)
			Expect(backendType).To(Equal(expectedType))
		},
		Entry("bk_ticket", Config{BackendType: BackendBkTicket}, BackendBkTicket),
		Entry("bk_token", Config{BackendType: BackendBkToken}, BackendBkToken),
		Entry("默认使用 bk_token", Config{}, BackendBkToken),
	)
})

func serveWithMiddleware(middlewares ...gin.HandlerFunc) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router := gin.New()
	router.Use(middlewares...)
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	return recorder
}
