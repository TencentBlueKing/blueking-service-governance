# PolarisConfig 动态下发

## 1. 问题背景

一份 PolarisConfig 会在两个地方产生影响：

- **工作负载**：部分字段要写进容器环境变量和 Service；
- **集群里的 PolarisConfig CR**：北极星侧的注册行为由这个 CR 决定。

用户改配置时，这两类影响的处理方式完全不同。只碰 CR 的字段可以立刻推到集群生效；碰工作负载的字段必须等应用重新部署，由部署流程整体下发。

所以每次配置变更都要回答一个问题：**这次改动能直接推 CR 吗？**

判断依据是"这个环境上次部署时用的是什么配置"。系统不额外维护流程状态机，而是在每次变更后，拿当前配置和环境的部署快照现场比一次。

### 1.1 两种注册模式

上面这套快照机制的前提是"配置会影响工作负载"。但有一类业务并不消费北极星 token 相关的环境变量，只是想把实例注册到北极星上，让它们等一次部署纯属多余。

于是配置上有一个创建后不可修改的 `registerMode`：

| 模式 | 含义 |
| --- | --- |
| `on_deploy` | 配置参与工作负载渲染，CR 随应用部署下发。缺省值，也是本节其余部分描述的行为 |
| `immediate` | 配置**完全不参与**工作负载渲染，绑定环境时由平台直接下发 CR 与配套 Service 完成注册 |

`immediate` 的关键在于"完全不参与"：不注入 `{instanceKey}_polarisToken` / `{instanceKey}_serviceport` 环境变量，也不往 tRPC 框架配置注入 `plugins.registry.polaris.service`。这样一来这份配置对 Pod 再无任何诉求，CR 下发成功即代表配置已完全生效，不需要引入"已注册但 Pod 还没拿到 token"这种中间状态，四种环境状态和部署快照的语义都能原样复用。

`enableHealthCheck` 在两种模式下都原样透传给 CR。跳过框架配置注入不影响心跳上报——那段配置并非上报的必要条件，确有需要的业务自行在配置文件里书写即可，Patcher 检测到已有该路径就不会覆盖。

存量数据没有 `registerMode` 字段，代码里一律按 `on_deploy` 解释（`PolarisConfig.IsImmediateRegister()`），因此老配置和滚动发布窗口内新建的配置行为都不变。

模式不可修改是刻意的：从 `immediate` 切到 `on_deploy` 会让下一次部署突然多出环境变量，反向切换则会让正在被业务读取的环境变量凭空消失，两个方向都能打挂线上业务。

## 2. 字段分成两类

### 2.1 需要重新部署的字段

这三个字段会影响工作负载，改了必须重新部署：

| 字段 | 为什么必须重新部署 |
| --- | --- |
| `instanceKey` | 决定 Polaris 环境变量的名字 |
| `polarisToken` | 既写进容器环境变量，也写进 CR |
| `servicePort` | 既写进环境变量和 Service，也写进 CR |

它们组成部署快照：

```go
type RedeployRequiredFields struct {
    InstanceKey  string
    PolarisToken string
    ServicePort  int32
}
```

### 2.2 可以动态下发的字段

这些字段只落在 CR 上，条件满足时可以直接推到集群：

- `direct`
- `keepNotReadyPod`
- `enableHealthCheck`
- `serviceLabels`

一次 PATCH 可以同时改两类字段。系统不去分析请求里具体改了哪些字段，只看**改完之后**的三个部署关联字段和环境快照是否还一致。

环境权重不再属于配置 PATCH。它只能通过环境级 PUT 修改，并走独立的 JSON Patch 下发路径。

### 2.3 权重因子与 spec.polaris.dynamicWeight

动态权重是两级开关：配置级的 `enableWeightFactor` 决定用户能否为这个北极星配置启用权重因子，环境级的 `envDynamicWeights[env]` 决定具体哪些环境开。

CR 上的 `dynamicWeight.enable` **直接取环境级开关**，平台不再叠加 `enableWeightFactor` 做二次判断。两级开关的组合关系在用户输入侧就已经决定，下发时照搬结果即可。这样即使出现「总开关关着、某环境开关仍为 `true`」的组合，平台也照常下发 `enable: true`：`preserveServiceConfig` 固定为 `true` 时 operator 只给实例打权重因子标记，是否真的按机型调权取决于北极星服务侧的 `internal-enable-dynamic-weight`（见下文），因此这个组合不会造成实际影响，不值得为它在两条下发路径上各写一次判断。

