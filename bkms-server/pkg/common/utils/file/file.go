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

// Package file 提供文件与路径相关的工具，包括受限根目录下的安全读取（避免路径逃逸与符号链接攻击）和路径比较
package file

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// IsSameAbsPath 判断两个路径是否指向同一绝对路径，比较前会做 filepath.Clean（去掉多余的 / 和 . / ..）
// 任一方是相对路径时返回 false：相对路径依赖工作目录，单凭字符串无法确定指向
func IsSameAbsPath(a, b string) bool {
	if !filepath.IsAbs(a) || !filepath.IsAbs(b) {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// SafeReadFile 安全地读取 baseDir 下的文件，底层使用 os.Root 以防止路径逃逸和符号链接攻击
func SafeReadFile(baseDir, name string) ([]byte, error) {
	root, err := os.OpenRoot(baseDir)
	if err != nil {
		return nil, errors.Wrap(err, "opening root directory")
	}
	defer root.Close()

	return root.ReadFile(name)
}
