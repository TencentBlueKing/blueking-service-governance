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

// Package framework 提供 e2e 基础框架功能
package framework

import (
	"encoding/json"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// CommandSpec 描述一个 CLI 命令的测试规格，用于自动生成共享测试用例
type CommandSpec struct {
	// Path 命令路径，如 []string{"app", "deploy", "list"}
	Path []string
	// RequiredFlags 执行命令所需的 flags 及其有效值
	RequiredFlags map[string]string
	// SupportsOutput 是否支持 -o flag
	SupportsOutput bool
	// RequiresAuth 是否需要认证（未认证时应报错）
	RequiresAuth bool
}

// RunSharedTests 根据 CommandSpec 自动生成一组共享的测试用例。
// 包含：基本成功执行、-o json 输出合法性、未认证报错。
func RunSharedTests(cli *CLI, cfg *EnvConfig, spec CommandSpec) {
	args := buildArgs(spec)

	ginkgo.It("exits with code 0", func() {
		cli.Run(args...).ExpectSuccess()
	})

	if spec.SupportsOutput {
		ginkgo.It("supports -o json with valid JSON output", func() {
			jsonArgs := append(append([]string{}, args...), "-o", "json")
			result := cli.Run(jsonArgs...)
			result.ExpectSuccess()
			var data any
			gomega.Expect(json.Unmarshal([]byte(result.Stdout), &data)).To(gomega.Succeed())
		})
	}

	if spec.RequiresAuth {
		ginkgo.It("without login exits with non-zero code and shows auth hint", func() {
			RunWithoutAuth(cli, cfg, func() {
				result := cli.Run(args...)
				result.ExpectFailure().
					ExpectOutputContains("unauthorized")
			})
		})
	}
}

// RunMissingFlagTests 测试缺少必需 flag 时的错误行为。
// missingFlags 是需要测试的 flag 名称列表（如 "--app", "--env"），
// 每个都会生成一个子用例验证缺少该 flag 时命令失败且输出包含 "required"。
func RunMissingFlagTests(cli *CLI, spec CommandSpec, missingFlags []string) {
	for _, flag := range missingFlags {
		ginkgo.It("without "+flag+" exits with non-zero code", func() {
			args := buildArgsExcluding(spec, flag)
			cli.Run(args...).ExpectFailure().ExpectOutputContains("required")
		})
	}
}

// buildArgs 根据 CommandSpec 构建完整的命令参数列表
func buildArgs(spec CommandSpec) []string {
	args := make([]string, 0, len(spec.Path)+len(spec.RequiredFlags)*2)
	args = append(args, spec.Path...)
	for flag, value := range spec.RequiredFlags {
		args = append(args, flag, value)
	}
	return args
}

// buildArgsExcluding 构建参数列表，排除指定的 flag
func buildArgsExcluding(spec CommandSpec, excludeFlag string) []string {
	args := make([]string, 0, len(spec.Path)+len(spec.RequiredFlags)*2)
	args = append(args, spec.Path...)
	for flag, value := range spec.RequiredFlags {
		if flag == excludeFlag {
			continue
		}
		args = append(args, flag, value)
	}
	return args
}