关闭 `enableWeightFactor` 不清理环境级开关：它们记录的是用户对每个环境的意图，重新打开后各环境按原样恢复。

环境级开关不进部署快照，通过环境级 PUT 即时下发；配置级 `enableWeightFactor` 同样不进快照，但它也不参与 CR 组装，单独 PATCH 它不会触发动态下发（与 `operator` 同类）。

CR 片段的形状很简单，且**始终下发**：

```json
{"enable": "<envDynamicWeights[env]，缺省 false>", "preserveServiceConfig": true}
```

关闭时下发 `enable: false`，而不是省略 `dynamicWeight` 这个键。省略的做法看起来更干净，但清理要靠 Server-Side Apply 回收字段——SSA 只删「本 FieldManager 通过 Apply 拥有、本次又省略」的字段，而环境级即时下发走的是 `types.JSONPatchType`，属于 Update 类操作，会把 `/spec/polaris/dynamicWeight` 的所有权转出 `bkms-server` 的 Apply 条目。一旦某个环境用过即时下发，后续关闭开关时 SSA 就删不掉这个字段，CR 上会残留 `enable: true`。让字段常驻可以让值始终反映真实状态，不依赖 SSA 的删除语义。

部署下发（`workload.go`）与环境级 JSON Patch（`applier.go`）各自拼这个片段，两处必须保持一致。部署侧直接取 `cfg.EnvDynamicWeights[env]`；PUT 侧用 `lo.FromPtrOr(dynamicWeight, config.EnvDynamicWeights[envName])`：请求带了新值就用新值，没带则沿用库中现值。

平台只下发这两个字段，公式参数（`func` / `a` / `b` / `min` / `max`）既不存储也不下发，原因有两条：

- `preserveServiceConfig` 固定为 `true`，operator 因此只给实例标记权重因子，不改写北极星服务上的服务级配置（`internal-enable-dynamic-weight`、`internal-dynamic-weight-config`）。平台对北极星服务的配置由平台自己维护，不被底层控制器覆盖；相应地，服务级的动态权重开关与公式需要在北极星侧自行配置。
- CRD 规定 `preserveServiceConfig=true` 时公式参数必须留空。即便要下发，产品要求的默认值 `{"func":"linear","params":{"a":1,"b":1,"min":-1,"max":1.5}}` 也逐项等于 CRD 字段默认值，平台没有理由再维护一份。

## 3. 能不能动态下发完整 CR

对配置 `config` 和环境 `env`：

```text
DesiredFields = config 当前的 instanceKey + polarisToken + servicePort
AppliedFields = config.EnvStates[env].AppliedFields
```

三个条件全部成立才允许动态下发：

```text
env ∈ config.ScopeEnvNames
AND AppliedFields != nil            // 这个环境部署过
AND AppliedFields == DesiredFields  // 部署关联字段没变
```

其余情况一律等应用部署。

这一节只适用于 `on_deploy` 配置。`immediate` 配置不影响工作负载，没有"等部署"这回事，scope 内的环境一律在请求内同步下发，见 §6.1。

这条规则带来几个不需要额外代码维护的好性质：

- 首次部署前没有快照，任何改动都不会误碰集群；
- 只改 CR 字段时快照仍然相等，可以直接推；
- 改了部署关联字段，快照不相等，自然落到"等部署"；
- 用户把部署关联字段改回集群里的值，下次 PATCH 又能满足条件；
- 某次动态下发失败了，下次 PATCH 按同样规则重算，不需要失败状态位。

环境权重 PUT 不受这条快照一致性规则限制：只要环境已经部署过（包括 `pendingModify` 和 `pendingDelete`），就在请求内同步 Patch 权重；scope 内尚未部署的环境只持久化，等首次部署时随完整 CR 生效。

## 4. 环境维度的数据

PolarisConfig 用环境名做 key 存三份环境级数据：

```go
type PolarisConfig struct {
    // ...其他字段
    ScopeEnvNames     []string                   // 生效环境
    EnvStates         map[string]PolarisEnvState // 部署快照 + 下发错误
    EnvWeights        map[string]int32           // 单环境权重
    EnvDynamicWeights map[string]bool            // 单环境动态权重开关
}

type PolarisEnvState struct {
    AppliedFields *RedeployRequiredFields
    LastError     string
    UpdatedAt     time.Time
}
```

含义：

