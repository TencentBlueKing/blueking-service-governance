# 数据层多租户：Collection Wrapper

同一张 Mongo 集合用 `tenant_id` 做逻辑隔离，不按租户拆物理表。

Store 是进程单例，不能把租户 ID 存进自己。用 `pkg/infras/database/tenant.Collection` 包一层 `*mongo.Collection`：业务照常 `Find` / `InsertOne`，Wrapper 自动补 `tenant_id`。

- **租户表**：从 ctx 读取租户并注入。开关开不开都一样——入口中间件已经把结果写进 ctx（关开关是 `default`）。
- **平台表**（静态名单，见 `platform.go`）：不注入。当前为 `cluster_addon_defs`、`depservice_services`。
- ctx 无租户：`ErrTenantIDRequired`，不回落 `default`。

请求头和租户校验仍在 `pkg/infras/tenant`，本文只写 DAL。Migration 只加字段和索引，存量回填 `default`。先试点 `workspaces`，不改 `id` 全局唯一。

---

## 1. 总体结构

```
HTTP
  X-Bk-Tenant-Id
       │
       ▼
pkg/infras/tenant.Required          ← F1，已落地
  tenant.WithTenantID(ctx, id)
       │
       ▼
WorkspaceStore / 其他 Store
  collection.Find / InsertOne / …
       │
       ▼
pkg/infras/database/tenant.Collection
  平台表（静态名单）  → 不注入
  ctx 无租户         → ErrTenantIDRequired（不回落 default）
  ctx 有租户         → 用 ctx 租户写 tenant_id（业务自带的值无效）
       │
       ▼
mongo.Collection（同一 workspaces 集合）
```

| 包 | 职责 |
|----|------|
| `pkg/infras/tenant` | 请求租户：header、mode、校验、`WithTenantID` / `GetTenantID`；字段名 `FieldTenantID` |
| `pkg/infras/database/tenant` | DAL：`Wrap`、`IsPlatform`（静态平台表名单） |
| `pkg/common/config.TenantConfig` | 总开关 `EnableMultiTenantMode` |

保留租户常量（`pkg/infras/tenant`）：

| 常量 | 值 | 用途 |
|------|-----|------|
| `FieldTenantID` | `tenant_id` | 业务集合隔离字段 |
| `DefaultTenantID` | `default` | 单租兼容、存量回填 |
| `SystemTenantID` | `system` | 平台 / 系统保留 |
| `SuperTenantID` | `superadmin` | 超级租户保留（已定义，业务暂未用） |

---

## 2. Wrapper 行为

包路径：`pkg/infras/database/tenant`。

```go
coll := dbtenant.Wrap(client.Database(dbName).Collection("workspaces"))
```

Wrapper **不读** `EnableMultiTenantMode`。模式只在 `tenant.Required` 里决定写入 ctx 的值。

### 2.1 何时注入

| 条件 | 行为 |
|------|------|
| 集合在平台表名单中 | 不改 filter / document / pipeline |
| ctx 有租户 | 用 ctx 租户写 `tenant_id`；业务自己带的 `tenant_id` 无效 |
| ctx 无租户 | 返回 `tenant.ErrTenantIDRequired`，禁止裸查整表，也 **不回落 default** |

关开关时入口写的是 `default`，所以 DAL 注入的也是 `default`，和存量回填同形。后台任务、单测没有走中间件时，调用方自己 `WithTenantID`。

### 2.2 各操作

| 方法 | 注入位置 |
|------|----------|
| `Find` / `FindOne` / `CountDocuments` / `DeleteOne` / `DeleteMany` / `UpdateOne` / `UpdateMany` / `FindOneAndUpdate` / `FindOneAndDelete` | 只改 **filter** |
| `InsertOne` | 只改 **document**（`bson.M` 拷贝；struct 按 bson tag 写字段，不整文档编解码） |
| `InsertMany` | 对 slice 里每条 **document** 注入 |
| `ReplaceOne` | **filter** 与 **replacement** 都注入 |
| `BulkWrite` | `InsertOne` 注入 document；`ReplaceOne` 注入 filter + replacement；Update/Delete 只注入 filter |
| `Aggregate` | 管道 **最前面** 追加 `{ $match: { tenant_id: <ctx> } }` |
| update payload（`UpdateOne` / `UpdateMany` / `FindOneAndUpdate` / BulkWrite 的 Update*） | **不改**，避免把 `$set` 里的业务字段冲掉；租户迁移不走这条路径 |

`FindOne` / `FindOneAndUpdate` / `FindOneAndDelete` 在注入失败时用 `mongo.NewSingleResultFromDocument(nil, err, nil)`，错误在 `Decode` 时返回。

