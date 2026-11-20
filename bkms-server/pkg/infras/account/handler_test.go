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

package account

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
)

var _ = Describe("GetCurrentUser", func() {
	var router *gin.Engine

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		optionalAuth, err := auth.Optional(auth.Config{
			BackendType:          auth.BackendBkToken,
			LoginURL:             "http://login.example.com",
			AllowSetUserInHeader: true,
		}, nil)
		Expect(err).NotTo(HaveOccurred())

		router = gin.New()
		Register(router.Group(""), NewHandler(Config{}, nil), optionalAuth)
	})

	request := func(userInfo string) (int, map[string]any) {
		req := httptest.NewRequest(http.MethodGet, "/simple_account/info", nil)
		if userInfo != "" {
			req.Header.Set(auth.BKAuthKey, userInfo)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		body := map[string]any{}
		Expect(json.Unmarshal(rec.Body.Bytes(), &body)).To(Succeed())
		return rec.Code, body
	}

	It("returns the tenant of the authenticated user", func() {
		code, body := request(`{"userId":"alice","tenantId":"tenant-a"}`)

		Expect(code).To(Equal(http.StatusOK))
		Expect(body).To(Equal(map[string]any{"user_id": "alice", "tenant_id": "tenant-a"}))
	})

	It("returns an empty tenant when the auth upstream does not provide one", func() {
		code, body := request(`{"userId":"alice"}`)

		Expect(code).To(Equal(http.StatusOK))
		Expect(body).To(Equal(map[string]any{"user_id": "alice", "tenant_id": ""}))
	})
})
