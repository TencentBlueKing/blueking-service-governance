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
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testUnauthenticatedError struct{}

func (testUnauthenticatedError) Error() string {
	return "auth user not found in request headers"
}

func (testUnauthenticatedError) IsUnauthenticated() bool {
	return true
}

var _ = Describe("Gin error handling", func() {
	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
	})

	makeRequest := func(err error) *httptest.ResponseRecorder {
		router := gin.New()
		router.Use(ErrorHandler())
		router.GET("/ping", func(c *gin.Context) {
			AbortWithErr(c, err)
		})

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
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

	Context("when handler reports an unauthenticated marker error", func() {
		It("should return unauthenticated error response", func() {
			rec := makeRequest(testUnauthenticatedError{})

			Expect(rec.Code).To(Equal(http.StatusUnauthorized))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "UNAUTHENTICATED",
					"message": "get auth user from request failed: auth user not found in request headers",
					"system": "bkms",
					"module": "bkms-server",
					"details": null
				}
			}`))
		})
	})

	Context("when handler reports a generic error", func() {
		It("should return internal error response", func() {
			rec := makeRequest(errors.New("boom"))

			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			Expect(rec.Body.String()).To(MatchJSON(`{
				"error": {
					"code": "INTERNAL_ERROR",
					"message": "boom",
					"system": "bkms",
					"module": "bkms-server",
					"details": null
				}
			}`))
		})
	})
})
