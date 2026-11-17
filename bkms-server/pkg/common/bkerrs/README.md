# bkerrs 使用指南

## 背景

`bkerrs` 提供已分类错误 `bkerrs.Error`：错误码（`ErrCode`，决定 HTTP 状态）+ 错误详情码（`ErrDetailCode`，稳定的机器可读键，
输出为 `details[].code`，同时是 i18n key）+ 参数。`ErrorHandler` / `WriteError` 在出口把它转换为新版蓝鲸 HTTP API 协议的响应，并按请求语言翻译 `message`。

i18n 的通用规则（语言协商、文案资源、非错误文案）见 [`pkg/common/i18n/README.md`](../i18n/README.md)。

## 核心规则

| 操作 | 用什么 | 谁来做 |
| - | - | - |
| 定码 | `bkerrs.Define` 定义，`With` 附加参数后返回 | 第一个知道错误业务含义的层，通常是 service / 领域包 |
| 加上下文 | `github.com/pkg/errors` 的 `Wrap` / `Wrapf` | 任何一层，不带错误码 |
| 改判 | `ErrXxx.With(...).WithCause(err)` | 外层知道下层错误在当前场景下的真实含义时 |

- 定码与加上下文是两件事：**加上下文时不要用 `bkerrs.Wrap` / `Wrapf`**，它会新建一个带码错误盖住内层的错误码
- 出口用 `errors.As` 取错误链上**第一个** `*bkerrs.Error`：外层没有新建带码错误时，内层定的码会原样返回；`WithCause` 新建的错误排在最前，因此改判生效
- 链上没有任何 `*bkerrs.Error` 时，按错误声明的类别转换（见下文 store 约定）；仍未命中则返回 500 + `INTERNAL_ERROR`，并记录错误日志
- 通用组件（工具函数、外部系统客户端封装、解析器等）不定码，返回普通错误或哨兵错误，由调用方判断
- handler 可以定码（它是知道业务含义的那一层时），但必须**有条件**：先 `errors.Is` / `errors.As` 判断出具体错误，再用 `Define` 的错误 + `WithCause`；不要无条件地给所有错误套一个码

## 什么时候需要 Define

- **需要**：用户可能触发、需要返回 4xx 或特定提示的业务错误（重名、参数不合法、状态不允许、资源不存在等），以及上层需要 `errors.Is` 区分处理的业务错误
- **需要**：把链上已有的 4xx 错误改判为 5xx 时（见下文改判）。只用 `errors.Wrap` 会保留内层的 4xx
- **不需要**：系统故障类错误（数据库超时、外部系统调用失败等），保持普通错误并用 `errors.Wrap` 加上下文，出口自动按 500 处理并记日志

## 用法

以下示例均取自环境模块（`pkg/core/env`）。

### 定义错误

在错误产生的包内以包级变量定义，新增的错误详情码直接写在 `Define` 调用处：

```go
var ErrEnvHasDeployedAppsCannotUpdateCluster = bkerrs.Define(
	bkerrs.ErrCodeInvalidRequest,
	"CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER",
	"environment has deployed apps, cannot update cluster",
)
```

- 错误详情码全局唯一，重复或为空会在启动时 panic；建议以模块名为前缀（如 `CORE_ENV_`），避免冲突。已发布的码（前端或接口测试可能依赖）不要改名
- `err_detail_code.go` 中的常量是存量写法留下的，已有的码可以直接传给 `Define`，新增的码不要再加到这里
- `msg` 是给开发者看的英文描述，用户看到的文案来自 i18n 资源（见下文错误文案）
- 每个 `Define` 都必须在所属业务域的语言文件中提供同名文案，否则 `pkg/server` 的翻译完整性测试会失败

### 返回与判断

```go
// 返回：参数会输出到 details[].extras，并用于渲染用户文案
return ErrEnvHasDeployedAppsCannotUpdateCluster.With("appCount", strconv.Itoa(appCount))

// 加上下文：不影响错误码
return errors.Wrapf(err, "create env %s", input.Name)

// 判断：派生出的实例与定义满足 errors.Is
if errors.Is(err, envmodel.ErrEnvNotFound) { ... }

// 读取参数（如测试中断言）
params := bkerrs.FromError(err).Params()
```

- 直接返回 `Define` 的错误即可，**不要再为它写 `WrapXxx` 包装函数，也不要手工 `SetDetails`**：出口会根据错误详情码与参数自动生成一条详情（`code` 为错误详情码，`extras` 为参数，`message` 为完整错误链）。手工设置的详情会替换这条自动详情，出口还会额外追加一条以错误码为 `code` 的详情，造成重复
- 前端需要的结构化信息通过 `With` 传入，参数名即 `details[].extras` 的 key（如 `ErrEnvClusterNamespaceOccupied` 的 `clusterID`、`namespace`）

