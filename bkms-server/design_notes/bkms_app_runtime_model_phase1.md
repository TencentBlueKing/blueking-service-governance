# BKMSApp 运行时能力集合（runtime mod）：M1 方案

状态：已按 PR 评审意见调整。声明改为函数式，能力范围收紧为"只描述因技术栈不同而变化的部分"，具体能力与基础框架分目录。

对应父文档 [bkms_app_runtime_model.md](./bkms_app_runtime_model.md) 的阶段 2，依赖已冻结的 [阶段 0](./bkms_app_runtime_model_phase0.md) 与 [里程碑](./bkms_app_runtime_model_milestones.md) M1。

本文件只解决两件事：**哪些功能算 Mod、怎样声明**，以及**能力代码如何划分**。API、前端、数据迁移不在本阶段范围。

---

## 1. 决策

| 问题 | 结论 |
| --- | --- |
| 什么算 Mod | 支持与否**或**参数因 framework / language 不同而变化的功能。对所有技术栈一致的功能属基座，不进 ModSet |
| 声明形式 | 一个解析函数 `func(framework, language) Mod[P]`。简单能力用工厂函数，复杂能力自行实现 |
| 解析位置 | 编译进代码；`Of(app)` 每次现算，不落库、不缓存 |
| 组装方式 | 强类型 `Set`，集中组装；不使用 init / FX 动态注册 |
| 解析失败语义 | 技术栈非法 → error；能力不支持 → `Supported=false` + `reasonCode` |
| 语言为空 | 允许（桥接期）；按框架的能力照常支持，按语言的能力报 `incomplete_language` |
| Mod ID | 常量集中定义，不使用字面量 |

---

## 2. 声明与解析

包：`pkg/core/appruntime/runtimemod`。只依赖 `appruntime` 与标准库。

```go
type Spec[P Props] struct {
    ID      ModID
    Resolve func(framework, language string) Mod[P]
}

// 按框架查表：表中有的框架支持，与语言无关，语言为空也支持
func ByFramework[P Props](table map[string]P) func(string, string) Mod[P]
// 按语言查表：表中有的语言支持；语言为空时无法判断，返回 incomplete_language
func ByLanguage[P Props](table map[string]P) func(string, string) Mod[P]
```

简单能力一行即可：

```go
var FrameworkFile = runtimemod.Spec[FrameworkFileProps]{
    ID: runtimemod.ModFrameworkFile,
    Resolve: runtimemod.ByFramework(map[string]FrameworkFileProps{
        appruntime.FrameworkTRPC: {Format: FrameworkFileYAML},
        appruntime.FrameworkTAF:  {Format: FrameworkFileTAF},
    }),
}
```

参数随框架与语言组合变化的能力自行实现，例如 adminCommand 按语言选配置键布局、platformBuild 限定 trpc + go。**不引入 Pair 表、优先级算法、语言匹配器**：这类组合逻辑直接写在解析函数里，读起来就是它的实际行为。

解析规则：

1. `Type != bkmsApp` → 所有能力 `unsupported_app_type`，在 `Spec.ResolveStack` 中统一处理。
2. 技术栈本身非法（未知框架、未知语言、非法组合）由 `Of` / `Resolve` 返回 error，不进入能力解析。
3. 能力不支持不是错误；执行器调用 `Mod.Require()` 时才转换为 `ErrModUnsupported`。
4. `reasonCode` 取值：`unsupported_app_type`、`unsupported_stack`、`incomplete_language`。

### 2.1 技术栈来源

`mods.Of(app)` 读取 Application 字段，经 `appruntime.CapabilityStack` 规范化为技术栈后解析：

```go
stack, err := appruntime.CapabilityStack(app.RuntimeSnapshot())
```

`CapabilityStack` 与 `Normalize` 的区别是缺少语言不算错误：存量 TAF 应用都没有记录语言，部分 tRPC 应用的语言只记录在 workload 上。技术栈本身的冲突（类型与框架不一致、语言冲突、非法组合）仍然报错。

---

## 3. 能力目录

8 个。

| ID | 支持的框架 / 语言 | props（执行器用） |
| --- | --- | --- |
| `frameworkFile` | trpc（yaml）、taf（taf） | format |
| `devMode` | trpc、taf | scriptSet、workPath、mountPath、binaryPath |
| `platformBuild` | trpc + go | dockerfileLanguage |
| `adminCommand` | trpc（按语言选布局）、taf | protocol、configLayout |
| `apmServiceName` | trpc（yaml）、taf（xml） | parser |
| `polarisServiceCheck` | trpc | — |
| `polarisFrameworkPatch` | trpc | — |
| `bscpWorkloadMeta` | trpc | —（工作负载类型取实际渲染结果） |

`adminCommand` 登记 go / cpp / java / nodejs / python 五种布局：go、python 读 `server.admin.ip`，cpp、nodejs 读 `server.admin_ip`，java 读 `server.admin.admin_ip`。java、nodejs、python 不在框架目录中，暂不可达，纳入目录后自动生效。

### 3.0 不属于 ModSet 的功能

判据是**是否因技术栈不同而变化**，不是"是否重要"。

