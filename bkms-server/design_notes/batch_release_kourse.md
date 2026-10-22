# 分批发布

分批发布让 appmodel 应用部署新版本时支持按自定义策略，逐批替换实例，每批之间可以暂停、人工确认，出问题时回滚。

平台**不自己实现分批逻辑**，而是依赖 BCS Kourse 的 Rollout 控制器（`bcs-kourse-rollout`）：平台把用户选择的发布策略翻译成 Rollout CR 下发到集群，由控制器按批次调整 GameDeployment 的 `partition` 来推进实例更新(**和现在平台灰度发布的逻辑类似**)；平台通过读取 Rollout 的 status 判断进度，通过修改 status 子资源实现继续、暂停、跳过、回滚等人工干预。

## 1. 整体方案

| 职责 | 平台（bkms-server） | Kourse Rollout 控制器 |
|------|------|------|
| 批次节奏 | 空间级发布策略的增删改查 | — |
| 下发 | 策略 → Rollout CR，与主工作负载同名、同命名空间 | — |
| 批次推进 | — | 按 `steps` 调整 GameDeployment `partition`，处理暂停 |
| 人工干预 | 校验前置条件后修改 Rollout status 子资源 | 响应 `paused`、`currentStepIndex` 的变化 |
| 进度与终态 | 轮询任务读取 Rollout + GameDeployment 状态，写入部署记录 | 维护 Rollout status |
| 冲突拦截 | 分批发布中拒绝新部署、下架、扩缩容等操作 | — |

```
用户选择策略部署
  → 平台生成 Rollout CR 并下发
  → 平台更新 GameDeployment（webhook 注入 partition = replicas，新版本 Pod 暂不创建）
  → 控制器逐批降低 partition，按策略暂停
  → 平台轮询 Rollout / GameDeployment，写入部署记录的 batchProgress
  → 用户查看进度，继续 / 暂停 / 回退 / 跳过 / 回滚（平台修改 Rollout status）
  → Rollout 进入 Healthy 且工作负载已是新版本 → 部署记录置为 deployed
    或回滚完成 → 部署记录置为 rolled-back
```

## 2. 一期功能范围

| 功能 | 说明 | 状态 |
|------|------|------|
| 公共模型与 Rollout 封装 | 发布策略存储、部署记录扩展字段、Rollout CR 生成 / 读写 / status 修改与解析、状态判定 | 已完成 |
| 发布策略增删改查 | 空间级自定义策略，空间下所有应用可选用；权限同空间设置 | 已完成 |
| 按策略分批部署 | 部署接口新增可选的 `releaseStrategyID`，传入时下发 Rollout 并分批更新 | 已完成 |
| 全量部署清理 | 不传策略时，先删除遗留的 Rollout CR 和 `rollout` 标签，再一次更新全部实例；实例更新（全量 / 灰度）修改镜像前同样清理 | 已完成 |
| 进度同步与查询 | 部署轮询任务同步 Rollout 状态到部署记录；进度查询接口只读库 | 待实现 |
| 终态判定 | 已部署 / 已回滚；分批发布记录不受轮询超时限制 | 待实现 |
| 手动操作 | 继续、暂停、回退上一批、跳过剩余批次、回滚 | 待实现（前置条件校验已完成） |
| 冲突拦截 | 分批发布中拒绝新部署、下架、实例灰度更新、删除实例、手动扩缩容 | 待实现（判定方法已完成） |
| 审计 | 策略增删改、分批部署、手动操作写入空间操作记录 | 部分完成（策略增删改、分批部署） |

一期**不包含**：

- 流量调整(涉及北极星的流量权重调整)。
- 批次后置检查（hook）、失败自动回滚。

## 3. 数据模型

### 3.1 发布策略

集合 `release_strategies`，归属空间，`(workspaceID, name)` 唯一。

| 字段 | 说明 |
|------|------|
| `name` / `description` | 名称 1 ~ 32 字符，空间内唯一；描述不超过 200 字符 |
| `batches` | 1 ~ 10 个批次 |
| `batches[].target` | 本批**累计更新到**的实例数：正整数（如 `"1"`）或百分比（`1%` ~ `100%`），每批可独立选择单位 |
| `batches[].pauseMode` | 本批完成后的暂停方式：`none` 不暂停 / `duration` 定时暂停 / `permanent` 永久暂停（需人工继续） |
| `batches[].pauseSeconds` | 仅 `duration` 时有效，60 ~ 86400 秒 |

