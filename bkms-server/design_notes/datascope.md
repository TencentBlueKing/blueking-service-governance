# datascope：Store 显式租户范围

本轮只迁 `workspaces`。其它 Store 不变，后续按同样方式逐个迁移。

## 背景

多租户要求业务表按 `tenant_id` 隔离。实现方式有两种：

- **隐式注入**：包装 `*mongo.Collection`，在 Find / Insert / Update / Delete / Aggregate 执行时自动补上 `tenant_id`。Store 代码几乎不用改，但租户逻辑藏在包装层里，读 Store 代码看不出有没有按租户过滤；包装层还要跟着 driver 的每种操作扩展，复杂度高。
- **显式注入**：Store 继续持有原生 `*mongo.Collection`，在每个方法里调用公共小函数拿到带租户的条件，再交给 driver。租户逻辑写在 Store 方法里，一眼能看到。

本方案采用显式注入，并参考 GORM 的 Scopes，让 Store 共享同一套租户范围的写法：

```go
wsStore.Get(ctx, id)                                         // 默认：从 ctx 读租户
wsStore.Scopes(datascope.ForTenant("t1")).Get(ctx, id)       // 指定租户
wsStore.Scopes(datascope.AllTenants()).List(ctx, opts)       // 不限租户
wsStore.Scopes(datascope.FromContext()).Get(ctx, id)         // 显式从 ctx 读租户，与默认相同
```

## 分层结构

```text
┌────────────────────────────────────────────────────────────────────────┐
│ 入口：确定 ctx 里的租户                                                 │
│   HTTP 请求    tenant 中间件解析 X-Bk-Tenant-Id，写入 ctx                │
│   taskq 任务   投递时把租户写进 envelope 的 _tenantId，消费时写回 ctx    │
│   迁移命令     不依赖 ctx，用 Scopes(datascope.AllTenants())             │
└────────────────────────────────┬───────────────────────────────────────┘
                                 │ ctx
┌────────────────────────────────▼───────────────────────────────────────┐
│ handler / service                                                      │
│   普通请求   reg.WorkspaceStore.Get(ctx, id)                            │
│   平台管理   reg.WorkspaceStore.Scopes(datascope.AllTenants()).List(…)  │
└────────────────────────────────┬───────────────────────────────────────┘
                                 │ 领域方法
┌────────────────────────────────▼───────────────────────────────────────┐
│ 领域 Store（pkg/core/workspace/store.go）                               │
│   WorkspaceStore 接口    领域方法 + Scopes(datascope.Option) WorkspaceStore │
│   WorkspaceStoreMongo    持有原生 *mongo.Collection 和 datascope.Scope     │
│                          手写 Scopes                                     │
│   每个方法：                                                            │
│     读 / 改 / 删   filter, err := s.scope.TenantFilter(ctx, 业务条件)    │
│     写            tenantID, err := s.scope.TenantForWrite(ctx)          │
│     然后直接调 driver：Find / FindOne / CountDocuments / UpdateOne …     │
└───────────────┬────────────────────────────────────────┬───────────────┘
                │ 只读取租户范围                          │ 带 tenant_id 的条件和文档
┌───────────────▼────────────────────┐   ┌───────────────▼───────────────┐
│ datascope（pkg/infras/database/     │   │ mongo-driver                   │
│ datascope）                         │   │ *mongo.Collection              │
│   Scope       本次访问的租户范围     │   └───────────────────────────────┘
│   ForTenant / AllTenants  范围选项  │
│   ApplyScope        设置范围        │
│   TenantFilter / TenantForWrite     │
└────────────────────────────────────┘
```

和隐式注入相比，`datascope` 不在 Store 和 driver 之间，它不执行任何数据库操作。Store 直接调用 driver，`datascope` 只回答两个问题：这次访问的条件里要加哪个 `tenant_id`；写入时用哪个租户。

## datascope 提供什么

