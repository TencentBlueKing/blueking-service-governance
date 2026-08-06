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

// 对应 cmd/config/
var _ = Describe("Config", Ordered, func() {
	BeforeAll(func() {
		framework.GenerateConfigFile(envCfg, true)
		cli.RunWithStdin(envCfg.Token+"\n", "login", "--access-token")
	})

	Context("View", func() {
		// config view 退出码为 0 且输出包含 username 和 bkmsBaseUrl
		It("config view exits with code 0 and output contains username and bkmsBaseUrl", func() {
			result := cli.Run("config", "view")
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring("username"))
			Expect(result.CombinedOutput()).To(ContainSubstring("bkmsBaseUrl"))
		})

		// 登出后 config view 凭证已被清除
		It("config view shows cleared credentials after logout", func() {
			// 先登出
			cli.Run("logout")

			result := cli.Run("config", "view")
			Expect(result.ExitCode).To(Equal(0))
			// 登出后 accessToken 应为空（不应包含有效的 token 值）
			Expect(result.CombinedOutput()).NotTo(MatchRegexp(`accessToken: [A-Za-z0-9]+`))

			// 恢复登录状态
			cli.RunWithStdin(envCfg.Token+"\n", "login", "--access-token")
		})
	})
})
