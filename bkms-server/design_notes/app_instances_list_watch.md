# 应用实例 List 全量与 Watch 增量

## 1. 背景

实例列表是「一次 List 拿全量首包，之后由 Watch 推增量」。筛选排序放在前端本地做，服务端不提供筛排参数。CLI 等非 UI 调用方继续走分页 List，两种模式共存于同一个接口。

## 2. 两个接口如何衔接

衔接点是 `resourceVersion`——集群 List 响应里的位点，Watch 带着它才能不重不漏地接上。它由 List 成功响应返回，Watch 请求必填。

```mermaid
sequenceDiagram
    participant UI as 前端
    participant Server as bkms-server
    participant K8s as 集群

    UI->>Server: List all=true
    Server->>K8s: List Pods（按部署记录的 LabelSelector）
    K8s-->>Server: Pods + resourceVersion
    Server-->>UI: 全量投影 + resourceVersion
    UI->>Server: Watch（带 resourceVersion）
    Server->>K8s: Watch from resourceVersion
    loop 连接存活
        K8s-->>Server: Pod 变更
        Server-->>UI: Pod 投影事件
        Server-->>UI: PLUGIN 事件（约 15s，仅有差异时）
    end
```

两个接口的作用域完全一致，都以该环境/泳道的**最新部署记录**为准：集群、命名空间、LabelSelector 都取自部署记录，不从请求参数推断。鉴权、非 AppModel、无部署记录这三种拒绝也走同一套，因此 Watch 建不成连接的条件与 List 失败的条件相同。

服务端拉集群 Pod 时可能翻多页，`resourceVersion` 固定取**第一次**响应的值，后续页不覆盖它。否则位点会往前跳，漏掉中间发生的变更。

## 3. List 全量

`all=true` 一次返回该应用环境下匹配的全部实例，与 `page` / `pageSize` 互斥。

两种模式对坏数据的口径不同：全量模式下单个 Pod 投影失败只跳过该实例并记入 `skipped`，`count` 只算成功投影的条数；分页模式下当前页任一 Pod 解析失败即整次请求失败，避免 CLI 拿到残缺页。

分页的 `page` 有上界，因为 `(page-1) * pageSize` 在极大页码下会整数溢出成负的起始下标。上界在参数校验阶段拦截，投影时不再重复判断。

北极星在 List 里内嵌在每条实例的 `polarisInfos` 上，匹配与降级见 §5。

## 4. Watch 投影流

Watch 把集群 Pod 变更转成**平台投影事件**：事件里的 `object` 是 `AppInstanceOutputObj`，与 List 的元素同构，不是原生 Pod JSON。只推增量，不推首包快照。

传输用 SSE。同一条流上有两类事件，信封不同：

- **Pod 事件** `ADDED` / `MODIFIED` / `DELETED` / `ENDED`，由基础层产出。`DELETED` 只保证 `id`；`ENDED` 的 `object` 为 null。Pod 事件**不承载附属数据**，`polarisInfos` 恒为空数组
- **附属数据事件** `PLUGIN`，由插件层产出，带 `plugin` 标识来源，`object` 为 `{id, data}`。它不是实例的增删改，只更新已有行上某个插件的数据

`watch.Manager` 建流后进一条 select 循环，同时处理集群事件、定时 tick（心跳 + 插件）、连接硬上限和连接取消。异常口径为：

- 单个 Pod 投影失败：跳过该事件，流继续，不推 `skipped`
- BOOKMARK：只推进位点，不向前端转发
- 集群通道关闭或返回 Error 事件：先推 `ENDED`（`reason=cluster watch interrupted`）再收流
- 连接达到 2 分钟硬上限：先推 `ENDED`（`reason=watch timeout`）再收流
- 集群 Watch 建不起来（此时尚未写响应头）：位点过期（410 Gone / Expired）返回 409，其他集群故障返回 500，都不写成 200 的事件流

### 4.1 断流后必须重新 List

收到 `ENDED` / `onerror` / `onclose` 后，前端**必须重新 List 拿新位点**再建 Watch，不能复用旧的。

集群侧位点有保留窗口，过期后 apiserver 直接拒绝，拿旧位点重试只会立刻又收到 `ENDED`，形成重连死循环。重新 List 同时补回了断流期间丢失的增量。

### 4.2 长连接超时

有两层超时，职责不同：

- **单次写 30s**：`http.Server.WriteTimeout` 是整条响应的写入上限而不是空闲超时，心跳挡不住它。每次写 SSE 前用 `http.NewResponseController` 把写超时重置为「单次写 30s」，避免客户端连着不读时写缓冲写满、goroutine 永久阻塞
- **连接总时长 2 分钟**：不对客户端承诺更长的 SSE。到期推 `ENDED`（`reason=watch timeout`），同时给集群 Watch 带上同样量级的 `TimeoutSeconds`。前端按 §4.1 重新 List 再建 Watch

响应头写出之后就改不了 HTTP 状态码，因此流中途的写失败只落 WARN 日志，不再向调用方返回错误。