| 名称 | 作用 |
|---|---|
| `ForTenant(id)` | 范围选项：限定为指定租户，不再读 ctx |
| `AllTenants()` | 范围选项：不限定租户，只用于平台管理和迁移命令 |
| `FromContext()` | 范围选项：每次执行从 ctx 读租户，与默认相同 |
| `Scope` | 租户范围，Store 作为字段持有；零值从 ctx 读租户 |
| `ApplyScope(scope, opt)` | 返回设置了范围的新 `Scope`；opt 为 nil 或已经设置过范围时，错误在后续 `TenantFilter` / `TenantForWrite` 返回 |
| `Scope.TenantFilter(ctx, filter)` | 返回带 `tenant_id` 的 filter 副本，覆盖调用方传入的值；`AllTenants` 下不加 |
| `Scope.TenantForWrite(ctx)` | 返回写入用的租户；`AllTenants` 下返回 `ErrWriteAllTenants` |

几个取舍：

- **默认读 ctx。** 不调用 `Scopes` 时，每次执行从 ctx 读租户，普通业务代码不用改调用方式。需要在代码里写明时用 `FromContext()`。
- **租户无关的表不持有 `Scope`。** 这类 Store 没有 `Scopes` 方法，按类型就能区分，不需要维护全局表名单。
- **每个 Store 手写 `Scopes`，不用公共泛型。** Go 里通过接口调用的方法必须在接口里声明，接口里写一行即可。实现是三行：复制 Store、用 `ApplyScope` 换掉 `scope` 字段、返回副本。nil 检查和"只能设置一次"由 `ApplyScope` 负责。
- **`AllTenants` 不允许写入。** 新建的数据必须落到一个确定的租户。跨租户的更新和删除允许，因为条件里仍有业务主键。

## 租户怎么决定

```text
Store 方法调用 TenantFilter / TenantForWrite
  │
  ├─ Scopes(AllTenants())？ ─────── 是 → TenantFilter：不加租户条件
  │                                     TenantForWrite：返回 ErrWriteAllTenants
  │
  ├─ Scopes(ForTenant(id))？ ────── 是 → id 为空：返回 ErrTenantIDRequired
  │                                     id 非空：使用 id，忽略 ctx
  │
  └─ 未指定或 Scopes(FromContext()) → 读 ctx
        ├─ ctx 有租户 → 使用该租户
        └─ ctx 没有   → 返回 ErrTenantIDRequired，不回落到 default
```

`Scopes` 返回新的 Store，原 Store 不受影响。`Scopes` 只接受一个范围，同一个 Store 只能调用一次。再次调用或传入 nil 时，后续操作返回 `ErrScopeAlreadySet` 或 `ErrScopeRequired`。

## Store 接入

```go
type WorkspaceStore interface {
	Scopes(opt datascope.Option) WorkspaceStore
	Get(ctx context.Context, id string) (*Workspace, error)
	// ...
}

type WorkspaceStoreMongo struct {
	collection *mongo.Collection
	scope      datascope.Scope
}

func NewWorkspaceStoreMongo(client *mongo.Client, dbName string) (*WorkspaceStoreMongo, error) {
	coll := client.Database(dbName).Collection(workspaceCollectionName)
	return &WorkspaceStoreMongo{collection: coll}, nil
}

func (s *WorkspaceStoreMongo) Scopes(opt datascope.Option) WorkspaceStore {
	next := *s
	next.scope = datascope.ApplyScope(s.scope, opt)
	return &next
}
```

`scope` 零值就是从 ctx 读租户，构造时不用设置。构造函数签名不变，registry 和 fx 的注册方式都不用改。

各类操作的写法：

| 操作 | 写法 |
|---|---|
| 查询 / 计数 | `filter, err := s.scope.TenantFilter(ctx, 业务条件)`，再 `Find` / `FindOne` / `CountDocuments` |
| 新建 | `tenantID, err := s.scope.TenantForWrite(ctx)`，赋给结构体的 `TenantID` 后 `InsertOne`，调用方传入的值被覆盖 |
| 更新 | 用 `TenantFilter` 生成过滤条件；`$set` 里不写 `tenant_id` |
| 删除 | 用 `TenantFilter` 生成过滤条件 |
| 聚合 | 把 `TenantFilter` 的结果放进第一个 `$match` |

