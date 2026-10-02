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

package version

import (
	"fmt"
	"runtime"
	"runtime/debug"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserAgent", func() {
	var originalVersion string

	BeforeEach(func() {
		originalVersion = Version
		DeferCleanup(func() { Version = originalVersion })
	})

	DescribeTable("assembles product/version (os/arch)",
		func(version, wantVersion string) {
			Version = version
			Expect(
				UserAgent(),
			).To(Equal(fmt.Sprintf("bkms-cli/%s (%s/%s)", wantVersion, runtime.GOOS, runtime.GOARCH)))
		},
		Entry("release version", "1.2.3", "1.2.3"),
		Entry("strips leading v", "v1.2.3", "1.2.3"),
		Entry("empty version uses dev", "", "dev"),
	)
})

var _ = Describe("Build metadata", func() {
	BeforeEach(func() {
		oldVersion, oldHash, oldTime := Version, GitHash, BuildTime
		DeferCleanup(func() {
			Version, GitHash, BuildTime = oldVersion, oldHash, oldTime
		})
		Version, GitHash, BuildTime = "", "", ""
	})

	It("uses the Go module version when build flags are absent", func() {
		resolveBuildInfo(&debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}})
		Expect(Version).To(Equal("1.2.3"))
		Expect(GitHash).To(Equal("unknown"))
		Expect(BuildTime).To(Equal("unknown"))
	})

	It("preserves all explicitly injected release metadata", func() {
		Version, GitHash, BuildTime = "2.0.0", "release-hash", "build-time"
		resolveBuildInfo(&debug.BuildInfo{
			Main:     debug.Module{Version: "v1.2.3"},
			Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "source-hash"}},
		})
		Expect(Version).To(Equal("2.0.0"))
		Expect(GitHash).To(Equal("release-hash"))
		Expect(BuildTime).To(Equal("build-time"))
	})

	It("uses local VCS revision without mistaking commit time for build time", func() {
		resolveBuildInfo(&debug.BuildInfo{
			Main: debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "source-hash"},
				{Key: "vcs.time", Value: "2026-09-10T00:00:00Z"},
			},
		})
		Expect(Version).To(Equal("dev"))
		Expect(GitHash).To(Equal("source-hash"))
		Expect(BuildTime).To(Equal("unknown"))
	})
})
