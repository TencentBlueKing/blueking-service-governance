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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// semverRegexp 匹配标准 semver 格式的镜像 Tag
// 支持必选的 v 前缀和可选的 pre-release 后缀
var semverRegexp = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)(?:-[\w.]+)?$`)

// invalidTagCharsRegexp 匹配一个或多个连续的不符合 Docker 镜像 Tag 规范的字符（只保留字母、数字、.、-、_）
var invalidTagCharsRegexp = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// semverVersion 语义化版本号三元组
type semverVersion struct {
	Major int
	Minor int
	Patch int
}

// lessThan 判断当前版本是否小于 other
func (v semverVersion) lessThan(other semverVersion) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

// GenerateRecommendedSemverImageTag 根据镜像仓库已有的 Tag 列表，生成推荐的语义化版本镜像 TAG
// 找到最大的 semver 版本号后 Patch+1 返回，无有效 Tag 时返回 v1.0.0
func GenerateRecommendedSemverImageTag(tags []string) string {
	var maxVer *semverVersion

	for _, tag := range tags {
		matches := semverRegexp.FindStringSubmatch(tag)
		if matches == nil {
			continue
		}

		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		patch, _ := strconv.Atoi(matches[3])

		ver := semverVersion{Major: major, Minor: minor, Patch: patch}
		if maxVer == nil || maxVer.lessThan(ver) {
			maxVer = &ver
		}
	}

	if maxVer == nil {
		return "v1.0.0"
	}
	return fmt.Sprintf("v%d.%d.%d", maxVer.Major, maxVer.Minor, maxVer.Patch+1)
}

// maxBranchNameLen 分支名最大长度
const maxBranchNameLen = 128

// GenerateRecommendedCustomImageTag 生成推荐的自定义版本号
// branch 为可选的分支/Tag 名称，now 为当前时间
// opts 为自定义 Tag 选项，为 nil 时使用默认行为（withRevision=true, withBuildTime=true, prefix=""）
// 按 {prefix}-{branch}-{timestamp} 顺序拼接已启用的字段，用 - 分隔，跳过未启用的部分
// 分支名中不符合 Docker 镜像 Tag 规范的字符会被替换为 -，且长度截断至 128 字符
func GenerateRecommendedCustomImageTag(branch string, now time.Time, opts *CustomTagOpts) string {
	// opts 为 nil 时使用默认行为
	if opts == nil {
		opts = &CustomTagOpts{WithRevision: true, WithBuildTime: true}
	}

	var parts []string

	// 拼接 prefix
	if opts.Prefix != "" {
		parts = append(parts, opts.Prefix)
	}

	// 拼接分支/Tag 名称
	if opts.WithRevision && branch != "" {
		// 将连续的非法字符替换为单个 -，并去除首尾 -
		branch = invalidTagCharsRegexp.ReplaceAllString(branch, "-")
		branch = strings.Trim(branch, "-")
		// 截断分支名长度
		if len(branch) > maxBranchNameLen {
			branch = branch[:maxBranchNameLen]
		}
		if branch != "" {
			parts = append(parts, branch)
		}
	}

	// 拼接构建时间戳
	if opts.WithBuildTime {
		parts = append(parts, now.Format("200601021504"))
	}

	return strings.Join(parts, "-")
}
