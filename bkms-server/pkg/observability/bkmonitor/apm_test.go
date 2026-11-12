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

package bkmonitor_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/appruntime"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor"
)

var _ = Describe("GetApmServiceName", func() {
	// ==================== GetApmServiceName 入口分发 ====================

	Describe("appType dispatch", func() {
		It("should return error for an unknown app type", func() {
			_, err := bkmonitor.GetApmServiceName(&app.Application{Type: "unknown"}, "", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("apmServiceName: invalid_stack"))
		})

		It("should return error for empty app type", func() {
			_, err := bkmonitor.GetApmServiceName(&app.Application{}, "", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("apmServiceName: invalid_stack"))
		})

		It("should dispatch to trpc handler for trpc app type", func() {
			content := `
server:
  app: myapp
  server: myserver
plugins:
  telemetry:
    galileo:
      ocp_addr: "http://ocp.example.com"
`
			serviceName, err := bkmonitor.GetApmServiceName(
				&app.Application{Type: appruntime.AppTypeTRPC},
				content,
				nil,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(serviceName).To(Equal("myapp.myserver"))
		})

		It("should dispatch to taf handler for taf app type", func() {
			content := `
<taf>
	<application>
		<server>
			app=MyApp
			server=MyServer
		</server>
	</application>
</taf>`
			serviceName, err := bkmonitor.GetApmServiceName(
				&app.Application{Type: appruntime.AppTypeTAF},
				content,
				nil,
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(serviceName).To(Equal("MyApp.MyServer"))
		})
	})
})
