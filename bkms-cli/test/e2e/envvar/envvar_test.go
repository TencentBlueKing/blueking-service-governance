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
package envvar_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// 对应 cmd/envvar/
var _ = Describe("Envvar", Ordered, Label("destructive"), func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== List ====================
	Context("List", func() {
		It("envvar list scoped exits with code 0", func() {
			cli.Run("envvar", "list", "scoped").ExpectSuccess()
		})

		It("envvar list app exits with code 0", func() {
			cli.Run("envvar", "list", "app", "--app", envCfg.AppID).ExpectSuccess()
		})

		It("envvar list env exits with code 0", func() {
			cli.Run("envvar", "list", "env", "--env", envCfg.EnvName).ExpectSuccess()
		})

		It("envvar list app without --app fails", func() {
			cli.Run("envvar", "list", "app").ExpectFailure().ExpectOutputContains("required")
		})

		It("envvar list env without --env fails", func() {
			cli.Run("envvar", "list", "env").ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== Scoped CRUD ====================
	Context("Scoped CRUD", func() {
		testKey := "E2E_TEST_SCOPED_VAR"

		It("create scoped envvar", func() {
			cli.Run("envvar", "create", "scoped",
				"--key", testKey, "--value", "test-value").
				ExpectSuccess()
		})

		It("update scoped envvar", func() {
			cli.Run("envvar", "update", "scoped",
				"--key", testKey, "--value", "updated-value").
				ExpectSuccess()
		})

		It("delete scoped envvar", func() {
			cli.Run("envvar", "delete", "scoped", "--key", testKey).
				ExpectSuccess()
		})
	})

	// ==================== App CRUD ====================
	Context("App CRUD", func() {
		testKey := "E2E_TEST_APP_VAR"

		It("create app envvar", func() {
			cli.Run("envvar", "create", "app",
				"--app", envCfg.AppID,
				"--key", testKey, "--value", "test-value").
				ExpectSuccess()
		})

		It("update app envvar", func() {
			cli.Run("envvar", "update", "app",
				"--app", envCfg.AppID,
				"--key", testKey, "--value", "updated-value").
				ExpectSuccess()
		})

		It("delete app envvar", func() {
			cli.Run("envvar", "delete", "app",
				"--app", envCfg.AppID,
				"--key", testKey).
				ExpectSuccess()
		})
	})

	// ==================== Env CRUD ====================
	Context("Env CRUD", func() {
		testKey := "E2E_TEST_ENV_VAR"

		It("create env envvar", func() {
			cli.Run("envvar", "create", "env",
				"--env", envCfg.EnvName,
				"--key", testKey, "--value", "test-value").
				ExpectSuccess()
		})

		It("update env envvar", func() {
			cli.Run("envvar", "update", "env",
				"--env", envCfg.EnvName,
				"--key", testKey, "--value", "updated-value").
				ExpectSuccess()
		})

		It("delete env envvar", func() {
			cli.Run("envvar", "delete", "env",
				"--env", envCfg.EnvName,
				"--key", testKey).
				ExpectSuccess()
		})
	})

	// ==================== Import/Export ====================
	Context("Export and Import", func() {
		It("export scoped envvars exits with code 0", func() {
			cli.Run("envvar", "export", "scoped").ExpectSuccess()
		})

		It("export app envvars exits with code 0", func() {
			cli.Run("envvar", "export", "app", "--app", envCfg.AppID).ExpectSuccess()
		})

		It("export env envvars exits with code 0", func() {
			cli.Run("envvar", "export", "env", "--env", envCfg.EnvName).ExpectSuccess()
		})
	})

	// ==================== Error Cases ====================
	Context("Error Cases", func() {
		It("create without --key fails", func() {
			cli.Run("envvar", "create", "scoped", "--value", "val").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("delete without --key fails", func() {
			cli.Run("envvar", "delete", "scoped").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("delete nonexistent key fails", func() {
			cli.Run("envvar", "delete", "scoped",
				"--key", "NONEXISTENT_KEY_E2E_XYZ").
				ExpectFailure()
		})
	})
})
