# 多租户 DB 层隔离方案

在 `GenericStore`（`pkg/store`）内部实现租户隔离：把 `tenantID` 做成 store 自动维护的字段，
与 `_id` / `createdAt` / `updatedAt` 同级。所有模型共用这一套实现，领域包不需要写任何隔离代码。

## 1. 核心约定

- 租户字段统一为 `tenantID`（bson / camelCase），由 store 写入和过滤，业务代码不手动赋值。
- 集合是否隔离由 `Meta.TenantIsolated` 显式声明，并与模型字段交叉校验。
- 租户默认取自 ctx。ctx 里没有租户时直接报错，不回落到 `default`，避免漏挂中间件时写出错误数据。
- 不使用 ctx 租户时，在 opts 中指定租户范围：`store.AllTenants()` 跨租户，`store.ForTenant(id)` 指定某个租户。
  租户范围必须写在调用点，不能藏进 `Combine`。
- 业务代码的调用写法不变，例如 `appStore.Get(ctx, bkmsapp.ByName(wsID, name))`：不传租户选项时按 ctx 中的租户过滤；
  需要时追加租户选项即可，如 `appStore.Get(ctx, bkmsapp.ByName(wsID, name), store.ForTenant("tenant-a"))`。

## 2. 数据模型

### 2.1 集合分类

| 类型 | 声明 | 说明 |
|---|---|---|
| 租户集合 | 模型有 `TenantID string \`bson:"tenantID"\``，`TenantIsolated: true` | 绝大多数业务集合 |
| 不隔离集合 | 无该字段，`TenantIsolated` 为零值 false | 全系统一份，如 `cluster_addon_defs`、`plat_admin_role_bindings`、`component_defs`、`runtime_images` |
| 计数器 | `store.Counter`，文档为 `{_id: key, seq}` | 不经过 Meta，隔离由 key 承担，见 7.2 |

### 2.2 每个租户集合都冗余 `tenantID`

不采用"只在 `workspaces` 上标租户、其余集合靠 `workspaceID` 间接隔离"：
现有代码大量存在不带 `workspaceID` 的查询（如按记录 ID、按 appID 查），store 层无法据此判断归属。
全表冗余后，隔离完全在 store 内完成；代价是一次性回填，以及同一 workspace 下数据的 `tenantID` 需保持一致（由回填保证）。

### 2.3 字段定义

```go
// TenantID 所属租户，由 store 层自动维护，业务代码不应手动赋值
TenantID string `json:"-" bson:"tenantID"`
```

- `json:"-"`：租户由请求头确定，响应不回显。
- 不加 `omitempty`：空租户写成空字符串，能被索引发现。

## 3. store 层机制

### 3.1 声明与校验

- `Meta.TenantIsolated bool`：零值 false 表示不隔离，因此可以逐个集合迁入，未声明的模型行为不变。
- `schema` 识别 `tenantID` 字段并要求类型为 `string`。
- `store.New` 交叉校验：声明隔离但没有字段、或有字段但没声明，都直接报错（`Of` 会 panic，模型单测即可发现）。

### 3.2 租户范围

```go
// 普通业务：租户来自 ctx
app, err := apps.Get(ctx, bkmsapp.ByName(wsID, name))

// 平台管理 / 运维：跨租户
list, err := workspaces.ListPage(ctx, page, size, opts, store.AllTenants())

// 指定租户（忽略 ctx）
err := apps.Create(ctx, &app, store.ForTenant("tenant-a"))
```

一次操作的租户按以下顺序确定：

1. 集合不隔离：不注入；若传了租户选项，报 `ErrTenantScopeNotSupported`。
2. 传了 `AllTenants()`：不按租户过滤。
3. 传了 `ForTenant(id)`：用 `id`，为空报 `ErrTenantIDRequired`。
4. 都没传：ctx 带显式范围标记时报 `ErrTenantScopeRequired`（见 5.2）；否则取 ctx 租户，取不到报 `ErrTenantIDRequired`。

约束：

- 租户选项只能传一个，重复传报 `ErrTenantScopeDuplicated`。
- `AllTenants()` 只用于读、局部更新（`Update` / `Modify`）和删除；`Create` / `Replace` / `Upsert` 只接受 `ForTenant`，
  因为新文档必须落在确定的租户。由 `optKind` 的类别检查实现，无需额外规则。
- 被 `Combine` 包住的租户选项报 `ErrTenantScopeCombined`，保证跨租户在调用点可见、可被 `grep store.AllTenants()` 找全。
- 租户范围只存在于每次调用新建的 `query` 中，`GenericStore` 实例不持有任何租户状态，多处共用同一实例互不影响。
  不要在 store 上增加会修改实例状态的方法。

### 3.3 注入点

所有读写都经过 `parseOpts`（写操作经 `writeQuery`），租户条件在这里统一注入：
字段名解析完成后，向 `q.conds` 追加 `tenantID = <租户>`。

- 调用方自己在条件里写 `tenantID` 会报错，而不是覆盖或丢弃：重复 key 会被 `filter()` 合成 `$and` 仍参与匹配，
  静默丢弃又会掩盖调用方的误解。`AllTenants()` 下允许按 `tenantID` 筛选，这是平台管理的正常用法。
- 写操作"必须带过滤条件"的保护改为看 `callerCondCount`（注入前调用方条件数），否则注入后条件恒非空，
  `DeleteMany(ctx)` 会变成删光当前租户的数据。`Upsert` 把对象自带的 `_id` 计入调用方条件。

写入路径：

