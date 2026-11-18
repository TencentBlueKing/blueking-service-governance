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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagConfig.IsAutoGenerateEnabled", func() {
	DescribeTable("detects whether recommended image tag is configured",
		func(cfg TagConfig, want bool) {
			Expect(cfg.IsAutoGenerateEnabled()).To(Equal(want))
		},
		Entry("empty type", TagConfig{}, false),
		Entry("semver", TagConfig{Type: VersionTypeSemver}, true),
		Entry("custom", TagConfig{Type: VersionTypeCustom}, true),
		Entry("unknown type", TagConfig{Type: VersionType("manual")}, false),
	)
})

var _ = Describe("RepositoryConfig image build mode", func() {
	It("returns repository dockerfile mode for empty config", func() {
		cfg := &RepositoryConfig{}

		Expect(cfg.EffectiveImageBuildMode()).To(Equal(ImageBuildModeRepositoryDockerfile))
	})

	It("returns platform mode for platform configs", func() {
		cfg := &RepositoryConfig{
			ImageBuildMode: ImageBuildModePlatform,
			PlatformBuildConfig: &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
				Commands: &BuildCommands{
					Build: []string{"go build -o app ./cmd/server"},
					Start: "./app",
				},
			},
		}

		Expect(cfg.EffectiveImageBuildMode()).To(Equal(ImageBuildModePlatform))
	})

	It("infers platform mode when platform build config exists", func() {
		cfg := &RepositoryConfig{
			PlatformBuildConfig: &PlatformBuildConfig{
				BuilderImage: "golang:1.24",
				RunnerImage:  "debian:12",
			},
		}

		Expect(cfg.EffectiveImageBuildMode()).To(Equal(ImageBuildModePlatform))
	})
})

var _ = Describe("RepositoryConfig.ValidateDockerBuildArgs", func() {
	It("accepts nil config and empty args", func() {
		var nilCfg *RepositoryConfig
		Expect(nilCfg.ValidateDockerBuildArgs()).To(Succeed())
		Expect((&RepositoryConfig{}).ValidateDockerBuildArgs()).To(Succeed())
	})

	It("accepts values with spaces, quotes and dollar signs", func() {
		cfg := &RepositoryConfig{DockerBuildArgs: map[string]string{
			"GREETING": "it's hello world",
			"PRICE":    "$5",
			"EMPTY":    "",
		}}

		Expect(cfg.ValidateDockerBuildArgs()).To(Succeed())
	})

	DescribeTable("rejects args that break line-based passing",
		func(args map[string]string, msg string) {
			err := (&RepositoryConfig{DockerBuildArgs: args}).ValidateDockerBuildArgs()

			Expect(err).To(MatchError(ContainSubstring(msg)))
		},
		Entry("empty key", map[string]string{"": "1"}, `key ""`),
		Entry("key with equal sign", map[string]string{"A=B": "1"}, `key "A=B"`),
		Entry("key with newline", map[string]string{"A\nB": "1"}, `key "A\nB"`),
		// 值里独占一行的结束标记会提前闭合脚本中的 heredoc
		Entry(
			"value closing the heredoc",
			map[string]string{"A": "1\nBKMS_BUILD_ARGS_EOF\ntouch /tmp/pwned"},
			"value of A must not contain line breaks",
		),
		Entry("value with carriage return", map[string]string{"A": "1\r"}, "value of A must not contain line breaks"),
	)
})
