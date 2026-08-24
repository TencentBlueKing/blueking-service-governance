# 部署记录：回填 workloadKind

## 背景

联邦环境开始把主工作负载写成原生 `Deployment` 后，部署记录增加了 `workloadKind`（`GameDeployment` 或 `Deployment`）。新写入的记录会带上该字段；本迁移补齐启动前的存量文档，避免各读路径反复说明「字段可能为空」。

读路径 `Record.MainWorkload()` 仍保留空值时从 `resourceKeys` 推断，作为防御。

## 迁移语句说明

`up` 是一条 `update` 命令，`multi: true`，`u` 使用聚合管道。

**筛选条件 `q`**

只处理 `workloadKind` 缺失、空串或 null，且 `resourceKeys` 里已有 `Deployment` 或 `GameDeployment` 的文档。推断不出主工作负载的文档不改，避免写成空串。

**管道：`$set workloadKind`**

与 `MainWorkload()` 一致：先取 `resourceKeys` 中第一个 `kind=Deployment`，否则取第一个 `kind=GameDeployment`。

已有非空 `workloadKind` 的文档不在筛选范围内，重复执行不会覆盖。

## down

无法区分「本迁移回填」和「新代码写入」的 `workloadKind`，回滚不能安全 `$unset`。`down.json` 为空命令数组，只回退 `schema_migrations` 版本号，数据保持迁移后的状态。

## 验证

```js
// 有主工作负载但 workloadKind 仍空的文档，应为 0
db.app_model_deploy_records.countDocuments({
  $and: [
    { $or: [{ workloadKind: { $exists: false } }, { workloadKind: "" }, { workloadKind: null }] },
    { "resourceKeys.kind": { $in: ["Deployment", "GameDeployment"] } }
  ]
})

db.app_model_deploy_records.find({}, { workloadKind: 1, resourceKeys: 1 }).limit(20)
```
