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

// 对应 cmd/app/build/
var _ = Describe("App Build", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== cmd/app/build/list.go ====================
	Context("List", Label("readonly"), func() {
		It("app build list exits with code 0", func() {
			cli.Run("app", "build", "list", "--app", envCfg.AppID).ExpectSuccess()
		})

		It("app build list supports -o json", func() {
			cli.Run("app", "build", "list", "--app", envCfg.AppID, "-o", "json").
				ExpectSuccess()
		})

		It("app build list without --app fails", func() {
			cli.Run("app", "build", "list").
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== cmd/app/build/create.go ====================
	Context("Create", Label("destructive"), func() {
		It("app build create without --app fails", func() {
			cli.Run("app", "build", "create",
				"--branch", "main",
				"--image-tag", "v1.0.0").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app build create without --branch fails", func() {
			cli.Run("app", "build", "create",
				"--app", envCfg.AppID,
				"--image-tag", "v1.0.0").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app build create without --image-tag fails", func() {
			cli.Run("app", "build", "create",
				"--app", envCfg.AppID,
				"--branch", "main").
				ExpectFailure().ExpectOutputContains("required")
		})
	})
})
