# dal：带租户注入的 Mongo 查询层

工作空间先接到这一层。其它 Store 仍直接使用 `*mongo.Collection`，后续逐个迁移。

## 为什么要加这一层

迁移前，每个 Store 持有 `*mongo.Collection`，查询条件和写入文档里的 `tenant_id` 都要 Store 自己拼。每个方法都得记得加，漏一处就会读到或改到别的租户的数据，而且这种遗漏在代码评审里很难发现。

`dal` 把"这次访问用哪个租户、怎么写进条件"集中到一处：

- Store 只写领域查询，例如按 `id` 查、按关键字搜，不再接触 `tenant_id`。
- 租户在执行时由 `dal` 统一注入。调用方传进来的 `tenant_id` 会被覆盖，伪造不了。
- 平台管理和迁移命令需要跨租户时，显式调用 `Scopes`，在代码里一眼能看出来。

## 分层结构

从上到下的调用层级：

```text
┌──────────────────────────────────────────────────────────────────────┐
│ 入口：确定 ctx 里的租户                                               │
│   HTTP 请求    tenant 中间件解析租户，写入 ctx                         │
│   taskq 任务   投递时把租户放进 envelope，消费时写回 ctx               │
│   迁移命令     不依赖 ctx，需要时用 Scopes(dal.AllTenants())           │
└───────────────────────────────┬──────────────────────────────────────┘
                                │ ctx
┌───────────────────────────────▼──────────────────────────────────────┐
│ handler / service：业务逻辑                                           │
│   普通请求   reg.WorkspaceStore.Get(ctx, id)                          │
│   平台管理   reg.WorkspaceStore.Scopes(dal.AllTenants()).List(ctx, …) │
│   不直接调用 db.Table(...)                                            │
└───────────────────────────────┬──────────────────────────────────────┘
                                │ 领域方法
┌───────────────────────────────▼──────────────────────────────────────┐
│ 领域 Store（pkg/core/workspace/store.go）                             │
│   WorkspaceStore 接口     领域方法 + Scopes(dal.Scope) WorkspaceStore  │
│   WorkspaceStoreMongo     持有 table dal.Table，手写 Scope              │
│   只拼业务条件，例如 {"id": id}，通过 s.table 访问表                    │
└───────────────────────────────┬──────────────────────────────────────┘
                                │ s.table
┌───────────────────────────────▼──────────────────────────────────────┐
│ dal 包（pkg/infras/database/dal）                                     │
│                                                                      │
│   Scope            scoped.go   范围选项和 ApplyScope                   │
│        │                                                             │
│   Table            table.go    一张表 + 本次访问的租户范围             │
│        │                       Insert / Update / Delete / Aggregate   │
│        │                       Find(filter) → Query                   │
│   Query            query.go    Sort / Skip / Limit                    │
│        │                       One / All / Count 执行时注入租户        │
│        │                                                             │
│   辅助             global.go   全局表名单                              │
│                    inject.go   把 tenant_id 写进 filter / document 副本 │
│                    pipeline.go 聚合补 $match，更新禁止改 tenant_id     │
│                                                                      │
│   DB               db.go       持有 client 和库名，只负责 Table(name)  │
└───────────────────────────────┬──────────────────────────────────────┘
                                │ 已注入 tenant_id 的 filter / document
┌───────────────────────────────▼──────────────────────────────────────┐
│ mongo-driver：*mongo.Collection                                       │
└──────────────────────────────────────────────────────────────────────┘
```

DB 有两个构造入口，拿到的是同一种对象：

```text
registry.initStores   dal.New(mongoClient, dbName)   启动时已经拿到 client，直接传入
业务代码 / 测试       database.DB()                   包装全局 client 和库名
```

## 每一层负责什么

| 层 | 负责 | 不负责 |
|---|---|---|
| handler / service | 业务逻辑，决定要不要跨租户 | 拼 `tenant_id` |
| 领域 Store | 领域方法和业务查询条件 | 租户解析、租户注入 |
| `Scopes` / `ApplyScope` | 给 Table 换访问范围 | 具体查询 |
| `Table` | 记住这次访问的租户范围（只能设置一次），执行写操作和聚合 | 业务条件 |
| `Query` | 链式组装查询，执行时注入租户 | 写操作 |
| `DB` | 按表名打开 `Table` | 租户范围 |

几个取舍：

- **范围放在 Table 上，DB 不带范围。** 同一个连接会打开很多张表，范围跟着具体的那一次访问走，更容易看清楚是哪张表、哪次调用跨了租户。
- **Find 时不注入，执行时注入。** `Find(filter)` 只记下条件，`One`、`All`、`Count` 执行时才读 ctx 或指定的租户。这样 Query 可以先组装好再执行，租户总是按执行那一刻的范围来定。
- **每个 Store 手写 `Scopes`，不用公共泛型。** Go 里通过接口调用的方法必须在接口里声明，接口里写一行 `Scopes(scope dal.Scope) WorkspaceStore`。实现是三行：复制 Store、用 `dal.ApplyScope` 换掉 Table、返回副本。nil 检查和"只能设置一次"都由 `dal` 负责，Store 不用重复。
- **领域方法留在 Store。** handler 不直接 `db.Table("workspaces")`，查询逻辑集中在 Store，方便测试和复用。

