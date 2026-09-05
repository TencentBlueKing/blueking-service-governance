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
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// ExpectSuccess 断言命令执行成功（ExitCode == 0），失败时打印完整输出便于排查
func (r Result) ExpectSuccess() Result {
	if r.ExitCode != 0 {
		ginkgo.Fail(fmt.Sprintf("expected exit code 0, got %d\nstdout: %s\nstderr: %s",
			r.ExitCode, r.Stdout, r.Stderr))
	}
	return r
}

// ExpectFailure 断言命令执行失败（ExitCode != 0）
func (r Result) ExpectFailure() Result {
	if r.ExitCode == 0 {
		ginkgo.Fail(fmt.Sprintf("expected non-zero exit code, got 0\nstdout: %s", r.Stdout))
	}
	return r
}

// ExpectExitCode 断言精确退出码
func (r Result) ExpectExitCode(code int) Result {
	if r.ExitCode != code {
		ginkgo.Fail(fmt.Sprintf("expected exit code %d, got %d\nstdout: %s\nstderr: %s",
			code, r.ExitCode, r.Stdout, r.Stderr))
	}
	return r
}

// ExpectStdoutContains 断言标准输出包含指定子串
func (r Result) ExpectStdoutContains(s string) Result {
	if !strings.Contains(r.Stdout, s) {
		ginkgo.Fail(fmt.Sprintf("expected stdout to contain %q, got:\n%s", s, r.Stdout))
	}
	return r
}

// ExpectStdoutNotContains 断言标准输出不包含指定子串
func (r Result) ExpectStdoutNotContains(s string) Result {
	if strings.Contains(r.Stdout, s) {
		ginkgo.Fail(fmt.Sprintf("expected stdout NOT to contain %q, got:\n%s", s, r.Stdout))
	}
	return r
}

// ExpectStderrContains 断言标准错误输出包含指定子串
func (r Result) ExpectStderrContains(s string) Result {
	if !strings.Contains(r.Stderr, s) {
		ginkgo.Fail(fmt.Sprintf("expected stderr to contain %q, got:\n%s", s, r.Stderr))
	}
	return r
}

// ExpectOutputContains 断言合并输出（stdout+stderr）包含指定子串
func (r Result) ExpectOutputContains(s string) Result {
	combined := r.CombinedOutput()
	if !strings.Contains(combined, s) {
		ginkgo.Fail(fmt.Sprintf("expected combined output to contain %q, got:\n%s", s, combined))
	}
	return r
}

// ExpectJSON 将 stdout 解析为 JSON 并传入回调函数进行断言
func (r Result) ExpectJSON(fn func(data any)) Result {
	r.ExpectSuccess()
	var data any
	gomega.Expect(json.Unmarshal([]byte(r.Stdout), &data)).To(gomega.Succeed(),
		"failed to parse stdout as JSON:\n%s", r.Stdout)
	fn(data)
	return r
}

// ExpectJSONArray 将 stdout 解析为 JSON 数组并传入回调函数进行断言
func (r Result) ExpectJSONArray(fn func(arr []any)) Result {
	r.ExpectSuccess()
	var arr []any
	gomega.Expect(json.Unmarshal([]byte(r.Stdout), &arr)).To(gomega.Succeed(),
		"failed to parse stdout as JSON array:\n%s", r.Stdout)
	fn(arr)
	return r
}
