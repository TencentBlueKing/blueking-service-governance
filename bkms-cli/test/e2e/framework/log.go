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
	"fmt"

	"github.com/onsi/ginkgo/v2"
)

// Logf 向 GinkgoWriter 输出带标签的格式化日志。
// 测试通过时日志自动静默，失败时自动输出，与 Ginkgo 生态完全兼容。
//
// 用法：
//
//	Logf("CMD", "executing: %s", cmdStr)
//	// 输出: [CMD] executing: bkms-cli version
func Logf(tag, format string, args ...any) {
	_, _ = fmt.Fprintf(ginkgo.GinkgoWriter, "[%s] %s\n", tag, fmt.Sprintf(format, args...))
}