| 字段 | 说明 |
| --- | --- |
| `AppliedFields` | 该环境最近一次**应用部署**实际下发的部署关联字段；`nil` 表示还没部署过 |
| `LastError` | 最近一次异步动态下发的错误；部署完成或下发成功后清空 |
| `UpdatedAt` | 环境信息更新时间，由 Store 统一写 |
| `EnvWeights[env]` | 该环境的单实例权重；没有这一项时固定回退到 `DefaultEnvWeight = 100` |
| `EnvDynamicWeights[env]` | 该环境是否按机型动态调权；没有这一项时按关闭处理 |

创建配置时不需要为 scope 里的环境预建 `EnvStates`。Go 读 map 里不存在的 key 会拿到零值（`AppliedFields == nil`），所以判定逻辑不用区分"没有这条记录"和"有记录但没部署过"。

判断是否部署过统一走 `PolarisEnvState.IsDeployed()`，不要直接比 `AppliedFields != nil`。

`EnvDynamicWeights` 与 `EnvWeights` 同属「环境级配置」——由用户按环境设置、共享同一套清理时机（见 §8.2），在 store 层由 `envSettingFields` 统一登记、`RemoveEnvSettings` 一起清理。两者只有两点不同：

- **加入 scope 时不预建默认值**。权重必须有个具体数值才能下发，所以补 `100`；动态权重开关缺省即关闭，预建 `false` 只会多出一条无意义的记录，新环境自动按关闭处理更安全。
- **独立决定 CR 上的开关**。它的取值直接下发为 `dynamicWeight.enable`，不与配置级 `enableWeightFactor` 做二次组合；关闭后者也不会清除这里的取值（见 §2.3）。

## 5. 三个组件的分工

```mermaid
flowchart LR
    Patch["PATCH 配置"] --> Service["PolarisConfigService"]
    PutWeight["PUT 环境权重"] --> Service
    Service --> ConfigStore["PolarisConfigStore"]
    Service --> StateManager["PolarisEnvStateManager"]
    Service --> Queue["asynq polaris.dynamic_apply"]
    Service --> Applier["CRApplier"]
    Queue --> DynamicApply["DynamicApplyService"]
    DynamicApply --> Applier
    Applier --> Cluster["目标集群 PolarisConfig CR"]

    Deploy["应用部署完成"] --> StateManager
    Uninstall["应用卸载完成"] --> StateManager
    StateManager --> ConfigStore
```

### PolarisConfigService

配置变更的入口，负责流程编排：

- 创建配置，按需让平台创建托管的北极星服务；
- 创建时为 scope 环境补齐默认权重；PATCH 修改 scope 时规范化 `envWeights`，落库后再算出哪些环境可以动态下发；
- 按 `registerMode` 分叉：`immediate` 走 §6.1 的同步收敛，`on_deploy` 为每个可下发环境投递一条 `polaris.dynamic_apply` asynq 任务；
- PUT 权重时，待部署环境直接持久化；已部署环境先同步调用权重专用 Patch，成功后再持久化，失败时记录日志并返回错误；
- asynq 任务执行时由 `DynamicApplyService` 现读最新配置并下发，按配置 `UpdatedAt` 记录结果，避免旧任务覆盖新状态；
- 删除配置，并处理托管北极星服务的生命周期。

Service 不含资源构建算法，也不含环境状态算法。托管服务的增删查由 `PolarisPlatformManager` 封装。

### PolarisEnvStateManager

管环境快照、环境级设置的生命周期和下发结果：

- `reconcileEnvSettingsForScope`：规范化 `envWeights` 与 `envDynamicWeights`（清理离域项，只给权重补默认值）；`immediate` 配置离域即删，因为它没有"等下次部署清理"这个阶段；
- `PrepareDynamicApply`：清掉没有部署事实的离域 `EnvState`，返回可动态下发的环境名；
- `PolarisConfig.EnvNamesOutsideScope` / `TrackedEnvNames`：离域环境、以及 scope ∪ 仍有记录的环境；
- `RecordDynamicApplyResult`：仅在配置顶层 `UpdatedAt` 仍匹配时记录该环境下发结果，版本已变则跳过；
- `RecordImmediateApplyResult`：记录 `immediate` 下发结果，成功时一并写 `AppliedFields`；
- `ReleaseEnv`：`immediate` 配置的集群资源删干净后，移除该环境的 `EnvState` 与环境级设置；
- `ReconcileAfterDeploy`：部署完成后写 `AppliedFields`，或清理离域数据；
- `ReconcileAfterUninstall`：卸载完成后删 `EnvState`，离域时连环境级设置一起删。