字段取值范围由接口层校验；`ParseBatchTarget` 负责目标值格式校验，生成 Rollout 时复用。

### 3.2 部署记录扩展

| 字段 | 说明 |
|------|------|
| `releaseMode` | `all` 一次更新全部 / `canary` 按策略分批。存量记录没有该字段，读取时通过 `GetReleaseMode()` 视为 `all`，无需迁移 |
| `releaseStrategy` | 发起时的策略快照（ID、名称、批次）。策略之后被修改或删除，进度查询仍以快照为准 |
| `batchProgress` | 轮询任务写入的进度：当前批次序号（缺失为 `nil`，与第一批的 `0` 区分）、批次状态、Rollout 原始 phase 与提示、暂停开始时间、已更新 / 总实例数、新旧镜像 |
| `status` | 新增 `rolled-back`，属于终态（`IsStable()` 为真） |

`Record.IsBatchReleasing()`：发布方式为 `canary` 且状态为 `deploying` 时视为"分批发布中"。冲突拦截和手动操作都以环境 / 泳道的**最新部署记录**为准，不实时请求集群。

`rolled-back` 的语义：Pod 已全部恢复旧版本，但工作负载 spec 和部署记录中的镜像**仍是本次部署的新版本**（见 4.6）。工作负载仍在运行，所以 `HasActiveDeployments` 把它算作活跃部署。

## 4. 与 Rollout 的交互

### 4.1 生成 Rollout CR

`BuildManifest` 由主工作负载和策略批次生成 CR：

- 主工作负载必须是 GameDeployment，否则返回 `ErrWorkloadNotSupported`。
- CR 与主工作负载同名，`spec.objectRef.workloadRef` 指向它。
- 批次目标原样写入 `steps[].replicas`，**百分比不换算为个数**，由 Kourse 按实际副本数计算。
- 暂停方式映射：

| pauseMode | 暂停方式 | `steps[].pause` |
|------|------|------|
| `none` | 本批完成后不暂停，直接进入下一批 | 不写 |
| `duration` | 定时暂停，到期后自动进入下一批 | `{duration: "<秒数>s"}` |
| `permanent` | 永久暂停，需人工继续 | `{}` |

示例：策略 [1 个 / 永久暂停，50% / 不暂停，80% / 定时 1800 秒] 生成：

```yaml
apiVersion: tkex.tencent.com/v1alpha1
kind: Rollout
metadata:
  name: demo            # 与主工作负载同名
spec:
  type: Canary
  failurePolicy: Present
  objectRef:
    workloadRef:
      apiVersion: tkex.tencent.com/v1alpha1
      kind: GameDeployment
      name: demo
  steps:
    - replicas: 1
      pause: {}
    - replicas: "50%"
    - replicas: "80%"
      pause:
        duration: 1800s
```

### 4.2 下发

**顺序很关键**。GameDeployment 的 pod template 变化时，Kourse 的 mutating webhook 会查找同命名空间中 `workloadRef` 指向它的 Rollout：找到才注入 `partition = replicas`（新版本 Pod 暂不创建）；找不到就直接放行，GameDeployment 按自身策略**整批更新**。webhook 与控制器共用同一个缓存，Rollout 刚创建时缓存可能还没同步。

整批更新后 Rollout 照样会走到 `Healthy`，从 Rollout status 上看不出没有分批。因此分批部署按以下顺序执行：

1. Apply Rollout CR（`RolloutClient.Apply`，Server-Side Apply）。
2. 等待控制器调谐过该 Rollout 的当前 spec：
   - Rollout `status.workloadRevision` 已写入，说明缓存中已有该 Rollout；
   - 如果本次修改了已有 Rollout 的 spec（Apply 前后 `metadata.generation` 不同），还要等 `status.rolloutRevision` 离开 Apply 前的值。上一轮遗留的 Rollout 早已写入 `workloadRevision`，单看它会立即返回。控制器若先看到工作负载版本变化，会按**旧批次**开始一轮，随后发现 spec 变化再从第 0 批重来，旧批次已更新的实例会被白白更新一次再退回（已在集群中复现）。
