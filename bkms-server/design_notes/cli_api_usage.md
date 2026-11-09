# CLI API 调用统计

## 核心诉求

bkms-cli 无自动升级，用户通常不主动升级，旧版本持续调用旧接口，导致这些接口无法下架。需要掌握各版本对 API 的实际调用情况，以判断哪些 API 仍被使用、哪些可以下架。

## 目标

server 为每个 CLI 请求输出一条结构化事件日志，经 APM 采集后离线计算，得到以下统计：

- 版本 × API 的最后调用时间（核心）
- 调用次数、设备、系统、用户等维度的分布

结论由人工评估得出。server 不建表，也不提供查询接口。

## 整体流程

```
bkms-cli ──请求（UA、X-Bkms-Device-Id）──▶ bkms-server 中间件
                                              │ 输出事件日志 event=cli_api_call
                                              ▼
                                    slog ──▶ stdout
                                         └─▶ APM（OTel log）──▶ 离线计算
```

## 请求识别

| 条件 | `cli_version` |
|---|---|
| UA 为 `bkms-cli/<semver>` | `<semver>` |
| UA 为 `bkms-cli/dev` | `dev` |
| UA 为 `bkms-cli/<非法版本>` | `invalid` |
| Bearer 认证 + UA 以 `go-resty/` 开头 | `<1.0.3` |
| 其他 | 不输出 |

依据：

- v1.0.3 起，CLI 发送 `bkms-cli/<version> (<os>/<arch>)` UA。
- v1.0.1、v1.0.2 使用 resty 默认 UA。
- 前端使用 cookie 认证，不带 Bearer。

## API 标识

使用 handler 方法名，例如 `ListHelmDeployRecords`。

- 服务启动后遍历 `r.Routes()`，建立 `c.HandlerName()` 到方法名的映射。
- 方法名重名时，改用去掉模块前缀的全名，例如 `deploy/handler.(*Handler).List`，并输出 warn 日志。
- 闭包 handler（名称中含 `funcN` 或纯数字段）无法取得方法名，使用 `METHOD route`。

## 事件日志

### 输出

- 中间件挂载在 `v1` 组上、`requiredAuth` 之前。在 `c.Next()` 之后输出，此时认证结果和状态码均已确定；认证失败的请求同样输出。
- 使用 `logging.InfoAttrs(c.Request.Context(), "cli_api_call", ...)` 输出 info 级别日志：
  - 经全局 slog 同时写入 stdout 和 APM。
  - `trace_id` / `span_id` 由 ctx 自动注入。
- 未匹配路由的请求不输出。

### 字段

| 字段 | 来源 | 说明 |
|---|---|---|
| `event` | 固定值 | `cli_api_call`，用于过滤 |
| `schema` | 固定值 | `1`，字段变更时递增 |
| `cli_version` | UA | 见「请求识别」 |
| `os` / `arch` | UA | v1.0.3 起才有 |
| `device_id` | `X-Bkms-Device-Id` | 64 位十六进制；不符合格式时记为 `invalid`，缺失时为空 |
| `username` / `tenant_id` | 认证结果 | 认证失败时为空 |
| `handler` | 见「API 标识」 | |
| `method` / `route` | 请求方法、`c.FullPath()` | |
| `status` | 响应状态码 | |
| `trace_id` / `span_id` | 自动注入 | APM 启用时才有 |

### 约束

- 日志级别须为 info 或更低，否则事件日志会被过滤。
- 须启用 APM，否则日志只写入 stdout。
- APM 日志采用批量异步上报，进程异常退出时，可能丢失尚未上报的少量日志。

## CLI 改动

### 设备 ID

- 使用 `github.com/keygen-sh/machineid`（MIT 许可，原 `denisbrodbeck/machineid` 的维护版 fork），读取操作系统机器标识：
  - Linux：`/var/lib/dbus/machine-id`、`/etc/machine-id`
  - macOS：`IOPlatformUUID`
  - Windows：注册表 `MachineGuid`
- 上报值为 `ProtectedID("bkms-cli")`，即 HMAC-SHA256 后的 64 位十六进制串，不暴露原始机器标识。
- 每个进程计算一次，不落盘，请求时放在 `X-Bkms-Device-Id` 中发送（包括 port-forward 的 WebSocket 请求）。
- 原始值须为 32 位十六进制、UUID 或 64 位容器 ID，否则不发送。原因是该库在 cgroup 回退路径上可能取到 `user-1000.slice` 之类会在不同机器间重复的值。
- 取值失败时不发送，不影响命令执行。

已知局限：

- 镜像克隆且未重新生成 machine-id 的机器，会得到相同的 ID。
- 容器取到的是容器 ID，每个容器各不相同。
- 重装系统后 ID 会变化。

离线计算时可结合 `username` 辅助识别。

### trace id

server 在每个响应的 `X-Trace-Id` 头中返回 trace id。CLI 的处理方式：

- 非 2xx 的错误信息附带 `trace_id`：
  - `env_var.go`、`dashboard.go`、`client.go` 中 21 处手工拼接的错误改为经 `formatAPIError` 构造。
  - `port_forward.go` 的权限预检与 WebSocket 拨号失败 2 处补充 trace id。
- 在 `--log-level debug` 下，每个请求输出 method、path、status、trace_id、耗时，通过 resty `OnAfterResponse` 实现。只输出 URL path，不含 query，避免 `access_token` 进入日志。

## 数据缺口

| 版本 | 版本号 | os/arch | 设备 ID | 用户 |
|---|---|---|---|---|
| < 1.0.3 | 仅 `<1.0.3` | 无 | 无 | 有 |
| 1.0.3 ～ 1.0.7 | 有 | 有 | 无 | 有 |
| 新版本 | 有 | 有 | 有 | 有 |

另外，上线前的调用和 `v1` 路由组以外的接口均无记录。

## 离线计算

以 `event=cli_api_call` 为过滤条件，示例指标如下：

- 按 `(cli_version, handler)` 计算 `max(time)`、`count(*)`
- 按 `cli_version` 计算 `count(distinct device_id)`、`count(distinct username)`
- 按 `device_id` 取最新的 `cli_version`

## 改动范围

- server：
  - 新增 `pkg/observability/cliusage`：UA 识别、handler 名称映射、中间件、单测。
  - `pkg/server/server.go`：挂载中间件。
- CLI：
  - 新增 `pkg/deviceid`：设备 ID。
  - 统一错误信息中的 trace id。
  - debug 模式下输出请求的 trace id。
  - 单测。

## 后续

- 积累数据后，制定 CLI 版本支持周期说明。
