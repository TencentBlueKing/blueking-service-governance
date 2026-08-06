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

// Package 端到端测试
package e2e_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// 对应 cmd/app/image/
var _ = Describe("App Image", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== cmd/app/image/list.go ====================
	Context("List", func() {
		// app image list 退出码为 0
		It("app image list exits with code 0", func() {
			result := cli.Run("app", "image", "list", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
		})

		// app image list 缺少 --app 退出码为非零且输出包含 required
		It("app image list without --app exits with non-zero code and output contains required", func() {
			result := cli.Run("app", "image", "list")
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring("required"))
		})
	})
})
