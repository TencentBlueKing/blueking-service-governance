# workspaces：回填 tenantID 并补查询索引

## 背景

`workspaces` 是第一个接入 store 层租户过滤的集合（`Meta.TenantScoped = true`）。
接入后 GenericStore 会在所有读写的过滤条件上自动补 `tenantID`，存量文档没有这个字段，
不回填就会一条都查不到，因此**本迁移必须先于新代码上线**。

单租兼容模式下所有存量数据都归属保留租户 `default`（对齐 `pkg/infras/tenant.DefaultTenantID`）。

本轮不改 `id` 的全局唯一索引：工作空间 ID 由用户自选，改成租户内唯一会影响 IAM 资源 ID
与外部系统（蓝盾、BCS、制品库）的命名，作为独立的产品决策推进。

## up

两条命令，先回填再建索引。

**回填**

条件 `{"tenantID": {"$exists": false}}` 只处理还没有该字段的文档，重复执行不会覆盖真实租户数据，
因此是幂等的。`multi: true` 覆盖全部存量文档。

**索引**

`tenantID_1`，非唯一。用于平台管理按租户筛选，以及将来按租户清理数据。
按 `id` 的点查仍然走已有的唯一索引，`tenantID` 只是附加的等值条件，不需要复合索引。

## down

先删 `tenantID_1`，再 `$unset` 值为 `default` 的 `tenantID`。
已经是其它租户值的文档不回滚，避免回退时误删真实的多租数据。

## 发布顺序

1. 执行本迁移。
2. 发布新版本代码。
3. **全部实例升级完成后，手动再回填一次**。步骤 1 到旧 Pod 全部退出之间，旧版本创建的 workspace 不带 `tenantID`，
   新代码查不到也改不了（`id` 仍全局唯一，用户用同一 ID 重建会报 Duplicate）。
   golang-migrate 不会重复执行已记录的版本，因此直接执行与 up 相同的回填命令，幂等、可安全重跑：

```js
db.workspaces.updateMany({ tenantID: { $exists: false } }, { $set: { tenantID: "default" } })
```

## 验证

在发布顺序第 3 步之后执行：

```js
db.workspaces.countDocuments({ tenantID: { $exists: false } }) // 应为 0
db.workspaces.getIndexes()
```
