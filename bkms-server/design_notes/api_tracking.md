# API 调用追踪

## 背景

bkms-cli 无自动升级，旧版本持续调用旧接口，导致接口无法下架。需要知道各 CLI 版本实际调用了哪些 API。

## 方案

访问日志（`apm.HTTPLogMiddleware`）附加追踪字段，经 APM 采集后离线计算。server 不建表，不提供查询接口。

- 每个请求一条日志，追踪字段位于 attributes 顶层，sloggin 原有字段位于 `request`、`response` 分组。
- 覆盖所有已匹配路由；`/healthz`、`/readyz`、`/metrics` 与未匹配路由除外。
- 认证失败的请求同样记录。

## 字段

| 字段 | 说明 |
|---|---|
| `event` | 固定为 `api_call`，用于过滤 |
| `schema` | 字段版本，当前为 `1` |
| `client` | `cli` / `web` / `token` / `unknown`，见下文 |
| `cli_version` | 仅 CLI，见下文 |
| `os` / `arch` | 仅 CLI，取自 UA，v1.0.3 起 |
| `device_id` | 仅 CLI，取自 `X-Bkms-Device-Id`，格式非法时为 `invalid` |
| `user_agent` | 最长 512 字节 |
| `username` / `tenant_id` | 认证失败时为空 |
| `client_ip` | 取自 `X-Forwarded-For`，仅供参考 |
| `handler` | handler 方法名，如 `ListHelmDeployRecords` |
| `method` / `route` / `path` | `route` 为路由模板 |
| `query` | 已脱敏 |
| `status` | 响应状态码 |
| `time` / `trace_id` / `span_id` | slog 与 OTel 自动注入 |

### 客户端识别

| 条件 | `client` | `cli_version` |
|---|---|---|
| UA 为 `bkms-cli/<semver>` | `cli` | `<semver>` |
| UA 为 `bkms-cli/dev` | `cli` | `dev` |
| UA 为 `bkms-cli/<非法版本>` | `cli` | `invalid` |
| Bearer 认证 + UA 以 `go-resty/` 开头 | `cli` | `<1.0.3` |
| bk_ticket / bk_token 认证 | `web` | |
| 其他 access token 认证 | `token` | |
| 其他 | `unknown` | |

v1.0.1、v1.0.2 未声明 UA，使用 resty 默认 UA。

### handler 名称

- 全部路由注册后，由 `r.Routes()` 建立映射。
- 方法名重名时，使用去掉模块前缀的全名，如 `deploy/handler.(*Handler).List`。
- 闭包 handler 使用 `METHOD route`。

## 隐私

- 记录明文 `username`，与 CloudTrail、K8s audit 等服务端审计一致；操作审计也已记录用户名。
- 不记录凭据请求头、请求体、响应体。
- query 与访问日志的 `request.query`、`request.referer` 均脱敏：参数名含 `token`、`ticket`、`secret`、`password`、`passwd`、`credential`，或以 `key` 结尾时，值替换为 `***`（`pkg/common/utils/urlmask`）。
- 设备 ID 只上报 HMAC 后的值。

## CLI 改动

- **设备 ID**：用 `keygen-sh/machineid` 读取系统机器标识，上报 `ProtectedID("bkms-cli")`（HMAC-SHA256）。原始值不是 32 位十六进制、UUID 或 64 位容器 ID 时不上报。不落盘，取不到不影响命令执行。
- **trace id**：非 2xx 错误附带响应头 `X-Trace-Id`；`--log-level debug` 下每个请求输出 method、path、status、trace_id。
- README 列出采集的字段。

设备 ID 的局限：镜像克隆的机器可能相同，容器每次不同，重装系统后变化。

## 数据缺口

| CLI 版本 | 版本号 | os/arch | 设备 ID |
|---|---|---|---|
| < 1.0.3 | 仅 `<1.0.3` | 无 | 无 |
| 1.0.3 ～ 1.0.7 | 有 | 有 | 无 |
| 1.0.8 起 | 有 | 有 | 有 |

上线前的调用无记录。

## 约束

- 日志级别须为 info 或更低。
- 须启用 APM，否则日志只在 stdout。
- 统计依赖日志而非链路，不受 trace 采样影响；APM 日志批量异步上报，进程异常退出时可能丢失少量日志。
- 前端请求量大，日志量随之增加。

## 离线计算示例

过滤条件 `event=api_call`：

- `client=cli`，按 `(cli_version, handler)` 求 `max(time)`、`count(*)`
- 按 `(handler, client)` 求 `max(time)`，判断接口是否仍被使用
- 按 `cli_version` 求 `count(distinct device_id)`、`count(distinct username)`

## 后续

- 积累数据后制定 CLI 版本支持周期。
