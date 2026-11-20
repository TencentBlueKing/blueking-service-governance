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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	// envDockerBuildTest 设为 1 才跑真实 docker build：需要本机 docker 与访问 Python 包仓库的网络，默认跳过
	envDockerBuildTest = "BKMS_DOCKERFILE_DOCKER_TEST"
	// envDockerBuildTestRunnerImage 覆盖运行镜像，拉不到 Docker Hub 时换成可访问的镜像地址
	envDockerBuildTestRunnerImage     = "BKMS_DOCKERFILE_DOCKER_TEST_RUNNER_IMAGE"
	defaultDockerBuildTestRunnerImage = "python:3.12-slim"

	// dockerBuildTestTimeout 单次 docker build / run 的上限，首次拉镜像、装工具都算在内
	dockerBuildTestTimeout = 10 * time.Minute
	dockerBuildTestImage   = "bkms-dockerfile-generator-test"
	pythonFixturesDir      = "testdata/python"
)

// pythonIndexBuildArgNames 本机设置了才透传给 docker build，内网环境靠它们走包仓库镜像源
var pythonIndexBuildArgNames = []string{"PIP_INDEX_URL", "UV_DEFAULT_INDEX"}

var _ = Describe("Python Dockerfile docker build", Label("docker"), func() {
	var runnerImage string

	BeforeEach(func() {
		if os.Getenv(envDockerBuildTest) != "1" {
			Skip(fmt.Sprintf("set %s=1 to run docker build tests", envDockerBuildTest))
		}

		_, err := exec.LookPath("docker")
		Expect(err).NotTo(HaveOccurred(), "docker CLI is required")

		runnerImage = os.Getenv(envDockerBuildTestRunnerImage)
		if runnerImage == "" {
			runnerImage = defaultDockerBuildTestRunnerImage
		}
	})

	DescribeTable("installs locked dependencies into the system Python",
		func(fixture string, preBuildCommands []string, excludedModule string) {
			// 拷一份夹具当构建目录，Dockerfile 写在目录外，不随 COPY . . 进镜像
			buildDir := filepath.Join(GinkgoT().TempDir(), fixture)
			Expect(os.CopyFS(buildDir, os.DirFS(filepath.Join(pythonFixturesDir, fixture)))).To(Succeed())

			argNames := collectPresentEnvNames(pythonIndexBuildArgNames)
			content, err := Render(Input{
				Language:            LanguagePython,
				RunnerImage:         runnerImage,
				PreBuildCommands:    encodeJSONStrings(preBuildCommands),
				DockerBuildArgNames: encodeJSONStrings(argNames),
				DockerBuildDir:      buildDir,
				ImageName:           fixture,
			})
			Expect(err).NotTo(HaveOccurred())

			dockerfilePath := filepath.Join(GinkgoT().TempDir(), "Dockerfile")
			Expect(os.WriteFile(dockerfilePath, []byte(content), testFileMode)).To(Succeed())

			// 构建参数只传名字，值由 docker 从当前环境读取，不在命令行里展开
			tag := fmt.Sprintf("%s:%s-%d", dockerBuildTestImage, fixture, GinkgoRandomSeed())
			buildArgs := []string{"build", "--file", dockerfilePath, "--tag", tag}
			for _, name := range argNames {
				buildArgs = append(buildArgs, "--build-arg", name)
			}
			Expect(runDocker(append(buildArgs, buildDir)...)).Error().NotTo(HaveOccurred())
			DeferCleanup(func() {
				_, _ = runDocker("image", "rm", "--force", tag)
			})

			// 不写启动命令时没有 ENTRYPOINT，直接用系统 python3 导入，验证依赖没有落进虚拟环境
			Expect(runDocker("run", "--rm", "--entrypoint", "python3", tag, "-c", "import six")).
				Error().NotTo(HaveOccurred())

			// dev 组依赖不应进镜像
			if excludedModule != "" {
				output, runErr := runDocker("run", "--rm", "--entrypoint", "python3", tag, "-c", "import "+excludedModule)
				Expect(runErr).To(HaveOccurred())
				Expect(output).To(ContainSubstring("ModuleNotFoundError"))
			}
		},
		Entry("requirements.txt", "requirements", nil, ""),
		Entry("uv.lock without dev group", "uv", nil, "iniconfig"),
		// 前置命令先建好 .venv，poetry 仍要装进系统 Python
		Entry("poetry.lock with an existing in-project virtualenv",
			"poetry", []string{"python3 -m venv .venv"}, "iniconfig"),
	)
})

// encodeJSONStrings 按生成器的入参格式编码为 JSON 字符串数组，空列表编码为空串，与流水线未填时一致
func encodeJSONStrings(values []string) string {
	if len(values) == 0 {
		return ""
	}

	raw, err := json.Marshal(values)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return string(raw)
}

// collectPresentEnvNames 返回当前环境里已设置且非空的变量名，保持入参顺序
func collectPresentEnvNames(names []string) []string {
	present := make([]string, 0, len(names))
	for _, name := range names {
		if os.Getenv(name) != "" {
			present = append(present, name)
		}
	}
	return present
}

// runDocker 执行 docker 命令并返回合并后的输出，同时写进 GinkgoWriter，失败时便于看构建日志
func runDocker(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dockerBuildTestTimeout)
	defer cancel()

	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	GinkgoWriter.Printf("$ docker %s\n%s\n", strings.Join(args, " "), output)
	return string(output), err
}
