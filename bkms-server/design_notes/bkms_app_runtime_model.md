# bkmsApp 应用模型与运行时能力集合

本文介绍平台托管应用（`bkmsApp`）的数据模型、运行时能力集合（ModSet）的设计、相关包的分工与扩展方式。
存量迁移、旧 API 兼容与待删除事项见 [bkms_app_runtime_model_migration.md](./bkms_app_runtime_model_migration.md)。

## 1. 应用类型与技术栈

| 应用类型 `type` | 管理方式 | 技术栈 |
| --- | --- | --- |
| `bkmsApp` | 平台按 AppModel 渲染工作负载 | 由 `framework` + `language` 描述 |
| `helm`、`agones` | 用户提供 Chart，平台按 Chart 部署 | 不涉及，`framework`、`language` 为空 |

`trpc`、`taf` 是旧的 `type` 取值，与 `bkmsApp` 同属 AppModel 管理，迁移完成前仍可读到，见迁移文档。
判断管理方式统一用 `appruntime.IsAppModelType` / `IsHelmBasedType`，不比较 `type` 字面值。

**框架目录**（`pkg/core/appruntime/catalog.go`）是合法组合的唯一来源，创建校验、能力矩阵和前端选项都取自它：

| framework | 支持的 language | 说明 |
| --- | --- | --- |
| `trpc` | go、cpp | |
| `taf` | cpp | |
| `blank` | go、cpp、nodejs、python | 不使用开发框架，由用户提供镜像。`blank` 是显式取值，不是空字符串 |

约束：

- 组合合法不代表支持全部功能，功能是否可用由能力集合决定（见 §3）。
- 技术栈创建后不能修改：它同时决定配置格式、脚本、构建方式与历史部署的含义。

## 2. 数据模型

| 对象 | 字段 | 说明 |
| --- | --- | --- |
| Application | `type` | `bkmsApp` / `helm` / `agones` |
| Application | `framework`、`language` | bkmsApp 必填，须为框架目录中的组合；helm / agones 为空 |
| AppModel.Workload | `frameworkConfig` | 框架专属配置，`map[string]any`；目前三种框架都没有字段，固定为空对象，不落库 |
| AppModel.Workload | `frameworkConfigVersion` | `frameworkConfig` 的结构版本，当前为 1 |
| 框架配置文件 def | `Name`、`MountDir` | 框架配置文件的文件名与挂载目录，与普通配置文件一致；内容走配置文件与版本体系 |

技术栈只存在 Application 上，不在 AppModel 重复保存。

**frameworkConfig 的约定：**

- 由各框架的 `DecodeConfig`（`appmodelcore/framework/<框架>/config.go`）严格解码：拒绝未知 key、错类型和未知版本，消费方不直接读 map。
- 修改时整体替换并重新校验，不做递归合并。
- 结构有变化时先发布能读新旧版本的程序，再写入新版本。

## 3. 运行时能力集合

### 3.1 什么算能力

能力（Mod）是**支持与否或参数因技术栈不同而变化**的功能。不满足这一条的不建模：

| 归属 | 例子 | 判断方式 |
| --- | --- | --- |
| 所有技术栈一致 | 普通配置文件、构建、一键构建部署、监控 | 不判断，或按管理方式判断 |
| 管理方式 | 部署、渲染、实例、AppSpec、组件、环境变量 | `IsAppModelType` |
| 环境 / 集群 | GPA、HostPort、端口池、集群组件 | 环境或集群属性 |
| 框架实现的选择 | 渲染插件 | 按 framework 查表，不是能力 |

### 3.2 三层职责

| 层 | 负责 | 位置 |
| --- | --- | --- |
| 声明 | 能不能做、用什么参数：规则、props、常量、错误约定 | `pkg/core/appruntime/capability/<能力>` |
| 框架实现 | 某个框架具体怎么做：配置解析、协议、渲染插件 | `pkg/workload/appmodelcore/framework/<框架>` |
| 使用方 | 何时做、动态条件（环境、启用状态、权限）是否满足、按 props 选实现并执行 | 各业务模块 |

