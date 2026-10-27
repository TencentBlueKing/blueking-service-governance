# 其余租户集合：回填 tenantID 并补查询索引

## 背景

继 `workspaces`（000024）之后，其余 47 个按租户划分的集合同时接入 store 层租户过滤（`Meta.TenantScoped = true`）。
接入后读写都会附加 `tenantID` 条件，存量文档没有这个字段就查不到，因此**本迁移必须先于新代码上线**。

存量数据全部归属保留租户 `default`。各集合的唯一索引保持不变，不改为租户内唯一。

保持全局、不在本迁移中的集合：`runtime_images`、`image_snapshots`、`image_tag_suppressions`、
`repo_snapshot_statuses`、`component_defs`、`cluster_addon_defs`、`depservice_services`、
`bkci_pipeline_templates`、`plat_admin_role_bindings`，以及计数器集合。

## up

对每个集合执行两条命令，与 000024 相同：

- 回填：`{"tenantID": {"$exists": false}}` 的文档写入 `tenantID: "default"`，只处理缺字段的文档，可重复执行。
- 索引：`tenantID_1`，非唯一，用于按租户筛选与清理。

## down

每个集合先删 `tenantID_1`，再 `$unset` 值为 `default` 的 `tenantID`；其它租户的值不回滚。

## 发布顺序

1. 执行本迁移。
2. 发布新版本代码。
3. **全部实例升级完成后，再回填一次**。旧 Pod 退出前写入的文档不带 `tenantID`，新代码看不到。
   golang-migrate 不会重复执行已记录的版本，需手动执行：

```js
db.getCollectionNames()
  .filter((c) => db[c].getIndexes().some((i) => i.name === "tenantID_1"))
  .forEach((c) => {
    const r = db[c].updateMany({ tenantID: { $exists: false } }, { $set: { tenantID: "default" } })
    print(c, r.modifiedCount)
  })
```

## 验证

第 3 步之后，上面脚本再执行一次，每个集合的 `modifiedCount` 应为 0。
