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
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/onsi/ginkgo/v2"
)

// Result 封装 CLI 命令执行结果
type Result struct {
	// ExitCode 命令退出码
	ExitCode int
	// Stdout 标准输出
	Stdout string
	// Stderr 标准错误输出
	Stderr string
}

// CombinedOutput 返回 stdout + stderr 合并输出
func (r Result) CombinedOutput() string {
	return r.Stdout + r.Stderr
}

// CLI 封装 bkms-cli 二进制文件的执行
type CLI struct {
	// BinPath 二进制文件路径
	BinPath string
}

// NewCLI 创建 CLI 执行器实例
// 按以下优先级查找二进制文件：
//  1. BKMS_CLI_BIN 环境变量指定的路径
//  2. build/bkms-cli-e2e（E2E 测试专用二进制，避免覆盖正式构建产物）
//  3. build/bkms-cli-{os}-{arch}（带平台后缀）
//  4. build/bkms-cli（默认名称，make build 不指定 GOOS/GOARCH 时的产物）
func NewCLI() *CLI {
	binPath := findBinary()
	validateBinary(binPath)
	Logf("CLI", "Using binary: %s", binPath)
	return &CLI{BinPath: binPath}
}

// Run 执行 CLI 命令并返回结果
func (c *CLI) Run(args ...string) Result {
	return c.execute("", args...)
}

// RunWithStdin 执行 CLI 命令，并通过 stdin 传入输入内容
func (c *CLI) RunWithStdin(stdin string, args ...string) Result {
	return c.execute(stdin, args...)
}

// execute 内部执行方法
func (c *CLI) execute(stdin string, args ...string) Result {
	cmd := exec.Command(c.BinPath, args...) //nolint:gosec // 测试框架中执行 E2E CLI 二进制

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	// 记录执行的命令
	cmdStr := fmt.Sprintf("%s %s", c.BinPath, strings.Join(args, " "))
	Logf("CMD", "%s", cmdStr)

	err := cmd.Run()

	result := Result{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// 非退出码错误（如二进制不存在），设为 -1
			result.ExitCode = -1
			Logf("CMD", "Execution error: %v", err)
		}
	}

	// 记录退出码
	Logf("CMD", "Exit code: %d", result.ExitCode)

	// 退出码非 0 时，自动输出 CLI 原始错误信息，便于排查是 CLI 还是 Server 的问题
	if result.ExitCode != 0 {
		Logf("CMD", "CLI raw output ↓↓↓")
		if result.Stdout != "" {
			Logf("STDOUT", "%s", result.Stdout)
		}
		if result.Stderr != "" {
			Logf("STDERR", "%s", result.Stderr)
		}
		Logf("CMD", "CLI raw output ↑↑↑")
	}

	return result
}

// findBinary 按优先级查找 bkms-cli 二进制文件
func findBinary() string {
	// 1. 优先使用环境变量指定的路径（显式指定时路径必须有效，否则直接 Fail）
	if binPath := os.Getenv("BKMS_CLI_BIN"); binPath != "" {
		if _, err := os.Stat(binPath); err != nil {
			ginkgo.Fail(fmt.Sprintf("BKMS_CLI_BIN is set but path does not exist: %s", binPath))
		}
		return binPath
	}

	// 获取项目 build 目录（相对于 test/e2e/framework/）
	buildDir := getBuildDir()

	// 按优先级构建候选二进制名称列表
	candidates := candidateBinNames()

	// 依次尝试查找候选路径
	tried := make([]string, 0, len(candidates))
	for _, name := range candidates {
		binPath := filepath.Join(buildDir, name)
		if _, err := os.Stat(binPath); err == nil {
			return binPath
		}
		tried = append(tried, binPath)
	}

	// 所有候选路径均未找到，直接 Fail
	ginkgo.Fail(fmt.Sprintf("binary not found in build dir %s, tried: %s",
		buildDir, strings.Join(tried, ", ")))

	return ""
}

// candidateBinNames 按优先级返回候选二进制文件名列表
func candidateBinNames() []string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return []string{
		// E2E 专用二进制（避免覆盖正式构建产物）
		fmt.Sprintf("bkms-cli-e2e%s", ext),
		// 带平台后缀的二进制
		fmt.Sprintf("bkms-cli-%s-%s%s", runtime.GOOS, runtime.GOARCH, ext),
		// 默认名称（make build 不指定 GOOS/GOARCH 时的产物）
		fmt.Sprintf("bkms-cli%s", ext),
	}
}

// validateBinary 校验二进制文件存在且可执行，不满足时直接 Fail
func validateBinary(binPath string) {
	info, err := os.Stat(binPath)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("binary file does not exist: %s", binPath))
	}
	if info.IsDir() {
		ginkgo.Fail(fmt.Sprintf("binary path is a directory, not a file: %s", binPath))
	}
	if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
		ginkgo.Fail(fmt.Sprintf("binary file is not executable: %s (mode: %s)", binPath, info.Mode()))
	}
}

// getBuildDir 获取 build 目录的绝对路径（必须通过 BKMS_CLI_BUILD_DIR 环境变量指定）
func getBuildDir() string {
	buildDir := os.Getenv("BKMS_CLI_BUILD_DIR")
	if buildDir == "" {
		ginkgo.Fail("BKMS_CLI_BUILD_DIR is not set; please set it to the absolute path of " +
			"the build directory (e.g. make e2e-go-test will set it automatically)")
	}

	info, err := os.Stat(buildDir)
	if err != nil || !info.IsDir() {
		ginkgo.Fail(fmt.Sprintf("BKMS_CLI_BUILD_DIR path does not exist or is not a directory: %s", buildDir))
	}

	return buildDir
}