| 操作 | 处理 |
|---|---|
| `Create` / `CreateMany` / `CreateManyIfAbsent` | 写入前用解析出的租户覆盖对象上的值，并回填到对象 |
| `Replace` | 同上；过滤条件已注入，跨租户替换返回 NotFound |
| `Update` / `Modify` | 拒绝修改 `tenantID`；全字段更新时跳过 `tenantID` |
| `Upsert` | 过滤条件中的 `tenantID` 等值条件在插入时由 MongoDB 写入，`$set` 不含 `tenantID` |
| `Delete*` / `GetAndDelete` | 仅靠过滤条件注入 |

### 3.4 聚合

`Aggregate` / `ListLatestBy` 把租户条件作为管道首个 `$match`。

租户集合上拒绝 `$lookup`、`$graphLookup`、`$unionWith`、`$merge`、`$out`：它们读写其他集合，不受首个 `$match` 约束。
目前业务代码没有用到这几种阶段（仅 `db/migrations` 和 `cmd/migration` 直接操作原始集合时使用，不受影响），
需要关联时分别经各自的 Store 查询再在代码里拼装。

已知缺口：只检查顶层阶段，`$facet` 子管道里的 `$lookup` / `$unionWith` 能绕过，应补递归检查。

将来确需放开时的方案（按需再做）：

- 前提：`store.New` 登记"集合名 → 是否隔离"，查不到登记的集合按拒绝处理。
- `$lookup` / `$unionWith`：在子管道首部插入 `{$match: {tenantID: <租户>}}` 并递归处理子管道。
  `$lookup` 同时使用 `localField` 与 `pipeline` 需 MongoDB 5.0+。
- `$graphLookup`：把租户条件并入 `restrictSearchWithMatch`。
- `$facet`：递归处理每个子管道。
- `$merge` / `$out`：继续拒绝，回填放在迁移里做。

## 4. 租户来源

数据层只认 ctx（或 `ForTenant`），每个入口都要保证 ctx 里有租户：

| 入口 | 处理 |
|---|---|
| HTTP 业务 API | `/bkms/v1/bkms-server` 路由组挂 `tenant.Required`，写入 ctx |
| 平台管理 API | 同上，另由 `RequirePlatformRole` 打显式范围标记，见 5.2 |
| 异步任务 | envelope 增加 `_tenantID`：投递时 ctx 无租户直接失败，消费时写回 ctx，详见 4.1 |
| `cmd/migration/*` 与命令行 | ctx 无租户，显式传 `AllTenants()`；需要投递任务的命令要给 ctx 设置租户 |
| 定时 / worker 自发逻辑 | 显式 `AllTenants()` 遍历，或按租户 `ForTenant` 循环 |
| 单测 | `testutil.WithDefaultTenant(ctx)` 补默认租户 |

### 4.1 存量异步任务

- 带 envelope 但没有 `_tenantID` 的任务（升级前入队）消费时回落到 `default`。前提是回填迁移已先执行。
- 没有 envelope 的更早期任务不会恢复租户，访问租户集合会失败。
- 滚动升级期间新旧版本混跑无影响：旧 worker 忽略 `_tenantID`；新 worker 对旧任务走回落。
- 轮询类任务在 handler 内重新投递时使用恢复后的 ctx，会自动带上 `_tenantID`。
- 回落逻辑可在 pending / scheduled / retry / archived 四类任务中都不再有缺 `_tenantID` 的任务后移除。

## 5. 跨租户访问

### 5.1 使用方

- 平台管理（`pkg/platmgt/**`）：已有调用点均显式传 `AllTenants()`。
- 运维命令（`cmd/migration/*`）：`refresh_workspace_bkmonitor_perms`、`backfill_default_alert_strategies`、
  `bind_bscp_project`、`list_bscp_projects`、`app_bscpcfg_mgr`。

谁有权跨租户由路由上的 `RequirePlatformRole` 决定；哪次查询跨租户由调用点的 `AllTenants()` 决定，store 不做权限判断。

### 5.2 防止漏传 `AllTenants()`

平台管理请求的 ctx 里是操作者自己的租户，漏传 `AllTenants()` 会静默只返回该租户的数据。

`RequirePlatformRole` 通过后用 `store.RequireExplicitTenantScope` 给 ctx 打标记。
带标记时，租户集合的操作若未传 `AllTenants()` / `ForTenant()`，返回 `ErrTenantScopeRequired`。

- 按 ctx 判断，经过多层领域函数调用也能拦住。
- 标记不授予跨租户能力，跨租户仍须在调用点写 `AllTenants()`。
- 不隔离的集合不受影响；`cmd/migration` 的 ctx 本来没有租户，漏传会报 `ErrTenantIDRequired`，无需标记。
- 平台管理 service 的用例在带标记的 ctx 下调用，漏传会直接失败。

## 6. 索引与迁移

### 6.1 索引

- 查询索引：现有索引多以 `workspaceID` 开头，选择性已足够，不批量加 `tenantID` 前缀，出现慢查询再调整。
- 唯一索引：`workspaces.id` 是用户自选的 slug，第一期保持全局唯一。
  改为租户内唯一涉及 IAM 资源 ID 和外部系统命名，作为产品决策单独评估。
  其余集合的唯一键多以 `workspaceID` / `appID` 开头，无需调整。

### 6.2 迁移

- 每个集合（或每批集合）一个迁移，回填 `tenantID = "default"`，条件 `{"tenantID": {"$exists": false}}` 保证幂等，并建 `tenantID_1` 索引。
- `down` 先删索引，再只 `$unset` 值为 `default` 的字段。
- webserver 启动不自动跑迁移。**回填迁移必须先于新代码上线**，否则存量文档按租户过滤会全部查不到。
