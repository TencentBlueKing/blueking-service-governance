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

// 对应 cmd/auth/
var _ = Describe("Auth", Ordered, Label("smoke"), func() {
	BeforeAll(func() {
		framework.GenerateConfigFile(envCfg, true)
	})

	Context("Login", func() {
		// 使用有效 AccessToken 登录
		It("login with valid AccessToken", func() {
			cli.Run("login", "--access-token", envCfg.Token).
				ExpectSuccess().ExpectOutputContains("Success")
		})

		// 使用无效 Token 登录退出码为非零
		It("login with invalid token exits with non-zero code", func() {
			cli.Run("login", "--access-token", "invalid-token-12345").
				ExpectFailure()
		})

		// login 同时使用 --access-token 和 --bk-ticket 退出码为非零
		It("login with both --access-token and --bk-ticket exits with non-zero code", func() {
			cli.Run("login", "--access-token", "some-token", "--bk-ticket").
				ExpectFailure().ExpectOutputContains("cannot use both")
		})
	})

	Context("Logout", func() {
		// logout 退出码为 0 且输出包含 success
		It("logout exits with code 0 and output contains success", func() {
			// 先确保已登录
			cli.Run("login", "--access-token", envCfg.Token)

			cli.Run("logout").ExpectSuccess().ExpectOutputContains("success")
		})
	})

	// 恢复登录状态，供后续测试使用
	AfterAll(func() {
		framework.GenerateConfigFile(envCfg, true)
	})
})
