## bkms-dockerfile-generator

`bkms-dockerfile-generator` 是 BKMS 镜像构建流程使用的 Dockerfile 生成工具。它从流水线注入的环境变量中读取构建配置，根据应用语言渲染平台默认
Dockerfile，并将生成结果写入 `BKMS_DOCKERFILE_PATH` 指定的位置。

### 适用场景

- **平台生成 Dockerfile**：当 `BKMS_DOCKERFILE_SOURCE_TYPE=bkms_generated` 时，工具会根据 `BKMS_DOCKERFILE_LANGUAGE`、构建镜像、运行镜像和构建命令等配置生成 Dockerfile。
- **仓库自带 Dockerfile**：当 `BKMS_DOCKERFILE_SOURCE_TYPE=repository` 时，工具会跳过生成逻辑，由后续镜像构建流程使用仓库中的 Dockerfile。
- **流水线工具链分发**：BKMS 流水线会下载该二进制并执行，由它在工作空间内生成最终参与镜像构建的 Dockerfile。

### 主要输入

工具通过环境变量读取配置，常用变量如下：

| 变量                                     | 说明                                                 |
|----------------------------------------|----------------------------------------------------|
| `BKMS_DOCKERFILE_SOURCE_TYPE`          | Dockerfile 来源类型，支持 `bkms_generated` 和 `repository` |
| `BKMS_DOCKERFILE_LANGUAGE`             | 平台生成模式下的应用语言，目前支持 `go`、`cpp`、`python`             |
| `BKMS_DOCKERFILE_PATH`                 | 生成 Dockerfile 的目标路径                                |
| `BKMS_DOCKERFILE_BUILDER_IMAGE`        | 构建阶段基础镜像；`python` 只有运行阶段，可为空                       |
| `BKMS_DOCKERFILE_RUNNER_IMAGE`         | 运行阶段基础镜像                                           |
| `BKMS_DOCKERFILE_PRE_BUILD_COMMANDS`   | 编译前置命令，多条命令使用换行分隔                                  |
| `BKMS_DOCKERFILE_BUILD_COMMANDS`       | 编译命令，多条命令使用换行分隔；为空时使用语言默认构建命令                      |
| `BKMS_DOCKERFILE_RUNTIME_ENV_COMMANDS` | 运行环境命令，多条命令使用换行分隔                                  |
| `BKMS_DOCKERFILE_START_COMMAND`        | 容器启动命令，可为空；为空时入口为 `/app/<镜像名>`                     |
| `BKMS_DOCKERFILE_ARTIFACT_DIR`         | 构建产物目录，可为空；为空或未定义时 `go`、`cpp` 使用 `/out`，`python` 使用 `/app` |
| `BKMS_IMAGE_NAME`                      | 镜像名称，同时用于默认构建产物名                                   |

在 `bkms_generated` 模式下，`BKMS_DOCKERFILE_LANGUAGE`、`BKMS_DOCKERFILE_PATH`、`BKMS_DOCKERFILE_RUNNER_IMAGE` 和
`BKMS_IMAGE_NAME` 为必填项；`go`、`cpp` 还要求 `BKMS_DOCKERFILE_BUILDER_IMAGE`。

### 产物目录与入口

- builder 阶段声明 `ENV BKMS_ARTIFACT_DIR=<产物目录>`，编译前置命令和构建命令可以用 `${BKMS_ARTIFACT_DIR}` 引用；runner 阶段不提供该变量。
- 未填构建命令时，默认构建命令把产物输出到 `<产物目录>/<镜像名>`，未配置产物目录时即 `/out/<镜像名>`。
- 构建命令执行后，产物目录必须存在且至少有一个条目（隐藏文件、空子目录都算），否则构建失败并输出
  `build output directory <产物目录> is missing or empty`。
- runner 阶段把产物目录内的全部内容拷到 `/app`，保持相对目录结构，例如 `/dist/conf/a.yaml` 落到 `/app/conf/a.yaml`。
- 产物目录非空时依次校验：长度不超过 128；只包含字母、数字、`.`、`_`、`-`、`/`；以 `/` 开头；不包含 `..` 路径段；
  规范化（去掉末尾 `/`、合并连续 `/`、去掉 `.` 段）后不是根目录 `/`。任一不满足即退出并报错，不生成 Dockerfile。
- 填写启动命令时入口为 `["/bin/sh", "-ec", <启动命令>]`。
- 未填写启动命令时入口为 `["/app/<镜像名>"]`。默认构建会把二进制放在这个路径。自定义构建命令若产出别的文件名，需要自行填写启动命令，否则容器会去执行 `/app/<镜像名>`。

以上是 `go`、`cpp` 的两阶段模板。

### Python 单阶段模板

- 只有基于运行镜像的一个阶段，不使用构建镜像。源码整体拷到产物目录（默认 `/app`），并以它作为工作目录；
  `BKMS_ARTIFACT_DIR` 等于该目录，在最终镜像中同样可见。
