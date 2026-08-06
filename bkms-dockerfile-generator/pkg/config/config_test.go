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

package config

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config load", func() {
	It("loads generated config with safe image name", func() {
		cfg, err := LoadFromEnviron(validGeneratedEnviron(" demo.api_1-2 "))

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.SourceType).To(Equal(SourceTypeBKMSGenerated))
		Expect(cfg.DockerBuildDir).To(Equal("/workspace/source"))
		Expect(cfg.DockerBuildArgNames).To(Equal(`["GOPROXY","GOSUMDB"]`))
		Expect(cfg.ImageName).To(Equal("demo.api_1-2"))
	})

	It("returns error when generated config image name is unsafe", func() {
		unsafeImageNames := []string{"demo/api", "demo:api", "demo api", ".", ".."}
		for _, unsafeImageName := range unsafeImageNames {
			_, err := LoadFromEnviron(validGeneratedEnviron(unsafeImageName))

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid image name"))
		}
	})

	It("does not validate image name for repository source type", func() {
		cfg, err := LoadFromEnviron([]string{
			EnvDockerfileSourceType + "=" + SourceTypeRepository,
			EnvImageName + "=demo/api",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.SourceType).To(Equal(SourceTypeRepository))
		Expect(cfg.ImageName).To(Equal("demo/api"))
	})
})

func validGeneratedEnviron(imageName string) []string {
	return []string{
		EnvDockerfileSourceType + "=" + SourceTypeBKMSGenerated,
		EnvDockerfileLanguage + "=go",
		EnvDockerfilePath + "=.bkms/Dockerfile.generated",
		EnvDockerBuildDir + "=/workspace/source",
		EnvDockerfileBuilderImage + "=golang:1.25",
		EnvDockerfileRunnerImage + "=alpine:3.20",
		EnvDockerBuildArgNames + `=["GOPROXY","GOSUMDB"]`,
		EnvImageName + "=" + imageName,
	}
}
