# bkms-cli

bkms-cli 是蓝鲸服务治理平台提供的命令行工具，支持查看应用基础信息、构建、部署和查询部署结果等功能。

## 安装

任选一种方式：

```shell
# 类 Unix
curl -fsSL https://raw.githubusercontent.com/TencentBlueKing/blueking-service-governance/main/bkms-cli/install.sh | sh
# 或
wget -qO- https://raw.githubusercontent.com/TencentBlueKing/blueking-service-governance/main/bkms-cli/install.sh | sh

# Windows（PowerShell 5.1+）
irm https://raw.githubusercontent.com/TencentBlueKing/blueking-service-governance/main/bkms-cli/install.ps1 | iex

# npm（Node.js 18+）
npm i -g @blueking/bkms-cli@latest

# Go 1.25.5+
go install github.com/TencentBlueKing/blueking-service-governance/bkms-cli@latest
```

脚本默认安装到 `~/.local/bin`（Windows 为 `%LOCALAPPDATA%\bkms-cli\bin`）。
可在 `sh` 后加 `-s -- --version 1.0.4` 指定版本，其他选项见 `--help`。

## 项目结构

```
bkms-cli/
├── main.go                  # 程序入口
├── Makefile                 # 构建 & 开发命令
├── .goreleaser.yaml         # 多架构发布构建（goreleaser）
├── install.sh               # macOS / Linux 安装脚本
├── install.ps1              # Windows PowerShell 安装脚本
├── latest.txt               # 已发布的 CLI 稳定版本号
├── npm/                     # @blueking/bkms-cli（现有 npm 分发包）
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
│   │   └── publish/         # BCS 发布逻辑
│   ├── version/             # 版本信息（ldflags 注入，Go 构建信息兜底）
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

- Go 1.25.5+
- Make

### Make 命令一览

运行 `make help` 查看所有可用命令，以下为常用命令：

| 命令 | 说明 |
|------|------|
| `make build` | 构建当前平台二进制（产物在 `./build/`） |
| `make build GOOS=linux GOARCH=amd64` | 构建指定平台 |
| `make build-all` | 用 goreleaser 构建所有平台（产物在 `./dist/`） |
| `make clean` | 清理 `build/` 与 `dist/` |
| `make tidy` | 执行 `go mod tidy` |
| `make vet` | 执行 `go vet` |
| `make fmt` | 代码格式化（golangci-lint） |
| `make lint` | 代码检查（golangci-lint） |
| `make test` | 运行单元测试（Ginkgo） |
| `make e2e-go-test` | 构建 E2E 专用二进制并运行 E2E 测试 |

### 构建

```shell
# 日常开发构建
make build

# 多架构出包（goreleaser snapshot，产物在 dist/）
make build-all
```

> API 地址不在编译期注入：由安装侧（npm 的 `bkmsCli` / `config set`，或安装脚本）写入 `~/.bkms/config.yaml`。

构建产物：
- `make build`：`./build/`
- `make build-all`：`./dist/`（`bkms-cli_{version}_{os}_{arch}.tar.gz`，Windows 为 `.zip`）

### 编译期注入参数

发布前手动同步 `bkms-cli/latest.txt` 和 `npm/package.json` 中的版本号，使其与 CLI tag 一致；CI 只校验，不修改或推送分支。

以下参数通过 `go build -ldflags -X` / goreleaser `ldflags` 在编译期注入：

| 参数 | 说明 |
|------|------|
| `pkg/version.Version` | 版本号（不带 `v`，如 `1.2.3`） |
| `pkg/version.GitHash` | Git commit hash |
| `pkg/version.BuildTime` | 构建时间 |

版本信息优先使用编译期注入值；未注入时读取 Go 模块版本和可用的 VCS revision。
缺少版本号时显示 `dev`，缺少 SHA 或构建时间时显示 `unknown`，不额外联网查询。

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