3. 更新 GameDeployment，并为其加上标签 `rollout: "true"`。控制器检查该标签，缺失时会直接放开 partition 整批更新（见 4.4）。
4. 校验 GameDeployment `spec.updateStrategy.partition == replicas`，确认 webhook 已注入；不满足时部署记录置为失败，不能当作分批发布继续轮询。

GameDeployment 首次创建时没有旧版本实例，无需分批：仍先下发 Rollout、带上标签，但跳过第 2、4 步（Rollout 指向不存在的工作负载时控制器不会写入 `workloadRevision`）。

**全量部署清理**：不传策略时，如果主工作负载存在同名 Rollout CR 或 `rollout` 标签，先删除 CR（CR 不存在或集群未安装 CRD 都视为没有遗留）并等待删除完成、移除标签，再走现有部署流程。否则遗留的 Rollout 会继续按旧策略分批。删除失败时整个部署失败，不更新工作负载。

**资源追踪**：Rollout 不写入部署记录的 `ResourceKeys`（其删除必须先于工作负载更新，通用的过期资源删除在更新之后执行），因此单独处理：分批部署时 Rollout CR 写入资源清单快照；下架时先删除与主工作负载同名的 Rollout，不以最新记录的发布方式判断（全量实例更新生成的新记录不带发布方式）。

**审计**：按策略分批部署在下发成功时写入空间操作记录（`deploy` / `app`，extras 含策略 ID 与名称），不等待轮询任务进入终态。

**不要在发布进行中重新 Apply 批次不同的 Rollout**：控制器发现 Rollout spec 变化后，会清理灰度资源并从第 0 批重新开始。

### 4.3 读取 status

`ParseStatus` 解析以下字段，字段缺失时为零值：

| 字段 | 含义 |
|------|------|
| `status.phase` | 整体阶段，取值和流转见 4.4 |
| `status.message` | 整体提示 |
| `status.paused` | 字符串：`true` 暂停在当前批次 / `false` / `temporary` 定时暂停中 / `manually` 立即冻结所有动作 |
| `status.workloadRevision` | Rollout 正在跟踪的工作负载版本，格式与 GameDeployment `status.updateRevision` 相同（`<name>-<hash>`），可直接逐字比较 |
| `status.rolloutRevision` | 控制器已观察到的 Rollout spec 的哈希，spec 相同则取值相同。修改 spec 后，该值变化说明控制器已看到新 spec |
| `status.stepStatus.currentStepIndex` | 当前批次序号，从 0 开始；`-1` 表示回滚。批次全部走完后整个 `stepStatus` 被清成 `{}`，该字段不存在，解析为 `nil`，**必须和 0 区分** |
| `status.stepStatus.currentStepPhase` | 当前批次阶段：`ScalingReplicas` 更新本批实例 / `PausedTemporarily` / `PausedPermanently` / `PausedManually` / `Error`。两批之间为空；回滚后不清理，残留回滚前的值 |
| `status.stepStatus.message` | 当前批次提示，批次出错时的原因写在这里 |
| `status.stepStatus.pauseStartTime` | 当前批次暂停开始时间 |


### 4.4 Rollout 控制器的行为

控制器每次调谐按以下顺序检查，前一步命中即返回：

1. `paused=manually`：什么都不做，`currentStepPhase` 置为 `PausedManually`。
2. 工作负载版本变化（`workloadRevision` 不等于 GameDeployment 的 `updateRevision`）：开始新一轮发布，见下方流转。
3. Rollout spec 变化（`rolloutRevision` 与当前 spec 的哈希不同）：清理灰度资源，从第 0 批重新开始，并更新 `rolloutRevision`。没有进行中的发布（`Healthy`）时只更新 `rolloutRevision`。
4. 检查 GameDeployment 状态：

| 条件（按顺序） | Rollout 行为 |
|------|------|
| 不存在 / 删除中 | 清理后置为 `Healthy` |
| 缺少标签 `rollout: "true"` | 放开 partition，**直接整批更新**并进入 `Healthy` |
| `generation != observedGeneration` | 跳过本次调谐 |
| 所有实例已是 `updateRevision` 且就绪 | 发布结束，见下方流转 |
| 设置了 partition 且 `currentRevision != updateRevision` | 继续推进批次 |
| 其他 | 跳过本次调谐 |

