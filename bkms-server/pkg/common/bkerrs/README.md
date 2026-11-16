# bkerrs 使用指南

## 背景

`bkerrs` 提供已分类错误 `bkerrs.Error`：错误码（`ErrCode`，决定 HTTP 状态）+ 错误详情码（`ErrDetailCode`，稳定的机器可读键，
输出为 `details[].code`，同时是 i18n key）+ 参数。`ErrorHandler` / `WriteError` 在出口把它转换为新版蓝鲸 HTTP API 协议的响应，并按请求语言翻译 `message`。

设计背景见 `design_notes/error_code_i18n.md`。

## 核心规则

| 操作 | 用什么 | 谁来做 |
| - | - | - |
| 定码 | `bkerrs.Define` 定义，`With` 附加参数后返回 | 第一个知道错误业务含义的层（通常是 service / 领域包；store 的通用错误由出口按其声明的类别定码，见下文 store 约定） |
| 加上下文 | `github.com/pkg/errors` 的 `Wrap` / `Wrapf`，或 `fmt.Errorf("...: %w", err)` | 任何一层，不带错误码 |
| 改判 | `ErrXxx.With(...).WithCause(err)` | 外层知道下层错误在当前场景下的真实含义时 |

- 定码与加上下文是两件事：**加上下文时不要用 `bkerrs.Wrap` / `Wrapf`**，它会新建一个带码错误盖住内层的错误码
- 出口用 `errors.As` 取错误链上**第一个** `*bkerrs.Error`：外层没有新建带码错误时，内层定的码会原样返回；`WithCause` 新建的错误排在最前，因此改判生效
- 链上没有任何 `*bkerrs.Error` 时，按错误声明的类别转换：实现 `IsNotFound() bool` 返回 404，实现 `IsAlreadyExists() bool` 返回 409（如 `store.ErrNotFound`）；仍未命中则返回 500 + `INTERNAL_ERROR`，并记录错误日志
- 通用组件（工具函数、外部系统客户端封装、解析器等）不定码，返回普通错误或哨兵错误，由调用方判断；确属「资源不存在 / 已存在」的通用错误可以实现上述标记方法，无需依赖 `bkerrs`
- handler 可以定码（它是知道业务含义的那一层时），但必须**有条件**：先 `errors.Is` / `errors.As` 判断出具体错误，再用 `Define` 的错误 + `WithCause`；不要无条件地给所有错误套一个码

## 什么时候需要 Define

- **需要**：用户可能触发、需要返回 4xx 或特定提示的业务错误（重名、参数不合法、状态不允许、资源不存在等），以及上层需要 `errors.Is` 区分处理的业务错误
- **不需要**：系统故障类错误（数据库超时、外部系统调用失败等），保持普通错误并用 `errors.Wrap` 加上下文，出口自动按 500 处理并记日志；通用组件内部的错误也不需要，由调用方判断

改造前后的完整链路对比（错误码为何会被覆盖、Define 解决了什么）见 `design_notes/error_code_i18n.md` 4.3 节。

## 用法

### 定义错误

在错误产生的包内以包级变量定义，错误详情码直接写在 `Define` 调用处，不需要在 `err_detail_code.go` 中声明常量：

```go
var ErrItemNameDuplicated = bkerrs.Define(
	bkerrs.ErrCodeInvalidArgument, "PORTPOOL_ITEM_NAME_DUPLICATED", "portpool item name duplicated",
)
```

- 错误详情码全局唯一，重复或为空会在启动时 panic；建议以模块名为前缀（如 `PORTPOOL_`），避免冲突
- `err_detail_code.go` 只保留 `NewDetail` 手工构造详情时使用的、跨包共享的详情码
- `msg` 是给开发者看的英文描述，用户看到的文案来自 i18n 资源（见下文）
- 定义是共享的，`With` / `WithCause` / `SetDetails` 都返回副本，不会修改定义本身

### 返回与判断

```go
// 返回：参数会输出到 details[].extras，并用于渲染用户文案
return ErrItemNameDuplicated.With("itemName", name)

// 加上下文：不影响错误码
return errors.Wrapf(err, "create portpool %s", cfg.Name)

// 判断：派生出的实例与定义满足 errors.Is
if errors.Is(err, portpool.ErrConfigNotFound) { ... }

// 取错误码 / 错误详情码
if bkErr := bkerrs.FromError(err); bkErr != nil {
	_ = bkErr.Code()
	_ = bkErr.DetailCode()
}
```

### 改判

同一个下层错误在不同场景含义不同时，由外层改判。例如刚从列表查到的端口池再取详情时不存在，说明数据不一致，应返回 500 而非 404：

```go
// ErrDataInconsistent 为示意，实际使用时按场景 Define
cfg, err := svc.Get(ctx, env, name)
if errors.Is(err, portpool.ErrConfigNotFound) {
	return ErrDataInconsistent.With("portpool", name).WithCause(err)
}
```

### handler

handler 只加上下文，不挑错误码：