## 租户怎么决定

一次访问要不要注入、注入哪个租户，由两件独立的事决定：

1. **表本身。** 租户表有 `tenant_id`。全局表没有，按表名识别：`cluster_addon_defs`、`depservice_services`、`plat_admin_role_bindings`。全局表即使调用 `ForTenant` 或 `AllTenants` 也不注入。
2. **这一次访问。** 租户表默认读 ctx；`Scopes` 可以改成指定租户或跨租户。

```text
执行 One / All / Count / Insert / Update / Delete / Aggregate
  │
  ├─ 全局表？ ───────────────────── 是 → 不注入，原样执行
  │
  ├─ Scopes(AllTenants())？ ────── 是 → 不注入，查所有租户
  │
  ├─ Scopes(ForTenant(id))？ ────── 是 → id 为空：返回 ErrTenantIDRequired
  │                                      id 非空：注入 id，忽略 ctx
  │
  └─ 未指定或 Scopes(FromContext()) → 读 ctx
        ├─ ctx 有租户 → 注入该租户
        └─ ctx 没有   → 返回 ErrTenantIDRequired，不回落到 default
```

注入的具体做法：

- filter 和 document 上的 `tenant_id` 被覆盖成解析出的租户。写的是副本，调用方传入的对象不变。
- 更新不能修改 `tenant_id`，也不支持 pipeline 形式的更新。
- 聚合在最前面补一个租户 `$match`，只允许单表阶段：`$match`、`$group`、`$sort`、`$project`、`$skip`、`$limit`、`$count`、`$unwind`。
- `Count` 只按 filter 计数，不看 `Sort`、`Skip`、`Limit`。

## 调用示例

普通请求，租户来自 ctx：

```text
handler: reg.WorkspaceStore.Get(ctx, "ws-1")
  → WorkspaceStoreMongo.Get: s.table.Find({"id": "ws-1"}).One(ctx, &ws)
    → Query.One: 读 ctx 得到 tenant-a，filter 变为 {"id": "ws-1", "tenant_id": "tenant-a"}
      → mongo FindOne
```

平台管理，跨租户：

```text
handler: reg.WorkspaceStore.Scopes(dal.AllTenants()).ListWithPagination(ctx, opts)
  → WorkspaceStoreMongo.Scope: 复制 Store，dal.ApplyScope 换成跨租户的 Table
    → 新 Store 的 ListWithPagination: s.table.Find(filter).Count / All
      → Query: 跨租户，不注入，filter 原样执行
```

`Scopes` 返回的是新 Store，registry 里那份不受影响，后面的请求照样读 ctx。`Scopes` 只接受一个范围，同一个 Store 只能调用一次。再次调用或传入 nil 时，后续操作返回 `ErrScopeAlreadySet` 或 `ErrScopeRequired`。

## Store 接入

领域 Store 接口声明 `Scopes`，实现持有 `dal.Table` 并手写 `Scopes`：

```go
type WorkspaceStore interface {
	Scopes(scope dal.Scope) WorkspaceStore
	Get(ctx context.Context, id string) (*Workspace, error)
	// ...
}

type WorkspaceStoreMongo struct {
	table dal.Table
}

func NewWorkspaceStoreMongo(db dal.DB) (*WorkspaceStoreMongo, error) {
	return &WorkspaceStoreMongo{table: db.Table(workspaceCollectionName)}, nil
}

func (s *WorkspaceStoreMongo) Scopes(scope dal.Scope) WorkspaceStore {
	next := *s
	next.table = dal.ApplyScope(s.table, scope)
	return &next
}
```

`ApplyScope` 在 scope 为 nil，或 Table 已经设置过范围时，把错误记在副本上，后续操作再返回。Store 有其他字段时 `next := *s` 会一起复制，不需要额外处理。

`Create` 用 `table.TenantID(ctx)` 把解析出的租户写回内存中的结构体，因为 `Insert` 写进库的是 BSON 副本。

## 当前使用情况

- 普通请求和 taskq 任务：用 registry 里的 Store，租户来自 ctx。没有 `_tenantId` 字段的存量任务按 `default` 恢复。
- 跨租户：平台空间的两个 handler，以及会扫全部空间的迁移命令，使用 `store.Scopes(dal.AllTenants())`。

## 边界

- 本轮只迁 `workspaces`。
- `id` 仍是全局唯一。
- 还没有 `BulkWrite`、`ReplaceOne`、`FindOneAndUpdate`。等有 Store 需要再加。