### 改判

同一个下层错误在不同场景含义不同时，才由外层改判。例如应用记录里引用的环境查不到，并非用户请求的资源不存在，而是数据不一致，应返回 500 而非 404：

```go
// ErrAppEnvInconsistent 为示意，实际使用时按场景 Define
env, err := envs.Get(ctx, store.ByID(app.EnvID))
if errors.Is(err, envmodel.ErrEnvNotFound) {
	return ErrAppEnvInconsistent.With("envID", app.EnvID).WithCause(err)
}
```

下层已经返回了正确的错误时，handler 不需要再判断，直接加上下文返回即可：

```go
if err = svc.Update(ctx, env.ID, updateData); err != nil {
	bkerrs.AbortWithErr(c, errors.Wrap(err, "update env cluster"))
	return
}
```

## store 约定

store 不依赖 `bkerrs`，不感知错误码。它的通用错误通过 `IsNotFound()` / `IsAlreadyExists()` 声明类别，出口据此定码：

| store 错误 | 声明的类别 | 错误码 | 错误详情码 | 文案 |
| - | - | - | - | - |
| `store.ErrNotFound` | `IsNotFound()` | `NOT_FOUND`（404） | `RESOURCE_NOT_FOUND` | 资源不存在 |
| `store.ErrDuplicate` | `IsAlreadyExists()` | `ALREADY_EXISTS`（409） | `RESOURCE_ALREADY_EXISTS` | 资源已存在 |

- 按类别定码只在出口生效，且优先级最低：错误链上已有 `*bkerrs.Error` 时以链上的为准；`bkerrs.FromError` 不受影响
- 其他通用组件中确属「资源不存在 / 已存在」的错误，也可以实现这两个方法，无需依赖 `bkerrs`
- `store.Meta` 的 `NotFound` / `Duplicate` 一般保持普通哨兵错误即可，供业务层 `errors.Is` 判断
- 需要比通用文案更具体的提示时，可以把 `Meta` 配置为 `Define` 定义的错误（如 `envmodel.ErrEnvAlreadyExists`），它在链上，优先于通用类别

## 错误文案

- 错误的 i18n key 即错误详情码（大写下划线，模块前缀），文案放在所属业务域的语言文件中（如 `core_env.yaml`）；
  `errors.yaml` 只放错误码通用文案、`RESOURCE_*` 类别文案，以及 `bkerrs` 包内定义的错误
- 新增 `Define` 必须同时新增文案；可执行 `make i18n` 把源码中缺失的 key 以骨架（英文预填 `Define` 的描述、中文留空）
  追加到所属模块的文件，再补全翻译
- 模板参数名与 `With` 的 key 一致。`pkg/common/i18n` 的单测会静态检查：各语言占位符一致、
  模板不引用调用处未传入的参数（`With` 的参数同时进入错误详情，允许只传不用）
- 业务代码只返回错误，翻译统一在出口完成

```yaml
CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER:
  zh: "环境已部署 {{.appCount}} 个应用，无法修改集群配置"
  en: "Environment has {{.appCount}} deployed app(s) and cannot update cluster configuration"
```

响应中顶层 `message` 与 `details` 的取值：

| 错误 | `message` | `details` |
| - | - | - |
| `Define` 定义 | 按错误详情码 + 参数翻译；缺翻译或参数时用错误码通用文案 | 一条自动生成的详情：错误详情码、参数与完整错误链 |
| 非 `Define` 定义的 5xx（含未分类错误） | 错误码通用文案，如"服务内部错误"，不暴露内部信息 | 原有详情 + 完整错误链 |
| 非 `Define` 定义的其他错误（存量写法） | 原始错误信息 `err.Error()`，不翻译 | 原有详情 |

## FAQ

##### 1. 忘记定码会怎样？

出口返回 500 + `INTERNAL_ERROR`，`message` 为"服务内部错误"，并在服务端记录包含路由与错误链的错误日志。若是用户输入导致的错误，应在知道业务含义的层补上 `Define`。

##### 2. ErrCodeInvalidArgument 和 ErrCodeInvalidRequest 怎么选？

如果是整个 `request.Validate`，选 `InvalidRequest`（不论是不是里面单个字段错误）。

如果是单独逻辑校验发现的某个字段有问题，比如 `objID` 不合法，选 `InvalidArgument`。

##### 3. ErrCodeIAMNoPermission 如何使用？

目前仍使用存量工具函数 `bkerrs.WrapIAMNoPermission`，调用时需提交 `workspaceID` 参数，其会自动填充权限中心用户组信息，便于用户在遇到没权限后自主申请。

##### 4. 什么时候记录日志？

出口统一记录：最终为 5xx 的请求记 error 日志（含完整错误链与调用栈），404 记 warn 日志。业务代码无需在返回错误前重复打印。
