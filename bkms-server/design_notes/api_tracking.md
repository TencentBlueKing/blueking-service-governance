# API 调用追踪

## 核心诉求

bkms-cli 无自动升级，用户通常不主动升级，旧版本持续调用旧接口，导致这些接口无法下架。需要掌握各版本对 API 的实际调用情况，以判断哪些 API 仍被使用、哪些可以下架。

## 目标

server 为每个 API 请求输出一条结构化事件日志，经 APM 采集后离线计算，得到以下统计：

- CLI 版本 × API 的最后调用时间（核心）
- 各 API 的调用方分布（CLI、前端、其他 token 调用方）
- 调用次数、设备、系统、用户等维度的分布

结论由人工评估得出。server 不建表，也不提供查询接口。

## 整体流程

```
bkms-cli / 前端 / 其他 ──请求──▶ bkms-server 中间件
                                        │ 输出事件日志 event=api_call
                                              ▼
                                    slog ──▶ stdout
                                         └─▶ APM（OTel log）──▶ 离线计算
```

## 客户端识别

| 条件 | `client` | `cli_version` |
|---|---|---|
| UA 为 `bkms-cli/<semver>` | `cli` | `<semver>` |
| UA 为 `bkms-cli/dev` | `cli` | `dev` |
| UA 为 `bkms-cli/<非法版本>` | `cli` | `invalid` |
| Bearer 认证 + UA 以 `go-resty/` 开头 | `cli` | `<1.0.3` |
| bk_ticket / bk_token 认证 | `web` | 空 |
| 其他 access token 认证 | `token` | 空 |
| 认证失败且非 CLI | `unknown` | 空 |

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
- 使用 `logging.InfoAttrs(c.Request.Context(), "api_call", ...)` 输出 info 级别日志：
  - 经全局 slog 同时写入 stdout 和 APM。
  - `trace_id` / `span_id` 由 ctx 自动注入。
- 未匹配路由的请求不输出。

### 字段

| 字段 | 来源 | 说明 |
|---|---|---|
| `event` | 固定值 | `api_call`，用于过滤 |
| `schema` | 固定值 | `1`，字段变更时递增 |
| `time` | slog | 事件时间 |
| `client` | 见「客户端识别」 | |
| `cli_version` | UA | 仅 CLI |
| `os` / `arch` | UA | 仅 CLI，v1.0.3 起才有 |
| `device_id` | `X-Bkms-Device-Id` | 仅 CLI；64 位十六进制，不符合格式时为 `invalid` |
| `user_agent` | UA | 最长 512 字节 |
| `username` / `tenant_id` | 认证结果 | 认证失败时为空 |
| `client_ip` | `c.ClientIP()` | 取自 `X-Forwarded-For`，可被客户端伪造，仅供参考 |
| `handler` | 见「API 标识」 | |
| `method` / `route` | 请求方法、`c.FullPath()` | |
| `path` | 实际请求路径 | |
| `query` | 脱敏后的查询参数 | 见「脱敏」 |
| `status` | 响应状态码 | |
| `trace_id` / `span_id` | 自动注入 | APM 启用时才有 |

### 脱敏

- 不记录 Authorization、Cookie 等凭据请求头，不记录请求体与响应体。
- `query` 中参数名含 `token`、`ticket`、`secret`、`password`、`passwd`、`credential`，或以 `key` 结尾（如 `api_key`，不含 `keyword`）时，值替换为 `***`；超过 1024 字节截断。规则实现于 `pkg/common/utils/urlmask`。
- 访问日志（`apm.HTTPLogMiddleware`）的 `request.query` 与 `request.referer` 使用同一规则脱敏。此前 `/user_token/validate?access_token=...` 的 token 会明文写入访问日志。

### 用户身份

记录明文 `username`，做法与 AWS CloudTrail（`userIdentity`）、K8s audit（`user`）等服务端审计一致：

- 产品遥测（.NET CLI、Homebrew 等）从用户机器收集数据，因此匿名且可关闭；本方案是服务端记录已认证的请求，身份明文是定位调用方的前提。
- 操作审计（`pkg/misc/audit`）已记录操作人用户名。
- 约束：只记录必要字段，凭据与请求体不记录，query 脱敏；设备 ID 只上报 HMAC 后的值。

### 采样与丢失

- 统计依据为日志，不依赖链路。OTel 采样只作用于 span，不作用于 log；当前 trace 采样器为 `AlwaysSample`。
- APM 日志由 `BatchProcessor` 批量异步上报，队列满或进程异常退出时可能丢失少量日志。
- stdout 中有同一份日志，可接入日志平台的容器标准输出采集作为兜底。

### 约束

- 日志级别须为 info 或更低，否则事件日志会被过滤。
- 须启用 APM，否则日志只写入 stdout。
- 前端请求量远大于 CLI，日志量随之增加。
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

以 `event=api_call` 为过滤条件，示例指标如下：

- `client=cli` 时，按 `(cli_version, handler)` 计算 `max(time)`、`count(*)`
- 按 `(handler, client)` 计算 `max(time)`、`count(*)`，判断接口是否仍被前端或其他调用方使用
- 按 `cli_version` 计算 `count(distinct device_id)`、`count(distinct username)`
- 按 `device_id` 取最新的 `cli_version`

## 改动范围

- server：
  - 新增 `pkg/observability/apitracking`：客户端识别、handler 名称映射、中间件、单测。
  - 新增 `pkg/common/utils/urlmask`：query 脱敏。
  - `pkg/observability/apm`：访问日志脱敏 query 与 referer。
  - `pkg/server/server.go`：挂载中间件。
- CLI：
  - 新增 `pkg/deviceid`：设备 ID。
  - 统一错误信息中的 trace id。
  - debug 模式下输出请求的 trace id。
  - README 说明采集的字段。
  - 单测。

## 后续

- 积累数据后，制定 CLI 版本支持周期说明。
