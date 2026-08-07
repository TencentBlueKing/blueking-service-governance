# bkms-cli

bkms-cli 是蓝鲸服务治理平台提供的命令行工具，支持查看应用基础信息、构建、部署和查询部署结果等功能。

## 项目结构

```
bkms-cli/
├── main.go                  # 程序入口
├── Makefile                 # 构建 & 开发命令
├── cmd/                     # 子命令定义（按功能分目录，每个目录对应一个子命令）
│   ├── root/                # 根命令
│   ├── version/             # version 子命令
│   ├── auth/                # login / logout
│   ├── config/              # config view
│   ├── workspace/           # workspace list / set / unset
│   ├── env/                 # env list
│   └── app/                 # app 相关
│       ├── list.go          # app list
│       ├── image/           # app image list
│       ├── deploy/          # app deploy list / create / update
│       ├── instance/        # app instance list
│       └── publish/         # app publish（BCS 发布）
├── pkg/                     # 内部共享包
│   ├── client/              # API 客户端（封装 HTTP 请求）
│   ├── config/              # 配置文件读写（~/.bkms/config.yaml）
│   ├── account/             # 用户账号管理
│   ├── handler/             # 业务处理器
│   │   ├── deploy/          # 部署相关逻辑
│   │   └── publish/         # BCS 发布逻辑（bcsAPIHost 在此注入）
│   ├── version/             # 版本信息（通过 ldflags 注入）
│   └── utils/               # 工具函数（命令行、控制台、环境变量、输出格式化、路径）
└── test/
    └── e2e/                 # E2E 功能测试（详见 test/e2e/README.md）
```

### 添加新子命令

1. 在 `cmd/` 下创建新目录（如 `cmd/mycommand/`）
2. 定义命令入口文件（参考 `cmd/env/env.go` 的模式）
3. 在 `cmd/root/root.go` 中注册新子命令
4. API 调用逻辑放在 `pkg/client/` 中，业务处理逻辑放在 `pkg/handler/` 中

## 开发指引

### 环境要求

- Go 1.21+
- Make

### Make 命令一览

运行 `make help` 查看所有可用命令，以下为常用命令：

| 命令 | 说明 |
|------|------|
| `make build` | 构建当前平台二进制（产物在 `./build/`） |
| `make build GOOS=linux GOARCH=amd64` | 构建指定平台 |
| `make build-all` | 构建所有平台（linux/darwin/windows × amd64/arm64） |
| `make build-all compress` | 构建所有平台 + UPX 压缩 |
| `make clean` | 清理构建产物 |
| `make tidy` | 执行 `go mod tidy` |
| `make vet` | 执行 `go vet` |
| `make fmt` | 代码格式化（golangci-lint） |
| `make lint` | 代码检查（golangci-lint） |
| `make test` | 运行单元测试（Ginkgo） |
| `make e2e-go-test` | 构建 E2E 专用二进制并运行 E2E 测试 |

### 构建

```shell
# 日常开发构建（BCS_API_HOST 已内置默认值，无需手动指定）
make build

# 出包（所有平台 + UPX 压缩）
make build-all compress
```

> `BCS_API_HOST` 默认为 `https://bcs-api.example.com`，实际使用时候需要可通过 `make build BCS_API_HOST=https://your-host` 指定。

构建产物位于 `./build/` 目录，命名规则：
- 当前平台：`bkms-cli`（或 `bkms-cli.exe`）
- 指定平台：`bkms-cli-{os}-{arch}`
- 平台发布构建：`bkms-cli-{os}-{arch}-{version}`

### 编译期注入参数

以下参数通过 `go build -ldflags -X` 在编译期注入：

| 参数 | 说明 |
|------|------|
| `pkg/version.Version` | 版本号（`git describe --always`） |
| `pkg/version.GitHash` | Git commit hash |
| `pkg/version.BuildTime` | 构建时间 |
| `pkg/updater.updateSource` | 更新源；`owner/repository` 使用 GitHub Releases，HTTP(S) URL 使用制品仓库目录 |
| `pkg/handler/publish.bcsAPIHost` | BCS API 网关地址 |

### 代码检查 & 格式化

```shell
make fmt    # 格式化代码
make lint   # 静态检查
make vet    # go vet
```

lint 规则配置见 `.golangci.yaml`。

### 测试

**单元测试：**

```shell
make test
```

使用 Ginkgo 框架，测试文件与源码同目录（`*_test.go`）。

#### 生成测试 Mock

项目使用 [mockery](https://github.com/vektra/mockery) 生成接口 mock，配置文件为 `.mockery.yml`，工具依赖通过 `tools.go` 记录。

前置安装：

```shell
go install github.com/vektra/mockery/v3@v3.7.1
```

生成 `pkg/client.Client` mock：

```shell
mockery --config .mockery.yml
```

**E2E 功能测试：**

```shell
# 需要先设置环境变量（BKMS_API_URL / BKMS_USERNAME / BKMS_TOKEN 等）
make e2e-go-test
```

E2E 测试会自动构建 `bkms-cli-e2e` 专用二进制（避免覆盖正式构建产物），详细说明见 [test/e2e/README.md](test/e2e/README.md)。
