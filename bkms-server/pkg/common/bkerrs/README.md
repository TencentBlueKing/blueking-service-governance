# bkerrs 使用指南

## 背景

`bkerrs` 包提供创建 / 包装错误的方法，目的是返回特定类型的错误（`bkerrs.Error`），其会包含更具体的错误信息以便
`BkErrHandler` 处理成特定格式的响应内容（新版蓝鲸 HTTP API 协议）

## 分层约定

- **model / service / store**：只返回领域错误（哨兵错误或携带参数的错误类型），不依赖 `bkerrs`，不感知 HTTP 状态码
- **handler**：把领域错误映射为 `bkerrs.Error`，决定错误码、面向用户的原文与错误详情
- **错误出口**（`WriteError`）：定码、翻译、记录日志，业务代码不需要关心

## 映射领域错误

handler 用 `bkerrs.Map` 按顺序匹配规则，返回第一条命中规则生成的 API 错误；都不命中时原样返回，交给错误出口处理：

```go
bkerrs.AbortWithErr(c, bkerrs.Map(errors.Wrap(err, "update cfg switch"),
    bkerrs.On(model.ErrUnknownSwitch, bkerrs.ErrCodeInvalidArgument, i18n.Msg("unknown cfg switch key")),
))
```

| 规则 | 用途 |
| - | - |
| `On(target, code, format, args...)` | 错误链上存在哨兵错误 `target` 时包装为 `code` |
| `OnAs[T](build)` | 错误链上存在 `T` 类型的错误时，由 `build` 取出其中的参数生成错误 |
| `OnFunc(target, build)` | 需要附带错误详情等，无法用 `On` 表达的场景 |
| `Otherwise(code, format, args...)` | 放在末尾兜底；不需要兜底时省略 |

- 同一领域在多个 handler 中复用的规则放在该领域的 `apierrs` 包，如 `pkg/core/env/apierrs`（环境下仍有应用、命名空间被占用等）
- 领域错误需要携带参数时定义错误类型，由 `OnAs` 取出，如 `bkmsenv.EnvHasAppsError`
- 也可以直接返回 `bkerrs.Error`，如 `ginutils.WrapBSCPDisabled`；需要在多处匹配时再导出对应的规则（`ginutils.BSCPDisabledRule`）

### 资源不存在 / 已存在

`store` 返回的错误声明了错误类别（`IsNotFound` / `IsAlreadyExists`），错误出口在错误链上没有 `bkerrs.Error` 时自动转换：

| 类别 | 错误码 | 原文 |
| - | - | - |
| 不存在 | `NOT_FOUND`（404） | `Meta.NotFound` 的文本，未设置时为 `resource not found` |
| 已存在 | `ALREADY_EXISTS`（409） | `Meta.Duplicate` 的文本，未设置时为 `resource already exists` |

因此 handler 不需要再为 `ErrXxxNotFound` 写映射。以下情况仍需显式映射，映射后以外层的 `bkerrs.Error` 为准：

- 不存在 / 重名不应返回 404 / 409，例如内部查询失败、系统生成的名称重复（见 `apierrs.FeatureEnvNameConflict`）
- 需要附带错误详情或使用其他文案

## 错误出口

`WriteError` 按以下顺序处理：

1. **定码**：取错误链上最外层的 `bkerrs.Error`；没有时按上述存储类别转换；仍没有时为 `INTERNAL_ERROR`（500）
2. **文案**：
    - 内部错误（`INTERNAL_ERROR` / `INTERNAL_SERVER_ERROR`）统一返回通用文案，不暴露错误链
    - 其他错误翻译最外层 `bkerrs.Error` 的原文（`New` / `Wrap` 的 `msg`，`Errorf` / `Wrapf` 的 `format`），再用原参数格式化；
      内层错误面向开发者，不参与翻译
    - 原文没有译文时，非源语言（中文）使用错误码的默认文案（`codeMessages`）；源语言（英文）或错误码没有默认文案时
      沿用完整的错误信息，与引入 i18n 前一致。英文下只要原文出现在 `.po` 中（无论是否已翻译）就视为有译文
3. **详情**：文案被翻译或使用默认文案时，完整的错误信息作为一条详情追加到末尾（详情码同错误码），已有详情保持在前
4. **日志**：内部错误记录 Error 日志；按存储类别自动转换的 404 记录 Warn 日志，便于发现漏映射的内部查询

翻译的维护方式见 [`pkg/common/i18n/README.md`](../i18n/README.md)。面向用户的原文需用 `i18n.Msg("...")` 显式标记，
`make i18n` 才会提取；未标记或动态拼接（如 `WrapIAMNoPermission`）的原文按未翻译处理（中文下使用错误码的默认文案）。

## FAQ

##### 1. 反复调用 bkerrs.Wrap / Wrapf 会有什么影响

不会导致错误，但是下层包装的 bkerrs 会丢失其 details 信息（ErrCode 还在），且最后转换的 HTTP Status Code 以外层 `bkerrs.Error` 指定的 ErrCode 映射值为准。
翻译同样只取最外层 `bkerrs.Error` 的原文。

##### 2. 如果忘记使用 bkerrs 包装错误会有什么影响？

store 声明了类别的错误会按上文转换为 404 / 409；其他错误为 HTTP 500 + ErrCode INTERNAL_ERROR，返回通用文案，原始错误链放在详情中，并记录 Error 日志。

##### 3. 在什么时机使用 bkerrs.New / Errorf / Wrap / Wrapf？

在尽可能上层的位置（handler -> store/infras/utils），除非下层返回的数据是明确需要分错误类型的，否则越上层越好。

##### 4. ErrCodeInvalidArgument 和 ErrCodeInvalidRequest 怎么选？

如果是整个 `request.Validate`，选 `InvalidRequest`（不论是不是里面单个字段错误）。

如果是单独逻辑校验发现的某个字段有问题，比如 `objID` 不合法，选 `InvalidArgument`。

##### 5. ErrCodeIAMNoPermission 如何使用？

bkerrs 包已提供 IAMNoPermission 专用的工具函数 `bkerrs.WrapIAMNoPermission`，调用时需提交 `workspaceID` 参数，其会自动填充权限中心用户组信息，便于用户在遇到没权限后自主申请。
