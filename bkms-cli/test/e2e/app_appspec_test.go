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
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// appspecTestdataPath 返回 testdata/appspec/ 目录下指定文件的绝对路径
func appspecTestdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", "appspec", name)
}

// 对应 cmd/app/appspec/
var _ = Describe("App AppSpec", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== appspec view (聚合查看) ====================
	Context("View All", func() {
		It("view all sections exits with code 0", func() {
			result := cli.Run("app", "appspec", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("=== Resources ==="))
			Expect(result.Stdout).To(ContainSubstring("=== Update Strategy ==="))
			Expect(result.Stdout).To(ContainSubstring("=== Lifecycle ==="))
		})

		It("view all with -o json outputs valid JSON", func() {
			result := cli.Run("app", "appspec", "view", "--app", envCfg.AppID, "-o", "json")
			Expect(result.ExitCode).To(Equal(0))
			var data map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &data)).To(Succeed())
			Expect(data).To(HaveKey("resources"))
			Expect(data).To(HaveKey("updateStrategy"))
		})

		It("view all with --env exits with code 0", func() {
			result := cli.Run("app", "appspec", "view", "--app", envCfg.AppID, "--env", envCfg.EnvName)
			Expect(result.ExitCode).To(Equal(0))
		})

		It("view all without --app exits with non-zero code", func() {
			result := cli.Run("app", "appspec", "view")
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring("required"))
		})
	})

	// ==================== appspec resources ====================
	Context("Resources", func() {
		It("view exits with code 0", func() {
			result := cli.Run("app", "appspec", "resources", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Replicas:"))
		})

		It("view with -o json outputs valid JSON", func() {
			result := cli.Run("app", "appspec", "resources", "view", "--app", envCfg.AppID, "-o", "json")
			Expect(result.ExitCode).To(Equal(0))
			var data map[string]any
			Expect(json.Unmarshal([]byte(result.Stdout), &data)).To(Succeed())
		})

		It("edit with valid YAML exits with code 0", func() {
			result := cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("resources.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit without -f exits with non-zero code", func() {
			result := cli.Run("app", "appspec", "resources", "edit", "--app", envCfg.AppID)
			Expect(result.ExitCode).NotTo(Equal(0))
		})

		It("edit with invalid YAML exits with non-zero code", func() {
			result := cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("invalid.yaml"))
			Expect(result.ExitCode).NotTo(Equal(0))
		})

		It("edit with --env sets env override", func() {
			result := cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", appspecTestdataPath("resources.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("reset removes env override", func() {
			result := cli.Run("app", "appspec", "resources", "reset",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName)
			Expect(result.ExitCode).To(Equal(0))
		})

		It("reset without --env exits with non-zero code", func() {
			result := cli.Run("app", "appspec", "resources", "reset", "--app", envCfg.AppID)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring("required"))
		})
	})

	// ==================== appspec update-strategy ====================
	Context("Update Strategy", func() {
		It("view exits with code 0", func() {
			result := cli.Run("app", "appspec", "update-strategy", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Max Surge:"))
		})

		It("edit with valid YAML exits with code 0", func() {
			result := cli.Run("app", "appspec", "update-strategy", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("update-strategy.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with --env and reset", func() {
			result := cli.Run("app", "appspec", "update-strategy", "edit",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", appspecTestdataPath("update-strategy.yaml"))
			Expect(result.ExitCode).To(Equal(0))

			resetResult := cli.Run("app", "appspec", "update-strategy", "reset",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName)
			Expect(resetResult.ExitCode).To(Equal(0))
		})
	})

	// ==================== appspec lifecycle ====================
	Context("Lifecycle", func() {
		It("view exits with code 0", func() {
			result := cli.Run("app", "appspec", "lifecycle", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with valid YAML exits with code 0", func() {
			result := cli.Run("app", "appspec", "lifecycle", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("lifecycle.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with --env and reset", func() {
			result := cli.Run("app", "appspec", "lifecycle", "edit",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", appspecTestdataPath("lifecycle.yaml"))
			Expect(result.ExitCode).To(Equal(0))

			resetResult := cli.Run("app", "appspec", "lifecycle", "reset",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName)
			Expect(resetResult.ExitCode).To(Equal(0))
		})
	})

	// ==================== appspec probe ====================
	Context("Probe", func() {
		It("view exits with code 0", func() {
			result := cli.Run("app", "appspec", "probe", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with valid YAML exits with code 0", func() {
			result := cli.Run("app", "appspec", "probe", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("probe.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with --env and reset", func() {
			result := cli.Run("app", "appspec", "probe", "edit",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", appspecTestdataPath("probe.yaml"))
			Expect(result.ExitCode).To(Equal(0))

			resetResult := cli.Run("app", "appspec", "probe", "reset",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName)
			Expect(resetResult.ExitCode).To(Equal(0))
		})
	})

	// ==================== appspec labels ====================
	Context("Labels", func() {
		It("view exits with code 0", func() {
			result := cli.Run("app", "appspec", "labels", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with valid YAML exits with code 0", func() {
			result := cli.Run("app", "appspec", "labels", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("labels.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with --env and reset", func() {
			result := cli.Run("app", "appspec", "labels", "edit",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", appspecTestdataPath("labels.yaml"))
			Expect(result.ExitCode).To(Equal(0))

			resetResult := cli.Run("app", "appspec", "labels", "reset",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName)
			Expect(resetResult.ExitCode).To(Equal(0))
		})
	})

	// ==================== appspec annotations ====================
	Context("Annotations", func() {
		It("view exits with code 0", func() {
			result := cli.Run("app", "appspec", "annotations", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with valid YAML exits with code 0", func() {
			result := cli.Run("app", "appspec", "annotations", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("annotations.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit with --env and reset", func() {
			result := cli.Run("app", "appspec", "annotations", "edit",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", appspecTestdataPath("annotations.yaml"))
			Expect(result.ExitCode).To(Equal(0))

			resetResult := cli.Run("app", "appspec", "annotations", "reset",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName)
			Expect(resetResult.ExitCode).To(Equal(0))
		})
	})

	// ==================== appspec start-command ====================
	Context("Start Command", func() {
		It("view exits with code 0", func() {
			result := cli.Run("app", "appspec", "start-command", "view", "--app", envCfg.AppID)
			Expect(result.ExitCode).To(Equal(0))
			Expect(result.Stdout).To(ContainSubstring("Command:"))
		})

		It("edit with valid YAML exits with code 0", func() {
			result := cli.Run("app", "appspec", "start-command", "edit",
				"--app", envCfg.AppID,
				"-f", appspecTestdataPath("start-command.yaml"))
			Expect(result.ExitCode).To(Equal(0))
		})

		It("edit without -f exits with non-zero code", func() {
			result := cli.Run("app", "appspec", "start-command", "edit", "--app", envCfg.AppID)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring("required"))
		})
	})

	// ==================== 边界场景 ====================
	Context("Edge Cases", func() {
		It("view with nonexistent app exits with code 0 but outputs empty sections", func() {
			// viewAll 设计为容忍部分失败（并发查询各 section 时静默忽略错误），
			// 所以不存在的 app 也会返回 exit 0，但所有 section 内容为空。
			result := cli.Run("app", "appspec", "view", "--app", "nonexistent-app-xyz123")
			Expect(result.ExitCode).To(Equal(0))
			// 不应包含实际的资源配置数据
			Expect(result.Stdout).NotTo(ContainSubstring("Replicas:"))
		})

		It("edit with nonexistent file exits with non-zero code", func() {
			result := cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"-f", "/tmp/nonexistent-file-xyz.yaml")
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring("not found"))
		})

		It("edit with empty file exits with non-zero code", func() {
			emptyFile := filepath.Join(os.TempDir(), "bkms-e2e-empty.yaml")
			_ = os.WriteFile(emptyFile, []byte(""), 0o644)
			defer os.Remove(emptyFile)

			result := cli.Run("app", "appspec", "resources", "edit",
				"--app", envCfg.AppID,
				"-f", emptyFile)
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring("empty"))
		})

		It("view env effective with nonexistent env exits with non-zero code", func() {
			result := cli.Run("app", "appspec", "resources", "view",
				"--app", envCfg.AppID,
				"--env", "nonexistent-env-xyz123")
			Expect(result.ExitCode).NotTo(Equal(0))
		})
	})
})