| 归属 | 典型功能 | 处理 |
| --- | --- | --- |
| 基座：所有技术栈一致 | plain 配置文件、一键构建部署、构建配置、构建记录、监控、集成类接口 | 直接在原模块判断，或按 `IsAppModelType` 判断 |
| 管理方式 | 部署、预检、状态、总览、实例、AppSpec、组件、环境变量 | `IsAppModelType` |
| 环境 / 集群 | GPA（需 AppModel 且非联邦）、HostPort（看联邦）、端口池、集群 addon | 由环境或集群属性决定 |
| 框架适配器 | 渲染插件的选择（trpcPlugin / tafPlugin 等） | 按 framework 取适配器，不是能力问题 |
| 独立链路 | Helm chart 来源、values、Helm 部署、chart 构建 | Helm / Agones 专属 |

**Mod 之间不互相调用。** 执行器可以同时读取多个 Mod；已知的依赖关系（如北极星注入依赖框架配置文件）由测试约束，不引入依赖字段。

### 3.1 props 要点

- `devMode`：路径与脚本集由 props 给出，不再从 `app.Type` / `Workload.Type` 推导；启动命令怎么传属于脚本集自身的行为。
- `platformBuild`：`DockerfileLanguage` 替代 `Application.TrpcSpec.Language` 的读取。
- `frameworkFile` 的 format 取值与存储层 `appcfg.FileFormat` 一致（`yaml` / `taf`），执行端不需要转换。
- 执行器内部 key（scriptSet、parser、configLayout、dockerfileLanguage）**不对外暴露**。

### 3.2 命名

`blank` 为 framework 字面值；`standard` 仅保留为内部 workload 插件名。

---

## 4. 对外 JSON

`Set.Public()` 的输出即后续 API 响应体，本阶段用 golden 测试锁定。

```json
{
  "schemaVersion": 1,
  "type": "bkmsApp",
  "framework": "trpc",
  "language": "cpp",
  "mods": {
    "frameworkFile":         { "supported": true,  "props": { "format": "yaml" } },
    "devMode":               { "supported": true,  "props": { "workPath": "/data/bkms/dev-mode/trpc", "mountPath": "/data/bkms/dev-mode/trpc/configmap-scripts" } },
    "platformBuild":         { "supported": false, "reasonCode": "unsupported_stack" },
    "adminCommand":          { "supported": true },
    "apmServiceName":        { "supported": true },
    "polarisServiceCheck":   { "supported": true },
    "polarisFrameworkPatch": { "supported": true },
    "bscpWorkloadMeta":      { "supported": true }
  }
}
```

规则：

- `mods` 用 map，`encoding/json` 按键排序，输出稳定。
- `props` 为空时整个键省略，不输出 `null`。
- `framework` / `language` 为空时省略键。
- helm / agones：`framework`、`language` 省略，全部 `unsupported_app_type`。
- `Matrix()` 对 catalog 全组合生成同样结构，供前端创建表单使用（挂 API 属 M6）。

---

## 5. 校验与测试

不设启动期校验。约束由测试承担。

M1 交付：

| 测试 | 保证 |
| --- | --- |
| golden JSON（5 组合 + Matrix） | 支持矩阵与对外契约不随重构漂移 |
| `Set` 字段与 `Public()` 输出对齐（反射） | 新增能力漏进 `all()` 时立即失败 |
| 工厂函数单测 | ByFramework / ByLanguage 的空语言、未知值语义 |
| import 约束 | `mods` 只依赖 appruntime、runtimemod、core/app，不引入其他领域包 |

M2 交付：执行器覆盖测试（各能力所在模块），保证凡 `Supported=true` 的组合都能构造执行器。

---

## 6. 能力代码的 pkg 划分

```
pkg/core/appruntime/runtimemod/        # 基础：ModID、ReasonCode、Mod、Spec、工厂、对外结构
pkg/core/appruntime/runtimemod/mods/   # 具体能力声明、Set、Of(app)
```

划分原则：基础包只放与具体能力无关的类型和工具；具体能力集中在 `mods`，每个能力一个文件。M3 再把各能力连同声明搬到 `pkg/capability/<能力>/`，届时 `Set` 的组装独立成一个薄包。

---

## 7. 实施

| 步骤 | 内容 | 状态 |
| --- | --- | --- |
| M1 | 声明、`Set`、`Of(app)`、测试、文档 | 本次 |
| M2 | 8 个能力的执行端改读 `Set` | 待做 |
| M3 | 能力搬到 `pkg/capability/*` | 待做 |

M2 的两个已知行为点：

- AppSpec 的 `workPath` / `mountPath` 今日实际不生效（`devmode.New` 会再次覆盖），props 作为唯一来源与现状一致；字段暂按原样读写，M9 清理。
- 路径来源今日不一致：`FromAppModel` 看 `Workload.Type`，其余三处看 `app.Type`。统一到 Application 后属修复，上线前用 `scan_app_runtime_model` 统计"taf 且 Workload.Type 为空"的记录数。
- `frameworkFile` 当前统一按 YAML 校验内容，含 `key: value` 行的合法 TAF 配置会被误拦，M2 按 props.format 选择校验器时一并修复。

---

## 8. 验收

- 8 个能力均有声明，`Matrix()` 覆盖 catalog 全部合法组合，5 份 golden JSON 通过。
- `trpc+cpp` 支持 devMode、不支持 platformBuild；`blank+go` 全部能力不支持；helm 全部 `unsupported_app_type`。
- M2 完成后，能力判断不再读取 `AppType` / `Workload.Type` / `TrpcSpec.Language`。
- `make lint`、相关包测试通过。
