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

package bkerrs

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pkgerrors "github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/i18n"
)

type testUnauthenticatedError struct{}

func (testUnauthenticatedError) Error() string {
	return "auth user not found in request headers"
}

func (testUnauthenticatedError) IsUnauthenticated() bool {
	return true
}

// testStoreError 模拟 store 声明了类别的错误
type testStoreError struct {
	notFound bool
	domain   error
}

func (e *testStoreError) Error() string         { return "document not found: collection widgets" }
func (e *testStoreError) IsNotFound() bool      { return e.notFound }
func (e *testStoreError) IsAlreadyExists() bool { return !e.notFound }
func (e *testStoreError) DomainError() error    { return e.domain }

var _ = Describe("Gin error handling", func() {
	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
	})

	request := func(err error, lang string) GinErrorOutput {
		router := gin.New()
		router.Use(i18n.Middleware(), ErrorHandler())
		router.GET("/ping", func(c *gin.Context) {
			AbortWithErr(c, err)
		})

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		if lang != "" {
			req.AddCookie(&http.Cookie{Name: i18n.LangCookieName, Value: lang})
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		var out GinErrorOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
		Expect(out.Error.Code.AsHTTPStatusCode()).To(Equal(rec.Code))
		return out
	}

	Context("when the outermost bk error message has no translation", func() {
		var err error

		BeforeEach(func() {
			inner := Errorf(ErrCodeInvalidRequest, "environment has %d deployed apps, cannot delete", 2)
			err = Wrap(inner, ErrCodeInvalidRequest, "untranslated outer")
		})

		It("should fall back to the default message of the error code instead of translating inner bk errors", func() {
			out := request(err, "zh")

			Expect(out.Error.Message).To(Equal("请求不符合业务规则"))
			Expect(out.Error.Details).To(HaveLen(1))
			Expect(out.Error.Details[0]["code"]).To(Equal(string(ErrCodeInvalidRequest)))
			Expect(out.Error.Details[0]["message"]).To(Equal(
				"untranslated outer: environment has 2 deployed apps, cannot delete",
			))
		})

		It("should keep existing details in front of the full error message", func() {
			withDetail := New(ErrCodeNoPermission, "untranslated outer").SetDetails(NewDetail("BIZ_CODE", "biz"))
			out := request(withDetail, "zh")

			Expect(out.Error.Message).To(Equal("无权限执行该操作"))
			Expect(out.Error.Details).To(HaveLen(2))
			Expect(out.Error.Details[0]["code"]).To(Equal("BIZ_CODE"))
			Expect(out.Error.Details[1]["message"]).To(Equal("untranslated outer"))
		})

		It("should keep the full error message and original details in the source language", func() {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			c.Request = req.WithContext(i18n.WithLang(req.Context(), i18n.LangEN))
			WriteError(c, New(ErrCodeInvalidRequest, "bad request"))

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "INVALID_REQUEST",
					"message": "bad request",
					"system": "bkms",
					"module": "bkms-server",
					"details": null
				}
			}`))
		})

		It("should keep the full error message when the error code has no default message", func() {
			out := request(New(ErrCode("CUSTOM_CODE"), "custom failure"), "zh")

			Expect(out.Error.Message).To(Equal("custom failure"))
			Expect(out.Error.Details).To(BeEmpty())
		})
	})

	Context("when the outermost bk error message has a translation", func() {
		var err error

		BeforeEach(func() {
			inner := Errorf(ErrCodeInvalidRequest, "environment has %d deployed apps, cannot delete", 2)
			err = pkgerrors.Wrap(inner, "delete environment")
		})

		It("should translate the message with its args in the request language", func() {
			Expect(request(err, "zh").Error.Message).To(Equal("环境已部署 2 个应用，无法删除"))
			Expect(request(err, "en").Error.Message).To(Equal("environment has 2 deployed apps, cannot delete"))
		})

		It("should append the full error message to details", func() {
			out := request(err, "")
			Expect(out.Error.Details).To(HaveLen(1))
			Expect(out.Error.Details[0]["code"]).To(Equal(string(ErrCodeInvalidRequest)))
			Expect(out.Error.Details[0]["message"]).To(Equal(err.Error()))
		})

		It("should keep existing details in front", func() {
			withDetail := Wrap(
				err,
				ErrCodeInvalidRequest,
				"resource not found",
			).SetDetails(NewDetail("BIZ_CODE", "biz"))
			out := request(withDetail, "")

			Expect(out.Error.Message).To(Equal("资源不存在"))
			Expect(out.Error.Details).To(HaveLen(2))
			Expect(out.Error.Details[0]["code"]).To(Equal("BIZ_CODE"))
			Expect(out.Error.Details[1]["code"]).To(Equal(string(ErrCodeInvalidRequest)))
		})
	})

	Context("when handler reports an unauthenticated marker error", func() {
		It("should return a translated unauthenticated error", func() {
			out := request(testUnauthenticatedError{}, "")

			Expect(out.Error.Code).To(Equal(ErrCodeUnauthenticated))
			Expect(out.Error.Message).To(Equal("获取登录用户信息失败，请重新登录"))
			Expect(out.Error.Details[0]["message"]).To(ContainSubstring("auth user not found in request headers"))
		})
	})

	Context("when the error declares a store category", func() {
		It("should use the domain error text when present", func() {
			cause := &testStoreError{notFound: true, domain: errors.New("resource not found")}
			out := request(pkgerrors.Wrap(cause, "get widget"), "")

			Expect(out.Error.Code).To(Equal(ErrCodeNotFound))
			Expect(out.Error.Message).To(Equal("资源不存在"))
			Expect(out.Error.Details[0]["message"]).To(ContainSubstring("get widget"))
		})

		It("should fall back to the generic text without a domain error", func() {
			out := request(&testStoreError{notFound: false}, "")

			Expect(out.Error.Code).To(Equal(ErrCodeAlreadyExists))
			Expect(out.Error.Message).To(Equal("资源已存在"))
		})

		It("should let an outer bk error take precedence", func() {
			cause := &testStoreError{notFound: true}
			out := request(Wrap(cause, ErrCodeInternalServerError, "lookup dependency"), "")

			Expect(out.Error.Code).To(Equal(ErrCodeInternalServerError))
		})
	})

	Context("when handler reports a generic error", func() {
		It("should return a generic internal error message and keep the chain in details", func() {
			out := request(errors.New("boom"), "")

			Expect(out.Error.Code).To(Equal(ErrCodeInternalError))
			Expect(out.Error.Message).To(Equal("服务内部错误，请稍后重试或联系管理员"))
			Expect(out.Error.Details).To(HaveLen(1))
			Expect(out.Error.Details[0]["code"]).To(Equal(string(ErrCodeInternalError)))
			Expect(out.Error.Details[0]["message"]).To(Equal("boom"))
		})
	})
})
