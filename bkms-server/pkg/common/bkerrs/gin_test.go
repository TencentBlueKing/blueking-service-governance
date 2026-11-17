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

var (
	// 没有对应翻译的详情码，用于验证回退到错误码通用文案
	errGinTestNameDuplicated = Define(ErrCodeAlreadyExists, "GIN_TEST_NAME_DUPLICATED", "resource name duplicated")
	// 复用已有翻译的详情码，用于验证按详情码翻译（本测试二进制不包含环境包，不会重复定义）
	errGinTestEnvHasApps = Define(
		ErrCodeInvalidArgument, "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER", "environment has deployed apps",
	)
)

// testCategorizedError 模拟不依赖 bkerrs 的通用组件（如 store）通过标记方法声明错误类别
type testCategorizedError struct {
	msg           string
	notFound      bool
	alreadyExists bool
}

func (e testCategorizedError) Error() string         { return e.msg }
func (e testCategorizedError) IsNotFound() bool      { return e.notFound }
func (e testCategorizedError) IsAlreadyExists() bool { return e.alreadyExists }

var _ = Describe("Gin error handling", func() {
	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
	})

	makeLocalizedRequest := func(err error, acceptLanguage string) *httptest.ResponseRecorder {
		router := gin.New()
		router.Use(i18n.Middleware(), ErrorHandler())
		router.GET("/ping", func(c *gin.Context) {
			AbortWithErr(c, err)
		})

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		if acceptLanguage != "" {
			req.Header.Set("Accept-Language", acceptLanguage)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	makeRequest := func(err error) *httptest.ResponseRecorder {
		return makeLocalizedRequest(err, "")
	}

	Context("when handler reports a bk error", func() {
		It("should return bk error response", func() {
			rec := makeRequest(New(ErrCodeInvalidRequest, "bad request"))

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
	})

	Context("when handler reports a defined error wrapped with context only", func() {
		err := pkgerrors.Wrap(errGinTestEnvHasApps.With("appCount", "2"), "update env cluster")
		expectedDetails := `[{
			"code": "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER",
			"message": "update env cluster: environment has deployed apps (appCount=2)",
			"system": "bkms",
			"module": "",
			"extras": {"appCount": "2"}
		}]`

		It("should localize the message by detail code in the default language", func() {
			rec := makeRequest(err)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "INVALID_ARGUMENT",
					"message": "环境已部署 2 个应用，无法修改集群配置",
					"system": "bkms",
					"module": "bkms-server",
					"details": ` + expectedDetails + `
				}
			}`))
		})

		It("should localize the message in the negotiated language", func() {
			rec := makeLocalizedRequest(err, "en-US,en;q=0.9")

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "INVALID_ARGUMENT",
					"message": "Environment has 2 deployed app(s) and cannot update cluster configuration",
					"system": "bkms",
					"module": "bkms-server",
					"details": ` + expectedDetails + `
				}
			}`))
		})

		It("should fall back to the code message when a template param is missing", func() {
			rec := makeRequest(errGinTestEnvHasApps)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(rec.Body.String()).To(ContainSubstring(`"message":"参数错误"`))
		})
	})

	Context("when handler reports a defined error without translation", func() {
		It("should fall back to the code message", func() {
			rec := makeRequest(pkgerrors.Wrap(errGinTestNameDuplicated.With("name", "a"), "create resource"))

			Expect(rec.Code).To(Equal(http.StatusConflict))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "ALREADY_EXISTS",
					"message": "资源已存在",
					"system": "bkms",
					"module": "bkms-server",
					"details": [{
						"code": "GIN_TEST_NAME_DUPLICATED",
						"message": "create resource: resource name duplicated (name=a)",
						"system": "bkms",
						"module": "",
						"extras": {"name": "a"}
					}]
				}
			}`))
		})
	})

	Context("when handler reports an error declaring its category", func() {
		notFound := testCategorizedError{msg: "document not found", notFound: true}

		It("should return not found for an error declaring IsNotFound", func() {
			rec := makeRequest(pkgerrors.Wrap(notFound, "get item"))

			Expect(rec.Code).To(Equal(http.StatusNotFound))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "NOT_FOUND",
					"message": "资源不存在",
					"system": "bkms",
					"module": "bkms-server",
					"details": [{
						"code": "RESOURCE_NOT_FOUND",
						"message": "get item: document not found",
						"system": "bkms",
						"module": "",
						"extras": null
					}]
				}
			}`))
		})

		It("should return already exists for an error declaring IsAlreadyExists", func() {
			rec := makeRequest(testCategorizedError{msg: "duplicate key", alreadyExists: true})

			Expect(rec.Code).To(Equal(http.StatusConflict))
			Expect(rec.Body.String()).To(ContainSubstring(`"message":"资源已存在"`))
			Expect(rec.Body.String()).To(ContainSubstring(`"code":"RESOURCE_ALREADY_EXISTS"`))
		})

		It("should treat the error as internal when the marker returns false", func() {
			rec := makeRequest(testCategorizedError{msg: "boom"})

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		})

		It("should prefer a classified error on the chain over the declared category", func() {
			err := errGinTestEnvHasApps.With("appCount", "2").WithCause(notFound)
			rec := makeRequest(err)

			Expect(rec.Code).To(Equal(http.StatusBadRequest))
			Expect(rec.Body.String()).To(ContainSubstring(`"message":"环境已部署 2 个应用，无法修改集群配置"`))
		})
	})

	Context("when a legacy bkerrs.Wrap covers a defined error", func() {
		It("should still use the outer code until the call site is migrated", func() {
			err := Wrap(errGinTestNameDuplicated, ErrCodeInternalServerError, "create resource")
			rec := makeRequest(err)

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("when handler reports an error caused by bscp being disabled", func() {
		It("should restore the BSCP_DISABLED detail even if wrapped by another bk error", func() {
			cause := pkgerrors.Wrap(ErrBSCPDisabled, "initializing bscp client")
			rec := makeRequest(Wrap(cause, ErrCodeInvalidArgument, "validating bscp config"))

			Expect(rec.Code).To(Equal(http.StatusNotImplemented))
			var out GinErrorOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
			Expect(out.Error.Code).To(Equal(ErrCodeNotImplemented))
			Expect(out.Error.Message).To(Equal("当前环境未启用 BSCP 配置服务"))
			Expect(out.Error.Details).NotTo(BeEmpty())
			Expect(out.Error.Details).To(HaveLen(1))
			Expect(out.Error.Details[0]["code"]).To(Equal(string(ErrDetailCodeBSCPDisabled)))
		})

		It("should return BSCP_DISABLED for the bare sentinel error", func() {
			rec := makeRequest(ErrBSCPDisabled)

			Expect(rec.Code).To(Equal(http.StatusNotImplemented))
			var out GinErrorOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
			Expect(out.Error.Message).To(Equal("当前环境未启用 BSCP 配置服务"))
			Expect(out.Error.Details).NotTo(BeEmpty())
			Expect(out.Error.Details[0]["code"]).To(Equal(string(ErrDetailCodeBSCPDisabled)))
		})
	})

	Context("when handler reports a generic error", func() {
		It("should hide the error chain from the message and keep it in details", func() {
			rec := makeLocalizedRequest(errors.New("boom"), "en")

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "INTERNAL_ERROR",
					"message": "Internal server error",
					"system": "bkms",
					"module": "bkms-server",
					"details": [{
						"code": "INTERNAL_ERROR",
						"message": "boom",
						"system": "bkms",
						"module": "",
						"extras": null
					}]
				}
			}`))
		})
	})

	Context("when handler reports a legacy 5xx error with details", func() {
		It("should keep existing details and append the error chain", func() {
			err := New(ErrCodeInternalServerError, "query db").
				SetDetails(NewDetail("LEGACY_DETAIL", "legacy detail"))
			rec := makeRequest(err)

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "INTERNAL_SERVER_ERROR",
					"message": "服务内部错误",
					"system": "bkms",
					"module": "bkms-server",
					"details": [{
						"code": "LEGACY_DETAIL",
						"message": "legacy detail",
						"system": "",
						"module": "",
						"extras": null
					}, {
						"code": "INTERNAL_SERVER_ERROR",
						"message": "query db",
						"system": "bkms",
						"module": "",
						"extras": null
					}]
				}
			}`))
		})
	})
})

var _ = Describe("Error code messages", func() {
	It("should have a localized message for every error code", func() {
		codes := []ErrCode{
			ErrCodeInvalidArgument, ErrCodeInvalidRequest, ErrCodeUnauthenticated, ErrCodeIAMNoPermission,
			ErrCodeNoPermission, ErrCodeNotFound, ErrCodeAlreadyExists, ErrCodeAborted, ErrCodeRateLimitExceeded,
			ErrCodeResourceExhausted, ErrCodeInternalError, ErrCodeInternalServerError, ErrCodeNotImplemented,
		}
		for _, code := range codes {
			Expect(i18n.HasKey(string(code))).To(BeTrue(), "missing message for error code %s", code)
		}
	})
})
