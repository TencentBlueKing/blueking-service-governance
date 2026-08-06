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
