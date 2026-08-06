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

// Package console 提供 bkms-cli 终端着色输出工具，用于错误、提示等场景
package console

import (
	"fmt"

	"github.com/fatih/color"
)

// Error 向终端输出错误类信息
func Error(format string, args ...any) {
	color.Red(format, args...)
}

// Tips 向终端输出提示类信息
func Tips(format string, args ...any) {
	color.Cyan(format, args...)
}

// Info 向终端输出信息
func Info(format string, args ...any) {
	fmt.Println(fmt.Sprintf(format, args...))
}
