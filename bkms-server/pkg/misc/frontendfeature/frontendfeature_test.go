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

package frontendfeature_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/frontendfeature"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/frontendfeature/handler"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/frontendfeature/serializer"
)

var _ = Describe("GetFrontendFeatures", func() {
	getFeatures := func(cfg config.Config) *slz.FrontendFeaturesOutput {
		origin := config.G
		config.G = &cfg
		DeferCleanup(func() { config.G = origin })

		r := gin.New()
		frontendfeature.Register(r.Group(""), handler.New())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/frontend-features", nil))

		Expect(w.Code).To(Equal(http.StatusOK))
		var resp slz.FrontendFeaturesResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		return resp.Data
	}

	It("should enable workspace access points when the config allows them", func() {
		features := getFeatures(config.Config{Polaris: config.PolarisConfig{AllowWorkspaceAccessPoints: true}})
		Expect(features.PolarisWorkspaceAccessPoint).To(BeTrue())
	})

	It("should disable workspace access points when the config forbids them", func() {
		features := getFeatures(config.Config{Polaris: config.PolarisConfig{AllowWorkspaceAccessPoints: false}})
		Expect(features.PolarisWorkspaceAccessPoint).To(BeFalse())
	})
})
