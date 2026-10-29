# 租户集合回填 tenantID，组件定义按租户划分

## 背景

`workspaces`、其余 50 个集合（含 4 个计数器集合）与 `component_defs` 接入 store 层租户过滤（`Meta.TenantScoped = true`），
方案见 `design_notes/tenant_isolation.md`。接入后读写都会附加 `tenantID` 条件，存量文档没有这个字段就查不到，
因此**本迁移必须先于新代码上线**。

存量环境都是单租户，数据全部归属保留租户 `default`（`pkg/infras/tenant.DefaultTenantID`）。
单租户模式下平台租户也是 `default`，内置组件定义无需搬迁。

保持全局、不在本迁移中的集合：`runtime_images`、`image_snapshots`、`image_tag_suppressions`、
`repo_snapshot_statuses`、`cluster_addon_defs`、`depservice_services`、`bkci_pipeline_templates`、
`plat_admin_role_bindings`。

## up

1. `workspaces` 与其余 50 个集合（清单见 `up.json`），每个集合两条命令：
   - 回填：`{"tenantID": {"$exists": false}}` 的文档写入 `tenantID: "default"`，只处理缺字段的文档，可重复执行。
   - 索引：`tenantID_1`，非唯一，用于按租户筛选与清理。唯一索引保持不变，不改为租户内唯一。
2. `component_defs`：
   - 回填 `tenantID`，同上。
   - 唯一索引：新建 `tenantID_1_name_1_version_1`，再删 `name_1_version_1`。先建后删，中间不会出现没有唯一约束的窗口。
   - 计数表：建唯一索引 `tenantID_1_defName_1_version_1`（同时创建集合），
     再把定义上非零的 `appCompInstanceCount`、`workspaceCompInstanceCount` 通过 `$merge` 写入计数表，已有记录不覆盖。
   - 从定义上 `$unset` 两个计数字段。

## down

按 up 的逆序执行：

1. `component_defs`：
   - 计数表通过 `$merge` 写回定义上的计数字段，再删除计数表。
   - 重建 `name_1_version_1`，删除 `tenantID_1_name_1_version_1`。不同租户存在同名定义时重建会失败，需要先人工处理。
2. 每个集合先删 `tenantID_1`，再 `$unset` 值为 `default` 的 `tenantID`；其它租户的值不回滚。

## 发布顺序

1. 升级前检查内置组件定义都已标记 `isBuiltin`。旧版本加载命令不强制该字段，未标记的内置定义在新版本重新加载时
   会被当作同名自建定义而报错：
   ```js
   db.component_defs.find({ creator: "admin", isBuiltin: { $ne: true } }, { name: 1, version: 1 })
   ```
   查出的如果是内置定义，先设为 `isBuiltin: true`。
2. 执行本迁移。
3. 发布新版本代码。
4. 全部实例升级完成后：
   - 再回填一次 `tenantID`。旧 Pod 退出前写入的文档不带该字段，新代码看不到；
     其中旧 Pod 新建的计数器文档还会让新代码对该应用取号时唯一键冲突，构建、创建特性环境失败，回填后恢复。
     golang-migrate 不会重复执行已记录的版本，需手动执行：
     ```js
     db.getCollectionNames()
       .filter((c) => db[c].getIndexes().some((i) => Object.keys(i.key)[0] === "tenantID"))
       .forEach((c) => {
         const r = db[c].updateMany({ tenantID: { $exists: false } }, { $set: { tenantID: "default" } })
         print(c, r.modifiedCount)
       })
     ```
   - 再执行一次 `load_builtin_component`，确保内置定义都带 `isBuiltin: true`。

混跑窗口内旧 Pod 写在组件定义上的计数增量不搬迁：窗口很短，期间增删组件的概率低，偏差可接受；
定义上残留的计数字段新代码不读取。

## 验证

第 4 步之后，上面的回填脚本再执行一次，每个集合的 `modifiedCount` 应为 0。

```js
db.component_defs.getIndexes()
db.component_def_instance_counts.getIndexes()
```
