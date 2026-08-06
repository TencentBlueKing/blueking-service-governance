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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// 对应 cmd/app/app.go 和 cmd/app/list.go
var _ = Describe("App", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== cmd/app/list.go ====================
	Context("List", func() {
		// app list 退出码为 0
		It("app list exits with code 0", func() {
			result := cli.Run("app", "list")
			Expect(result.ExitCode).To(Equal(0))
		})

		// 使用指定的 workspace 查询，而非配置文件中的默认值
		It("app list with explicit --workspace uses the specified workspace", func() {
			result := cli.Run("app", "list", "--workspace", uuid.New().String())
			Expect(result.ExitCode).To(Equal(0))
			// 指定一个不存在的 workspace，服务端返回空数据，输出为 null
			Expect(result.CombinedOutput()).To(ContainSubstring("null"))
		})

		// 未登录执行 app list 退出码为非零且输出包含认证提示
		It("app list without login exits with non-zero code and shows auth hint", func() {
			framework.RunWithoutAuth(cli, envCfg, func() {
				result := cli.Run("app", "list")
				Expect(result.ExitCode).NotTo(Equal(0))
				Expect(result.CombinedOutput()).To(
					MatchRegexp(`(?i)unauthorized|login|Unauthorized`))
			})
		})
	})
})
