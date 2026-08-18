# 北极星配置：回填 registerMode

## 背景

北极星配置新增了注册模式开关 `registerMode`（见 `design_notes/polaris_config_auto_sync.md` 的「注册模式」一节），取值两种：

- `on_deploy`：配置参与 Workload 渲染（环境变量、容器端口、tRPC 框架配置），CR 随应用部署下发，是本次改动之前的唯一行为；
- `immediate`：配置不参与 Workload 渲染，绑定环境时直接下发 PolarisConfig CR 与配套 Service 完成注册。

存量文档没有这个字段。代码侧 `PolarisConfig.IsImmediateRegister()` 只在字段等于 `immediate` 时返回 `true`，因此缺字段的文档天然按 `on_deploy` 处理，行为不会改变。本迁移把缺省值显式写进文档，让存量数据与新写入的数据形状一致，便于直接按 `registerMode` 查询和统计。

## 迁移语句说明

`up` 是一条 `update` 命令，`multi: true`。

**筛选条件 `q`**

```json
{ "registerMode": { "$exists": false } }
```

只处理还没有该字段的文档。已经显式写过 `registerMode` 的（包括新代码创建的 `immediate` 配置）一律跳过，因此迁移幂等，重复执行结果一致。

**更新 `u`**

```json
{ "$set": { "registerMode": "on_deploy" } }
```

写入缺省模式。这里不用聚合管道，因为回填值是常量，不依赖文档其他字段。

## down

`$unset` 掉 `registerMode` 为 `on_deploy` 的文档的该字段，回到迁移前的形状。用 `registerMode: "on_deploy"` 而不是无条件 `$unset`，是为了保留 `immediate` 配置的模式——那是用户显式选择的业务数据，删掉会让配置在回滚后静默变回 `on_deploy`，下一次部署时环境变量突然出现、CR 也不再由平台主动维护。

## 验证

迁移后在业务库执行：

```js
// 应为 0：不应再有文档缺少 registerMode
db.polaris_configs.countDocuments({ registerMode: { $exists: false } })

// 分布抽查
db.polaris_configs.aggregate([{ $group: { _id: "$registerMode", count: { $sum: 1 } } }])
```

第一条查询结果非 `0`，说明滚动发布窗口期内仍有旧版本 Pod 创建过配置（迁移 Job 先于新版本 Pod 启动，但旧 Pod 尚未完全下线）。这类文档按同样规则补一次即可，在补齐之前它们的行为也是 `on_deploy`，不会出错。
