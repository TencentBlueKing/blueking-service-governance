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
package app_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

var _ = Describe("App AppSpec", Ordered, Label("destructive"), func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== appspec view (聚合查看) ====================
	Context("View All", func() {
		It("view all sections exits with code 0", func() {
			cli.Run("app", "appspec", "view", "--app", envCfg.AppID).
				ExpectSuccess().
				ExpectStdoutContains("=== Resources ===").
				ExpectStdoutContains("=== Update Strategy ===").
				ExpectStdoutContains("=== Lifecycle ===")
		})

		It("view all with -o json outputs valid JSON", func() {
			cli.Run("app", "appspec", "view", "--app", envCfg.AppID, "-o", "json").
				ExpectJSON(func(data any) {
					m := data.(map[string]any)
					Expect(m).To(HaveKey("resources"))
					Expect(m).To(HaveKey("updateStrategy"))
					Expect(m).To(HaveKey("underlayIP"))
					Expect(m).NotTo(HaveKey("devMode"))
				})
		})

		It("view all with --env includes devMode", func() {
			cli.Run("app", "appspec", "view",
				"--app", envCfg.AppID, "--env", envCfg.EnvName, "-o", "json").
				ExpectJSON(func(data any) {
					m := data.(map[string]any)
					Expect(m).To(HaveKey("underlayIP"))
					Expect(m).To(HaveKey("devMode"))
				})
		})

		It("view all with --env exits with code 0", func() {
			cli.Run("app", "appspec", "view", "--app", envCfg.AppID, "--env", envCfg.EnvName).
				ExpectSuccess()
		})

		It("view all without --app exits with non-zero code", func() {
			cli.Run("app", "appspec", "view").
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== 6 个标准 section 参数化测试 ====================
	DescribeTable("Standard Section CRUD",
		func(section, fixture string) {
			By("view exits with code 0")
			cli.Run("app", "appspec", section, "view", "--app", envCfg.AppID).
				ExpectSuccess()

			By("view with -o json outputs valid JSON")
			cli.Run("app", "appspec", section, "view", "--app", envCfg.AppID, "-o", "json").
				ExpectJSON(func(_ any) {})

			By("edit with valid YAML exits with code 0")
			cli.Run("app", "appspec", section, "edit",
				"--app", envCfg.AppID,
				"-f", framework.TestdataPath("appspec", fixture)).
				ExpectSuccess()

			By("edit with --env sets env override")
			cli.Run("app", "appspec", section, "edit",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", framework.TestdataPath("appspec", fixture)).
				ExpectSuccess()

			By("reset removes env override")
			cli.Run("app", "appspec", section, "reset",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName).
				ExpectSuccess()
		},
		Entry("resources", "resources", "resources.yaml"),
		Entry("update-strategy", "update-strategy", "update-strategy.yaml"),
		Entry("lifecycle", "lifecycle", "lifecycle.yaml"),
		Entry("probe", "probe", "probe.yaml"),
		Entry("labels", "labels", "labels.yaml"),
		Entry("annotations", "annotations", "annotations.yaml"),
	)

	// ==================== resources 额外测试 ====================
	Context("Resources Extra", func() {
		It("edit without -f exits with non-zero code", func() {
			cli.Run("app", "appspec", "resources", "edit", "--app", envCfg.AppID).
				ExpectFailure()
		})

		It("edit with invalid YAML exits with non-zero code", func() {
			cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"-f", framework.TestdataPath("appspec", "invalid.yaml")).
				ExpectFailure()
		})

		It("reset without --env exits with non-zero code", func() {
			cli.Run("app", "appspec", "resources", "reset", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== appspec start-command ====================
	Context("Start Command", func() {
		It("view exits with code 0", func() {
			cli.Run("app", "appspec", "start-command", "view", "--app", envCfg.AppID).
				ExpectSuccess().ExpectStdoutContains("Command:")
		})

		It("edit with valid YAML exits with code 0", func() {
			cli.Run("app", "appspec", "start-command", "edit",
				"--app", envCfg.AppID,
				"-f", framework.TestdataPath("appspec", "start-command.yaml")).
				ExpectSuccess()
		})

		It("edit without -f exits with non-zero code", func() {
			cli.Run("app", "appspec", "start-command", "edit", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== appspec underlay-ip ====================
	Context("Underlay IP", func() {
		It("view exits with code 0", func() {
			cli.Run("app", "appspec", "underlay-ip", "view", "--app", envCfg.AppID).
				ExpectSuccess().ExpectStdoutContains("Enabled:")
		})

		It("view with -o json outputs valid JSON", func() {
			cli.Run("app", "appspec", "underlay-ip", "view", "--app", envCfg.AppID, "-o", "json").
				ExpectJSON(func(data any) {
					Expect(data.(map[string]any)).To(HaveKey("enabled"))
				})
		})

		It("enable then disable at default level", func() {
			cli.Run("app", "appspec", "underlay-ip", "enable", "--app", envCfg.AppID).
				ExpectSuccess()
			cli.Run("app", "appspec", "underlay-ip", "disable", "--app", envCfg.AppID).
				ExpectSuccess()
		})

		It("enable with --env then reset removes the override", func() {
			cli.Run("app", "appspec", "underlay-ip", "enable",
				"--app", envCfg.AppID, "--env", envCfg.EnvName).
				ExpectSuccess()
			cli.Run("app", "appspec", "underlay-ip", "reset",
				"--app", envCfg.AppID, "--env", envCfg.EnvName).
				ExpectSuccess()
		})

		It("reset without --env exits with non-zero code", func() {
			cli.Run("app", "appspec", "underlay-ip", "reset", "--app", envCfg.AppID).
				ExpectFailure()
		})
	})

	// ==================== appspec dev-mode（仅环境级） ====================
	Context("Dev Mode", func() {
		It("view with --env exits with code 0", func() {
			cli.Run("app", "appspec", "dev-mode", "view",
				"--app", envCfg.AppID, "--env", envCfg.EnvName).
				ExpectSuccess().
				ExpectStdoutContains("Enabled:").
				ExpectStdoutContains("Work Path:")
		})

		It("view without --env exits with non-zero code", func() {
			cli.Run("app", "appspec", "dev-mode", "view", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})

		It("enable without --env exits with non-zero code", func() {
			cli.Run("app", "appspec", "dev-mode", "enable", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})

		It("disable without --env exits with non-zero code", func() {
			cli.Run("app", "appspec", "dev-mode", "disable", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})

		It("enable then disable with --env", func() {
			cli.Run("app", "appspec", "dev-mode", "enable",
				"--app", envCfg.AppID, "--env", envCfg.EnvName).
				ExpectSuccess()
			cli.Run("app", "appspec", "dev-mode", "disable",
				"--app", envCfg.AppID, "--env", envCfg.EnvName).
				ExpectSuccess()
		})

		It("reset with --env exits with code 0", func() {
			cli.Run("app", "appspec", "dev-mode", "reset",
				"--app", envCfg.AppID, "--env", envCfg.EnvName).
				ExpectSuccess()
		})
	})

	// ==================== 边界场景 ====================
	Context("Edge Cases", func() {
		It("view with nonexistent app exits with non-zero code", func() {
			cli.Run("app", "appspec", "view", "--app", "nonexistent-app-xyz123").
				ExpectFailure().ExpectOutputContains("not found")
		})

		It("edit with nonexistent file exits with non-zero code", func() {
			cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"-f", "/tmp/nonexistent-file-xyz.yaml").
				ExpectFailure().ExpectOutputContains("not found")
		})

		It("edit with empty file exits with non-zero code", func() {
			emptyFile := filepath.Join(os.TempDir(), "bkms-e2e-empty.yaml")
			Expect(os.WriteFile(emptyFile, []byte(""), 0o644)).To(Succeed())
			DeferCleanup(os.Remove, emptyFile)

			cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"-f", emptyFile).
				ExpectFailure().ExpectOutputContains("empty")
		})

		It("view env effective with nonexistent env exits with non-zero code", func() {
			cli.Run("app", "appspec", "resources", "view",
				"--app", envCfg.AppID,
				"--env", "nonexistent-env-xyz123").
				ExpectFailure()
		})
	})
})
