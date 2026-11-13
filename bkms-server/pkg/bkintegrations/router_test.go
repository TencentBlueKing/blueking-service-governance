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

package bkintegrations_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

// fakeHandler 仅实现用到的方法，其余方法被调用时会因 nil 接口 panic
type fakeHandler struct {
	bkintegrations.Handler
}

func (fakeHandler) ListBSCPBizs(c *gin.Context) {
	ginutils.OK(c, map[string]string{"from": "bscp"})
}

func (fakeHandler) ListBKCCAuthorizedBusinesses(c *gin.Context) {
	ginutils.OK(c, map[string]string{"from": "bkcc"})
}

var _ = Describe("Register", func() {
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
		bkintegrations.Register(router.Group(""), fakeHandler{})
	})

	AfterEach(func() {
		config.G = originalCfg
	})

	serve := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}

	It("should serve bscp routes when bscp is enabled", func() {
		rec := serve("/bscp/bizs")
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(MatchJSON(`{"from": "bscp"}`))
	})

	It("should reject bscp routes with BSCP_DISABLED when bscp is disabled", func() {
		config.G.FeatureIntegrations.DisableBSCP = true

		rec := serve("/bscp/bizs")
		Expect(rec.Code).To(Equal(http.StatusNotImplemented))
		var out bkerrs.GinErrorOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
		Expect(out.Error.Details).To(HaveLen(1))
		Expect(out.Error.Details[0]["code"]).To(Equal(string(bkerrs.ErrDetailCodeBSCPDisabled)))
	})

	It("should not affect non-bscp routes when bscp is disabled", func() {
		config.G.FeatureIntegrations.DisableBSCP = true

		rec := serve("/bkcc/businesses/authorized")
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Body.String()).To(MatchJSON(`{"from": "bkcc"}`))
	})
})
