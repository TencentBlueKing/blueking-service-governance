# bkms-server

本项目是服务治理产品中与 Helm 应用相关的原生 Gin REST API 服务。

## 开发指引

### 环境初始化

```shell
make ginkgo golangci-lint swag migrate_binary
```

### 研发命令

```shell
make fmt      # 执行代码格式化
make lint     # 执行代码检查
make test     # 执行单元测试
make apidocs  # 根据 Gin API 注解生成 Swagger API 文档
```

### 日志使用

项目统一使用 `pkg/common/logging` 包输出日志，不直接使用标准库 `log`。请求链路、异步任务链路以及其他已经持有真实 `context.Context` 的场景，应优先传递真实上下文，以保留 trace、span 等链路字段：

```go
log.Info(ctx, "deploy started")
log.InfoAttrs(ctx, "deploy started", slog.String("app_id", appID))
```

对于进程启动、退出清理、测试辅助、一次性脚本等确实没有可用上下文的场景，可以使用 `NoContext` 系列接口，避免为了满足签名而传入 `context.TODO()`：

```go
log.InfoNoContext("scheduler starting")
log.WarnNoContextf("cleanup timeout after %s", timeout)
log.ErrorNoContextAttrs("cleanup failed", slog.String("workspace_id", workspaceID))
```

结构化日志请使用 `DebugAttrs`、`InfoAttrs`、`WarnAttrs`、`ErrorAttrs` 或对应的 `NoContextAttrs` 接口传入 `slog.Attr`。不要将结构化字段混入 `Infof`、`Warnf` 等格式化日志参数中，避免格式化参数和日志属性产生歧义。

### 构建命令

```shell
make build   # 构建可运行的二进制
make docker  # 构建 docker 镜像
```

### 本地服务运行

```shell
# 先执行 make build 后再运行如下命令
./build/bkms-server webserver --srvCfg ./configs/config.yaml
```

### 数据库迁移

数据库迁移文件位于 `db/migrations`，使用 golang-migrate 的 MongoDB JSON
格式，并在构建时嵌入 `bkms-server` 二进制。

常用命令示例（与官方 golang-migrate 保持一致）：

```bash
bkms-server migrate --srvCfg <config_path> up
bkms-server migrate --srvCfg <config_path> up 2
bkms-server migrate --srvCfg <config_path> goto 2
bkms-server migrate --srvCfg <config_path> down 1
```

- 数据库相关迁移（新建索引、旧数据清理等），建议优先使用该 migration 完成；
- 复杂的迁移任务，多增加一个 markdown 文件作为各项改动的注释（json 文件不支持注释，只能采取这种方式）；
- 对于特别复杂的数据迁移任务，json 文件无法胜任的，可以编写 Go 子命令完成；
- 测试数据库在 `SetUpGlobalDatabase` 初始化连接前会自动执行 `migrate up`；

#### seq 序号唯一性

迁移文件使用 golang-migrate 的 seq 命名（`000001_initial_idx.up.json`），序号必须全局唯一：
golang-migrate 在数据库中只保存**单个版本游标**，而不是逐条执行台账，一旦两个分支用了相同 seq，
后合入的那份 migration 会被视为"已执行"而被永久跳过，且不会报任何错误。

因此新增 migration 前，需确认序号未被目标分支占用。

#### 测试环境兼容高版本 migration

测试环境常出现"开发分支已执行更高 seq 的 migration，而 main 尚未合入该文件"的情况，
此时 golang-migrate 找不到数据库记录的版本，`migrate up` 会直接失败，导致 main 无法发布。
为此提供开关 `development.allowSkipNewerDBMigration`（默认 `false`）：

- 开启后，若数据库记录的版本高于当前二进制内嵌的最大版本，`migrate up` 会打印 WARN 日志并静默跳过；
- 数据库处于 dirty 状态时不会跳过，仍按原逻辑报错，避免掩盖真实的迁移失败；
- 只影响 `up`，`goto` / `down` / `force` 等人工运维命令行为不变。

**该开关禁止在生产环境开启**。它只保证本次发布不失败，并不会补执行本二进制内尚未执行的 migration：
例如测试环境游标已被开发分支推到 `7`，之后 main 合入了别人的 `000006`、`000007`，`migrate up`
会直接从 7 往后走，这两个 migration 永远不会执行且不报错。遇到这种情况只能人工
`bkms-server migrate force <N>` 把游标回退后重跑，或直接重建测试库。

#### 示例：创建索引

项目引入新的 collection 后，通常需要为其创建好初始化索引。可参考以下步骤添加 migration 文件。

```
# 创建空的 migration json 文件
./bin/migrate create -ext json -seq -dir db/migrations some_model_idx
```

之后，修改空 json 文件内容以添加索引（参考： https://github.com/golang-migrate/migrate/tree/master/database/mongodb/examples/migrations）。注意 up/down 两个 json 文件都需要提供。