## 调用示例

普通请求：

```text
handler: reg.WorkspaceStore.Get(ctx, "ws-1")
  → WorkspaceStoreMongo.Get
      filter = s.scope.TenantFilter(ctx, {"id": "ws-1"})   // ctx 里是 tenant-a
             = {"id": "ws-1", "tenant_id": "tenant-a"}
      s.collection.FindOne(ctx, filter)
```

平台管理：

```text
handler: reg.WorkspaceStore.Scopes(datascope.AllTenants()).ListWithPagination(ctx, opts)
  → WorkspaceStoreMongo.Scopes：复制 Store，ApplyScope 把 scope 设为 AllTenants
    → 新 Store 的 ListWithPagination
        filter = s.scope.TenantFilter(ctx, 业务条件)       // 不加 tenant_id
        s.collection.CountDocuments / Find
```

## 当前使用情况

- 普通请求和 taskq 任务：用 registry 里的 Store，租户来自 ctx。没有 `_tenantId` 的存量任务按 `default` 恢复。
- 跨租户（`Scopes(datascope.AllTenants())`）：
  - `pkg/platmgt/workspace/handler`、`pkg/platmgt/workspace/admin/handler`
  - `cmd/migration` 下的 `app_bscpcfg_mgr`、`backfill_default_alert_strategies`、`bind_bscp_project`、`list_bscp_projects`、`refresh_workspace_bkmonitor_perms`
- 数据迁移 `000018_workspaces_tenant_id`：把已有文档的 `tenant_id` 回填为 `default`，并建 `tenant_id_1` 索引。

## 测试

显式注入的主要风险是某个方法漏调 `TenantFilter`，编译和运行都不会报错。所以每个迁入的 Store 都要有一组租户隔离用例，逐个方法验证。`workspace/store_test.go` 的 `Tenant Scope` 覆盖了：

- 新建时写入解析出的租户，覆盖调用方伪造的 `tenant_id`；
- 用 B 租户的 ctx 调用 `Get`、`List`、`ListWithPagination`、`CountByState`、`Update`、`Delete`，看不到也改不到 A 租户的数据；
- ctx 没有租户时，每个方法都返回 `ErrTenantIDRequired`；
- `ForTenant` 优先于 ctx，原 Store 不受影响；
- `AllTenants` 能跨租户读，新建返回 `ErrWriteAllTenants`。

`datascope` 自身的用例不依赖数据库，覆盖范围解析、filter 副本，以及 `Scopes` 只能调用一次。

新 Store 迁入时，在它自己的测试里加同样一组用例。

## 和隐式注入对比

| | 隐式注入 | 显式注入（本方案） |
|---|---|---|
| 调用方写法 | `store.Scope(...)` | `store.Scopes(...)`，基本一样 |
| 租户条件在哪加 | 包装层执行时自动加 | Store 方法里调 `TenantFilter` |
| 读 Store 代码 | 看不出租户过滤 | 每个方法都能看到 |
| 漏加的风险 | Store 拿不到原生 collection，很难漏 | 靠评审和每个 Store 的隔离用例兜底 |
| driver 能力 | 每用一种新操作都要在包装层实现 | 直接用 |
| 全局表 | 维护表名单 | 不持有 `Scope`，按类型区分 |
| 公共代码量 | 包装层、查询链、注入、聚合校验 | 一个文件，一百多行 |
| Store 改动 | 很少 | 每个方法加几行 |

## 边界

- 本轮只迁 `workspaces`，`id` 仍是全局唯一。
- `TenantFilter` 只接受 `bson.M`。现有 Store 的条件基本都是 `bson.M`，等有需要再支持 `bson.D`。
- 聚合不做阶段白名单校验，由 Store 自己保证第一个阶段是带租户的 `$match`。
