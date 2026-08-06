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

package appcfg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseTrpcServiceNames", func() {
	It("should return service names from server service configs", func() {
		content := `
server:
  app: demo
  server: demo-server
  service:
    - name: trpc.demo.demo-server.ApiServer
      ip: 0.0.0.0
      port: 8080
      network: tcp
      protocol: trpc
    - name: trpc.demo.demo-server.Worker
      timeout: 60000
`

		names, err := parseTrpcServiceNames(content)

		Expect(err).NotTo(HaveOccurred())
		Expect(names).To(Equal([]string{
			"trpc.demo.demo-server.ApiServer",
			"trpc.demo.demo-server.Worker",
		}))
	})

	It("should ignore unused fields and empty service names", func() {
		content := `
global:
  namespace: Development
server:
  service:
    - name: trpc.bkms.valid.Service
      ip: 127.0.0.1
      port: 8000
      timeout: 60000
      extra:
        nested: value
    - ip: 127.0.0.1
      port: 8001
plugins:
  registry:
    etcd:
      address: 127.0.0.1:2379
`

		names, err := parseTrpcServiceNames(content)

		Expect(err).NotTo(HaveOccurred())
		Expect(names).To(Equal([]string{"trpc.bkms.valid.Service"}))
	})

	It("should return empty service names when server service is missing", func() {
		content := `
server:
  app: bkms
  server: bkmsserver
client:
  service:
    - name: trpc.bkms.registry.etcd
`

		names, err := parseTrpcServiceNames(content)

		Expect(err).NotTo(HaveOccurred())
		Expect(names).To(BeEmpty())
	})

	It("should return error when yaml is invalid", func() {
		_, err := parseTrpcServiceNames("server:\n  service:\n    - name: [")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unmarshal tRPC config YAML"))
	})
})