### 4.3 分层与插件

Watch 分两层，边界就是包边界：

- **基础层** `watch/`：Pod 的 list/watch、投影、SSE、心跳、`ENDED`、连接生命周期。不引用任何附属数据源（靠 Code Review 守住）
- **插件层** `watch/plugin/`：`Plugin` 接口与 `Runner`。每个 tick 把本连接当前存活的全量实例快照交给已注册插件，差异以 `PLUGIN` 事件推出
- **插件实现** `watch/plugin/polaris/`：当前唯一的生产插件；在 `handler.newWatchManager` 注册

```mermaid
graph LR
    handler[instance/handler] --> base[watch.Manager]
    handler --> polaris[watch/plugin/polaris]
    base --> runner[plugin.Runner]
    runner --> polaris
```

依赖单向：基础层只认识 `Plugin` 接口。新增一种附属数据 = 实现接口 + 在 handler 注册，基础层零改动，事件 `type` 也不用扩。

插件与 Runner 的契约：

- 每个周期拿到**全量存活快照**（`pushedInstances`：已成功推送且未 `DELETED` 的实例），不是本轮有变动的 Pod。附属数据变化可以与 Pod 无关
- 快照按值交给插件，改顶层字段不影响其它插件和 `pushed` 缓存
- 插件只返回当前值，不维护历史、不判断变化。集合类载荷必须定序
- Runner 拿上次成功推送的载荷做深比较，只有差异才推。深比较能处理载荷里的 map，但顺序不稳会被误判成变化
- 首次见到且载荷为空则只记账不推，避免未配置该数据源的应用刷出一串空事件；曾经推过非空再变空，必须推
- 单个插件失败（含超时）只跳过本轮该插件：不推、不拆流、不动已推送记录，也不影响其它插件与 Pod 事件。插件必须尊重 `ctx` 取消

插件调用留在 `consume` 的同一个 goroutine 里。`pushedInstances` 与 `Runner` 都不加锁，前提是 Pod 事件与周期任务是同一 goroutine 的两个 select case。心跳与全部插件共用同一 tick（15s，对齐北极星 SDK 缓存）；有独立节奏需求时再按插件拆。

### 4.4 前端如何合并两类事件

| 事件 | 前端动作 |
|------|---------|
| `ADDED` | 按 `object.id` 新增本地行；附属数据暂空，最多一个周期（约 15s）后由 `PLUGIN` 补齐 |
| `MODIFIED` | 按 `object.id` 更新 K8s 字段，**不要覆盖**该行已有的插件数据 |
| `DELETED` | 按 `object.id` 移除本地行及其插件数据 |
| `PLUGIN` | 按 `object.id` 找到本地行，用 `data` 覆盖 `plugin` 对应的附属数据；找不到行则忽略 |
| `ENDED` / `onerror` / `onclose` | 按 §4.1 重新 List 取新位点后重建 Watch |

附属数据的首包与增量来源不同：**首包读 List 响应里内嵌的 `polarisInfos`，之后的更新只看 `PLUGIN` 事件**。Watch 的 Pod 事件里 `polarisInfos` 恒为空数组，不要从那里取。

`PLUGIN` 的 `data` 由插件决定。`plugin=polaris` 时是 `PolarisInstanceInfoOutputObj` 列表，可以是空列表（该实例当前没有注册信息）。前端按 `plugin` 取值分派，信封结构不随新插件变化。

## 5. 北极星信息

北极星实例状态（健康 / 权重 / 隔离）按 Pod IP + 服务端口匹配。匹配与定序在 `addon/polaris.InstanceMatcher`，API 投影走 `PolarisInstanceInfoOutputObj.FromModel`：

- **List**：`MergePolarisInfoToAppInstances` 把投影写进实例的 `polarisInfos`
- **Watch**：北极星没有原生 Watch，只能周期拉取。插件每个 tick 用 Matcher 取领域结果，写出前 `FromModel`，有差异则推 `PLUGIN` 事件

周期 15s，与 SSE 心跳共用一个 ticker。间隔取自北极星 SDK 的缓存量级：拉得更密拿回来的是同一份数据。新实例最多等一个周期才拿到北极星信息。

`polarisInfos` 按 `(serviceNamespace, serviceName, port)` 定序。北极星侧返回顺序会漂移，不定序就会被 Runner 误判成变化、每轮重复推送。定序发生在 Matcher 里。

未命中返回**空切片而不是 nil**，与「配置被删后成功拉回空」同形，才能被识别为变化并推 `data: []`。

### 5.1 拉不到时的两种口径

北极星是旁路信息源，不可用不牵连 Pod 数据。List 与 Watch 的降级不同：

- **List**：降级为 `polarisInfos=[]`，K8s 字段照常返回。结果与「该应用确实没注册北极星」同形，服务端不做区分，前端一律展示为未知
- **Watch**：跳过本轮，不推事件、不拆流。页面保留上一轮拿到的真实状态；清成空是信息倒退。失败只落 WARN，并计 `bkms_instance_watch_plugin_fetch_total{plugin,result}`