`envWeights` 与 `envDynamicWeights` 的清理时机完全一致，三处清理点统一调 `store.RemoveEnvSettings`，避免将来只改了一半。

另外还提供两个纯函数供 serializer 计算展示状态：`PolarisEnvStatus` 和 `PolarisTokenChanged`。

Manager 不持有请求级状态，可以同时注入给 API Service 和 AppModel Deployer。它不读 AppModel、Environment、环境变量，也不碰 Kubernetes。

### CRApplier

只管单次下发：

- `Apply` 复用和应用部署相同的资源构建函数，再按调用方声明的 kind 过滤后 Upsert：on_deploy 动态下发只推 PolarisConfig CR，immediate 还要推配套 Service；
- 权重 PUT 生成只包含 service-name `test`、weight `add`，以及可选的 `dynamicWeight` `add` 的 JSON Patch；
- `immediate` 配置删除或离域时 `DeleteResources` 按资源名删掉 CR 与 Service；
- 所有路径都解析目标集群的 GVR，但权重路径使用 `types.JSONPatchType`，不会替换 `services` 数组。

immediate 要连 Service 一起推，是因为绑定的新环境从未走过部署流程，集群里根本没有那个 Service，只推 CR 完不成实例注册。`DeleteResources` 复用 `Client.Delete` 对 NotFound 返回成功的语义，所以重复删除是安全的。

输入由调用方准备好（Application、Environment、PolarisConfig、完整环境变量）。Applier 不持有 Store，不读写 `EnvStates`，只把资源构建或集群操作的错误返回上去。

**`on_deploy` 路径下 Applier 不更新部署快照**。CR Upsert 成功不代表工作负载已经用上新的部署关联字段，只有完整的应用部署才能刷新快照。

`immediate` 路径下这条不变式不成立，且是有意为之：这类配置对工作负载没有任何诉求，"下发成功"和"完全生效"是同一件事，所以 Service 会在 `Apply`（CR + Service）成功后立即写 `AppliedFields`（`RecordImmediateApplyResult`），环境状态直接落到 `deployed`。写快照的动作仍然在 Service / Manager 里，Applier 本身依旧不碰 Store。

## 6. PATCH 走一遍

```mermaid
sequenceDiagram
    participant API as Handler
    participant Service as PolarisConfigService
    participant Store as PolarisConfigStore
    participant Manager as PolarisEnvStateManager
    participant Queue as asynq
    participant Worker as DynamicApplyService
    participant Applier as CRApplier
    participant K8s as Kubernetes

    API->>Service: Update(app, config, patch)
    Service->>Manager: reconcileEnvWeightsForScope（涉及 scope 时）
    Service->>Store: 写入配置
    Service->>Store: 读回最新配置
    Service->>Manager: PrepareDynamicApply(config)
    Manager->>Store: 清理离域且未部署的环境记录
    Manager->>Manager: 逐环境比对快照与当前部署关联字段
    Manager-->>Service: 返回可下发的环境名
    Service->>Queue: 每个环境一条 polaris.dynamic_apply
    Service-->>API: 返回更新后的配置
    Queue->>Worker: handle(appID, configName, envName)
    Worker->>Store: 现读最新配置
    Worker->>Applier: Apply(app, env, config, envVars)
    Applier->>K8s: Upsert PolarisConfig CR
    Worker->>Manager: RecordDynamicApplyResult(updatedAt, error)
    Manager->>Store: 版本仍匹配时写入该环境 LastError
```

接口只等配置保存、条件计算和任务入队，不等集群操作。所以响应里的 `envStates.lastError` 可能还是上一次的值，要拿最新结果得重新请求列表接口。

asynq 任务只带业务主键。执行时 `DynamicApplyService` 现读最新数据，下发后再核对配置 `UpdatedAt`：版本变了就重试最新配置，避免旧渲染结果覆盖新状态。单个环境失败不影响其他环境。

下发成功清空 `LastError`，失败写入错误，**两种情况都不动 `AppliedFields`**。

### 6.1 immediate 配置的同步收敛

`immediate` 配置的 Create / PATCH 落库后，在**请求内**同步收敛集群资源，不走上面的异步路径：

