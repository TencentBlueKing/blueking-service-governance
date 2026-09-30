# bkms-cli E2E Tests

## 运行测试

```bash
# 1. 配置环境变量
vim test/e2e/.env

# 2. 一键构建 + 运行
make e2e-go-test
```

按标签选择性运行：

```bash
# 只跑只读测试（不修改服务端数据）
make e2e-go-test GINKGO_FLAGS="--label-filter=readonly"

# 跳过写操作测试
make e2e-go-test GINKGO_FLAGS="--label-filter='!destructive'"

# 只跑冒烟测试
make e2e-go-test GINKGO_FLAGS="--label-filter=smoke"
```

手动运行：

```bash
make e2e-build
set -a && . test/e2e/.env && set +a
BKMS_CLI_BIN=$(pwd)/build/bkms-cli-e2e \
BKMS_CLI_BUILD_DIR=$(pwd)/build \
go test -v ./test/e2e/...
```

## 环境变量

在 `test/e2e/.env` 中配置：

```bash
BKMS_API_URL="http://example.com"
BKMS_USERNAME="your-username"
BKMS_TOKEN="your-access-token"
BKMS_WORKSPACE_ID="your-workspace-id"
BKMS_APP_ID="your-app-id"
BKMS_ENV_NAME="your-env-name"
```

以上均为必填项，缺失任一测试直接失败。`BKMS_CLI_BUILD_DIR` 由 `make e2e-go-test` 自动注入。

可选项：
- `BKMS_CLI_BIN` — 自定义二进制路径（默认自动查找 build 目录）
- `BKMS_CLI_CONFIG` — 自定义配置文件路径（默认 `~/.bkms/e2e-config.yaml`）

## 测试标签

| 标签 | 含义 |
|------|------|
| `readonly` | list / view 类操作，不改数据 |
| `destructive` | create / update / delete 类操作，会改数据 |
| `smoke` | 冒烟测试，快速验证核心链路 |

## 编写新测试

在对应子目录下创建 `xxx_test.go`，使用该目录的包名和全局变量 `cli`、`envCfg`：

| 目录 | 包名 | 放什么 |
|------|------|--------|
| `base/` | `base_test` | root、auth、config、workspace、env |
| `app/` | `app_test` | app CRUD、appspec、image、instance |
| `deploy/` | `deploy_test` | deploy、build |
| `extension/` | `extension_test` | polaris、component、app-cfg-file |
| `envvar/` | `envvar_test` | 环境变量 CRUD |

只读测试：

```go
var _ = Describe("Foo List", Ordered, Label("readonly"), func() {
    BeforeAll(func() { framework.EnsureLoggedIn(cli, envCfg) })

    It("exits with code 0", func() {
        cli.Run("foo", "list", "--app", envCfg.AppID).
            ExpectSuccess()
    })

    It("supports -o json", func() {
        cli.Run("foo", "list", "--app", envCfg.AppID, "-o", "json").
            ExpectJSON(func(data any) { Expect(data).NotTo(BeNil()) })
    })
})
```

写操作测试：

```go
var _ = Describe("Foo CRUD", Ordered, Label("destructive"), func() {
    var name string

    BeforeAll(func() { framework.EnsureLoggedIn(cli, envCfg) })

    AfterAll(func() {
        if name != "" {
            cli.Run("foo", "delete", "--name", name, "--yes")
        }
    })

    It("should create", func() {
        f := framework.WriteFixtureFile(spec, "foo")
        DeferCleanup(framework.CleanupFixtureFile, f)
        cli.Run("foo", "create", "-f", f).ExpectSuccess()
    })
})
```

### 链式断言

| 方法 | 作用 |
|------|------|
| `ExpectSuccess()` | 断言退出码 0 |
| `ExpectFailure()` | 断言退出码非 0 |
| `ExpectExitCode(n)` | 断言精确退出码 |
| `ExpectStdoutContains(s)` | stdout 包含子串 |
| `ExpectOutputContains(s)` | stdout+stderr 包含子串 |
| `ExpectJSON(fn)` | 解析 stdout 为 JSON 后回调断言 |

所有方法返回 `Result`，可链式调用：

```go
cli.Run("app", "get", "--app", id, "-o", "json").
    ExpectSuccess().
    ExpectJSON(func(data any) { Expect(data).NotTo(BeNil()) })
```
