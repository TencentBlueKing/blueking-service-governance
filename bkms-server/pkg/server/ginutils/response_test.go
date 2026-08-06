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

package ginutils_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

type jsonResponse struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

var _ = Describe("JSON", func() {
	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
	})

	It("should render JSON response with zero values from struct tags", func() {
		router := gin.New()
		router.GET("/json", func(c *gin.Context) {
			ginutils.OK(c, jsonResponse{Name: "test"})
		})

		req := httptest.NewRequest(http.MethodGet, "/json", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(MatchJSON(`{
			"name": "test",
			"enabled": false,
			"description": ""
		}`))
	})
})