filter 支持 `bson.M`、`bson.D`、`nil`；struct 按 bson tag 写字段。调用方传入的 filter **会被拷贝**，不原地改业务 map。BulkWrite 会拷贝各 WriteModel，不改调用方传入的模型。

---

## 3. 平台表 vs 租户表

- **租户表**：走 Wrapper，注入 `tenant_id`。默认：业务集合都是租户表。
- **平台表**：写在 `platform.go` 的静态 map，开关打开也不注入。未列入即租户表。

当前名单：

| 分类 | 集合 | 说明 |
|------|------|------|
| 平台 | `cluster_addon_defs` | 集群 Addon 定义，全局 |
| 平台 | `depservice_services` | 依赖服务目录，全局 |
| 混合 | `component_defs` | 勿写入平台名单 |
| 租户 | 其余集合 | 含 `workspaces` |

子资源（环境、应用）本轮仍可通过 `workspaceID` 间接隔离；是否冗余 `tenant_id` 在铺开该表时再定。

---

## 4. 试点：workspaces

已落地范围：

1. `NewWorkspaceStoreMongo` 使用 `dbtenant.Wrap`
2. `Workspace.TenantID`，bson 字段 `tenant_id`
3. 迁移 `000017_workspaces_tenant_id`
   - 无字段文档回填 `"default"`
   - 普通索引 `tenant_id_1`
   - **不改** 唯一索引 `id_1`
4. 单测 ctx 写入 `default` 时 List / Get 仍能看到本租户空间
5. 换 ctx 租户则 List / Get / Count 隔离；无租户的写入失败，不回落 default

**明确不做（本试点）**：

- 不改 workspace id 为租户内唯一
- 不接其余 ~50 个 Store
- 不改 IAM / Redis / Asynq / 外部 Client（其它功能线）

---

## 5. 存量与索引

迁移风格与现有 golang-migrate JSON 一致：**只加字段和索引**。

```
up:
  update workspaces
    q: { tenant_id: { $exists: false } }
    u: { $set: { tenant_id: "default" } }
  createIndexes workspaces.tenant_id_1

down:
  dropIndexes tenant_id_1
  unset tenant_id where tenant_id == "default"
```

其它表铺开时：

1. model 加 `TenantID`
2. store 构造改为 `Wrap`
3. 独立 migration：回填 `default` + `tenant_id` 索引
4. **单独评估唯一索引**：仅当业务标识会跨租户撞车时，才改成 `(tenant_id, …)`。已由 `workspaceID` / `appID` 限定的唯一键通常不用改。

---

## 6. 开关与请求上下文

| 模式 | 请求入口 | DAL |
|------|----------|-----|
| `enableMultiTenantMode=false` | 不强制租户头（F1 已实现） | Wrapper 固定写 / 滤 `tenant_id = default`，忽略 ctx |
| `enableMultiTenantMode=true` | 中间件写入 ctx | Wrapper 强制 ctx 租户；缺则报错，**不**回落 `default` |

后台任务、migrate Job：多租户关闭时无需 ctx，自然落在 default；打开后必须显式 `WithTenantID`。开启模式下禁止回落 default，否则一次漏 ctx 会扫到单租存量或写错租户。

---

## 7. 测试约定

| 层级 | 覆盖 |
|------|------|
| Wrapper 单测 | 关闭时注入 `default`、开启时注入 ctx、覆盖伪造 `tenant_id`、开启且缺 ctx 报错、平台表跳过、insert 回填、aggregate 前置 `$match` |
| Store 集成 | 开启后跨租户 List/Get/Count 不可见；关闭时只看到 / 只写入 `default` |

集成测需要 `BKMS_SERVER_CONFIG_PATH` 与可达 Mongo。开启多租户时改 `config.G.Tenant.EnableMultiTenantMode`，并在用例结束时还原。

---

## 8. 铺开顺序（建议）

1. 已完成：Wrapper + `workspaces` + `000017`
2. 仍挂在 workspace 下、已有 `workspaceID` 的表：可晚一点，先靠空间 ID 隔离
3. **没有** workspace/app 外键、名称全局唯一的表（如部分 depservice / 组件）：优先接 Wrapper，并评估唯一索引
4. 混合表（`component_defs`）单独设计，不要写入平台名单
5. 每张表独立 migration，保持每周可上线

---

## 9. 相关代码

| 路径 | 内容 |
|------|------|
| `bkms-server/pkg/infras/database/tenant/collection.go` | Wrapper |
| `bkms-server/pkg/infras/tenant/` | 请求租户 |
| `bkms-server/pkg/core/workspace/store.go` | 试点 Store |
| `bkms-server/db/migrations/000017_workspaces_tenant_id.*` | 回填与索引 |
| `docs/reqs/多租户改造/多租户改造-功能线梳理与迭代计划.md` | 全功能线（F1–F8，本地草稿，git 忽略） |