```mermaid
sequenceDiagram
    participant API as Handler
    participant Service as PolarisConfigService
    participant Store as PolarisConfigStore
    participant Manager as PolarisEnvStateManager
    participant Applier as polarisCRApplier
    participant K8s as Kubernetes

    API->>Service: Create / Update
    Service->>Store: 写入配置
    Service->>Store: 读回最新配置
    loop 本次离开 scope 的环境
        Service->>Applier: deleteResources
        Applier->>K8s: 删除 PolarisConfig CR + Service
        alt 成功
            Service->>Manager: ReleaseEnv（删 EnvState 与 envWeights）
        else 失败
            Service->>Manager: RecordImmediateApplyResult(error)
        end
    end
    loop scope 内全部环境
        Service->>Applier: applyAll
        Applier->>K8s: Upsert PolarisConfig CR + Service
        Service->>Manager: RecordImmediateApplyResult(error 或 nil)
    end
    Service->>Store: 读回收敛后的配置
    Service-->>API: 配置 + 汇总错误
```

几个刻意的选择：

**下发覆盖 scope 内全部环境，而不只是新增环境。** `applyAll` 是幂等的 Upsert，一次保存就能同时覆盖"新绑定环境"、"上次失败重试"和"普通字段变更"三种情况，用户不需要额外的重试入口。

**单个环境失败不中断其余环境。** 每个环境的结果各自写进自己的 `EnvState`，全部处理完后把失败环境和原因汇总成一个 `ErrClusterSyncFailed` 返回。

**失败既写库又返回 HTTP 500。** 配置本身已经保存成功，Handler 据 `ErrClusterSyncFailed` 区分这一点：照常记录配置变更审计，然后返回 500，错误消息里带上每个失败环境的名字和原因。CLI 这类调用方天然报错退出，不需要再查一次接口；`EnvState.LastError` 里的记录供事后排查。

**离域和删除配置都要同步删集群资源。** `immediate` 配置不参与工作负载渲染，也就不会在下一次部署时被资源差异清理带走。若沿用 `on_deploy` 那套"等下次部署清理"，业务长期不部署就会造成北极星注册泄漏。删除配置时集群资源删不掉就不继续删配置记录，用户重试即可收敛；离域时删不掉则保留 `EnvState`，下次保存配置会重新进入清理列表。

### 6.2 环境权重 PUT

```mermaid
sequenceDiagram
    participant API as Handler
    participant Service as PolarisConfigService
    participant Store as PolarisConfigStore
    participant Applier as polarisCRApplier
    participant K8s as Kubernetes

    API->>API: 校验 inScope 或 IsDeployed
    API->>Service: UpdateEnvWeight(envName, weight, dynamicWeight)
    alt 尚未部署但在 scope
        Service->>Store: $set envWeights.envName + envDynamicWeights.envName
        Service->>Store: 读回最新配置
        Service-->>API: 返回配置（200）
    else 已部署（含 pendingModify / pendingDelete）
        Service->>Applier: patchWeight(weight, dynamicWeight)（同步）
        Applier->>K8s: JSON Patch test service name + add weight (+ add dynamicWeight)
        alt Patch 成功
            Applier-->>Service: nil
            Service->>Store: $set envWeights.envName + envDynamicWeights.envName
            Service->>Store: 读回最新配置
            Service-->>API: 返回配置（200）
        else Patch 失败
            Applier-->>Service: error
            Service->>Service: 记录错误日志
            Service-->>API: 返回 error（500）
        end
    end
```

JSON Patch 固定针对 `/spec/services/0`：先 `test /spec/services/0/name`，再 `add /spec/services/0/weight`。`add` 同时兼容 weight 字段存在和不存在。PUT 会等待本次同步尝试完成；service 名不匹配、CR 不存在或集群调用失败时记录错误日志并返回 500，不持久化新权重，也不修改 `LastError`。Patch 成功后才持久化并记录配置变更审计，返回 200，已有的 `LastError` 保持不变。Kubernetes 与 MongoDB 不具备跨系统事务；若 Patch 成功后持久化失败，接口记录错误并返回 500，此时集群权重可能已变化，后续重试可重新收敛。

Patch 固定带第三个 operation `add /spec/polaris/dynamicWeight`，权重与开关一次提交，不会出现权重已变、开关未变的中间态。这里替换的是整个 `dynamicWeight` 对象而不是它的 `enable` 子路径：存量 CR 可能还没有 `dynamicWeight` 父节点，子路径的 `add` 会直接失败。

下发值取 `lo.FromPtrOr(dynamicWeight, config.EnvDynamicWeights[envName])`：请求带了 `dynamicWeight` 就用请求值，没带则沿用库中现值，因此「只调权重」不会改变 CR 上的开关，同时顺带把存量 CR 缺失的片段补齐。

