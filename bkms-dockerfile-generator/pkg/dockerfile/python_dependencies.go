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
	"slices"

	"github.com/pkg/errors"
)

const (
	uvLockFile       = "uv.lock"
	poetryLockFile   = "poetry.lock"
	requirementsFile = "requirements.txt"

	// uvExportedRequirementsPath uv 锁文件导出的临时 requirements 路径，安装后删除，不留在镜像里
	uvExportedRequirementsPath = "/tmp/bkms-uv-requirements.txt"
)

// requirePython3Command 自动安装依赖前先确认运行镜像自带 python3
//
// python3 与 pip 属于运行镜像提供的语言环境，平台不用 apk / apt 等系统包管理器补装：
// 各发行版的包名、版本与 PEP 668 限制都不同，补装出来的解释器也不保证能跑用户代码。
// 缺失时尽早失败并说清楚缺什么，避免后续命令报出 pip 缺失之类的误导信息
const requirePython3Command = "command -v python3 >/dev/null 2>&1 || " +
	"(echo \"python3 is required to install Python dependencies, but it is not available in the runner image\" && exit 1)"

// ensurePythonToolCommandTemplate 运行镜像缺少 uv / poetry 时先用 pip 安装
//
// 统一用 python3 -m pip：很多镜像只有 pip3 或根本没有 pip 这个命令名；
// 连 pip 模块都没有时直接失败并说明缺少 pip，同样不用系统包管理器补装。
// 不锁定工具版本：pip 会按运行镜像的 Python 版本挑声明兼容的最新版；
// 需要特定版本时，用户在编译前置命令里先装好，这里检测到已存在就跳过
const ensurePythonToolCommandTemplate = "if command -v %[1]s >/dev/null 2>&1; then " +
	"echo \"%[1]s already present\"; " +
	"elif python3 -m pip --version >/dev/null 2>&1; then " +
	"python3 -m pip install --no-cache-dir %[1]s; " +
	"else " +
	"echo \"pip is required to install %[1]s, but python3 -m pip is not available in the runner image\" && exit 1; " +
	"fi"

// requirePipCommand requirements.txt 直接用 pip 安装，先确认 pip 可用
//
// 只放在 pip 路径上：uv / poetry 已在镜像里时不需要 pip，统一前置检查会误伤这类镜像
const requirePipCommand = "python3 -m pip --version >/dev/null 2>&1 || " +
	"(echo \"pip is required to install " + requirementsFile +
	", but python3 -m pip is not available in the runner image\" && exit 1)"

// 三种安装方式都装进运行镜像的系统 Python，启动命令直接用 python 即可，不需要额外激活虚拟环境。
// 单阶段镜像没有 builder 层隔离，安装都关掉下载缓存，避免缓存随依赖规模留进最终镜像
const (
	// uv 先把锁文件导出为 requirements 再装进系统 Python；
	// 不用 uv sync：它默认只往项目 .venv 装，切到系统前缀时又会卸掉锁文件以外的包（包括 pip 本身）。
	// --no-default-groups 排除 [tool.uv] default-groups 里的全部分组，--no-dev 只排除 dev 组；
	// 导出格式不显式指定，默认就是 requirements.txt
	uvInstallCommand = "uv export --frozen --no-default-groups --no-emit-project" +
		" --output-file " + uvExportedRequirementsPath +
		" && uv pip install --system --no-cache -r " + uvExportedRequirementsPath +
		" && rm -f " + uvExportedRequirementsPath
	// --no-root：没有打包配置的项目安装自身会失败，平台只负责装依赖
	poetryInstallCommand = "POETRY_VIRTUALENVS_CREATE=false poetry install --no-cache --no-interaction --no-ansi" +
		" --no-root --only main"
	pipInstallCommand = "python3 -m pip install --no-cache-dir -r " + requirementsFile
)

type pythonInstaller struct {
	lockFile string
	commands []string
}

// pythonInstallers 按优先级排列：uv.lock > poetry.lock > requirements.txt，只取第一个命中的
var pythonInstallers = []pythonInstaller{
	{
		lockFile: uvLockFile,
		commands: []string{fmt.Sprintf(ensurePythonToolCommandTemplate, "uv"), uvInstallCommand},
	},
	{
		lockFile: poetryLockFile,
		commands: []string{fmt.Sprintf(ensurePythonToolCommandTemplate, "poetry"), poetryInstallCommand},
	},
	{
		lockFile: requirementsFile,
		commands: []string{requirePipCommand, pipInstallCommand},
	},
}

// defaultPythonBuildCommands 未指定构建命令时，按构建目录根下的锁文件生成依赖安装命令
//
// 只看构建目录根上的普通文件，不递归；三种文件都没有时返回空列表，镜像里只有源码。
// 工具是否已装在运行镜像里要到 docker build 时才知道，所以检测写进 RUN，不在生成期判断
func defaultPythonBuildCommands(input Input, _ string, _ string) ([]string, error) {
	directory, err := resolveDockerBuildDir(input.DockerBuildDir)
	if err != nil {
		return nil, errors.Wrap(err, "resolve Docker build directory for Python dependency files")
	}

	for _, installer := range pythonInstallers {
		exists, checkErr := regularFileExists(directory, installer.lockFile)
		if checkErr != nil {
			return nil, errors.Wrap(checkErr, "detect Python dependency file")
		}
		// 三种安装方式都依赖 python3，命中后先插入 python3 检查
		if exists {
			return slices.Concat([]string{requirePython3Command}, installer.commands), nil
		}
	}

	// 没有依赖要装时不检查 python3：镜像里能否启动由启动命令自己决定
	return nil, nil
}
