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

package apm

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/apm/apitracking"
)

var _ = Describe("HTTPLogMiddleware", func() {
	var (
		logBuf   *bytes.Buffer
		prevSlog *slog.Logger
	)

	BeforeEach(func() {
		prevSlog = slog.Default()
		logBuf = &bytes.Buffer{}
		slog.SetDefault(slog.New(slog.NewJSONHandler(logBuf, nil)))
	})

	AfterEach(func() {
		slog.SetDefault(prevSlog)
	})

	It("masks sensitive query and referer parameters", func() {
		gin.SetMode(gin.TestMode)
		engine := gin.New()
		engine.Use(HTTPLogMiddleware(apitracking.NewHandlerNames()))
		var gotToken string
		engine.GET("/user_token/validate", func(c *gin.Context) {
			gotToken = c.Query("access_token")
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/user_token/validate?access_token=secret&page=1", nil)
		req.Header.Set("Referer", "https://bkms.example.com/apps?bk_token=cookie-secret&tab=env")
		engine.ServeHTTP(httptest.NewRecorder(), req)

		// handler 读取到原始参数，只有日志被脱敏
		Expect(gotToken).To(Equal("secret"))
		Expect(logBuf.String()).NotTo(ContainSubstring("secret"))

		record := map[string]any{}
		Expect(json.Unmarshal(logBuf.Bytes(), &record)).To(Succeed())
		Expect(record).To(HaveKeyWithValue("request", SatisfyAll(
			HaveKeyWithValue("query", "access_token=%2A%2A%2A&page=1"),
			HaveKeyWithValue("referer", "https://bkms.example.com/apps?bk_token=%2A%2A%2A&tab=env"),
			HaveKeyWithValue("path", "/user_token/validate"),
		)))
	})
})