- 源码已整体拷入，`BKMS_DOCKERFILE_EXTRA_FILES` 不再单独拷贝。
- 执行顺序：编译前置命令、构建命令（或自动安装依赖）、运行环境命令，最后做与 go 相同的产物目录非空检查。
- 填写了构建命令时只执行用户命令，不自动安装依赖。
- 构建命令留空时，按构建目录根下的锁文件选一种方式安装，优先级 `uv.lock` > `poetry.lock` > `requirements.txt`，
  不递归子目录；三种都没有时只拷源码。依赖都装进运行镜像的系统 Python，启动命令直接用 `python` 即可：
  - `uv.lock`：`uv export --no-default-groups` 导出锁定版本后用 `uv pip install --system` 安装，
    不装 `[tool.uv] default-groups` 里的任何分组（含 dev），也不装项目自身；
  - `poetry.lock`：`poetry install --no-root --only main`，只装主依赖，不创建虚拟环境；
  - `requirements.txt`：先确认 pip 可用，再 `python3 -m pip install -r requirements.txt`。

  三种方式都关闭下载缓存（uv / poetry 用 `--no-cache`，pip 用 `--no-cache-dir`）：单阶段镜像没有 builder 层隔离，
  缓存会直接留在最终镜像里。
- 依赖安装失败时构建失败，日志保留安装工具的原始输出。
- 未填写启动命令时不写 `ENTRYPOINT`，沿用运行镜像自带入口或部署时的容器启动命令。

#### 运行镜像与工具

- 运行镜像负责提供 `python3` 与 pip，模板不会用 apk / apt 等系统包管理器补装：各发行版的包名、版本与
  PEP 668 限制都不同，补装出来的解释器也不保证能跑用户代码。平台官方运行镜像提供 `python:X.Y.Z-slim` 系列，
  满足上述要求，可以直接使用。
- 自动安装依赖前先检查 `python3`，缺失时构建失败并提示运行镜像没有 python3；没有依赖文件时不做该检查。
- 运行镜像缺少 uv / poetry 时先用 `python3 -m pip` 安装；连 pip 都没有时构建失败并提示缺少 pip。
  `requirements.txt` 路径同样先检查 pip；uv / poetry 已在镜像里时不需要 pip，因此不做统一的前置 pip 检查。
  只安装命中的那一种工具，镜像里已有时直接跳过。
- uv / poetry 不锁定版本：pip 会按运行镜像的 Python 版本挑声明兼容的最新版。项目需要特定版本时，
  在编译前置命令里先装好，例如 `python3 -m pip install poetry==1.8.5`，后面检测到已存在就不会覆盖。
- 自动安装需要访问 Python 包仓库。内网构建可通过构建参数传入镜像源：pip 读 `PIP_INDEX_URL`
  （安装 uv / poetry 本身也走它），uv 读 `UV_DEFAULT_INDEX`，不读 pip 的变量；poetry 的包源由项目
  `pyproject.toml` 自行声明。构建参数在单阶段里声明为 `ARG`，`RUN` 中可直接读取。
- 限制：Debian / Ubuntu 的系统 Python 按 PEP 668 拒绝 pip 往系统环境安装，自动安装会失败。
  使用官方 `python:*` 镜像即可避开，或自行填写构建命令。

### 主要输出

- 生成结果会写入 `BKMS_DOCKERFILE_PATH` 指定的路径。
- 如果目标路径包含不存在的父目录，工具会自动创建父目录。
- 如果目标文件已存在，工具会直接覆盖，便于流水线重复执行。

### 通过 Docker build 产出二进制

该目录下的 `Dockerfile` 用于在容器化 Go 环境中构建 `bkms-dockerfile-generator` 二进制，避免依赖宿主机操作系统、CPU 架构或本机
Go 工具链。构建目标固定为 `linux/amd64`，即使在 macOS 或其他架构机器上执行，也会产出 Linux AMD64 二进制。

#### 前置条件

- 已安装 Docker。
- 使用支持 BuildKit `--output` 的 Docker 版本。
- 推荐在 `bkms-dockerfile-generator` 目录执行以下命令。

#### 本地构建命令

```bash
cd bkms-dockerfile-generator
mkdir -p ./bin
DOCKER_BUILDKIT=1 docker build --target artifact --output type=local,dest=./bin .
```

构建完成后，产物路径为：

```bash
./bin/bkms-dockerfile-generator
```

可以使用以下命令验证产物存在并查看平台信息：

```bash
ls -l ./bin/bkms-dockerfile-generator
file ./bin/bkms-dockerfile-generator
```

#### CI 集成命令

CI 中可以直接使用非交互式命令导出产物：

```bash
cd bkms-dockerfile-generator && mkdir -p ./bin && DOCKER_BUILDKIT=1 docker build --target artifact --output type=local,dest=./bin .
```

如需将二进制上传到工具链分发地址，可在该命令成功后使用 `./bin/bkms-dockerfile-generator` 作为上传源文件。
