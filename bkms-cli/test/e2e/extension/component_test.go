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
package extension_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// 对应 cmd/app/component/
var _ = Describe("App Component", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== list ====================
	Context("List", Label("readonly"), func() {
		It("app component list exits with code 0", func() {
			cli.Run("app", "component", "list", "--app", envCfg.AppID).
				ExpectSuccess()
		})

		It("app component list supports -o json", func() {
			cli.Run("app", "component", "list", "--app", envCfg.AppID, "-o", "json").
				ExpectSuccess()
		})

		It("app component list without --app fails", func() {
			cli.Run("app", "component", "list").
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== create (error cases) ====================
	Context("Create", Label("destructive"), func() {
		It("app component create without --app fails", func() {
			cli.Run("app", "component", "create", "--ref", "some-ref").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app component create without --ref fails", func() {
			cli.Run("app", "component", "create", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== delete (error cases) ====================
	Context("Delete", Label("destructive"), func() {
		It("app component delete without --app fails", func() {
			cli.Run("app", "component", "delete", "--name", "some-name").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app component delete without --name fails", func() {
			cli.Run("app", "component", "delete", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})
	})
})
