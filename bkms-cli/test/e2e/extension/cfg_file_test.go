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

// 对应 cmd/app/appcfgfile/
var _ = Describe("App Config File", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== View ====================
	Context("View", Label("readonly"), func() {
		It("app app-cfg-file view exits with code 0", func() {
			cli.Run("app", "app-cfg-file", "view", "--app", envCfg.AppID).ExpectSuccess()
		})

		It("app app-cfg-file view supports -o json", func() {
			cli.Run("app", "app-cfg-file", "view", "--app", envCfg.AppID, "-o", "json").
				ExpectSuccess()
		})

		It("app app-cfg-file view without --app fails", func() {
			cli.Run("app", "app-cfg-file", "view").
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== Edit (error cases) ====================
	Context("Edit", Label("destructive"), func() {
		It("app app-cfg-file edit without --app fails", func() {
			cli.Run("app", "app-cfg-file", "edit", "-f", "/tmp/nonexistent.txt").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app app-cfg-file edit without -f or --file-content fails", func() {
			cli.Run("app", "app-cfg-file", "edit", "--app", envCfg.AppID).
				ExpectFailure()
		})

		It("app app-cfg-file edit with nonexistent file fails", func() {
			cli.Run("app", "app-cfg-file", "edit",
				"--app", envCfg.AppID,
				"-f", "/tmp/nonexistent-cfg-file-xyz.txt").
				ExpectFailure()
		})
	})

	// ==================== List Versions ====================
	Context("List Versions", Label("readonly"), func() {
		It("app app-cfg-file list-versions exits with code 0", func() {
			cli.Run("app", "app-cfg-file", "list-versions", "--app", envCfg.AppID).
				ExpectSuccess()
		})

		It("app app-cfg-file list-versions supports -o json", func() {
			cli.Run("app", "app-cfg-file", "list-versions", "--app", envCfg.AppID, "-o", "json").
				ExpectSuccess()
		})

		It("app app-cfg-file list-versions without --app fails", func() {
			cli.Run("app", "app-cfg-file", "list-versions").
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== View Version (error cases) ====================
	Context("View Version", Label("readonly"), func() {
		It("app app-cfg-file view-version without --app fails", func() {
			cli.Run("app", "app-cfg-file", "view-version", "--version", "1").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app app-cfg-file view-version without version ref fails", func() {
			cli.Run("app", "app-cfg-file", "view-version", "--app", envCfg.AppID).
				ExpectFailure()
		})
	})

	// ==================== Rollback Version (error cases) ====================
	Context("Rollback Version", Label("destructive"), func() {
		It("app app-cfg-file rollback-version without --app fails", func() {
			cli.Run("app", "app-cfg-file", "rollback-version", "--version", "1").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app app-cfg-file rollback-version without version ref fails", func() {
			cli.Run("app", "app-cfg-file", "rollback-version", "--app", envCfg.AppID).
				ExpectFailure()
		})
	})

	// ==================== Delete Version (error cases) ====================
	Context("Delete Version", Label("destructive"), func() {
		It("app app-cfg-file delete-version without --app fails", func() {
			cli.Run("app", "app-cfg-file", "delete-version", "--version", "1").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app app-cfg-file delete-version without version ref fails", func() {
			cli.Run("app", "app-cfg-file", "delete-version", "--app", envCfg.AppID).
				ExpectFailure()
		})
	})
})
