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
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Python Dockerfile render", func() {
	const (
		runnerImage = "python:3.12-slim"
		imageName   = "demo-py"
		// maxRenderDuration 单次生成的耗时上限，对应性能验收要求
		maxRenderDuration = time.Second
	)

	requirePython3Run := "RUN " + requirePython3Command + "\n"
	uvEnsureCommand := "RUN " + fmt.Sprintf(ensurePythonToolCommandTemplate, "uv") + "\n"
	poetryEnsureCommand := "RUN " + fmt.Sprintf(ensurePythonToolCommandTemplate, "poetry") + "\n"
	uvInstallRun := "RUN " + uvInstallCommand + "\n"
	poetryInstallRun := "RUN " + poetryInstallCommand + "\n"
	pipInstallRun := "RUN " + pipInstallCommand + "\n"
	requirePipRun := "RUN " + requirePipCommand + "\n"

	makePythonInput := func(files ...string) Input {
		return Input{
			Language:       LanguagePython,
			RunnerImage:    runnerImage,
			DockerBuildDir: createPythonProjectDir(files...),
			ImageName:      imageName,
		}
	}

	It("renders a single runner stage without builder image", func() {
		content, err := Render(makePythonInput())
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("FROM python:3.12-slim AS runner\n"))
		Expect(content).NotTo(ContainSubstring("AS builder"))
		Expect(content).NotTo(ContainSubstring("--from=builder"))
		Expect(content).To(ContainSubstring("ENV BKMS_ARTIFACT_DIR=/app\n"))
		Expect(content).To(ContainSubstring("WORKDIR /app\n"))
		Expect(content).To(ContainSubstring("COPY . .\n"))
		Expect(content).To(ContainSubstring(artifactDirCheck("/app")))
		// 三种锁文件都没有：只拷源码，不安装依赖
		Expect(content).NotTo(ContainSubstring("pip install"))
		Expect(content).NotTo(ContainSubstring("poetry install"))
		Expect(content).NotTo(ContainSubstring("uv pip install"))
		// 没有依赖要装时不要求运行镜像带 python3
		Expect(content).NotTo(ContainSubstring(requirePython3Run))
		// 不填启动命令时不写入口，与 go / cpp 的默认入口刻意不同
		Expect(content).NotTo(ContainSubstring("ENTRYPOINT"))

		Expect(indexOf(content, "ENV BKMS_ARTIFACT_DIR=/app")).To(BeNumerically("<", indexOf(content, "WORKDIR /app")))
		Expect(indexOf(content, "WORKDIR /app")).To(BeNumerically("<", indexOf(content, "COPY . .")))
	})

	DescribeTable("selects one installer by lock file priority",
		func(files []string, expected []string, absent []string) {
			content, err := Render(makePythonInput(files...))
			Expect(err).NotTo(HaveOccurred())

			for _, fragment := range expected {
				Expect(content).To(ContainSubstring(fragment))
			}
			for _, fragment := range absent {
				Expect(content).NotTo(ContainSubstring(fragment))
			}
			// 先确认 python3，再按顺序补工具、装依赖
			Expect(indexOf(content, requirePython3Run)).To(BeNumerically("<", indexOf(content, expected[0])))
			for i := 1; i < len(expected); i++ {
				Expect(indexOf(content, expected[i-1])).To(BeNumerically("<", indexOf(content, expected[i])))
			}
		},
		Entry("uv wins over poetry and pip",
			[]string{uvLockFile, poetryLockFile, requirementsFile},
			[]string{uvEnsureCommand, uvInstallRun},
			[]string{poetryEnsureCommand, poetryInstallRun, requirePipRun, pipInstallRun},
		),
		Entry("poetry wins over pip",
			[]string{poetryLockFile, requirementsFile},
			[]string{poetryEnsureCommand, poetryInstallRun},
			[]string{uvEnsureCommand, uvInstallRun, requirePipRun, pipInstallRun},
		),
		Entry("pip only",
			[]string{requirementsFile},
			[]string{requirePipRun, pipInstallRun},
			[]string{uvEnsureCommand, poetryEnsureCommand},
		),
	)

	It("only runs user build commands and skips dependency detection", func() {
		input := makePythonInput(uvLockFile, requirementsFile)
		input.BuildCommands = encodeCommandsParam([]string{"pip install -e ."})

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("RUN pip install -e .\n"))
		Expect(content).NotTo(ContainSubstring(uvEnsureCommand))
		Expect(content).NotTo(ContainSubstring(pipInstallRun))
	})

	It("does not inspect build directory when user build commands are set", func() {
		input := makePythonInput()
		input.DockerBuildDir = "missing"
		input.BuildCommands = encodeCommandsParam([]string{"pip install -e ."})

		_, err := Render(input)

		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error when build directory is missing and dependencies need detecting", func() {
		input := makePythonInput()
		input.DockerBuildDir = ""

		_, err := Render(input)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("missing Docker build directory"))
	})

	It("fails with a missing python3 hint before any installer runs", func() {
		content, err := Render(makePythonInput(requirementsFile))
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring(
			"echo \"python3 is required to install Python dependencies, but it is not available in the runner image\"",
		))
		Expect(indexOf(content, requirePython3Run)).To(BeNumerically("<", indexOf(content, pipInstallRun)))
	})

	It("does not add a python3 check when user build commands are set", func() {
		input := makePythonInput(requirementsFile)
		input.BuildCommands = encodeCommandsParam([]string{"pip install -e ."})

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).NotTo(ContainSubstring(requirePython3Run))
	})

	It("fails with a missing pip hint before installing requirements.txt", func() {
		content, err := Render(makePythonInput(requirementsFile))
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring(
			"echo \"pip is required to install requirements.txt, but python3 -m pip is not available in the runner image\"",
		))
		Expect(indexOf(content, requirePython3Run)).To(BeNumerically("<", indexOf(content, requirePipRun)))
	})

	It("excludes every default dependency group and disables download caches", func() {
		uvContent, err := Render(makePythonInput(uvLockFile))
		Expect(err).NotTo(HaveOccurred())
		// --no-dev 只排除 dev 组，[tool.uv] default-groups 里的其他分组仍会被装进镜像
		Expect(uvContent).To(ContainSubstring("uv export --frozen --no-default-groups --no-emit-project"))
		Expect(uvContent).NotTo(ContainSubstring("--no-dev"))
		Expect(uvContent).To(ContainSubstring("uv pip install --system --no-cache -r"))

		poetryContent, err := Render(makePythonInput(poetryLockFile))
		Expect(err).NotTo(HaveOccurred())
		Expect(poetryContent).To(ContainSubstring("poetry install --no-cache"))
	})

	It("fails with a missing pip hint when the tool cannot be installed", func() {
		content, err := Render(makePythonInput(uvLockFile))
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("if command -v uv >/dev/null 2>&1"))
		Expect(content).To(ContainSubstring("python3 -m pip install --no-cache-dir uv"))
		Expect(content).To(ContainSubstring(
			"echo \"pip is required to install uv, but python3 -m pip is not available in the runner image\" && exit 1",
		))
	})

	It("uses custom artifact dir as working directory and shell entrypoint", func() {
		input := makePythonInput(requirementsFile)
		input.ArtifactDir = "/srv/app/"
		input.StartCommand = "python main.py"

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).To(ContainSubstring("ENV BKMS_ARTIFACT_DIR=/srv/app\n"))
		Expect(content).To(ContainSubstring("WORKDIR /srv/app\n"))
		Expect(content).To(ContainSubstring(artifactDirCheck("/srv/app")))
		Expect(content).To(ContainSubstring("ENTRYPOINT [\"/bin/sh\", \"-ec\", \"python main.py\"]\n"))
		Expect(content).NotTo(ContainSubstring("/app/" + imageName))
	})

	It("runs commands in order and checks artifact dir after runtime env commands", func() {
		input := makePythonInput(requirementsFile)
		input.PreBuildCommands = encodeCommandsParam([]string{"pip config set global.index-url https://mirror.example.com"})
		input.RuntimeEnvCommands = encodeCommandsParam([]string{"python -m compileall ."})

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(indexOf(content, "COPY . .")).To(BeNumerically("<", indexOf(content, "RUN pip config set")))
		Expect(indexOf(content, "RUN pip config set")).To(BeNumerically("<", indexOf(content, pipInstallRun)))
		Expect(indexOf(content, pipInstallRun)).To(BeNumerically("<", indexOf(content, "RUN python -m compileall .")))
		Expect(indexOf(content, "RUN python -m compileall .")).To(BeNumerically(
			"<", indexOf(content, artifactDirCheck("/app")),
		))
	})

	It("does not copy extra files separately", func() {
		input := makePythonInput()
		input.ExtraFiles = encodeCommandsParam([]string{"data/key.pem", "*.json"})

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(content).NotTo(ContainSubstring("COPY data/key.pem"))
		Expect(content).NotTo(ContainSubstring("COPY *.json"))
	})

	It("declares build args in the single stage", func() {
		input := makePythonInput()
		input.DockerBuildArgNames = `["PIP_INDEX_URL"]`

		content, err := Render(input)
		Expect(err).NotTo(HaveOccurred())

		Expect(indexOf(content, "FROM python:3.12-slim AS runner")).To(BeNumerically(
			"<", indexOf(content, "ARG PIP_INDEX_URL"),
		))
	})

	It("renders within the time limit when all lock files exist", func() {
		input := makePythonInput(uvLockFile, poetryLockFile, requirementsFile)

		start := time.Now()
		_, err := Render(input)
		elapsed := time.Since(start)

		Expect(err).NotTo(HaveOccurred())
		Expect(elapsed).To(BeNumerically("<", maxRenderDuration))
	})

	It("reports builder image requirement per language", func() {
		Expect(RequiresBuilderImage(LanguagePython)).To(BeFalse())
		Expect(RequiresBuilderImage(LanguageGo)).To(BeTrue())
		Expect(RequiresBuilderImage(LanguageCpp)).To(BeTrue())
		// 未注册的语言仍要求构建镜像，交给 Render 报 unsupported language
		Expect(RequiresBuilderImage("java")).To(BeTrue())
	})
})

// createPythonProjectDir 在临时目录里创建指定的依赖文件，模拟 python 项目的构建目录
func createPythonProjectDir(files ...string) string {
	directory := GinkgoT().TempDir()
	for _, file := range files {
		ExpectWithOffset(1, os.WriteFile(filepath.Join(directory, file), []byte("# demo\n"), testFileMode)).To(Succeed())
	}
	return directory
}
