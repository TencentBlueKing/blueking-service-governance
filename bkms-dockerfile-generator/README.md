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
| `BKMS_DOCKERFILE_LANGUAGE`             | 平台生成模式下的应用语言，目前支持 `go`、`cpp`                       |
| `BKMS_DOCKERFILE_PATH`                 | 生成 Dockerfile 的目标路径                                |
| `BKMS_DOCKERFILE_BUILDER_IMAGE`        | 构建阶段基础镜像                                           |
| `BKMS_DOCKERFILE_RUNNER_IMAGE`         | 运行阶段基础镜像                                           |
| `BKMS_DOCKERFILE_PRE_BUILD_COMMANDS`   | 编译前置命令，多条命令使用换行分隔                                  |
| `BKMS_DOCKERFILE_BUILD_COMMANDS`       | 编译命令，多条命令使用换行分隔；为空时使用语言默认构建命令                      |
| `BKMS_DOCKERFILE_RUNTIME_ENV_COMMANDS` | 运行环境命令，多条命令使用换行分隔                                  |
| `BKMS_DOCKERFILE_START_COMMAND`        | 容器启动命令，可为空；为空时入口为 `/app/<镜像名>`                     |
| `BKMS_DOCKERFILE_ARTIFACT_DIR`         | 构建产物目录，可为空；为空或未定义时 `go`、`cpp` 使用 `/out`              |
| `BKMS_IMAGE_NAME`                      | 镜像名称，同时用于默认构建产物名                                   |

在 `bkms_generated` 模式下，`BKMS_DOCKERFILE_LANGUAGE`、`BKMS_DOCKERFILE_PATH`、`BKMS_DOCKERFILE_BUILDER_IMAGE`、
`BKMS_DOCKERFILE_RUNNER_IMAGE` 和 `BKMS_IMAGE_NAME` 为必填项。

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
