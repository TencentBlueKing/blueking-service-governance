# i18n 使用指南

## 背景

`i18n` 提供面向用户文案的国际化：按 key 查找内嵌的多语言模板，并按请求语言、以命名参数渲染。
错误文案由 `bkerrs` 在出口统一翻译，规则见 [`pkg/common/bkerrs/README.md`](../bkerrs/README.md)；本文说明通用规则与非错误文案。

## 语言协商

- 中间件按 `blueking_language` cookie → `Accept-Language` → 中文 的顺序协商语言，结果放入请求的 `context`
- 业务代码只需传递 `ctx`，不要自行解析语言

## 文案资源

文案放在 `locales/` 下，按业务域分文件（如 `core_env.yaml`），启动时加载，以下情况会直接启动失败：

- key 没有同时提供 `zh` / `en`
- key 在多个文件中重复
- 模板语法错误

```yaml
cluster_addon.addon.agones.description:
  zh: "Agones 是 Kubernetes 上的游戏服务器管理器，用于管理游戏服务器实例的部署、伸缩和调度。"
  en: "Agones is a Kubernetes-based game server manager for deploying, scaling, and scheduling game server instances."
```

- key 一经发布不再改名；按业务域命名，不按 Go 包路径命名，避免移动代码时 key 失效
- 错误的 key 是错误详情码（大写下划线）；非错误文案的 key 用小写点分（如 `cluster_addon.addon.agones.description`），两者不混用
- 模板参数用 `{{.name}}`，渲染时缺少参数视为翻译失败

## 谁来翻译

先按文案归属决定：

- **跟着界面交互走的文案**（如「卸载成功」这类操作结果提示、按钮、表单提示）：由前端 i18n 负责，后端不返回、也不翻译
- **跟着后端数据走的文案**（如随代码发布的内置插件描述）：由后端翻译，避免前端跟着后端的定义同步维护一份
- **用户输入的内容**：不翻译。即使初始值由系统写入，只要用户可以修改（如环境展示名），也按用户输入处理
- **不随代码发布的数据**（如部署时自行配置的内容）：无法保证有翻译，不走 i18n，直接展示原文

## 后端翻译非错误文案

成功响应不经过统一出口，翻译只在组装返回数据的地方做（handler / serializer，或直接生成用户可见内容的 service），model 与 store 不依赖 i18n。
业务代码统一使用 `i18n.T(ctx, key, params)`：

| 文案来源 | key 的写法 | 示例 |
| - | - | - |
| 后端代码中的固定文本 | 字符串字面量 | 暂无 |
| 随代码发布的内置数据（用户不能修改定义本身） | `前缀.` + 稳定标识 + `.字段` | 内置集群插件描述（`clusteraddon/serializer`），key 为 `cluster_addon.addon.<name>.description` |

`T` 在翻译失败时返回 key 本身，页面上会直接显示 key。翻译是否齐全由测试保证，漏翻会在测试中失败，而不是在运行时出现：

- 字面量 key：本包的扫描测试（`usage_test.go`）检查所有 `i18n.T` 调用
- 拼接的 key：无法静态扫描，由所在包的测试遍历定义表检查（如 `clusteraddon` 遍历 `assets/addons`）

需要按翻译结果走不同分支时，用 `TryT`，它在翻译失败时返回 `false`。例如：

- 错误出口：按详情码翻译失败时降级到错误码的通用文案
- 内置插件描述：插件定义目录可由部署配置指定，数据库中也可能残留已移出仓库的定义，这些插件没有内嵌翻译，回退到定义中的原文
