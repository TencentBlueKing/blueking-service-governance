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

package cliusage_test

import (
	"context"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/cliusage"
)

var _ = Describe("HandlerNames", func() {
	const (
		basePath   = "/bkms/v1/bkms-server"
		modulePath = "github.com/TencentBlueKing/blueking-service-governance/bkms-server"
		deployList = modulePath + "/pkg/deploy/handler.(*Handler).List-fm"
		envList    = modulePath + "/pkg/core/env/handler.(*Handler).List-fm"
		getApp     = modulePath + "/pkg/core/app/handler.(*Handler).GetApp-fm"
		closure    = modulePath + "/pkg/core/app.Register.func1"
		nested     = modulePath + "/pkg/core/app.Register.func1.2"
		healthz    = modulePath + "/pkg/server/basic/handler.(*Handler).List-fm"
	)

	var names *cliusage.HandlerNames

	BeforeEach(func() {
		names = cliusage.NewHandlerNames()
		names.Load(context.Background(), gin.RoutesInfo{
			{Method: "GET", Path: basePath + "/deploys", Handler: deployList},
			{Method: "GET", Path: basePath + "/envs", Handler: envList},
			{Method: "GET", Path: basePath + "/apps/:appID", Handler: getApp},
			{Method: "GET", Path: basePath + "/v2/apps/:appID", Handler: getApp},
			{Method: "GET", Path: basePath + "/closure", Handler: closure},
			{Method: "GET", Path: basePath + "/nested", Handler: nested},
			// basePath 以外，不参与重名判断
			{Method: "GET", Path: "/healthz", Handler: healthz},
		}, basePath)
	})

	It("uses the method name when it is unique", func() {
		Expect(names.Lookup(getApp, "GET", basePath+"/apps/:appID")).To(Equal("GetApp"))
		Expect(names.Lookup(getApp, "GET", basePath+"/v2/apps/:appID")).To(Equal("GetApp"))
	})

	It("uses qualified names for duplicate method names", func() {
		Expect(names.Lookup(deployList, "GET", basePath+"/deploys")).To(Equal("deploy/handler.(*Handler).List"))
		Expect(names.Lookup(envList, "GET", basePath+"/envs")).To(Equal("core/env/handler.(*Handler).List"))
	})

	It("falls back to method and route for closures", func() {
		Expect(names.Lookup(closure, "GET", basePath+"/closure")).To(Equal("GET " + basePath + "/closure"))
		Expect(names.Lookup(nested, "GET", basePath+"/nested")).To(Equal("GET " + basePath + "/nested"))
	})

	It("falls back to method and route before loading", func() {
		empty := cliusage.NewHandlerNames()
		Expect(empty.Lookup(getApp, "GET", basePath+"/apps/:appID")).To(Equal("GET " + basePath + "/apps/:appID"))
	})
})