能力集合只反映技术栈决定的静态支持情况，不包含环境、配置、权限等运行时条件；前端据此展示入口，执行入口仍需自行校验。
使用方不得再按 `framework` / `language` 字面值判断能力，这类分支只允许出现在能力声明和框架实现内部。

### 3.3 声明与解析

```go
type Spec[P Props] struct {
    ID      ModID
    Resolve func(framework, language string) Mod[P]
}

type Mod[P Props] struct {
    ID         ModID
    Supported  bool
    ReasonCode ReasonCode
    Props      P
}
```

- 只随框架或只随语言变化的能力用 `runtimemod.ByFramework` / `ByLanguage` 查表；随组合变化的自行实现解析函数，不引入规则表或优先级。
- `mods.Of(app)` 从 Application 字段得出技术栈，再解析全部能力，返回强类型的 `Set`。每次现算，不落库、不缓存，无 I/O。
- 技术栈非法时记录告警，全部能力为不支持（`invalid_stack`），不返回错误。
- 能力不支持不是错误；执行器通过 `Mod.Require()` 取 props，不支持时得到 `runtimemod.ErrModUnsupported`。
- 能力之间不互相调用；执行器可以同时读多个能力。

不支持的原因：

| reasonCode | 含义 |
| --- | --- |
| `unsupported_app_type` | helm / agones 应用 |
| `unsupported_stack` | 当前框架或组合不支持 |
| `incomplete_language` | 能力取决于语言，但应用未记录语言（只有存量应用会出现） |
| `invalid_stack` | 技术栈数据非法，全部能力不可用 |

### 3.4 能力目录

| 能力 ID | trpc | taf | blank | props（内部） | 对外 props | 使用方 |
| --- | --- | --- | --- | --- | --- | --- |
| `frameworkFile` | ✅ yaml | ✅ taf | ❌ | 文件格式 | `format` | 创建流程；配置文件定义接口拒绝为不支持的技术栈新建框架文件 |
| `devMode` | ✅ | ✅ | ❌ | 脚本集、工作目录、挂载目录、二进制目录 | `workPath`、`mountPath` | `extension/component/devmode`、AppSpec |
| `platformBuild` | ✅ 仅 go | ❌ | ❌ | Dockerfile 语言 | — | `build/image`、`build/build` |
| `adminCommand` | ✅ | ✅ | ❌ | 协议；tRPC 按语言区分 admin 地址写法 | — | `workload/instance` |
| `apmServiceName` | ✅ | ✅ | ❌ | 解析器 | — | `observability/bkmonitor` |
| `polarisServiceCheck` | ✅ | ❌ | ❌ | — | — | `extension/addon/polaris` |
| `polarisFrameworkPatch` | ✅ | ❌ | ❌ | — | — | tRPC 渲染插件 |
| `bscpWorkloadMeta` | ✅ | ❌ | ❌ | — | — | `extension/bscpcfg` |
| `configFileMerge` | ✅ 保留空值 | ✅ 保留空值 | ✅ k8s 默认 | 合并策略 | — | `core/app/appcfg` |

`adminCommand` 已预置 java、nodejs、python 的 tRPC 地址写法，这些语言加入框架目录后自动生效。

### 3.5 对外结构

`GET /app-runtime-definitions` 返回框架目录与全部合法组合的能力矩阵，`GET /apps/:appID/runtime-mods` 返回单个应用的能力集合，结构相同：

```json
{
  "schemaVersion": 1,
  "type": "bkmsApp",
  "framework": "trpc",
  "language": "cpp",
  "mods": {
    "frameworkFile": { "supported": true, "props": { "format": "yaml" } },
    "platformBuild": { "supported": false, "reasonCode": "unsupported_stack" }
  }
}
```

- `mods` 包含全部能力；新增能力只增加 key，客户端忽略不认识的 key。
- 只输出 `Props.Public()` 返回的内容，执行器内部参数不对外；没有可展示内容时省略 `props`。
- helm / agones 应用不输出 `framework`、`language`，全部能力为 `unsupported_app_type`。

## 4. 包划分与依赖

