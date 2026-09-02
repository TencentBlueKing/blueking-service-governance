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
- `dynamicWeight.enable`

一次 PATCH 可以同时改两类字段。系统不去分析请求里具体改了哪些字段，只看**改完之后**的三个部署关联字段和环境快照是否还一致。

环境权重不再属于配置 PATCH。它只能通过环境级 PUT 修改，并走独立的 JSON Patch 下发路径。

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

这一节只适用于 `on_deploy` 配置。`immediate` 配置不影响工作负载，没有"等部署"这回事，scope 内的环境一律在请求内同步下发

## 4. 行为速查
下表描述 `on_deploy` 配置。`immediate` 配置的行为集中在最后一组。

| 场景 | 是否动态下发 | 后续处理 |
| --- | --- | --- |
| 首次部署前改配置 | 否 | 等应用部署 |
| 已部署，只改普通 CR 字段 | 是（快照匹配时） | 异步 Upsert 完整 CR |
| 开关 `enableWeightFactor` | 否 | 平台创建的服务先同步北极星服务 metadata（开启写入固定公式，关闭删除两项），再落库；引入的服务 PATCH 该开关直接报错。不触发 CR 动态下发 |
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