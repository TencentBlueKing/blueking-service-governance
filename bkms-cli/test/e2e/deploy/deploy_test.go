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
package deploy_test

import (
	. "github.com/onsi/ginkgo/v2"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// 对应 cmd/app/deploy/
var _ = Describe("App Deploy", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== cmd/app/deploy/list.go ====================
	Context("List", Label("readonly"), func() {
		It("app deploy list exits with code 0", func() {
			cli.Run("app", "deploy", "list",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName).
				ExpectSuccess()
		})

		It("app deploy list without --app exits with non-zero code and output contains required", func() {
			cli.Run("app", "deploy", "list", "--env", "test").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app deploy list with comma-separated env names exits with code 0", func() {
			multiEnv := envCfg.EnvName + "," + envCfg.EnvName
			cli.Run("app", "deploy", "list",
				"--app", envCfg.AppID,
				"--env", multiEnv).
				ExpectSuccess()
		})

		It("app deploy list with invalid env name exits with non-zero code and output contains not found", func() {
			cli.Run("app", "deploy", "list",
				"--app", envCfg.AppID,
				"--env", "nonexistent-env-12345").
				ExpectFailure().ExpectOutputContains("not found")
		})

		It("app deploy list with mixed valid and invalid env names exits with non-zero code", func() {
			mixedEnv := envCfg.EnvName + ",nonexistent-env-12345"
			cli.Run("app", "deploy", "list",
				"--app", envCfg.AppID,
				"--env", mixedEnv).
				ExpectFailure().ExpectOutputContains("not found")
		})
	})

	// ==================== cmd/app/deploy/precheck.go ====================
	Context("Precheck", Label("readonly"), func() {
		It("app deploy precheck runs without error", func() {
			// precheck 可能返回 0（全部变量已定义）或非零（存在未定义变量），两者都是合法行为；
			// 这里仅验证命令不会崩溃，不断言退出码
			cli.Run("app", "deploy", "precheck",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName)
		})

		It("app deploy precheck without --app exits with non-zero code", func() {
			cli.Run("app", "deploy", "precheck", "--env", envCfg.EnvName).
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app deploy precheck without --env exits with non-zero code", func() {
			cli.Run("app", "deploy", "precheck", "--app", envCfg.AppID).
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== cmd/app/deploy/create.go ====================
	Context("Create", Label("destructive"), func() {
		It("app deploy create without --app exits with non-zero code", func() {
			cli.Run("app", "deploy", "create",
				"--env", envCfg.EnvName,
				"-f", "/tmp/nonexistent.yaml").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app deploy create without -f exits with non-zero code", func() {
			cli.Run("app", "deploy", "create",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName).
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app deploy create with nonexistent spec file exits with non-zero code", func() {
			cli.Run("app", "deploy", "create",
				"--app", envCfg.AppID,
				"--env", envCfg.EnvName,
				"-f", "/tmp/nonexistent-deploy-spec-xyz.yaml").
				ExpectFailure().ExpectOutputContains("not found")
		})
	})

	// ==================== cmd/app/deploy/delete.go ====================
	Context("Delete", Label("destructive"), func() {
		It("app deploy delete without --app exits with non-zero code", func() {
			cli.Run("app", "deploy", "delete",
				"--env", envCfg.EnvName, "--yes").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app deploy delete without --env exits with non-zero code", func() {
			cli.Run("app", "deploy", "delete",
				"--app", envCfg.AppID, "--yes").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app deploy delete with nonexistent env exits with non-zero code", func() {
			cli.Run("app", "deploy", "delete",
				"--app", envCfg.AppID,
				"--env", "nonexistent-env-12345",
				"--yes").
				ExpectFailure()
		})
	})
})