```
pkg/core/appruntime/                    技术栈值类型、框架目录、技术栈校验
pkg/core/appruntime/runtimemod/         ModID、ReasonCode、Mod、Spec、查表工厂、对外结构
pkg/core/appruntime/runtimemod/mods/    组装：Set、Of(app)、Resolve、Matrix
pkg/core/appruntime/capability/<能力>/   能力声明、props、常量与错误约定，一个能力一个包
pkg/workload/appmodelcore/framework/<框架>/   frameworkConfig 解码、渲染插件、协议与配置解析
pkg/workload/appmodelcore/lifecycle/   bkmsApp 应用的创建与修改编排
```

依赖约束：

- 能力包只依赖 `appruntime`、`runtimemod`；不依赖 `mods`，也不依赖 workload、extension 等业务层，不放执行代码。
- 框架包可以依赖能力包（取 props、错误约定）。需要被能力包以外的基础层引用的解析代码放在叶子子包（如 `trpcconf`、`tafconf`），避免循环引用。
- `core/app` 不依赖 `mods`，也不 import 具体能力的执行模块。

判断一段代码放在哪里：

- 换一个框架就得重写 → 框架包；
- 只回答"能不能做、参数是什么" → 能力包；
- 负责流程、动态条件或执行入口 → 使用方。

包位置遵循《pkg 模块结构重构迁移总说明》（iwiki 4020906210）：`pkg` 下不新增顶级目录，`core` 只放基础对象，框架配置的解释归 `workload/appmodelcore`。

## 5. API

| 接口 | 说明 |
| --- | --- |
| `POST /workspaces/:workspaceID/bkmsapps` | 创建 bkmsApp 应用 |
| `PUT /apps/:appID/startup-config` | 修改启动命令、参数、脚本与 `frameworkConfig`；不能修改技术栈 |
| `GET /app-runtime-definitions` | 框架目录与能力矩阵 |
| `GET /apps/:appID/runtime-mods` | 应用的能力集合 |
| `/apps/:appID/envs/:envName/bkmsapp-deploys[...]` | 部署记录、预检、部署、下架、资源快照、最新状态；所有 AppModel 应用通用 |
| `POST /apps/:appID/envs/:envName/build-deploys` | 一键构建部署；所有 AppModel 应用通用 |
| 详情 `GET /apps/:appID`、列表 `GET /workspaces/:workspaceID/apps` | 返回 `framework`、`language`；详情另返回 `appModelSpec.frameworkConfig` |

**创建**（`lifecycle.Service.Create`）：

1. 校验技术栈、`frameworkConfig`、框架配置文件输入，全部通过后再写库；
2. 写构建配置；
3. `frameworkFile` 支持时，按其格式创建框架配置文件 def 与默认内容（框架支持时必填，blank 传入即报错）；
4. 写 AppModel（启动配置、环境变量、`frameworkConfigVersion=1`）与 AppSpec（按工作空间默认规则）；
5. 最后写 Application，之前任一步失败都不会留下可见的应用；
6. 异步任务与审计：自定义镜像记录、默认告警策略、操作记录。

**修改 `startup-config`**：`command`、`args`、`shCommand` 整体替换，未传的字段被清空；`frameworkConfig` 未传不修改，传 `null` 拒绝，传对象（含 `{}`）整体替换并重新校验。

## 6. 扩展方式

| 场景 | 需要改动 |
| --- | --- |
| 框架支持新语言 | 框架目录加语言；逐个核对能力声明中按语言区分的部分（如 `adminCommand`、`platformBuild`） |
| 新增框架 | 框架目录加一项；`framework/<框架>` 提供 `DecodeConfig`，需要框架资源时提供渲染插件并在插件表登记；逐个能力决定是否支持并补充规则；按需补充框架实现（配置解析、协议） |
| 新增能力 | 新建 `capability/<能力>` 声明 props 与规则；在 `runtimemod` 加 ModID；在 `mods.Set` 的字段、`resolve`、`invalid`、`all` 四处登记（测试会检查四者一致）；使用方通过 `mods.Of(app).<能力>` 取 props |
| 给 blank 开放某项能力 | 在该能力的规则中加 blank；已有实现可复用时只改规则，否则先在框架包或使用方补实现 |

测试约束：能力矩阵测试覆盖框架目录的全部合法组合；每个 `Supported=true` 的组合都有执行器覆盖测试。
