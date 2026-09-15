# AppModel 应用类型体系

## 应用模型

AppModel 族只有一种应用，不再按框架分类型。构成如下：

```jsonc
{
  "id": "demo-app-a1b2c3",
  "workspaceID": "bkms-workspace",
  "name": "demo-app",
  "displayName": "demo-app",
  // 唯一值，不参与任何功能判别（名字待确认）
  "type": "appmodel",

  // 编程语言：只影响构建
  "language": "go",
  // 编程框架：只影响差异能力；blank = 无预设框架
  "framework": "trpc-go",
}
```

- `language` 是构建必备信息，只参与构建。
- `framework` 只参与「差异能力」，取值映射到框架预设。

> 说明：`buildConfig` 与 `configFiles` 不属于 Application 模型本身，各自独立存储、通过 `appID` 关联：
> - 构建配置 → 构建配置表（build_configs）
> - 配置文件 → plain 配置表（app_config_file_defs / app_config_files / app_config_file_versions）

## 框架预设（Framework Preset）

`framework` 字段的值映射到一个**预设**。预设是一份可注册、可扩展的数据，决定两件事：

1. 框架配置文件以及文件格式
2. 启用哪些治理能力

| framework | 配置文件格式 | 启用的治理能力 |
|-----------|------------|--------------|
| `trpc-go` | yaml | admin / devmode / apm |
| `trpc-cpp` | yaml | admin / devmode / apm |
| `taf` | xml | admin / devmode / apm |
| `blank` | 无 | 无 |

预设的落地方式：

- 新增框架 = 注册一个新预设（如 Python 的 `blueapps`），不改功能代码分支。
- 各功能模块只读取 `framework` 预设，不读取 `type`、不读取 `language`。
- 预设内容数据驱动：配置文件格式、治理能力清单，都由预设数据决定。

## 能力分层与建设

能力分三类，各自有不同的落位与启用方式：

### 1. 通用能力（内建于基座，所有 app 适用）

组件管理、构建、配置文件、特性环境、可观测 + APM、镜像管理、实例管理、port-forward。

- 不读 `type`，不读 `framework`。
- 作为 AppModel 基座的一部分，任何 framework / language 都可用。

### 2. 开放能力（可叠加，数据驱动）

北极星、BSCP（动态配置管理）、GPA（自动扩缩容）、联邦集群 + hostPort、端口池、devmode。

- 作为「可叠加能力」，按需启用，不硬编码在某类 app 上。
- 建设原则：**尽可能少依赖 appType 判断**，靠能力自身的数据配置决定是否生效、如何生效。
- **模块内部数据驱动**：能力的行为由配置数据（而非代码分支）决定。

### 3. 框架特有（按框架预设启用）

admin（管理命令）。

- 唯一真正与框架绑定的能力，通过 `framework` 预设启用（如 trpc-go / trpc-cpp / taf 有 admin，blank 无）。

### 建设原则落地

1. 新能力先判定归属：通用 / 开放 / 框架特有。
2. 除框架特有外，一律按「支持所有类型 app」设计，能力启用与行为走**数据配置**而非 `appType` 分支。

## 配置文件

- 通用配置文件使用 **plain 方式**：同一应用可多个、可选挂载环境（1:n）、存储完整内容、按环境配置（可一键恢复默认）、可移除挂载。
- 框架配置文件由 `framework` 预设决定格式：`trpc-go` / `trpc-cpp` = yaml，`taf` = xml，`blank` 无框架配置文件。

## 构建

三种构建方式：`codeRepository`（源码仓库）/ `imageRegistry`（镜像仓库）/ `pipeline`（流水线）。

平台通用构建（platform）按 framework / language 判定：`trpc`（trpc-go / trpc-cpp）支持，`taf` 不支持。
