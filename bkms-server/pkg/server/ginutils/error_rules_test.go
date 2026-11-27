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
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp"
	bscpapierrs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bscp/apierrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils"
)

var _ = Describe("GlobalErrorRules", func() {
	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
	})

	request := func(abort func(c *gin.Context)) (int, bkerrs.GinErrorOutput) {
		router := gin.New()
		router.Use(bkerrs.ErrorHandler(ginutils.GlobalErrorRules...))
		router.GET("/ping", abort)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
		var out bkerrs.GinErrorOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &out)).To(Succeed())
		return rec.Code, out
	}

	It("should map bscp no permission only with the explicit rule", func() {
		noPermission := errors.Wrap(bscp.ErrNoPermission, "call bscp api")

		status, _ := request(func(c *gin.Context) {
			bkerrs.AbortInternal(c, errors.Wrap(noPermission, "render config content"))
		})
		Expect(status).To(Equal(http.StatusInternalServerError))

		status, out := request(func(c *gin.Context) {
			bkerrs.AbortWithErr(c, errors.Wrap(noPermission, "list bscp user bizs"), bscpapierrs.NoPermission)
		})
		Expect(status).To(Equal(http.StatusForbidden))
		Expect(out.Error.Code).To(Equal(bkerrs.ErrCodeNoPermission))
		Expect(out.Error.Message).To(Equal("无权访问 BSCP 资源，请先申请对应业务的权限"))
		Expect(out.Error.Details[0]["code"]).To(Equal(string(bkerrs.ErrDetailCodeBSCPNoPermission)))
	})

	It("should map bscp disabled even if the handler wrapped it as invalid argument", func() {
		status, out := request(func(c *gin.Context) {
			bkerrs.AbortInvalidArg(c, bscp.ErrDisabled, "validate BSCP config")
		})

		Expect(status).To(Equal(http.StatusNotImplemented))
		Expect(out.Error.Message).To(Equal("当前环境未启用 BSCP 配置服务"))
		Expect(out.Error.Details[0]["code"]).To(Equal(string(bkerrs.ErrDetailCodeBSCPDisabled)))
	})

	It("should leave other errors to the handler", func() {
		status, out := request(func(c *gin.Context) {
			bkerrs.AbortInternal(c, errors.Wrap(errors.New("connection refused"), "list bscp user bizs"))
		})

		Expect(status).To(Equal(http.StatusInternalServerError))
		Expect(out.Error.Code).To(Equal(bkerrs.ErrCodeInternalServerError))
	})
})