```go
if err != nil {
	bkerrs.AbortWithErr(c, errors.Wrap(err, "create portpool config"))
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
- `store.Meta` 的 `NotFound` / `Duplicate` 一般保持普通哨兵错误即可，供业务层 `errors.Is` 判断，不需要逐个用 `Define` 定义
- 需要比通用文案更具体的提示时：可以把 `Meta` 配置为 `Define` 定义的错误（如 `envmodel.ErrEnvAlreadyExists`），它在链上，优先于通用类别；需要带参数（资源名称等）时由 service 层改判：`ErrEnvAlreadyExists.With("name", name).WithCause(err)`
- 内部依赖数据查不到时（并非用户请求的资源不存在），出口给出的 404 是错的，需在边界上改判为 500

## i18n

文案资源在 `pkg/common/i18n/locales/` 下，按模块分文件（如 `core_app.yaml`、`workload.yaml`、`extension.yaml`）；
`errors.yaml` 只放 `bkerrs` 自身的错误码通用文案、`RESOURCE_*` 类别文案与 `bkerrs` 内置工具函数的文案。
每个 key 必须同时提供 `zh` / `en`，且不能跨文件重复，否则启动时 panic：

```yaml
PORTPOOL_ITEM_NAME_DUPLICATED:
  zh: "端口池条目名称 {{.itemName}} 重复"
  en: "Port pool item name {{.itemName}} is duplicated"
```

- **新增 `Define` 必须同时在所属模块的文件中新增文案**，否则 `pkg/server` 的翻译完整性测试会失败
- 错误文案的 key 即错误详情码（大写下划线，模块前缀）；非错误的展示文案用小写点分 key（如 `workload.envvars.scopedBuiltin.<KEY>.description`），两者不混用
- 模板参数名与 `With` 的 key 一致；运行时缺少参数时回退到错误码通用文案
- 语言按 `blueking_language` cookie → `Accept-Language` → 中文 的顺序协商

### 非错误的展示文案

接口返回的固定展示文本（枚举名称、内置变量说明、告警提示等）同样不能硬编码某一种语言：

| 场景 | 用法 | 翻译失败时 |
| - | - | - |
| 代码中的固定文本 | `i18n.LocalizeOr(ctx, key, params, fallback)` | 返回 `fallback` |
| 系统写入存储的默认值（默认环境名、内置策略名、初始版本描述等） | `i18n.LocalizeStored(ctx, key, params, stored)`，key 由记录的稳定标识（如内置项 ID、环境类型）拼出 | 存储值与该 key 的中文文案不一致（用户改过）或 key 不存在时，原样返回存储值 |

- 写入存储的默认值统一使用中文文案（与 `LocalizeStored` 的比对基准一致），读取时再按请求语言翻译
- 用户输入的内容不翻译

响应中顶层 `message` 的取值：

| 错误 | `message` | `details` |
| - | - | - |
| `Define` 定义 | 按错误详情码 + 参数翻译；缺翻译或参数时用错误码通用文案 | 包含错误详情码、参数与完整错误链 |
| 非 `Define` 定义的 5xx（含未分类错误） | 错误码通用文案，如"服务内部错误"，不暴露内部信息 | 原有详情 + 完整错误链 |
| 非 `Define` 定义的其他错误（存量写法） | 原始错误信息 `err.Error()` | 原有详情 |

## 存量写法（迁移中）

- `bkerrs.Wrap` / `Wrapf`：新代码不要使用。4xx 的存量调用已全部改为 `Define`；仍有 `ErrCodeInternal*` 的存量调用，修改到所在代码时替换为 `errors.Wrap` / `Wrapf`。它们排在错误链外层时会盖住内层 `Define` 的 4xx，出现「本应 4xx 却返回 500」时优先排查这里
- `bkerrs.New` / `Errorf`：没有错误详情码，无法翻译，新代码优先用 `Define`
- 请求参数绑定失败统一为 `ginutils.ErrInvalidRequestParams`（`REQUEST_PARAMS_INVALID`），校验器给出的具体原因（英文）保留在 `details` 中，尚未逐字段翻译
- 接口测试（`tests/apis/**/*.bru`）断言 `error.code` 与 `error.details[0].code`；需要定位具体原因时断言 `error.details[0].message` 中的原始错误，不要断言会随语言变化的 `error.message`

## FAQ

##### 1. 忘记定码会怎样？

出口返回 500 + `INTERNAL_ERROR`，`message` 为"服务内部错误"，并在服务端记录包含路由与错误链的错误日志。若是用户输入导致的错误，应在知道业务含义的层补上 `Define`。

##### 2. ErrCodeInvalidArgument 和 ErrCodeInvalidRequest 怎么选？

如果是整个 `request.Validate`，选 `InvalidRequest`（不论是不是里面单个字段错误）。

如果是单独逻辑校验发现的某个字段有问题，比如 `objID` 不合法，选 `InvalidArgument`。

##### 3. ErrCodeIAMNoPermission 如何使用？

使用专用的工具函数 `perm.WrapIAMNoPermission`（`pkg/infras/perm`），调用时需提交 `workspaceID` 参数，其会自动填充权限中心用户组信息，便于用户在遇到没权限后自主申请。

##### 4. 什么时候记录日志？

出口统一记录：最终为 5xx 的请求记 error 日志（含完整错误链与调用栈），404 记 warn 日志。业务代码无需在返回错误前重复打印。
