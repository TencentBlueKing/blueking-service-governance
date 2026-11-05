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
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

var _ = Describe("RequireBSCP", func() {
	var (
		originalCfg *config.Config
		router      *gin.Engine
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		originalCfg = config.G
		config.G = &config.Config{}

		router = gin.New()
		router.Use(bkerrs.ErrorHandler())
		router.GET("/bscp", ginutils.RequireBSCP(), func(c *gin.Context) {
			ginutils.OK(c, map[string]string{"status": "ok"})
		})
	})

	AfterEach(func() {
		config.G = originalCfg
	})

	It("should pass through when bscp is not disabled", func() {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/bscp", nil))

		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(MatchJSON(`{"status": "ok"}`))
	})

	It("should reject with BSCP_DISABLED when bscp is disabled", func() {
		config.G.FeatureIntegrations.DisableBSCP = true

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/bscp", nil))

		Expect(rec.Code).To(Equal(http.StatusBadRequest))
		var out bkerrs.GinErrorOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
		Expect(out.Error.Code).To(Equal(bkerrs.ErrCodeInvalidRequest))
		Expect(out.Error.Details).To(HaveLen(1))
		Expect(out.Error.Details[0]["code"]).To(Equal(string(bkerrs.ErrDetailCodeBSCPDisabled)))
	})
})