文件准备就绪后，重新编译程序并执行 `bkms-server migrate up` 来测试。

### Swagger 文档

Gin API 使用 [swaggo/gin-swagger](https://github.com/swaggo/gin-swagger) 生成 Swagger 2.0 文档。

- 为 API 增加 Swagger 注解时，参考 `pkg/workload/envvars/handler/scoped_env_var.go` 中的各函数的注释。
- 更新文档时，在仓库根目录执行 `make apidocs`，生成内容位于 `docs/apis`。
- 本地启动服务后，可访问 Swagger UI：`/swagger/index.html`。
- Swagger JSON 地址：`/swagger/doc.json`。

### 运行单元测试

项目的单元测试依赖 Mongo 数据库、Helm 仓库、Git 仓库等外部服务，因此，在执行单元测试前，请完成以下操作：

- 设置项目配置文件：通过 `BKMS_SERVER_CONFIG_PATH` 环境变量，来设置一个有效的项目配置文件（效果和 webserver 命令所接收的 `--srvCfg` flag 一样）
  - 示例，执行 `export BKMS_SERVER_CONFIG_PATH=/configs/config.local.yaml`（建议使用绝对路径）
  - 测试将不会直接使用配置中所定义的 Mongo 数据库，而是会使用其鉴权信息创建一个新库，避免数据误写入
  - 默认创建名为 `{dbname}_for_test` 的数据库，其中 `{dbname}` 是配置文件中指定的库名
  - 其他依赖配置比如 Redis、MQ 等，目前将直接使用配置文件里的值
- 启动位于 `tests/utdeps` 中的所有依赖服务，其中包含 helm registry、container registry 和 Git server 3 种依赖服务
  - 切换到 `tests/utdeps`，执行 `just up`
  - 更多说明，查看 `tests/utdeps` 目录下的 README 文件

#### 依赖 Kubernetes 集群的测试

除了上面提到的，少量单元测试还会访问一个外部的 Kubernetes 集群（主要是 apiserver）——比如在 `pkg/deploy/helm/secret_test.go` 中的测试。默认情况下，这些测试将会被跳过。

通过以下两种方式配置有效的 Kubernetes 集群地址后，测试将被启用：

1. 配置 kubeconfig 文件地址：将 `FOR_TEST_KUBE_CONFIG_PATH` 环境变量设为有效的 kubeconfig 文件路径，测试将使用文件中 current_context 所定义的集群；
2. 分别配置 apiserver、ca、token 配置项：设置 `FOR_TEST_KUBE_APISERVER_URL`、`FOR_TEST_KUBE_CA_DATA`、`FOR_TEST_KUBE_TOKEN_VALUE` 环境变量，测试将使用这些信息来访问集群；

关于 Kubernetes 集群配置的更多详情，可查看 `testutil/deps.go` 文件。开发环境中建议使用 kind 启动测试集群。

一切准备就绪后：

- 执行 `make test` 运行所有的单元测试
- 执行 `./bin/ginkgo run -gcflags="all=-N -l" pkg/infras/registry` 来执行部分单元测试

**注意：完成代码改动，创建 PR 前，请你务必确认所有测试可正常通过。**

### 配置文件指南

本地开发环境，通常不具备访问有效鉴权服务的条件，为了便于调试，可以通过以下配置来关闭鉴权（总是允许所有鉴权请求）：

```yaml
development:
  useStubPerm: true
```

#### 为请求设置用户身份

bkms-server 服务的绝大多数接口为用户态接口，需要有效的用户身份才能调用。在线上环境中，用户身份认证由 bkms-server 的用户认证模块完成。

而在本地开发时，为了方便调试，你可以直接设置以下请求头来为设置用户身份信息：

- 首先在配置中设置 `development.allowSetUserInHeader: true`。该配置会绕过真实用户认证，禁止在生产环境中开启。
- `X-Bk-Authed-User-Info`：用户身份，值为 `{"userId":"USER_ID"}`（替换 USER_ID 为有效用户 ID）
- `X-Bk-Authed-User-Credential`：可选，用于认证的原始凭证信息，如接口逻辑并不依赖本字段，值可以随意填写，比如：`{"bkTicket":"xxx"}`

### 基于 fx 依赖注入框架简化对象链式构造

项目中的部分对象，构造时需要创建复杂的依赖链条（DBClient -> Store -> Service -> ...）。为简化该过程，项目引入了 [fx](https://github.com/uber-go/fx) 依赖注入框架。其常见使用方式为：

- 各模块 `module.go` 中提供 `FxModule` 变量，完成本模块内用于 DI 的对象定义
  - 这类对象通常为依赖数据库的 Store、依赖 Store 的领域 Service，等等
- 在测试用例的 BeforeEach 阶段，利用 `fxtest.New` 自动触发对象构造并赋值
