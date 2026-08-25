# plain cfg files 设计说明

## 核心建模

plain cfg files 在现有应用配置文件模型上增加了一条新的语义分支：`configKind=plain`。

- `framework`
  - 表示原有框架主配置文件
  - 仍然服务于 `trpc` / `taf` 等工作负载主配置渲染
  - 支持统一配置和按环境独立配置两种模式

- `plain`
  - 表示额外挂载到容器内某个路径的附加配置文件
  - 必须是 `normal + local`
  - 必须指定 `mountPath`
  - 不支持 `overlay`
  - 不支持 `bscp`

一份 plain 文件在数据上会拆成两类记录：

- 默认记录（`envName=""`）
  - 保存公共元数据，例如文件名、`mountPath`、`fileFormat`、`isUnifiedConfig`、`mountedEnvNames`
  - 同时承载"共用配置"的内容，所有引用状态的环境都使用此内容
- 环境记录（`envName!=""`）
  - 仅在用户**首次独立修改**该环境的内容时才创建（copy-on-write）
  - 通过 `DefaultAppConfigFileID` 关联回默认记录
  - 创建后拥有自己独立的内容和版本历史
  - 未创建环境记录的环境，查询时动态取默认记录的内容（即"引用"状态）

### 引用模型

环境配置采用**惰性引用 + 首次修改独立**的 copy-on-write 模型：

```
                    ┌──────────────┐
                    │  默认记录     │
                    │ (共用配置)    │
                    └──────┬───────┘
                           │
            ┌──────────────┼──────────────┐
            │              │              │
     ┌──────▼──────┐ ┌────▼────┐ ┌───────▼──────┐
     │ prod (引用)  │ │ stag    │ │ dev (引用)   │
     │ 无DB记录     │ │ (独立)  │ │ 无DB记录     │
     │ 内容=默认记录 │ │ 有DB记录│ │ 内容=默认记录 │
     └─────────────┘ │ 有版本   │ └──────────────┘
                     └─────────┘
```

- **引用状态**：环境没有独立的 env instance 记录，查询时返回默认记录的内容。修改默认记录的内容会自动影响所有引用环境。
- **独立状态**：用户修改了该环境的内容后，创建独立的 env instance 记录。此后该环境拥有自己的内容和版本历史，不再跟随默认记录变化。
- **恢复共用**：用户可以将已独立的环境恢复为引用状态，删除其独立记录和版本历史。

### 字段模型

环境配置策略拆分为两个独立字段，分别描述"挂载范围"和"内容模式"：

| 字段 | 类型 | 含义 |
|------|------|------|
| `isUnifiedConfig` | `bool` | `true` = 统一配置，所有挂载环境共用同一份内容；`false` = 按环境独立配置 |
| `mountedEnvNames` | `[]string` | `nil` = 对所有环境生效；`[]`（非 nil 空切片）= 不挂载到任何环境；非空 = 仅对列出的环境生效 |

四种组合的语义：

| `isUnifiedConfig` | `mountedEnvNames` | 含义 |
|---|---|---|
| `true` | 空 | 统一配置，对所有环境生效（默认模式） |
| `true` | `["prod","stag"]` | 统一配置，仅对指定环境生效 |
| `false` | 空 | 按环境独立配置，对所有环境生效 |
| `false` | `["prod","stag"]` | 按环境独立配置，仅对指定环境生效 |

framework 文件：
- 支持 `isUnifiedConfig=true`（统一配置）和 `isUnifiedConfig=false`（按环境独立配置）
- `mountedEnvNames` 始终为空（框架配置对所有环境生效）

plain 文件：
- 四种组合全部支持

运行时渲染时调整：

- `plain` 文件会先转换成 `runtimerender.ConfigFileParams`
- 主框架配置文件与 plain 文件在 `runtimerender` 内统一视为一组待下发文件
- 最终通过同一个 ConfigMap / init container / emptyDir 渲染并挂载到容器内

## 关键流程

### 1. 创建 plain 文件

创建文件时先根据 `configKind` 分流。

- `framework` 走原有语义
- `plain` 强制校验：
  - `type=normal`
  - `contentSourceType=local`
  - `mountPath` 必填
  - 不允许 `baseAppConfigFileID`
  - 不允许 `overlayContent`

创建时**始终**按统一配置处理（`isUnifiedConfig=true`），只创建默认记录。可以通过 `mountedEnvNames` 指定挂载范围，不传时对所有环境生效。无论选择全部环境还是部分环境，初始都是一份统一配置。按环境独立配置需要创建后通过 `env-config-policy` 接口开启。

在同一个应用里，`mountPath` 可以理解为这份 plain 文件在容器内占用的位置：

- 两份不同的 plain 文件不能使用同一个 `mountPath`
- 同一份 plain 文件的默认记录和环境记录共用同一个 `mountPath`

### 2. 切换 env-config-policy

通过 `UpdateEnvConfig` 统一处理 framework/plain 的模式切换。

framework（原配置文件）：

- `isUnifiedConfig=true`
  - 回到统一配置模式
  - 删除环境级派生记录
- `isUnifiedConfig=false`
  - 开启按环境独立配置
  - 每个环境最终会以环境配置参与渲染

plain：

