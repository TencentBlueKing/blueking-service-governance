# bkerrs 使用指南

handler 用 `bkerrs` 决定**错误码**与**给用户看的文案**；`bkerrs.ErrorHandler` 中间件（错误出口）统一把错误渲染为蓝鲸 HTTP API 错误格式、
按请求语言翻译文案、记录日志。

## 分层

- **model / service / store**：只返回领域错误（哨兵错误或携带参数的错误类型），用 `errors.Wrap` 补充上下文，不依赖 `bkerrs`
- **handler**：通过 `bkerrs` 的快捷函数或规则决定错误码与文案
- **错误出口**：定码、翻译、记录日志，业务代码不需要关心

## handler 中如何返回错误

按顺序判断，使用第一个符合的写法：

| 场景 | 写法 |
| - | - |
| 错误已由下层定好错误码，如 `ginutils.BindURI`、`perm.ValidateXxx` | `AbortWithErr(c, err)`，原样透传，不要再包装 |
| **默认**：service / store 调用返回的错误 | `AbortWithErr(c, errors.Wrap(err, "<在做什么>"))`，不需要选错误码：store 的"不存在 / 已存在"返回 404 / 409，其余为 500 |
| 部分领域错误应返回特定的 4xx | 同上再传入规则：`AbortWithErr(c, errors.Wrap(err, "<在做什么>"), rules...)`，见[规则](#规则) |
| 错误整体表示用户输入不合法 / 用户请求的资源不存在 | `AbortInvalidArg` / `AbortNotFound(c, err, i18n.Msg("<用户文案>"))` |
| 失败不可能由用户造成，如调用下游接口、按系统生成的 ID 查询 | `AbortInternal(c, errors.Wrap(err, "<在做什么>"))` |
| 同上，但部分领域错误仍要按规则返回 4xx，如系统创建默认环境时命名空间被占用 | `AbortWithErr(c, bkerrs.Wrap(err, ErrCodeInternalServerError, "<在做什么>"), rules...)` |

`errors` 指 `github.com/pkg/errors`。快捷函数的第三个参数一定是用户文案；`AbortInternal` 没有这个参数，因为内部错误统一返回通用文案，开发者上下文和默认写法一样用 `errors.Wrap` 附加。

判断是否用 `AbortInternal` 时问一句："用户传了错误的 ID 时会不会走到这里？"会的话用默认写法，否则用户会收到 500 而不是 404。
现有代码中大量 `bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "...")` 是历史写法，新代码不要照抄。

## msg 怎么写

`bkerrs` 中的字符串分为两种，写法不同：

| | 用户文案 | 开发者上下文 |
| - | - | - |
| 出现在 | `AbortInvalidArg` / `AbortNotFound` 的 `msg`；`WhenIs` / `Errorf` / `Wrapf` 的 `format`；错误码不是内部错误的 `New` / `Wrap` 的 `msg` | `errors.Wrap` / `Wrapf`；错误码为内部错误的 `New` / `Wrap` |
| 谁会看到 | 用户（作为响应的 `message`） | 只出现在日志与错误详情中 |
| 怎么写 | 站在用户角度、完整描述问题的一句话 | 简短说明"在做什么"，用 `Wrapf` 带上标识 |
| `i18n.Msg` | **必须标记**，并运行 `make i18n` 补充 `zh-cn.po` 译文 | **不要标记** |
| 示例 | `i18n.Msg("invalid version ID")`、`i18n.Msg("environment has %d deployed apps, cannot delete")` | `"create default env %s"`、`"list bscp user bizs"` |

用户文案不要写成开发者上下文那样的半句话：

```go
// ✗ 英文下用户看到 "parse version ID: the provided hex string is not a valid ObjectID"，中文下只能显示默认文案"请求参数不合法"
bkerrs.AbortInvalidArg(c, err, "parse version ID")

// ✓ 用户看到"版本 ID 不合法"，完整错误链仍在错误详情中
bkerrs.AbortInvalidArg(c, err, i18n.Msg("invalid version ID"))
```

- 动态值作为参数传入（`Errorf` / `Wrapf` / `WhenIs` 的 `args`），不要用 `fmt.Sprintf` 预先拼接，否则无法匹配译文；参数本身不翻译
- `NewDetail` 的 message 作为错误详情原样返回，不翻译
- `store.Meta` 的 `NotFound` / `Duplicate` 哨兵错误文本由 `make i18n` 自动提取，不需要标记

翻译文件的维护见 [`pkg/common/i18n/README.md`](../i18n/README.md)。

## 错误码优先级

错误出口按以下顺序定码，取第一个成立的：

| 优先级 | 错误链上有 | 结果 | 典型来源 |
| - | - | - | - |
| 1 | 命中全局规则的错误 | 全局规则的错误码，即使 handler 已包装为其他错误码 | `bscp.ErrDisabled` |
| 2 | `bkerrs.Error` | **最外层**那个的错误码、文案与详情，内层的全部失效 | 快捷函数、命中的规则、`New` / `Wrap` |
| 3 | store 声明的类别 | 不存在为 404，已存在为 409 | `store` 的 `ErrXxxNotFound` / 唯一键冲突 |
| 4 | 以上都没有 | 500，返回通用文案 | 未处理的错误 |

由此可以推出：

- `AbortInternal` 在最外层包了 500，store 的类别不再转换；全局规则仍然生效
- 规则生成的错误在最外层，会覆盖 handler 预先包好的 500，见上表最后一种写法
- handler 给出的错误码与全局规则相同时，保留 handler 的错误（如自定义的文案与详情）

## 规则

规则（`Rule`）在错误链命中时返回 API 错误，未命中时返回 nil。传给 `AbortWithErr` 的多条规则按顺序匹配，第一条命中的生效：

```go
bkerrs.AbortWithErr(c, errors.Wrap(err, "update cfg switch"),
    bkerrs.WhenIs(model.ErrUnknownSwitch, bkerrs.ErrCodeInvalidArgument, i18n.Msg("unknown cfg switch key")),
)
```

| 函数 | 用途 |
| - | - |
| `WhenIs(target, code, format, args...)` | 错误链上有哨兵错误 `target` 时，包装为 `code` 与用户文案 |
| `WhenIsFunc(target, build)` | 同上，由 `build` 生成错误，用于附带错误详情 |
| `WhenAs[T](build)` | 错误链上有 `T` 类型的错误时，由 `build` 取出其中的参数生成错误，如 `bkmsenv.EnvHasAppsError` |
| `Rules(rules...)` | 把一组规则组合为一条，便于命名与复用 |
| `Match(err, rules...)` | 只匹配、不中止请求，如在测试中断言映射结果 |

签名为 `func(err error) *bkerrs.Error` 的普通函数也可以直接作为规则，如 `envapierrs.NamespaceOccupied`。

规则放在哪里：

| 范围 | 位置 | 示例 |
| - | - | - |
| 只在一个 handler 包中使用 | handler 包内的私有变量 | `cfgswitch/handler` 的 `cfgSwitchRules` |
| 同一领域的多个 handler 复用 | 该领域的 `apierrs` 包 | `pkg/core/env/apierrs`、`pkg/core/app/appcfg/apierrs` |
| 外部系统 client 的错误，多个领域复用 | client 包下的 `apierrs` 包 | `pkg/infras/cloudapi/bscp/apierrs` |
| 在任何接口中含义都相同 | 规则定义在上述 `apierrs` 包，登记到 `ginutils.GlobalErrorRules`，由 `server.go` 注册到 `ErrorHandler` | BSCP 未启用 |

全局规则优先于 handler 的映射，影响所有接口，只有错误码与文案对任何调用场景都成立时才能加入。
反例：BSCP 返回的 401 / 403 可能是用户无业务权限，也可能是平台凭证或网关授权问题，因此 `bscpapierrs.NoPermission`
只在用户浏览自己 BSCP 资源的接口中显式使用。

只映射确定由用户输入导致的错误，不要把调用下游失败等内部错误映射为 4xx。store 的"不存在 / 已存在"已自动转换，
只有不应返回 404 / 409（如系统生成的名称重复，见 `envapierrs.FeatureEnvNameConflict`）或需要其他文案、详情时才写规则。

## 错误出口的行为

- **文案**：内部错误一律返回通用文案。其他错误翻译最外层 `bkerrs.Error` 的原文；没有译文时，中文使用错误码的默认文案，
  英文沿用完整的错误信息
- **详情**：`details[].message` 只给开发者看，不翻译，不要用 `i18n.Msg` 标记；前端只按 `details[].code` 判断错误类型，展示用顶层 `message`。
  返回翻译后的文案或默认文案时，完整的错误信息追加为最后一条详情（`code` 为错误码本身），便于排查；已有详情保持在前，前端按 `details[0]` 取值不受影响
- **日志**：内部错误记录 Error 日志；由 store 类别自动转换的 404 记录 Warn 日志，便于发现漏映射的内部查询

## 注意事项

- 不要写 `bkerrs.New(code, err.Error())`：丢弃了原始错误链，文案也无法翻译。改用 `WhenIs` 规则或 `Wrap(err, code, i18n.Msg("..."))`
- 不要给开发者上下文加 `i18n.Msg`：不会展示给用户，只会产生无用的翻译条目
- 不要把已经是 `bkerrs.Error` 的错误再包装为其他错误码：内层的错误码、文案与详情都会丢失
- 不要在 model / service / store 中使用 `bkerrs`：错误码由 handler 决定

## FAQ

##### 1. ErrCodeInvalidArgument 和 ErrCodeInvalidRequest 怎么选？

整个请求不满足业务规则（如 `request.Validate` 失败、状态不允许当前操作）选 `InvalidRequest`；
单个参数本身不合法（如 ID 格式错误）选 `InvalidArgument`。

##### 2. ErrCodeIAMNoPermission 如何使用？

使用 `bkerrs.WrapIAMNoPermission`，传入 `workspaceID`，它会填充权限中心的用户组信息，便于用户自主申请权限。
