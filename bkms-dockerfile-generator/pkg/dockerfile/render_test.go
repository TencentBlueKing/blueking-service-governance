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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDockerfile(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dockerfile Suite")
}

const testFileMode = 0o600

var _ = Describe("Dockerfile render", func() {
	const (
		goBuilderImage  = "golang:1.25"
		cppBuilderImage = "ubuntu:24.04"
		runnerImage     = "alpine:3.20"
		imageName       = "demo-api"
	)

	defaultInput := func(language string) Input {
		builderImage := goBuilderImage
		if language == LanguageCpp {
			builderImage = cppBuilderImage
		}
		return Input{
			Language:       language,
			BuilderImage:   builderImage,
			RunnerImage:    runnerImage,
			DockerBuildDir: defaultDockerBuildDir(language),
			ImageName:      imageName,
		}
	}

	AfterEach(func() {
		templateFiles = embeddedTemplates
		languageSpecs = map[string]languageSpec{
			LanguageGo: {
				templatePath:        goTemplatePath,
				defaultBuildCommand: defaultGoBuildCommand,
				prepareTemplateData: prepareGoTemplateData,
			},
			LanguageCpp: {
				templatePath:        cppTemplatePath,
				defaultBuildCommand: defaultCppBuildCommand,
			},
		}
	})

	It("renders default Go builder and runner stages", func() {
		content, err := Render(defaultInput(LanguageGo))
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("FROM golang:1.25 AS builder"))
		Expect(content).To(ContainSubstring("WORKDIR /workspace"))
		Expect(content).To(ContainSubstring("RUN mkdir -p /out"))
		Expect(content).To(ContainSubstring("command -v git"))
		Expect(content).To(ContainSubstring("apk add --no-cache ca-certificates git"))
		Expect(content).To(ContainSubstring("apt-get update && apt-get install -y --no-install-recommends ca-certificates git"))
		Expect(content).To(ContainSubstring("yum install -y ca-certificates git"))
		Expect(content).To(ContainSubstring("dnf install -y ca-certificates git"))
		Expect(content).NotTo(ContainSubstring("microdnf"))
		Expect(indexOf(content, "dnf install -y ca-certificates git")).To(BeNumerically(
			"<", indexOf(content, "yum install -y ca-certificates git"),
		))
		Expect(content).To(ContainSubstring("COPY go.mod ./"))
		Expect(content).To(ContainSubstring("COPY go.sum ./"))
		Expect(content).NotTo(ContainSubstring("COPY go.mod go.sum ./"))
		Expect(content).To(ContainSubstring("RUN go mod download"))
		Expect(content).To(ContainSubstring("COPY . ."))
		Expect(content).To(ContainSubstring("RUN CGO_ENABLED=0 go build -trimpath -o /out/demo-api ."))
		Expect(content).To(ContainSubstring(
			"RUN test -f /out/demo-api || (echo \"build artifact /out/demo-api not found\" && exit 1)",
		))
		Expect(content).To(ContainSubstring("FROM alpine:3.20 AS runner"))
		Expect(content).To(ContainSubstring("WORKDIR /app"))
		Expect(content).To(ContainSubstring("COPY --from=builder /out/demo-api /app/demo-api"))
		Expect(content).To(ContainSubstring("ENTRYPOINT [\"/app/demo-api\"]"))

		Expect(indexOf(content, "RUN mkdir -p /out")).To(BeNumerically("<", indexOf(content, "command -v git")))
		Expect(indexOf(content, "command -v git")).To(BeNumerically("<", indexOf(content, "COPY go.mod ./")))
		Expect(indexOf(content, "COPY go.mod ./")).To(BeNumerically("<", indexOf(content, "COPY go.sum ./")))
		Expect(indexOf(content, "COPY go.sum ./")).To(BeNumerically("<", indexOf(content, "RUN go mod download")))
		Expect(indexOf(content, "RUN go mod download")).To(BeNumerically("<", indexOf(content, "COPY . .")))
		Expect(indexOf(content, "COPY . .")).To(BeNumerically("<", indexOf(content, "RUN CGO_ENABLED=0 go build")))
		Expect(indexOf(content, "RUN CGO_ENABLED=0 go build")).To(BeNumerically(
			"<", indexOf(content, "RUN test -f /out/demo-api"),
		))
	})

	It("does not check Go module files for C++ templates", func() {
		input := defaultInput(LanguageCpp)
		input.DockerBuildDir = "missing"

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("FROM ubuntu:24.04 AS builder"))
		Expect(content).NotTo(ContainSubstring("go.mod"))
		Expect(content).NotTo(ContainSubstring("go.sum"))
	})

	DescribeTable("renders package-manager-agnostic git install for Go builder images",
		func(builderImage string) {
			input := defaultInput(LanguageGo)
			input.BuilderImage = builderImage

			content, err := Render(input)
			Expect(err).NotTo(HaveOccurred())

			Expect(content).To(ContainSubstring("FROM " + builderImage + " AS builder"))
			Expect(content).To(ContainSubstring("command -v git"))
			Expect(content).To(ContainSubstring("apk add --no-cache ca-certificates git"))
			Expect(content).To(ContainSubstring("apt-get update && apt-get install -y --no-install-recommends ca-certificates git"))
			Expect(content).To(ContainSubstring("dnf install -y ca-certificates git"))
			Expect(content).To(ContainSubstring("yum install -y ca-certificates git"))
			Expect(content).NotTo(ContainSubstring("microdnf"))
			Expect(indexOf(content, "dnf install -y ca-certificates git")).To(BeNumerically(
				"<", indexOf(content, "yum install -y ca-certificates git"),
			))
		},
		Entry("Debian golang tag", "golang:1.25.3"),
		Entry("Alpine golang tag", "golang:1.25.3-alpine3.22"),
		Entry("Alpine tag with digest", "golang:1.25.3-alpine3.22@sha256:abcd"),
		Entry("namespace contains alpine", "registry.example.com/alpine/golang:1.25"),
		Entry("custom tlinux compile image", "docker.bkrepo.woa.com/sgameai/repo/compile/visual_processor-leonyue:0.0.1"),
	)

	It("renders advanced Go commands at expected positions", func() {
		input := defaultInput(LanguageGo)
		input.PreBuildCommands = encodeCommandsParam([]string{
			"go env -w GOPROXY=https://goproxy.cn,direct",
			"go generate ./...",
		})
		input.BuildCommands = encodeCommandsParam([]string{
			"go build -o /out/demo-api ./cmd/server",
			"chmod +x /out/demo-api",
		})
		input.RuntimeEnvCommands = encodeCommandsParam([]string{
			"apk add --no-cache ca-certificates",
			"mkdir -p /app/config",
		})
		input.StartCommand = "/app/demo-api --config /app/config/config.yaml"

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("RUN go env -w GOPROXY=https://goproxy.cn,direct\n"))
		Expect(content).To(ContainSubstring("RUN go generate ./...\n"))
		Expect(content).To(ContainSubstring("RUN go build -o /out/demo-api ./cmd/server\n"))
		Expect(content).To(ContainSubstring("RUN chmod +x /out/demo-api\n"))
		Expect(content).To(ContainSubstring("RUN apk add --no-cache ca-certificates\n"))
		Expect(content).To(ContainSubstring("RUN mkdir -p /app/config\n"))
		Expect(content).To(ContainSubstring(
			"ENTRYPOINT [\"/bin/sh\", \"-ec\", \"/app/demo-api --config /app/config/config.yaml\"]",
		))
		Expect(content).NotTo(ContainSubstring("RUN CGO_ENABLED=0 go build -trimpath -o /out/demo-api ."))

		Expect(indexOf(content, "RUN go mod download")).To(BeNumerically("<", indexOf(content, "COPY . .")))
		Expect(indexOf(content, "COPY . .")).To(BeNumerically("<", indexOf(content, "RUN go env -w GOPROXY")))
		Expect(indexOf(content, "RUN go generate ./...")).To(BeNumerically(
			"<", indexOf(content, "RUN go build -o /out/demo-api ./cmd/server"),
		))
		Expect(indexOf(content, "RUN chmod +x /out/demo-api")).To(BeNumerically(
			"<", indexOf(content, "RUN test -f /out/demo-api"),
		))
		Expect(indexOf(content, "WORKDIR /app")).To(BeNumerically(
			"<", indexOf(content, "RUN apk add --no-cache ca-certificates\n"),
		))
		Expect(indexOf(content, "RUN mkdir -p /app/config")).To(BeNumerically(
			"<", indexOf(content, "COPY --from=builder /out/demo-api /app/demo-api"),
		))
	})

	It("renders commands from JSON string array", func() {
		input := defaultInput(LanguageGo)
		input.PreBuildCommands = encodeCommandsParam([]string{
			` go env -w GOPROXY="https://goproxy.cn,direct" `,
			`go env -w GOPRIVATE=""`,
			"",
			`go env -w GOSUMDB="sum.goproxy.cn+xxxxxxxx+xxxxxxxxxxxx"`,
		})

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring(`RUN go env -w GOPROXY="https://goproxy.cn,direct"` + "\n"))
		Expect(content).To(ContainSubstring(`RUN go env -w GOPRIVATE=""` + "\n"))
		Expect(content).To(ContainSubstring(`RUN go env -w GOSUMDB="sum.goproxy.cn+xxxxxxxx+xxxxxxxxxxxx"` + "\n"))
		Expect(content).NotTo(ContainSubstring("RUN  go env"))
		Expect(content).NotTo(ContainSubstring("RUN \n"))
		Expect(indexOf(content, `RUN go env -w GOPROXY`)).To(BeNumerically("<", indexOf(content, `RUN go env -w GOPRIVATE`)))
		Expect(indexOf(content, `RUN go env -w GOPRIVATE`)).To(BeNumerically("<", indexOf(content, `RUN go env -w GOSUMDB`)))
	})

	It("renders Docker build arg names as stage arguments", func() {
		input := defaultInput(LanguageGo)
		input.DockerBuildArgNames = `["GOPRIVATE","GOPROXY","GOSUMDB","GOPROXY","invalid-name",""]`

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(strings.Count(content, "ARG GOPRIVATE\n")).To(Equal(1))
		Expect(strings.Count(content, "ARG GOPROXY\n")).To(Equal(1))
		Expect(strings.Count(content, "ARG GOSUMDB\n")).To(Equal(1))
		Expect(content).NotTo(ContainSubstring("invalid-name"))
		Expect(content).NotTo(ContainSubstring("GOSUMDB=off"))

		Expect(indexOf(content, "ARG GOPRIVATE")).To(BeNumerically("<", indexOf(content, "ARG GOPROXY")))
		Expect(indexOf(content, "ARG GOPROXY")).To(BeNumerically("<", indexOf(content, "ARG GOSUMDB")))
		Expect(indexOf(content, "ARG GOSUMDB")).To(BeNumerically("<", indexOf(content, "WORKDIR /workspace")))
	})

	It("returns error when Docker build arg names are invalid JSON", func() {
		input := defaultInput(LanguageGo)
		input.DockerBuildArgNames = `["GOPROXY"`

		content, err := Render(input)

		Expect(content).To(Equal(""))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unmarshal Docker build arg names as JSON string array"))
	})

	It("uses default build command when JSON build commands become empty", func() {
		input := defaultInput(LanguageGo)
		input.BuildCommands = encodeCommandsParam([]string{" ", ""})

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("RUN CGO_ENABLED=0 go build -trimpath -o /out/demo-api ."))
	})

	It("returns error when commands are not a JSON array", func() {
		input := defaultInput(LanguageGo)
		input.PreBuildCommands = "go mod download"

		content, err := Render(input)

		Expect(content).To(Equal(""))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Dockerfile pre-build commands must be JSON string array"))
		Expect(err.Error()).NotTo(ContainSubstring(input.PreBuildCommands))
	})

	It("returns error when commands are invalid JSON", func() {
		input := defaultInput(LanguageGo)
		input.PreBuildCommands = `["token-value"`

		content, err := Render(input)

		Expect(content).To(Equal(""))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unmarshal Dockerfile pre-build commands as JSON string array"))
		Expect(err.Error()).NotTo(ContainSubstring("token-value"))
	})

	It("returns error when commands are JSON object", func() {
		input := defaultInput(LanguageGo)
		input.RuntimeEnvCommands = `{"command":"apk add bash"}`

		content, err := Render(input)

		Expect(content).To(Equal(""))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Dockerfile runtime env commands must be JSON string array"))
		Expect(err.Error()).NotTo(ContainSubstring("apk add bash"))
	})

	It("returns error when command array contains non-string item", func() {
		input := defaultInput(LanguageGo)
		input.BuildCommands = `["go build",123]`

		content, err := Render(input)

		Expect(content).To(Equal(""))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unmarshal Dockerfile build commands as JSON string array"))
		Expect(err.Error()).NotTo(ContainSubstring("go build"))
	})

	It("renders default C++ builder and runner stages", func() {
		content, err := Render(defaultInput(LanguageCpp))
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("FROM ubuntu:24.04 AS builder"))
		Expect(content).To(ContainSubstring("WORKDIR /workspace"))
		Expect(content).To(ContainSubstring("RUN mkdir -p /out"))
		Expect(content).To(ContainSubstring("COPY . ."))
		Expect(content).To(ContainSubstring(
			"RUN cmake -S . -B build -DCMAKE_BUILD_TYPE=Release" +
				" && cmake --build build --target demo-api -j$(nproc)" +
				" && cp build/demo-api /out/demo-api",
		))
		Expect(content).To(ContainSubstring(
			"RUN test -f /out/demo-api || (echo \"build artifact /out/demo-api not found\" && exit 1)",
		))
		Expect(content).To(ContainSubstring("FROM alpine:3.20 AS runner"))
		Expect(content).To(ContainSubstring("COPY --from=builder /out/demo-api /app/demo-api"))
		Expect(content).To(ContainSubstring("ENTRYPOINT [\"/app/demo-api\"]"))

		Expect(indexOf(content, "RUN mkdir -p /out")).To(BeNumerically("<", indexOf(content, "COPY . .")))
		Expect(indexOf(content, "COPY . .")).To(BeNumerically("<", indexOf(content, "RUN cmake -S . -B build")))
		Expect(indexOf(content, "RUN cmake -S . -B build")).To(BeNumerically(
			"<", indexOf(content, "RUN test -f /out/demo-api"),
		))
	})

	It("uses custom C++ build commands instead of the default command", func() {
		input := defaultInput(LanguageCpp)
		input.PreBuildCommands = encodeCommandsParam([]string{
			"apt-get update",
			"apt-get install -y cmake",
		})
		input.BuildCommands = encodeCommandsParam([]string{
			"bazel build //cmd:demo-api",
			"cp bazel-bin/cmd/demo-api /out/demo-api",
		})
		input.RuntimeEnvCommands = encodeCommandsParam([]string{"apk add --no-cache libstdc++"})
		input.StartCommand = "/app/demo-api --config /app/config.yaml"

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("RUN apt-get update\n"))
		Expect(content).To(ContainSubstring("RUN apt-get install -y cmake\n"))
		Expect(content).To(ContainSubstring("RUN bazel build //cmd:demo-api\n"))
		Expect(content).To(ContainSubstring("RUN cp bazel-bin/cmd/demo-api /out/demo-api\n"))
		Expect(content).To(ContainSubstring("RUN apk add --no-cache libstdc++\n"))
		Expect(content).To(ContainSubstring(
			"ENTRYPOINT [\"/bin/sh\", \"-ec\", \"/app/demo-api --config /app/config.yaml\"]",
		))
		Expect(content).NotTo(ContainSubstring("RUN cmake -S . -B build"))
	})

	It("does not inject extra BKMS platform variables", func() {
		content, err := Render(defaultInput(LanguageGo))
		Expect(err).NotTo(HaveOccurred())

		Expect(content).NotTo(ContainSubstring("BKMS_GOPROXY"))
		Expect(content).NotTo(ContainSubstring("BKMS_GOSUMDB"))
		Expect(content).NotTo(ContainSubstring("BKMS_APP_DIR"))
		Expect(content).NotTo(ContainSubstring("BKMS_BIN_DIR"))
	})

	It("uses trimmed image name as app name for generated artifact paths", func() {
		input := defaultInput(LanguageGo)
		input.ImageName = " demo.api_1-2 "

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("RUN CGO_ENABLED=0 go build -trimpath -o /out/demo.api_1-2 ."))
		Expect(content).To(ContainSubstring("RUN test -f /out/demo.api_1-2"))
		Expect(content).To(ContainSubstring("COPY --from=builder /out/demo.api_1-2 /app/demo.api_1-2"))
		Expect(content).To(ContainSubstring("ENTRYPOINT [\"/app/demo.api_1-2\"]"))
		Expect(content).NotTo(ContainSubstring("/out/ demo.api_1-2 "))
	})

	It("returns error when image name is unsafe", func() {
		unsafeImageNames := []string{"demo/api", "demo:api", "demo api", ".", ".."}
		for _, unsafeImageName := range unsafeImageNames {
			input := defaultInput(LanguageGo)
			input.ImageName = unsafeImageName

			_, err := Render(input)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid image name"))
		}
	})

	It("returns error when language is empty", func() {
		_, err := Render(defaultInput(""))

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing Dockerfile language"))
	})

	It("returns error when language is unsupported", func() {
		_, err := Render(defaultInput("java"))

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unsupported Dockerfile language \"java\""))
	})

	It("returns error when template file is missing", func() {
		templateFiles = fstest.MapFS{}

		_, err := Render(defaultInput(LanguageGo))

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("read Dockerfile template for language \"go\""))
		Expect(err.Error()).To(ContainSubstring(goTemplatePath))
	})

	It("returns error when template parsing fails", func() {
		templateFiles = fstest.MapFS{
			goTemplatePath: &fstest.MapFile{Data: []byte("{{ if }}")},
		}

		_, err := Render(defaultInput(LanguageGo))

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("parse Dockerfile template for language \"go\""))
	})
})

func indexOf(content string, fragment string) int {
	index := strings.Index(content, fragment)
	ExpectWithOffset(1, index).NotTo(Equal(-1), "fragment %q should exist", fragment)
	return index
}

func encodeCommandsParam(commands []string) string {
	data, err := json.Marshal(commands)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return string(data)
}

func defaultDockerBuildDir(language string) string {
	if language != LanguageGo {
		return ""
	}
	return createGoModuleDir(goModFile, goSumFile)
}

func createGoModuleDir(files ...string) string {
	directory := GinkgoT().TempDir()
	for _, file := range files {
		ExpectWithOffset(1, os.WriteFile(filepath.Join(directory, file), []byte("module demo\n"), testFileMode)).To(Succeed())
	}
	return directory
}
