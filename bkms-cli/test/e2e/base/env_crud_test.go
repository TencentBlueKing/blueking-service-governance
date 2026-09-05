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

// 对应 cmd/env/ 的 get / create / update / delete
var _ = Describe("Env CRUD", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== env get ====================
	Context("Get", Label("readonly"), func() {
		It("env get exits with code 0", func() {
			cli.Run("env", "get", "--env", envCfg.EnvName).
				ExpectSuccess().
				ExpectOutputContains(envCfg.EnvName)
		})

		It("env get supports -o json", func() {
			cli.Run("env", "get", "--env", envCfg.EnvName, "-o", "json").
				ExpectSuccess()
		})

		It("env get without --env fails", func() {
			cli.Run("env", "get").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("env get with nonexistent env fails", func() {
			cli.Run("env", "get", "--env", "nonexistent-env-xyz-e2e").
				ExpectFailure()
		})
	})

	// ==================== env create (error cases) ====================
	Context("Create", Label("destructive"), func() {
		It("env create without -f fails", func() {
			cli.Run("env", "create").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("env create with nonexistent file fails", func() {
			cli.Run("env", "create", "-f", "/tmp/nonexistent-env-spec-xyz.yaml").
				ExpectFailure()
		})
	})

	// ==================== env update (error cases) ====================
	Context("Update", Label("destructive"), func() {
		It("env update without --env fails", func() {
			cli.Run("env", "update", "--display-name", "test").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("env update without --display-name or --type fails", func() {
			cli.Run("env", "update", "--env", envCfg.EnvName).
				ExpectFailure()
		})
	})

	// ==================== env delete (error cases) ====================
	Context("Delete", Label("destructive"), func() {
		It("env delete without --env fails", func() {
			cli.Run("env", "delete", "--yes").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("env delete with nonexistent env fails", func() {
			cli.Run("env", "delete", "--env", "nonexistent-env-xyz-e2e", "--yes").
				ExpectFailure()
		})
	})
})