**接口不因 `enableWeightFactor` 关闭而拒绝写入环境级开关**，也不因此改变下发值（见 §2.3）。请求不带 `dynamicWeight` 时不写库，开关保持原值，旧客户端行为不变。

## 7. 部署和卸载之后

### 7.1 应用部署完成

AppModel Deployer 处理完额外资源、主工作负载和过期资源后调用：

```go
PolarisEnvStateManager.ReconcileAfterDeploy(ctx, app, env)
```

Manager 遍历该应用下所有 PolarisConfig：

- 配置在这个环境仍然生效：写入 `AppliedFields`，清空 `LastError`；
- 配置已经不在这个环境生效：删掉该环境的 `EnvState`、`envWeights` 和 `envDynamicWeights`。

这一步在完整资源下发之后执行，所以 `AppliedFields` 代表的是"部署流程处理过的配置"，而不是某次动态 CR 更新的结果。

### 7.2 应用卸载完成

环境卸载完成后 Deployer 调用：

```go
PolarisEnvStateManager.ReconcileAfterUninstall(ctx, app, envName)
```

Manager 从该应用所有 PolarisConfig 里删掉这个环境的 `EnvState`。`envWeights` 和 `envDynamicWeights` 则看环境是否还在 scope 内：

- **还在 scope 内**：保留权重与动态权重开关，下次部署继续用；
- **已经离域**：一起删掉。

这两个同步动作都在部署/卸载主体完成后执行，同步失败只记日志，不影响部署或卸载的最终结果。

## 8. Scope 变化和配置删除

### 8.1 环境加入 scope

- 一般情况下还没有 `AppliedFields`，要等这个环境首次部署才建立快照；
- 如果这个环境之前是"已部署状态离开 scope"且还没被清理，会直接复用原有的 `AppliedFields`、`envWeights` 和 `envDynamicWeights`；
- 否则 `reconcileEnvSettingsForScope` 给它补上固定默认权重 `100`，动态权重开关不预建、缺省即关闭。

### 8.2 环境离开 scope

`EnvStates`、`envWeights` 和 `envDynamicWeights` 共享同一套生命周期：

| 环境状态 | 离开 scope 时 | 最终清理时机 |
| --- | --- | --- |
| 没部署过 | 立即删除 `EnvState` 和环境级设置 | 即时 |
| 部署过 | 保留（`status = pendingDelete`） | 该环境下次**部署**且仍离域，或**卸载**且仍离域 |

保留是为了环境再次加入 scope 时不丢快照、自定义权重和动态权重开关。

`immediate` 配置不适用这张表：它在保存请求内就同步删掉了集群资源，删成功即连 `EnvState` 和环境级设置一起清掉，没有 `pendingDelete` 这个停留期。删失败才保留 `EnvState`（带 `LastError`），等下次保存重试。

### 8.3 删除整条配置

`on_deploy` 配置直接删 PolarisConfig 记录。集群里由它生成的资源，交给应用下一次部署按资源差异清理，不保留配置级的删除状态。

`immediate` 配置先同步删掉所有相关环境（scope 内的加上仍有记录的）的 CR 与 Service，全部删干净后才删配置记录。任一环境删除失败就中止，配置保留，用户重试即可——否则配置记录没了，集群里的注册就再没人能清理。

## 9. Store 的更新语义

环境维度数据通过四个接口维护：

```go
UpsertEnvState(ctx, appID, configName, envName, update)
RemoveEnvStates(ctx, appID, configName, envNames)
UpsertEnvWeight(ctx, appID, configName, envName, weight, dynamicWeight)
RemoveEnvSettings(ctx, appID, configName, envNames)
```

- `UpsertEnvState` 定位到 `envStates.<envName>`，只更新调用方传的字段，并刷新 `UpdatedAt`；
- `UpsertEnvWeight` 在一次 `$set` 里写 `envWeights.<envName>` 和 `envDynamicWeights.<envName>` 并刷新 `UpdatedAt`，PUT 单环境权重走这条。`dynamicWeight` 传 `nil` 时不带动态权重字段，开关保持原值；
- `RemoveEnvSettings` 在一次 `$unset` 里删掉这些环境在**全部**环境级配置字段下的条目，不会出现只清理了一半的中间态。字段清单由 store 层的 `envSettingFields` 登记，新增环境级配置字段时只改这一处；
- 两个 `Remove*` 都用 `$unset` 批量删，传空列表或重复删除都返回成功；
- Create 会为全部 scope 环境初始化权重 `100`，但不初始化动态权重开关；PATCH 改到 `scopeEnvNames` 时，Service 先经 Manager 规范化，再把两个 map 分别整体 `$set` 覆盖。

