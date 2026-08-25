# 配置文件：回填 configKind

## 背景

新增 `configKind` 字段用于区分 `framework`（框架配置）和 `plain`（容器挂载文件）两种语义。新写入的记录已携带该字段；本迁移补齐存量文档，使所有记录拥有显式的 `configKind` 值。

存量数据在引入 `plain` 类型之前创建，全部属于 `framework` 语义。

读路径 `GetConfigKind()` 仍保留空值时返回 `framework` 的防御逻辑，迁移完成后可移除该 fallback。

## 迁移语句说明

`up` 包含两条 `update` 命令，分别针对 `app_config_files` 和 `app_config_file_versions` 集合。

**筛选条件 `q`**

只处理 `configKind` 缺失、空串或 null 的文档。已有非空 `configKind` 的文档不在筛选范围内，重复执行幂等。

**更新：`$set configKind: "framework"`**

将缺失的 `configKind` 统一设为 `"framework"`，与 `GetConfigKind()` 的 fallback 行为一致。

## down

无法区分「本迁移回填」和「新代码写入」的 `configKind`，回滚不能安全 `$unset`。`down.json` 为空命令数组，只回退 `schema_migrations` 版本号，数据保持迁移后的状态。

## 验证

```js
// configKind 仍为空的文档，应为 0
db.app_config_files.countDocuments({
  $or: [
    { configKind: { $exists: false } },
    { configKind: "" },
    { configKind: null }
  ]
})

db.app_config_file_versions.countDocuments({
  $or: [
    { configKind: { $exists: false } },
    { configKind: "" },
    { configKind: null }
  ]
})

// 抽样检查
db.app_config_files.find({}, { configKind: 1, name: 1 }).limit(20)
db.app_config_file_versions.find({}, { configKind: 1, name: 1, version: 1 }).limit(20)
```
