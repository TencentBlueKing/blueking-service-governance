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
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/test/e2e/framework"
)

// invalidYAMLPath 返回 testdata/app/ 目录下指定文件的绝对路径
func invalidYAMLPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", "app", name)
}

// 对应 cmd/app/create.go 的输入校验
var _ = Describe("App Create Validation", Ordered, func() {
	BeforeAll(func() {
		framework.EnsureLoggedIn(cli, envCfg)
	})

	// ==================== required + app_name + oneof 校验 ====================
	DescribeTable("input validation",
		func(file, expectedErr string) {
			result := cli.Run("app", "create", "-f", invalidYAMLPath(file))
			Expect(result.ExitCode).NotTo(Equal(0))
			Expect(result.CombinedOutput()).To(ContainSubstring(expectedErr))
		},
		// required 校验
		Entry("missing name", "missing-name.yaml", "'name' is required"),
		Entry("missing build section", "missing-build.yaml", "'buildConfig' is required"),
		// app_name 正则校验
		Entry("invalid name format", "invalid-name-format.yaml", "'name' is invalid"),
		// oneof 枚举校验
		Entry("invalid type", "invalid-type.yaml", "'type' is invalid"),
		Entry("invalid build.source", "invalid-build-source.yaml", "is invalid"),
		Entry("invalid trpc.language", "invalid-trpc-language.yaml", "is invalid"),
		Entry("invalid helm.repoType", "invalid-helm-repo-type.yaml", "is invalid"),
	)
})