`envStates` 不在 `envSettingFields` 里：它记录的是系统观测到的部署事实，写入方与清理时机都和用户设置的环境级配置不同，所以保留独立的 `RemoveEnvStates`。

环境名直接当MongoDB 子文档字段名用。`envFieldPrefix` 会拒绝空串和含 `.`、`$` 的环境名，保证字段路径安全。

## 10. API 表现

### 10.1 响应结构

列表和 PATCH 响应里同时有顶层的 `scopeEnvNames`、`envWeights`，以及带 `status` 的 `envStates`：

```json
{
  "registerMode": "on_deploy",
  "scopeEnvNames": ["stag"],
  "envWeights": {
    "stag": 35
  },
  "envStates": {
    "stag": {
      "appliedFields": {
        "instanceKey": "demo",
        "polarisToken": "******",
        "servicePort": 8080
      },
      "polarisTokenChanged": false,
      "lastError": "",
      "updatedAt": "2026-07-27T08:30:00Z",
      "status": "deployed"
    }
  }
}
```

`status` 的取值规则：

| 条件 | `status` |
| --- | --- |
| 在 scope 内，没有记录或没有快照 | `pendingCreate` |
| 在 scope 内，部署关联字段和快照不同 | `pendingModify` |
| 在 scope 内，部署关联字段和快照一致 | `deployed` |
| 已离开 scope，但还留着快照 | `pendingDelete` |

几个细节：

- scope 内还没有环境记录时，响应会补一条 `appliedFields: null` 的 `pendingCreate`；
- `immediate` 配置只有下发失败的环境会停在 `pendingCreate`，成功下发即 `deployed`；
- 存量数据缺 `registerMode` 时，响应固定补成 `on_deploy`，不会出现空串；
- 没有任何相关环境时 `envStates` 是 `{}`，不是 `null`；
- 模型中的 `EnvWeights` 即使为 `nil`，响应里的 `envWeights` 也固定序列化为 `{}`，`envDynamicWeights` 同理；
- `lastError` 不参与 `status` 计算；
- `appliedFields.polarisToken` 固定返回 `******`，前端靠 `polarisTokenChanged` 判断 Token 有没有变。

### 10.2 写入路径

| 操作 | 字段 / 路径 | 约束 |
| --- | --- | --- |
| Create | `scopeEnvNames` | 环境维度生效范围；所有 scope 环境自动初始化权重 `100` |
| Create | `registerMode` | 可选，`immediate` 或 `on_deploy`，缺省 `on_deploy`；显式传空串会被拒绝 |
| Create | `enableWeightFactor` | 可选，缺省 `false`；`envDynamicWeights` 不预建 |
| PATCH | `scopeEnvNames` | 传了就全量替换，`[]` 清空，不传（`nil`）表示不更新 |
| PATCH | `enableWeightFactor` | 不传（`nil`）表示不更新；改为 `false` 不动 `envDynamicWeights`，不改变 CR，也不触发动态下发 |
| PATCH | `registerMode` | 不暴露，创建后不可修改 |
| PUT | `/envs/{envName}/weight` | 允许 scope 内环境或任何已部署环境；待部署只持久化，已部署同步 Patch 集群 |
| PUT | `/envs/{envName}/weight` 的 `dynamicWeight` | 可选，不传保持原值；不受配置的 `enableWeightFactor` 约束 |

PUT 的 `weight` 取值范围是 `0 - 10000`，`0` 是合法值（表示不接流量）。没有显式环境值时固定使用 `DefaultEnvWeight = 100`。`weight` 始终必填，接口不提供只改 `dynamicWeight` 的形态：产品交互上基准权重和动态权重开关是一次保存提交的。

Create/PATCH serializer 不声明 `weight` 或 `envWeights`；旧客户端继续传这两个字段时，按当前 Gin JSON 绑定行为静默忽略。响应也不再包含配置级 `weight`。

### 10.3 PUT 示例

```json
{
  "weight": 20,
  "dynamicWeight": true
}
```

## 11. 行为速查

下表描述 `on_deploy` 配置。`immediate` 配置的行为集中在最后一组。

