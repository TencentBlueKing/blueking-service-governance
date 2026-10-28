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

package dockerfile

import (
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

const (
	// defaultArtifactDir go / cpp 模板未配置产物目录时使用的默认值，与历史固定路径 /out 保持一致
	defaultArtifactDir = "/out"
	// maxArtifactDirLen 产物目录允许的最大长度，与 bkms-server 保存配置时的上限保持一致
	maxArtifactDirLen = 128
)

// safeArtifactDirRegexp 限定产物目录可用字符
//
// 产物目录会原样拼进 Dockerfile 的 ENV / RUN / COPY 指令，只放行路径必需的字符，
// 避免空格、引号、分号、$ 等字符破坏指令结构或引入命令注入
var safeArtifactDirRegexp = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// resolveArtifactDir 返回渲染使用的产物目录
//
// 为空时返回语言默认值；非空时按与 bkms-server 一致的规则校验，并返回规范化后的路径。
// 服务端保存前已经校验过一次，这里再校验是为了挡住绕过服务端直接注入的流水线变量
func resolveArtifactDir(raw string, defaultPath string) (string, error) {
	if raw == "" {
		return defaultPath, nil
	}

	if len(raw) > maxArtifactDirLen {
		return "", errors.Errorf(
			"invalid artifact dir %q: length must not exceed %d", raw, maxArtifactDirLen,
		)
	}

	if !safeArtifactDirRegexp.MatchString(raw) {
		return "", errors.Errorf(
			"invalid artifact dir %q: only letters, digits, dot, underscore, hyphen and slash are allowed", raw,
		)
	}

	if !strings.HasPrefix(raw, "/") {
		return "", errors.Errorf("invalid artifact dir %q: must be an absolute path", raw)
	}

	// 必须在规范化之前检查 ..：path.Clean 会把 /out/../etc 折叠成 /etc，先清理再检查就会放行越界路径
	if lo.Contains(strings.Split(raw, "/"), "..") {
		return "", errors.Errorf("invalid artifact dir %q: must not contain '..' path segment", raw)
	}

	// 去掉末尾 /、合并连续 /、去掉 . 段；/./ 这类输入清理后才暴露为根目录，所以根目录检查放在最后
	cleaned := path.Clean(raw)
	if cleaned == "/" {
		return "", errors.Errorf("invalid artifact dir %q: must not be the root directory", raw)
	}

	return cleaned, nil
}