- `isUnifiedConfig=true`（切回统一配置）
  - 直接使用默认记录当前内容作为统一配置
  - 删除所有 env instance 及其版本历史
  - 不需要选择来源环境

- `isUnifiedConfig=false`（开启/调整按环境独立配置）
  - 更新默认记录的 `isUnifiedConfig=false` 和 `mountedEnvNames`
  - **不创建** env instance（引用模型，环境默认引用默认记录内容）
  - 若调整挂载范围导致某些已有 env instance 不在新范围内，删除这些 env instance

- `fallbackConfigEnv`（单环境回退为引用状态）
  - 仅在文件当前已处于按环境独立配置模式下有效
  - 指定要回退的环境名称，删除该环境的 env instance 及其版本历史
  - 该环境恢复为引用默认记录内容的状态，其他环境的独立配置不受影响

### 3. 环境内容更新（copy-on-write）

当用户修改某个环境的 plain 配置内容时：

- 若目标是**默认记录**（root）：直接更新默认记录内容，所有引用状态的环境自动生效
- 若目标是**已存在的 env instance**：直接更新该实例内容
- 若目标环境**没有 env instance**（引用状态）且处于独立配置模式：
  - 首次修改时自动创建 env instance，内容为用户提交的新内容
  - 此后该环境变为独立状态

### 4. 环境创建与删除联动

plain 引入后，环境生命周期会影响配置文件实例生命周期。

- 创建环境时：
  - 不需要自动创建 env instance
  - 新环境自动处于引用状态，使用默认记录的内容
  - 用户需要为该环境定制内容时，通过内容更新接口触发 lazy create

- 删除环境时：
  - 清理该环境对应的 plain env instance（若存在）
  - 同步回收 root 上的 `mountedEnvNames` 元数据
  - 回收后 `mountedEnvNames` 可能变成空切片（`[]`），表示不挂载到任何环境，不会自动切回统一配置
  - `mountedEnvNames` 的 nil/空切片语义区分：`nil` = 对所有环境生效；`[]`（非 nil 空切片）= 不挂载到任何环境

### 5. 运行时渲染

工作负载构建时：

1. `appcfg.ListEnvPlainContents` 先解析当前环境下真正生效的 plain 文件
   - 有独立 env instance → 使用其内容
   - 没有 env instance → 使用默认记录内容（引用）
2. `plainfiles.BuildRuntimePlainConfigFiles` 将其转成运行时渲染输入
3. `runtimerender.BuildConfig` 将主框架配置文件与 plain 文件统一下发

### 6. 版本查看

- 引用状态的环境：当前版本查询接口不会按引用关系自动回退到默认记录历史；若按环境名精确筛选且该环境没有 env instance，结果可能为空
- 独立状态的环境：查看版本时显示该 env instance 自己的版本历史
- 恢复共用配置后：独立期间的版本历史被删除

## 接口与行为变化

### 1. 文件模型相关接口扩展

以下接口增加了 plain 相关字段或语义：

- 创建配置文件
- 更新配置文件基础属性
- 配置文件列表

新增或扩展的核心字段包括：

- `configKind`
- `mountPath`（仅文件接口，版本接口不包含）
- `isUnifiedConfig`（仅文件接口，版本接口不包含）
- `mountedEnvNames`（仅文件接口，版本接口不包含）

其中：

- `framework` 默认兼容旧语义
- `plain` 明确表示容器额外挂载文件

> 注：环境配置策略字段（`mountPath`、`isUnifiedConfig`、`mountedEnvNames`）仅存在于
> `AppConfigFile`，不写入版本记录。版本记录只保存身份字段与可版本化内容
> （`VersionedContent`），回滚操作也只恢复内容部分，不影响策略配置。

### 2. 创建接口

创建接口始终按统一配置处理（`isUnifiedConfig=true`），但可以传入 `mountedEnvNames` 指定挂载范围。不传 `mountedEnvNames` 时对所有环境生效。

### 3. 环境配置策略接口

`PUT /apps/{appID}/app-config-files/{id}/env-config-policy`

该接口负责：

- 切换是否启用按环境独立配置（`isUnifiedConfig`）
- 调整挂载环境范围（`mountedEnvNames`）
- 回退指定环境为共用配置（`fallbackConfigEnv`）

### 4. 删除行为

plain 文件删除采用"只允许删除 logical root"的语义：

- 不允许单独删除 plain env instance
- 删除 plain root 时，级联删除其全部 env instance 及对应版本记录

### 5. 内容更新行为

- framework 内容更新后，仍会做框架配置校验与编排预校验
- plain 内容更新只校验 plain 自身约束，不参与 framework 编排校验
- plain 不支持 overlay，因此 overlay 相关接口对 plain 直接拒绝
- plain 独立配置模式下，首次修改某环境内容时自动创建 env instance（copy-on-write）

### 6. 运行时挂载行为

在 `trpc` / `taf` 场景下：

- 主框架配置文件仍作为主配置参与运行时渲染
- plain 文件作为额外文件一起进入 `runtimerender`
- 最终生成多文件 ConfigMap + init container 渲染逻辑

结果上，应用可以同时拥有：

- 一个 framework 主配置文件
- 多个按 `mountPath` 区分的 plain 附加文件
