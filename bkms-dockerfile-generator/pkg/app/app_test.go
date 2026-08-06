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

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-dockerfile-generator/pkg/config"
)

const testImageName = "demo-api"

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Suite")
}

// 覆盖 Run 的核心分支：正常生成、跳过 repository 类型、必填变量缺失、
// 非法 SourceType、父目录不可创建等错误路径
var _ = Describe("App run", func() {
	var workDir string

	BeforeEach(func() {
		var err error
		workDir, err = os.MkdirTemp("", "bkms-dockerfile-generator-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(workDir)).To(Succeed())
	})

	It("writes generated Dockerfile to dot bkms path", func() {
		dockerfilePath := filepath.Join(workDir, ".bkms", "Dockerfile.generated")
		var out bytes.Buffer

		err := Run(validEnviron(dockerfilePath), &out)
		Expect(err).NotTo(HaveOccurred())

		content, err := os.ReadFile(dockerfilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("FROM golang:1.25 AS builder"))
		Expect(string(content)).To(ContainSubstring("ENTRYPOINT [\"/app/demo-api\"]"))
		Expect(out.String()).To(ContainSubstring("generated Dockerfile: " + dockerfilePath))

		dirInfo, err := os.Stat(filepath.Dir(dockerfilePath))
		Expect(err).NotTo(HaveOccurred())
		Expect(dirInfo.Mode().Perm()).To(Equal(dockerfileDirMode))
	})

	It("writes generated Dockerfile to nested source path", func() {
		dockerfilePath := filepath.Join(workDir, "src", ".bkms", "Dockerfile.generated")
		var out bytes.Buffer

		err := Run(validEnviron(dockerfilePath), &out)
		Expect(err).NotTo(HaveOccurred())

		content, err := os.ReadFile(dockerfilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("COPY --from=builder /out/demo-api /app/demo-api"))
	})

	It("writes generated Dockerfile when go.sum is missing", func() {
		dockerfilePath := filepath.Join(workDir, ".bkms", "Dockerfile.generated")
		environ := validEnvironWithGoModuleFiles(dockerfilePath, "go.mod")
		var out bytes.Buffer

		err := Run(environ, &out)
		Expect(err).NotTo(HaveOccurred())

		content, err := os.ReadFile(dockerfilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("COPY go.mod ./"))
		Expect(string(content)).NotTo(ContainSubstring("COPY go.sum ./"))
		Expect(string(content)).NotTo(ContainSubstring("COPY go.mod go.sum ./"))
		Expect(string(content)).To(ContainSubstring("RUN go mod download"))
	})

	It("writes Docker build args to generated Dockerfile", func() {
		dockerfilePath := filepath.Join(workDir, ".bkms", "Dockerfile.generated")
		environ := append(validEnviron(dockerfilePath),
			config.EnvDockerBuildArgNames+`=["GOPRIVATE","GOPROXY","GOSUMDB"]`,
		)
		var out bytes.Buffer

		err := Run(environ, &out)
		Expect(err).NotTo(HaveOccurred())

		content, err := os.ReadFile(dockerfilePath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(ContainSubstring("ARG GOPRIVATE\nARG GOPROXY\nARG GOSUMDB\nWORKDIR /workspace"))
		Expect(string(content)).NotTo(ContainSubstring("ARG GOPRIVATE\nARG GOPROXY\nARG GOSUMDB\nWORKDIR /app"))
		Expect(string(content)).NotTo(ContainSubstring("https://goproxy.cn"))

		Expect(string(content)).NotTo(ContainSubstring("GOSUMDB=off"))
	})

	It("skips repository source type without writing Dockerfile", func() {
		dockerfilePath := filepath.Join(workDir, "Dockerfile")
		var out bytes.Buffer

		err := Run([]string{config.EnvDockerfileSourceType + "=" + config.SourceTypeRepository}, &out)
		Expect(err).NotTo(HaveOccurred())
		Expect(out.String()).To(ContainSubstring("skip Dockerfile generation"))
		Expect(dockerfilePath).NotTo(BeAnExistingFile())
	})

	It("returns readable error for missing required variable", func() {
		var out bytes.Buffer

		err := Run([]string{config.EnvDockerfileSourceType + "=" + config.SourceTypeBKMSGenerated}, &out)
		Expect(err).To(HaveOccurred())
		// validateBKMSGeneratedConfig 内部使用 map 遍历必填项，具体先命中哪个变量取决于 map 顺序，
		// 因此这里只断言通用的错误前缀，避免测试偶发失败
		Expect(err.Error()).To(ContainSubstring("missing required environment variable"))
	})

	It("returns readable error for unsupported source type", func() {
		var out bytes.Buffer

		err := Run([]string{config.EnvDockerfileSourceType + "=unknown"}, &out)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(config.EnvDockerfileSourceType))
		Expect(err.Error()).To(ContainSubstring("unknown"))
	})

	It("returns readable error when image name is unsafe", func() {
		dockerfilePath := filepath.Join(workDir, ".bkms", "Dockerfile.generated")
		environ := validEnviron(dockerfilePath)
		environ[len(environ)-1] = config.EnvImageName + "=demo/api"
		var out bytes.Buffer

		err := Run(environ, &out)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid image name"))
		Expect(dockerfilePath).NotTo(BeAnExistingFile())
	})

	It("returns readable error when target path parent is a file", func() {
		parentFile := filepath.Join(workDir, "src")
		Expect(os.WriteFile(parentFile, []byte("not directory"), dockerfileFileMode)).To(Succeed())
		var out bytes.Buffer

		err := Run(validEnviron(filepath.Join(parentFile, "Dockerfile.generated")), &out)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("create Dockerfile directory"))
		Expect(err.Error()).To(ContainSubstring(parentFile))
	})
})

// validEnviron 构造一份最小可用、能通过 config 校验的流水线环境变量集合
// 用于跑通"正常生成"分支，其它错误路径的用例会手工组合各自的环境变量
func validEnviron(dockerfilePath string) []string {
	return validEnvironWithGoModuleFiles(dockerfilePath, "go.mod", "go.sum")
}

func validEnvironWithGoModuleFiles(dockerfilePath string, files ...string) []string {
	return []string{
		config.EnvDockerfileSourceType + "=" + config.SourceTypeBKMSGenerated,
		config.EnvDockerfileLanguage + "=go",
		config.EnvDockerfilePath + "=" + dockerfilePath,
		config.EnvDockerBuildDir + "=" + createGoModuleDir(files...),
		config.EnvDockerfileBuilderImage + "=golang:1.25",
		config.EnvDockerfileRunnerImage + "=alpine:3.20",
		config.EnvImageName + "=" + testImageName,
	}
}

func createGoModuleDir(files ...string) string {
	directory := GinkgoT().TempDir()
	for _, file := range files {
		ExpectWithOffset(1, os.WriteFile(filepath.Join(directory, file), []byte("module demo\n"), dockerfileFileMode)).To(Succeed())
	}
	return directory
}
