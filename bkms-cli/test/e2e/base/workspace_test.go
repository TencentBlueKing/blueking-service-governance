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
package base_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// 对应 cmd/workspace/
var _ = Describe("Workspace", Ordered, Label("readonly"), func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	Context("List", func() {
		// workspace list 退出码为 0
		It("workspace list exits with code 0", func() {
			cli.Run("workspace", "list").ExpectSuccess()
		})

		// 未登录执行 workspace list 退出码为非零且输出包含认证提示
		It("workspace list without login exits with non-zero code and shows auth hint", func() {
			framework.RunWithoutAuth(cli, envCfg, func() {
				cli.Run("workspace", "list").
					ExpectFailure().ExpectOutputContains("unauthorized")
			})
		})
	})

	Context("Set", func() {
		// workspace set 退出码为 0 且输出包含 successfully
		It("workspace set exits with code 0 and output contains successfully", func() {
			cli.Run("workspace", "set", envCfg.WorkspaceID).
				ExpectSuccess().ExpectOutputContains("successfully")
		})

		// workspace set 不带参数退出码为非零
		It("workspace set without args exits with non-zero code", func() {
			cli.Run("workspace", "set").ExpectFailure()
		})
	})

	Context("Unset", func() {
		// workspace unset 退出码为 0 且输出包含 successfully
		It("workspace unset exits with code 0 and output contains successfully", func() {
			// 先 set 再 unset
			cli.Run("workspace", "set", envCfg.WorkspaceID)

			cli.Run("workspace", "unset").
				ExpectSuccess().ExpectOutputContains("successfully")
		})
	})

	// 恢复 workspace 设置，供后续测试使用
	AfterAll(func() {
		if envCfg.WorkspaceID != "" {
			cli.Run("workspace", "set", envCfg.WorkspaceID)
		}
	})
})
