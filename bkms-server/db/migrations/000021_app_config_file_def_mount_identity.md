# app_config_file_defs：挂载路径改为 appID + mountDir + name 唯一

## 行为

- 先删除 `appID + name` 和旧的 partial `appID + mountDir + name` 唯一索引，再回填。依赖 `000018` 已经建过这两个索引；索引不存在时，删除会在改数据之前失败。
- 回填并去掉末尾斜杠之后，再建非 partial 的 `appID + mountDir + name` 唯一索引。同一应用下，不同目录可以有同名文件。
- `appID + mountDir + name` 改为非 partial 唯一索引。`mountDir` 不落库时按空值参与唯一约束，Helm / Agones 的 values 文件名在同一应用下仍然唯一。
- 只有一条 framework def 的 trpc/taf 应用：把 `app_models` 上的 `fileName` / `filePath` 回填到该 def，并去掉末尾斜杠。回填成功后，从 `app_models` 去掉这两个字段。
- 有多条 framework def 的应用不回填、也不清 `app_models`。需要先跑框架 def 修复，再单独处理。
- `..` 这类路径只在后续写入时由 `path.Clean` 收齐，本迁移只去掉末尾 `/`。本次上线前已确认没有这类存量。

## 迁移前检查

前两类查询应该没有结果。有结果时先处理，再跑本迁移。改名冲突按下面的说明单独看。

`app_config_file_defs` 上要已经存在 `appID_1_name_1` 和 `appID_1_mountDir_1_name_1`。本迁移一开始就删除它们。

framework def 不是恰好一条的 trpc/taf 应用：

```javascript
db.app_models.aggregate([
  { $match: { "workload.type": { $in: ["trpc", "taf"] } } },
  { $lookup: {
      from: "app_config_file_defs",
      let: { appID: "$appID" },
      pipeline: [{ $match: { $expr: { $and: [
        { $eq: ["$appID", "$$appID"] },
        { $eq: ["$configKind", "framework"] }
      ] } } }],
      as: "defs"
  } },
  { $match: { $expr: { $ne: [{ $size: "$defs" }, 1] } } },
  { $project: { appID: 1, defCount: { $size: "$defs" } } }
])
```

恰好一条 framework def，但 model 的 `fileName` 或 `filePath` 为空。回填会整行跳过，上线后挂载名或目录仍是 def 上的旧值：

```javascript
db.app_models.aggregate([
  { $match: { "workload.type": { $in: ["trpc", "taf"] } } },
  { $lookup: {
      from: "app_config_file_defs",
      let: { appID: "$appID" },
      pipeline: [{ $match: { $expr: { $and: [
        { $eq: ["$appID", "$$appID"] },
        { $eq: ["$configKind", "framework"] }
      ] } } }],
      as: "defs"
  } },
  { $match: { $expr: { $eq: [{ $size: "$defs" }, 1] } } },
  { $project: {
      appID: 1,
      fileName: { $cond: [
        { $eq: ["$workload.type", "trpc"] },
        "$workload.trpcConfig.fileName",
        "$workload.tafConfig.fileName"
      ] },
      filePath: { $cond: [
        { $eq: ["$workload.type", "trpc"] },
        "$workload.trpcConfig.filePath",
        "$workload.tafConfig.filePath"
      ] }
  } },
  { $match: { $or: [
      { fileName: { $in: [null, ""] } },
      { filePath: { $in: [null, ""] } }
  ] } }
])
```

回填会把 framework def 改成 model 上的文件名，而同应用已经有另一条 def 使用这个名字。目录不同时，删掉旧的 `appID + name` 索引后可以迁移；目录也相同则先改名：

```javascript
db.app_models.aggregate([
  { $match: { "workload.type": { $in: ["trpc", "taf"] } } },
  { $lookup: {
      from: "app_config_file_defs",
      let: { appID: "$appID" },
      pipeline: [{ $match: { $expr: { $and: [
        { $eq: ["$appID", "$$appID"] },
        { $eq: ["$configKind", "framework"] }
      ] } } }],
      as: "defs"
  } },
  { $match: { $expr: { $eq: [{ $size: "$defs" }, 1] } } },
  { $project: {
      appID: 1,
      defID: { $arrayElemAt: ["$defs._id", 0] },
      oldName: { $arrayElemAt: ["$defs.name", 0] },
      newName: { $cond: [
        { $eq: ["$workload.type", "trpc"] },
        "$workload.trpcConfig.fileName",
        "$workload.tafConfig.fileName"
      ] }
  } },
  { $match: { newName: { $nin: [null, ""] }, $expr: { $ne: ["$oldName", "$newName"] } } },
  { $lookup: {
      from: "app_config_file_defs",
      let: { appID: "$appID", name: "$newName", defID: "$defID" },
      pipeline: [{ $match: { $expr: { $and: [
        { $eq: ["$appID", "$$appID"] },
        { $eq: ["$name", "$$name"] },
        { $ne: ["$_id", "$$defID"] }
      ] } } }],
      as: "clash"
  } },
  { $match: { "clash.0": { $exists: true } } }
])
```

下面两条都应该没有结果。有结果时先处理，再跑本迁移。

含 `.`、`..` 或连续斜杠的目录：

```javascript
db.app_config_file_defs.find(
  { mountDir: { $regex: "(^|/)\\.\\.(/|$)|(^|/)\\.(/|$)|//" } },
  { appID: 1, name: 1, mountDir: 1, configKind: 1 }
)
```

纯斜杠目录。`$rtrim` 会把它们收成空字符串再删掉字段，新写入的 `path.Clean` 则会保存为 `/`：

```javascript
db.app_config_file_defs.find(
  { mountDir: { $regex: "^/+$" } },
  { appID: 1, name: 1, mountDir: 1, configKind: 1 }
)
```

## 迁移后核对

末尾斜杠、空字符串和 `null` 都应该没有结果：

```javascript
db.app_config_file_defs.find(
  { $or: [
      { mountDir: { $regex: ".+/$" } },
      { mountDir: "" },
      { mountDir: null }
  ] },
  { appID: 1, name: 1, mountDir: 1, configKind: 1 }
)
```

## down

只恢复旧索引。已经回填的 def 和已经删除的 `app_models` 字段不会改回去。down 会删除本迁移创建的 `appID_1_mountDir_1_name_1`，所以只在本迁移已经跑完时执行。

若同一应用下已经存在同名文件，重建 `appID + name` 唯一索引会失败。
