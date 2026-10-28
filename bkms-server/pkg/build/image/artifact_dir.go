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

package build

import (
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
)

// MaxPlatformBuildArtifactDirLen 平台通用构建产物目录允许的最大长度，与 bkms-dockerfile-generator 保持一致
const MaxPlatformBuildArtifactDirLen = 128

// safeArtifactDirRegexp 限定产物目录可用字符
//
// 产物目录会原样拼进生成的 Dockerfile 的 ENV / RUN / COPY 指令，只放行路径必需的字符，
// 避免空格、引号、分号、$ 等字符破坏指令结构或引入命令注入
var safeArtifactDirRegexp = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)

// NormalizePlatformBuildArtifactDir 校验平台通用构建产物目录，并返回规范化后的路径
//
// 空串表示未配置，原样返回空串，由 Dockerfile 生成器按语言默认值处理；
// 非空时的校验规则与 bkms-dockerfile-generator 渲染前的二次校验保持一致，两边需要同步修改
func NormalizePlatformBuildArtifactDir(artifactDir string) (string, error) {
	if artifactDir == "" {
		return "", nil
	}

	if len(artifactDir) > MaxPlatformBuildArtifactDirLen {
		return "", errors.Errorf("length must not exceed %d", MaxPlatformBuildArtifactDirLen)
	}

	if !safeArtifactDirRegexp.MatchString(artifactDir) {
		return "", errors.New("only letters, digits, dot, underscore, hyphen and slash are allowed")
	}

	if !strings.HasPrefix(artifactDir, "/") {
		return "", errors.New("must be an absolute path")
	}

	// 必须在规范化之前检查 ..：path.Clean 会把 /out/../etc 折叠成 /etc，先清理再检查就会放行越界路径
	if lo.Contains(strings.Split(artifactDir, "/"), "..") {
		return "", errors.New("must not contain '..' path segment")
	}

	// 去掉末尾 /、合并连续 /、去掉 . 段；/./ 这类输入清理后才暴露为根目录，所以根目录检查放在最后
	cleaned := path.Clean(artifactDir)
	if cleaned == "/" {
		return "", errors.New("must not be the root directory")
	}

	return cleaned, nil
}