| 场景 | 是否动态下发 | 后续处理 |
| --- | --- | --- |
| 首次部署前改配置 | 否 | 等应用部署 |
| 已部署，只改普通 CR 字段 | 是（快照匹配时） | 异步 Upsert 完整 CR |
| 开关 `enableWeightFactor` | 否 | 只落库，不触发动态下发；CR 不因此变化 |
| scope 内待部署环境 PUT 权重 | 否 | 仅持久化，下次部署写进 CR |
| 已部署环境 PUT 权重 | 是 | 不受 readiness 限制，同步 JSON Patch 该环境的 weight |
| scope 外且未部署环境 PUT 权重 | 不适用 | 返回 400，不持久化 |
| 已部署环境 PUT 动态权重 | 是 | 与 weight 在同一次同步 JSON Patch 里提交，下发值不受 `enableWeightFactor` 影响 |
| PUT 请求不带 `dynamicWeight` | 是 | 开关不写库、保持原值，Patch 按库中现值下发 |
| 改部署关联字段 | 否 | 等应用部署更新完整资源和快照 |
| 把部署关联字段改回快照的值 | 是 | 下次 PATCH 重新满足条件 |
| 新环境加入 scope | 否 | 首次部署后建快照；权重补默认值，动态权重开关不预建 |
| 已部署环境离开 scope | 否 | 保留快照和环境级设置，下次离域部署时清理 |
| 未部署环境离开 scope | 不适用 | 立即删 `EnvState`、`envWeights` 和 `envDynamicWeights` |
| 环境卸载（仍在 scope） | 不适用 | 删 `EnvState`，保留 `envWeights` 和 `envDynamicWeights` |
| 环境卸载（已离域） | 不适用 | `EnvState` 与两份环境级设置都删 |
| 删除整条配置 | 不适用 | 下次部署按资源差异清理集群资源 |
| **以下为 immediate 配置** | | |
| 创建配置 / 新环境加入 scope | 是（同步） | Upsert CR + Service，成功写快照落到 `deployed`，失败写 `LastError` 并返回 500 |
| 改任意 CR 字段 | 是（同步） | 对 scope 内全部环境重新 Upsert，幂等 |
| 环境离开 scope | 不适用 | 同步删 CR + Service，成功后删 `EnvState` 与环境级设置，失败保留记录待重试 |
| 删除整条配置 | 不适用 | 先同步删所有相关环境的集群资源，全部成功才删配置记录 |
| 应用部署 | 不适用 | 部署仍会下发同样的 CR + Service，结果与平台主动下发一致 |

## 12. 存量数据迁移与发布顺序

### 12.1 registerMode 回填

`000007_polaris_register_mode_backfill` 给缺少该字段的文档写入 `registerMode: "on_deploy"`。

这次迁移不影响行为，只是让数据形状统一：`IsImmediateRegister()` 只在字段等于 `immediate` 时返回 `true`，缺字段的文档本来就按 `on_deploy` 处理。回填之后可以直接按 `registerMode` 查询和统计。`down` 只 `$unset` 值为 `on_deploy` 的字段，`immediate` 是用户显式选择的业务数据，回滚时必须保留。

### 12.2 envWeights 回填

迁移由 golang-migrate 的 `000003_polaris_env_weights_backfill` 完成，语句逐段说明见同目录的 `.md`。

**回填规则**：对每条 `polaris_configs`，取 `scopeEnvNames` 与 `envStates` 的 key 的并集作为目标环境，仅为缺失的 `envWeights.<env>` 写入该配置旧的顶层 `weight`，随后在同一条聚合管道里删除顶层 `weight`。已显式设置过的环境权重原样保留，因此重复执行结果一致。顶层 `weight` 字段缺失的文档整条跳过（属于新代码写入的数据）；`weight` 为 `0` 是显式配置，照常回填。

**继承旧值而不是统一回填 100**，因为旧代码写进 PolarisConfig CR 的 `spec.services[0].weight` 就是配置级 `weight`（接口默认 `10`）。若回填 `100`，平台认知与现网就会不一致，任何一次 CR patch（改 scope、改权重、重新部署）都会把线上单实例权重从 `10` 抬到 `100`。

**发布顺序**由 Helm 保证：`migrate-job` 先执行，web / worker 的 initContainer 等待该 Job 完成后才启动，因此迁移一定早于新代码生效。

**残留风险**：滚动更新窗口内，尚未下线的旧版本 Pod 若新建配置，仍会写顶层 `weight` 而不写 `envWeights`，这类数据迁移已经跑过、不会被覆盖，其环境会回落到 `DefaultEnvWeight`。发布后执行 `db.polaris_configs.countDocuments({ weight: { $exists: true } })` 核对，结果应为 `0`，非 `0` 则按同样规则手工补齐。
