# workspaces：回填 tenant_id 并补查询索引

## 背景

多租户试点在同一 `workspaces` 集合上增加 `tenant_id` 字段，由 Collection Wrapper 在开启 `enableMultiTenantMode` 时注入过滤与写入。存量工作空间没有该字段。单租兼容保留租户 ID 为 `default`（对齐 `pkg/infras/tenant.DefaultTenantID`）。

本轮不改 `id` 全局唯一索引，只补 `tenant_id` 普通索引，方便按租户扫描。

## 迁移语句说明

`up` 两条命令，先回填再建索引。

**回填 `q`**

```json
{ "tenant_id": { "$exists": false } }
```

只处理还没有该字段的文档，幂等。

**回填 `u`**

```json
{ "$set": { "tenant_id": "default" } }
```

**索引**

`tenant_id_1`，非唯一。

## down

先删 `tenant_id_1`，再 `$unset` `tenant_id` 为 `default` 的字段。已是其他租户值的文档不回滚，避免误删真实多租数据。

## 验证

```js
db.workspaces.countDocuments({ tenant_id: { $exists: false } }) // 应为 0
db.workspaces.getIndexes()
```
