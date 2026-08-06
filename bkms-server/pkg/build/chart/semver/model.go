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

// Package semver 提供 Helm Chart semver 版本号生成功能（并发安全）
package semver

import "fmt"

// BumpType semver 递增段类型
type BumpType string

const (
	// BumpPatch 递增 patch 段（默认）
	BumpPatch BumpType = "patch"
	// BumpMinor 递增 minor 段，patch 归零
	BumpMinor BumpType = "minor"
	// BumpMajor 递增 major 段，minor+patch 归零
	BumpMajor BumpType = "major"
)

// Counter semver 计数器文档
type Counter struct {
	// AppID 应用 ID（作为文档主键）
	AppID string `bson:"_id"`
	// Major 主版本号
	Major int64 `bson:"major"`
	// Minor 次版本号
	Minor int64 `bson:"minor"`
	// Patch 修订版本号
	Patch int64 `bson:"patch"`
}

// FormatSemver 将 Counter 格式化为 semver 字符串（major.minor.patch）
func (c *Counter) FormatSemver() string {
	return fmt.Sprintf("%d.%d.%d", c.Major, c.Minor, c.Patch)
}

// PreviewNext 按 bumpType 纯内存计算下一个 semver 值（不修改数据库）
// 经典归零语义：递增 major 时 minor+patch 归零，递增 minor 时 patch 归零
func (c *Counter) PreviewNext(bumpType BumpType) *Counter {
	next := &Counter{
		AppID: c.AppID,
		Major: c.Major,
		Minor: c.Minor,
		Patch: c.Patch,
	}

	switch bumpType {
	case BumpPatch:
		next.Patch++
	case BumpMinor:
		next.Minor++
		next.Patch = 0
	case BumpMajor:
		next.Major++
		next.Minor = 0
		next.Patch = 0
	}

	return next
}