5. 推进批次：先判断回滚（`currentStepIndex < 0`），再按当前批次调整 `partition = replicas - 本批目标数`，本批 Pod 就绪后按策略暂停或进入下一批；序号等于批次总数时放开全部实例。

一轮分批发布的 phase 流转：

```
（工作负载版本变化）
CleaningCanary    清理上一轮灰度资源，workloadRevision 仍为旧值
  → Running         workloadRevision 切为新值，currentStepIndex 从 0 开始逐批推进，可能暂停
  → UpdatingPrimary 所有批次走完（或跳过），放开 partition 更新剩余实例，stepStatus = {}
  → CleaningCanary  所有实例已更新就绪，清理灰度资源，workloadRevision 为新值
  → Healthy
```

需要注意的点：

- `CleaningCanary` 在开头和结尾**各出现一次**，不能理解为"已是目标版本"。
- `workloadRevision` 在发布**开始**时就切到新值，不是结束时。
- `Healthy` 表示"没有进行中的发布"。Rollout 刚创建、工作负载还没更新时也是 `Healthy`。
- Rollout 首次调谐时直接取当前 `updateRevision`，不会触发一轮发布。这就是 4.2 要求先建 Rollout、后改工作负载的原因。
- 批次出错（调整副本失败、批次序号越界等）时 phase 仍为 `Running`，`currentStepPhase=Error`，控制器持续重试。
- `Terminating`、`RollbackPrevious`、`RollbackInPresent` 等 phase 只在 hook 失败时出现，平台不配置 hook，不会遇到。

### 4.5 批次状态与发布完成的判定

`Status.BatchPhase(workload)` 把 Rollout status 和 GameDeployment 快照映射为批次状态，按以下顺序判断：

| 顺序 | 条件 | 批次状态 |
|------|------|------|
| 1 | `currentStepIndex < 0` | `updating`（回滚中）。必须最先判断，否则残留的 `PausedPermanently` 会被当成暂停 |
| 2 | phase 为 `UpdatingPrimary` 或 `CleaningCanary` | `updating` |
| 3 | phase 为 `Healthy`，`workloadRevision` 等于 GameDeployment `updateRevision`，且 GameDeployment 已观察到最新 generation、所有实例已更新并就绪 | `completed` |
| 4 | phase 为 `Healthy` 的其他情况（如 Rollout 刚创建、工作负载尚未更新） | `pending` |
| 5 | `paused` 为 `true` / `manually`，或步骤阶段为 `PausedPermanently` | `manual-paused` |
| 6 | `paused` 为 `temporary`，或步骤阶段为 `PausedTemporarily` | `timed-paused` |
| 7 | 步骤阶段为空（两批之间） | `pending` |
| 8 | 其他（`ScalingReplicas`、`Error` 等） | `updating`；`Error` 时进度提示取 `stepStatus.message` |

第 3 条中 GameDeployment 的检查不能省：`workloadRevision` 在发布开始时就已是新值；缺少 `rollout` 标签时 Kourse 也会直接进入 `Healthy`。

批次状态为 `completed` 时，部署记录置为 `deployed`。

### 4.6 手动操作

手动操作通过 Merge Patch 修改 Rollout 的 **status 子资源**（`RolloutClient.UpdateStatus`）。`Status.ActionPatch(action, batchCount)` 负责校验前置条件并给出要修改的内容：

| 操作 | 前置条件 | 修改 status | 效果 |
|------|------|------|------|
| 继续 `continue` | 暂停中（手动 / 永久 / 定时） | `paused: "false"` | 进入下一批；定时暂停中执行则提前进入下一批 |
| 暂停 `pause` | 未暂停 | `paused: "true"` | 不再进入下一批，本批已开始更新的实例继续完成 |
| 回退上一批 `backward` | 当前批次序号 ≥ 1 | `paused: "false"`，`currentStepIndex` 减 1，`pauseStartTime: null` | 多出的新版本 Pod 退回旧版本，按上一批策略重新执行（清空 `pauseStartTime` 避免被 Kourse 当成已暂停过而直接跳回下一批） |
| 跳过剩余批次 `skip` | — | `paused: "false"`，`currentStepIndex` = 批次总数 | 放开 partition，所有实例更新为新版本，最终进入 `Healthy` |
| 回滚 `rollback` | — | `paused: "false"`，`currentStepIndex: -1` | 所有 Pod 恢复旧版本 |

