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

// Package version provides bkms-cli version info
package version

import (
	"cmp"
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// DevelopmentVersion identifies source builds without a release version.
const DevelopmentVersion = "dev"

var (
	// Version 版本号
	Version = ""
	// GitHash CommitID
	GitHash = ""
	// BuildTime 二进制构建时间
	BuildTime = ""
	// GoVersion Go 版本号
	GoVersion = runtime.Version()
)

func init() {
	info, _ := debug.ReadBuildInfo()
	resolveBuildInfo(info)
}

func resolveBuildInfo(info *debug.BuildInfo) {
	moduleVersion, revision := DevelopmentVersion, "unknown"
	if info != nil {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			moduleVersion = strings.TrimPrefix(info.Main.Version, "v")
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				revision = cmp.Or(setting.Value, revision)
			}
		}
	}

	// Explicit build flags take precedence over Go module metadata.
	Version = cmp.Or(Version, moduleVersion)
	GitHash = cmp.Or(GitHash, revision)
	BuildTime = cmp.Or(BuildTime, "unknown")
}

// UserAgent 返回访问 bkms-server 时使用的 User-Agent。
func UserAgent() string {
	v := strings.TrimPrefix(Version, "v")
	if v == "" {
		v = DevelopmentVersion
	}
	return fmt.Sprintf("bkms-cli/%s (%s/%s)", v, runtime.GOOS, runtime.GOARCH)
}

// GetVersion 获取版本信息
func GetVersion() string {
	return fmt.Sprintf(
		"\nVersion:   %s\nGitHash:   %s\nBuildTime: %s\nGoVersion: %s\n",
		Version, GitHash, BuildTime, GoVersion,
	)
}
