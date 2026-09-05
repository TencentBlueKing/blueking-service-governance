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

// 对应 cmd/app/polaris/
var _ = Describe("App Polaris", Ordered, Label("destructive"), func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== List ====================
	Context("List", Label("readonly"), func() {
		It("app polaris list exits with code 0", func() {
			cli.Run("app", "polaris", "list", "--app", envCfg.AppID).ExpectSuccess()
		})

		It("app polaris list without --app fails", func() {
			cli.Run("app", "polaris", "list").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris list supports -o json", func() {
			cli.Run("app", "polaris", "list", "--app", envCfg.AppID, "-o", "json").
				ExpectSuccess()
		})
	})

	// ==================== Create (error cases only - creating real polaris config requires external polaris service)
	// ====================
	Context("Create", func() {
		It("app polaris create without --app fails", func() {
			cli.Run("app", "polaris", "create", "-f", "/tmp/nonexistent.yaml").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris create without -f fails", func() {
			cli.Run("app", "polaris", "create", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris create with nonexistent file fails", func() {
			cli.Run("app", "polaris", "create",
				"--app", envCfg.AppID,
				"-f", "/tmp/nonexistent-polaris-spec-xyz.yaml").
				ExpectFailure().ExpectOutputContains("not found")
		})
	})

	// ==================== Update (error cases) ====================
	Context("Update", func() {
		It("app polaris update without --app fails", func() {
			cli.Run("app", "polaris", "update",
				"--name", "nonexistent",
				"-f", "/tmp/nonexistent.yaml").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris update without --name fails", func() {
			cli.Run("app", "polaris", "update",
				"--app", envCfg.AppID,
				"-f", "/tmp/nonexistent.yaml").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris update without -f fails", func() {
			cli.Run("app", "polaris", "update",
				"--app", envCfg.AppID,
				"--name", "nonexistent").
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== Delete (error cases) ====================
	Context("Delete", func() {
		It("app polaris delete without --app fails", func() {
			cli.Run("app", "polaris", "delete", "--name", "nonexistent").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris delete without --name fails", func() {
			cli.Run("app", "polaris", "delete", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== Weight (error cases) ====================
	Context("Weight", func() {
		It("app polaris weight without --app fails", func() {
			cli.Run("app", "polaris", "weight",
				"--config", "some-config",
				"--env", envCfg.EnvName,
				"--weight", "100").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris weight without --config fails", func() {
			cli.Run("app", "polaris", "weight",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"--weight", "100").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris weight without --env fails", func() {
			cli.Run("app", "polaris", "weight",
				"--app", envCfg.AppID,
				"--config", "some-config",
				"--weight", "100").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app polaris weight without --weight fails", func() {
			cli.Run("app", "polaris", "weight",
				"--app", envCfg.AppID,
				"--config", "some-config",
				"--env", envCfg.EnvName).
				ExpectFailure().ExpectOutputContains("required")
		})
	})
})