**回滚的语义与判定**

回滚是 Kourse 的"完全回滚"：控制器把 partition 恢复为副本数，已更新的 Pod 退回旧版本（`currentRevision`）。**工作负载 spec 和 `updateRevision` 仍是新版本**，只有 Pod 是旧版本。部署记录因此置为 `rolled-back`，镜像字段保留新镜像。

控制器一收到回滚就把 phase 置为 `RollbackCompleted`，并且在回滚的 Pod 就绪之前就提示"rolled back successfully"，所以不能以 phase 或提示作为完成依据。`Status.IsRolledBack(workload)` 要求同时满足：

- Rollout phase 为 `RollbackCompleted`，且 `currentStepIndex < 0`；
- GameDeployment 已观察到最新 generation；
- `updatedReplicas == 0`，且 `replicas`、`readyReplicas` 都等于 `spec.replicas`。

满足后部署记录置为 `rolled-back`。回滚后 Rollout 保持该状态，直到下一次工作负载版本变化。

## 5. 进度同步与冲突拦截

这两部分尚未实现，以下为设计约束。

**进度同步**（复用现有部署轮询任务）：

- 只处理 `releaseMode=batch` 且未进入终态的记录，每个轮询周期读取同名 Rollout 和 GameDeployment，按 4.5 计算批次状态，写入 `batchProgress`。`releaseMode=all` 的记录保持现有逻辑。
- 终态：批次状态为 `completed` 时置为 `deployed`；`IsRolledBack` 为真时置为 `rolled-back`。终态后停止轮询。
- **分批发布记录不受轮询超时窗口限制**（`TaskPoller.DeployStatus.Timeout`）。永久暂停可以无限期等待人工确认，如果沿用超时窗口，记录会被置为 `polling-timeout`，`IsBatchReleasing()` 随之返回 false，冲突拦截失效，手动操作也会因"没有进行中的分批发布"被拒绝。
- 读取 Rollout 失败时保留上一次进度，下个周期重试；连续失败沿用现有重试次数，耗尽后置为 `polling-broken`。
- 进度查询接口只读部署记录，不实时请求集群。

**冲突拦截**：环境 / 泳道的最新部署记录处于分批发布中时，拒绝新部署（全量和分批）、下架、实例灰度更新、删除实例、手动扩缩容，返回 `ErrBatchReleaseInProgress`。重启实例、执行管理命令、调整北极星权重 / 隔离不受影响。拦截只读库，不请求集群。

## 6. 已知风险与待办

- **`rolled-back` 状态的下游语义**：回滚后工作负载 spec 引用的仍是新镜像，但现有逻辑大多只认 `deployed`。以下几处需要在实现终态判定时一并处理：
  - 镜像 tag 删除保护：不把 `rolled-back` 视为占用，新 tag 被删后，Pod 重建会拉不到镜像。
  - 泳道部署前检查：要求基线泳道为 `deployed`。
  - 轮询收尾的审计与指标：只把 `deployed` 记为成功。
  - 前端状态枚举。
- **进度写入与其他写操作的竞争**：通用的 `UpdateRecord` 不写 `batchProgress`，进度由 `UpdateBatchProgress` 定向更新。轮询接入时必须走这个函数，不能把进度放进 `UpdateRecord` 的整记录写回。
- **webhook 缓存竞争**：4.2 的第 2、4 步是对 webhook 缓存未同步的兜底，需要在真实集群验证。
- **回滚后换策略会重新发布被回滚的版本**：回滚后 GameDeployment 的 `updateRevision` 仍是被回滚的版本。Rollout 处于 `RollbackCompleted` 时，修改其 spec 或删除后重建，控制器都会立即从第 0 批开始发布被回滚的版本，早于平台更新工作负载（已在集群中复现）。策略不变时 spec 不变，不受影响。处理方式待定。
- **轮询中断**：记录置为 `polling-broken` 后不再是"分批发布中"，但集群中的 Rollout 可能仍在进行，兜底方式另行设计。
