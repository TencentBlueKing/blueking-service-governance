# i18n 使用指南

## 背景

`i18n` 以 gettext 的方式提供面向用户文案的国际化：源码中的英文原文就是翻译的 key（msgid），译文放在 `locales/<lang>.po`。
错误文案由 `bkerrs` 在出口统一翻译，规则见 [`pkg/common/bkerrs/README.md`](../bkerrs/README.md)；本文说明通用规则与非错误文案。

## 语言协商

- 中间件按 `blueking_language` cookie → `Accept-Language` → 中文 的顺序协商语言，结果放入请求的 `context`
- 业务代码只需传递 `ctx`，不要自行解析语言

## 文案资源

- 源码中的文案即英文（源语言），只有中文需要翻译文件 `locales/zh.po`，启动时内嵌加载
- 原文是 printf 格式串，参数沿用原文中的动词；译文需要调整参数顺序时使用 `%[n]s`：

```po
#: pkg/core/env/apierrs/apierrs.go
msgid "cluster %s namespace %s is already bound to environment %q in workspace %s"
msgstr "集群 %[1]s 的命名空间 %[2]s 已被工作空间 %[4]s 下的环境 %[3]q 绑定"
```

以下情况会直接启动失败：

- 同一个 msgid 重复
- 译文与原文的参数不一致（参数序号与动词须一一对应）

msgstr 为空或标记为 `fuzzy` 的条目视为未翻译，运行时回退到原文。修改原文等于换了 key，原有译文随之失效，需要重新翻译。

## 维护流程

1. 在代码中直接写英文原文，原文必须是字符串字面量
2. 运行 `make i18n`：扫描代码中的原文，把 `.po` 中缺少的追加为待翻译条目（msgstr 留空），并列出已不再使用的条目
3. 填写 msgstr，删除不再使用的条目

`make i18n` 提取以下位置的原文（内部错误码统一返回通用文案，其原文不提取）：

| 位置 | 原文 |
| - | - |
| `bkerrs.New` / `Errorf` / `Wrap` / `Wrapf` / `On` / `Otherwise` | `msg` / `format` |
| `i18n.T` / `Translate` / `Noop` | `format` / `msgid` |
| `store.Meta` 的 `NotFound` / `Duplicate` | 对应哨兵错误 `errors.New` 的文本 |

原文需要在别处才翻译时（如作为参数传递），用 `i18n.Noop` 标记以便提取。本包的 `usage_test.go` 检查 `.po` 中每条已翻译的原文都仍被代码引用。

## 谁来翻译

先按文案归属决定：

- **跟着界面交互走的文案**（如「卸载成功」这类操作结果提示、按钮、表单提示）：由前端 i18n 负责，后端不返回、也不翻译
- **跟着后端数据走的文案**（如随代码发布的内置插件描述）：由后端翻译，避免前端跟着后端的定义同步维护一份
- **用户输入的内容**：不翻译。即使初始值由系统写入，只要用户可以修改（如环境展示名），也按用户输入处理
- **不随代码发布的数据**（如部署时自行配置的内容）：无法保证有翻译，不走 i18n，直接展示原文

## 后端翻译非错误文案

成功响应不经过统一出口，翻译只在组装返回数据的地方做（handler / serializer，或直接生成用户可见内容的 service），model 与 store 不依赖 i18n。

- `i18n.T(ctx, format, args...)`：翻译并格式化，没有译文时使用原文
- `i18n.Translate(ctx, msgid)`：需要按是否有译文走不同分支时使用，没有译文时返回 `false`
