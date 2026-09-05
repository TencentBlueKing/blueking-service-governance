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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// 对应 cmd/app/create.go, cmd/app/get.go, cmd/app/delete.go
var _ = Describe("App CRUD", Ordered, Label("destructive"), func() {
	var (
		appName  string
		specFile string
	)

	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)

		appName = fmt.Sprintf("e2e-test-%d", time.Now().UnixMilli())
		spec := map[string]any{
			"name": appName,
			"type": "trpc",
			"buildConfig": map[string]any{
				"sourceType": "imageRegistry",
				"imageBuildConfig": map[string]any{
					"name": "mirrors.example.com/test/e2e-image",
				},
			},
			"appModelSpec": map[string]any{
				"trpcSpec": map[string]any{
					"language": "go",
					"fileName": "trpc_go.yaml",
					"filePath": "/usr/local/trpc/conf",
				},
			},
		}
		specFile = framework.WriteFixtureFile(spec, "app-create")
	})

	AfterAll(func() {
		if appName != "" {
			cli.Run("app", "delete", "--app", appName, "--yes")
		}
		framework.CleanupFixtureFile(specFile)
	})

	// ==================== app create ====================
	Context("Create", func() {
		It("should create an app from spec file", func() {
			cli.Run("app", "create", "-f", specFile).
				ExpectSuccess().
				ExpectStdoutContains("App created successfully").
				ExpectStdoutContains(appName)
		})

		It("create without -f exits with non-zero code", func() {
			cli.Run("app", "create").
				ExpectFailure().ExpectOutputContains("required")
		})
	})

	// ==================== app get ====================
	Context("Get", func() {
		It("should get the created app", func() {
			cli.Run("app", "get", "--app", appName).
				ExpectSuccess().
				ExpectOutputContains(appName).
				ExpectOutputContains("trpc")
		})

		It("should get app with -o json", func() {
			cli.Run("app", "get", "--app", appName, "-o", "json").
				ExpectJSON(func(data any) {
					m := data.(map[string]any)
					Expect(m["name"]).To(Equal(appName))
					Expect(m["type"]).To(Equal("trpc"))
				})
		})

		It("app get without --app exits with non-zero code", func() {
			cli.Run("app", "get").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app get with nonexistent app exits with non-zero code", func() {
			cli.Run("app", "get", "--app", "nonexistent-app-xyz-e2e").
				ExpectFailure().ExpectOutputContains("not found")
		})
	})

	// ==================== app delete ====================
	Context("Delete", func() {
		It("app delete without --app exits with non-zero code", func() {
			cli.Run("app", "delete", "--yes").
				ExpectFailure().ExpectOutputContains("required")
		})

		It("app delete with nonexistent app exits with non-zero code", func() {
			cli.Run("app", "delete", "--app", "nonexistent-app-xyz-e2e", "--yes").
				ExpectFailure()
		})

		It("should delete the created app", func() {
			cli.Run("app", "delete", "--app", appName, "--yes").
				ExpectSuccess().
				ExpectOutputContains("deleted successfully")
			appName = ""
		})
	})
})
