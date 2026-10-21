# 多租户 DB 层隔离方案（基于 GenericStore）

本文给出在 `GenericStore` 改造（#272）之后，bkms-server 数据层做租户隔离的方案。

早期在 XStore 时代写过三版方案，分别对应 #249（Collection 包装）、#250（dal 查询层）、#251（datascope 显式注入），
它们的 design note 可以在对应 PR 分支的 `design_notes/tenant_collection.md`、`dal_table.md`、`datascope.md` 中找到。
那三版的核心矛盾是：当时每个领域 Store 自己持有 `*mongo.Collection` 并手写 driver 调用，
隔离逻辑要么下沉到一层新的包装 / 查询层（改造面大、要重新实现 driver 能力），
要么留在每个 Store 方法里（漏写不报错，只能靠评审和用例兜底）。

GenericStore 把这个矛盾消掉了：现在全部 51 个模型共用同一套 CRUD 实现，
`pkg/store` 本身就是那层"包装"，并且它已经收口了过滤条件与写入文档的构造。
所以本方案**不新增中间层**，而是把 `tenantID` 做成 GenericStore 的自动维护字段，
与 `_id` / `createdAt` / `updatedAt` 同级。

## 1. 结论先行

- 租户字段叫 `tenantID`（camelCase），与 `workspaceID`、`createdAt` 等现有 bson 字段保持一致。
  旧方案用的 `tenant_id` 与本仓库命名惯例不符，不沿用。
- 隔离在 `pkg/store` 内部完成，**领域包的公开 API 一行不改**：
  `apps.Get(ctx, ByName(...))` 的签名和写法都不变，差异只体现在 ctx 里有没有租户。
- 跨租户访问必须在调用点显式写出来：`apps.List(ctx, ..., store.AllTenants())`。
  租户范围是一个 `Opt`，但**不允许被 `Combine` 包起来**——它是安全边界，
  不该像分页、排序那样被领域包打包复用。选型过程见 5.3.1。
- 集合是否按租户隔离由 `Meta.TenantIsolated` 显式声明，并与模型字段交叉校验，两者不一致时 Store 构造即失败。
- ctx 里没有租户时直接报错，不回落 `default`：漏挂中间件应当是 500，而不是悄悄写出一批错数据。

## 2. 现状

### 2.1 已经有的

租户在**请求层**已经打通，但完全没有进入存储层：

- `pkg/infras/tenant/types.go`：`X-Bk-Tenant-Id` 头、`default` / `system` / `superadmin` 保留租户。
- `pkg/infras/tenant/middleware.go`：`tenant.Required(enableMultiTenantMode, verifier)`，
  单租模式统一为 `default`，多租模式校验 header 与 bk-user 归属，结果写入 `ctxkey.TenantID`。
- `pkg/server/server.go` 的 `/bkms/v1/bkms-server` 路由组已挂上该中间件。
- `pkg/infras/tenant/context.go`：`WithTenantID` / `GetTenantID`。

### 2.2 缺的

- 没有任何集合的文档带租户字段（全仓库搜不到 `bson:"tenant`）。
- `GenericStore` 的任何方法都不读 ctx 里的租户，filter 里也不会出现租户条件。
- 异步任务的 envelope（`pkg/infras/taskq/envelope.go`）只透传 `auth.User`，不带租户。
- 数据层目前的隔离轴是 `workspaceID` / `appID`，靠 handler 传条件 + IAM 鉴权保证，与租户无关。

### 2.3 GenericStore 的关键收口点

这几处是本方案能成立的基础，都在 `pkg/store` 内部，外部不可绕过：

| 收口点 | 位置 | 覆盖范围 |
|---|---|---|
| `parseOpts` | `query.go:87` | 所有操作的过滤条件，字段名在这里被解析成 bson 路径 |
| `writeQuery` | `query.go:128` | 写操作，额外要求必须带条件或 `All()` |
| `prepareInsert` | `write.go:111` | `Create` / `CreateMany` / `CreateManyIfAbsent` 的文档 |
| `Replace` | `write.go:76` | 整文档替换 |
| `prepareUpsert` | `upsert.go:137` | `Upsert` / `UpsertMany` 的 filter 与 `$setOnInsert` |
| `fieldSelection.paths` | `update.go:174` | `Update` / `Upsert` 的候选写入字段（已在此跳过 `_id`/`createdAt`/`updatedAt`） |
| `resolveUpdatePath` | `update.go:224` | 显式指定的更新字段（已在此拒绝 `_id`/`createdAt`） |
| `modifyDoc` | `modify.go:111` | `Modify` / `ModifyMany` 的原生操作符，字段同样过 `resolveUpdatePath` |
| `Aggregate` / `ListLatestBy` | `aggregate.go:106` / `:39` | 聚合管道的首个 `$match` |
| `schema` | `schema.go:44` | 反射出的字段索引，`fillID` / `fillCreated` 已是同类机制 |

没有一个公开方法绕开 `parseOpts`：`Get` / `GetByID` / `List` / `ListPage` / `ListLatestBy` /
`Exists` / `Count` / `Aggregate` / `Create*` / `Replace` / `Update*` / `Upsert*` / `Modify*` /
`Delete*` / `GetAndDelete` 全部经过它或 `writeQuery`。

唯一的逃生舱是 `GenericStore.Collection()`（`store.go:91`），见第 8 节。

## 3. 与三版旧方案的关系

| | #249 Collection 包装 | #250 dal 查询层 | #251 datascope 显式 | 本方案 |
|---|---|---|---|---|
| 新增公共层 | 714 行 Collection wrapper | 完整 Table/Query 链 | 150 行工具 | 0，改 `pkg/store` 内部 |
| 领域 Store 改动 | 构造函数换 Collection | 换成 `dal.Table`，手写 `Scopes` | 每个方法加 2-3 行，手写 `Scopes` | 无 |
| 每个模型要重复的代码 | 1 处 | 2 处（接口 + 实现） | 1 处 + 每方法 | 0 |
| 漏写导致越权 | 难 | 难 | 容易，靠用例兜底 | 不可能，没有绕过路径 |
| driver 能力缺口 | 要逐个补 | 要逐个补（`BulkWrite` 等当时缺失） | 无 | 无 |
| 全局表识别 | 维护表名单 | 维护表名单 | 按类型（无 `Scope` 字段） | `Meta.TenantIsolated` 声明 + 模型字段交叉校验 |

三版方案里反复出现的两句"每个 Store 手写 `Scopes`"和"本轮只迁 `workspaces`"，
根因都是当时没有统一的 Store 实现。现在隔离机制只需在 `GenericStore[T]` 上实现一次，
51 个模型同时获得；"只迁一张表"也不再是技术约束，而只是数据回填的节奏问题。

值得从旧方案继承的决策（本方案照搬）：

- 调用方传入的 `tenantID` 一律被覆盖，不可伪造。
- `AllTenants` 允许跨租户读 / 改 / 删，但**禁止写入新文档**（新数据必须落到确定的租户）。
- ctx 缺租户直接报 `ErrTenantIDRequired`，不回落 `default`。
- 更新操作不允许修改 `tenantID`。
- 平台管理（`pkg/platmgt/**`）与 `cmd/migration/*` 是跨租户访问的已知使用方。

## 4. 数据模型：哪些集合带 tenantID

### 4.1 三种集合

1. **租户集合**：模型有 `TenantID string \`bson:"tenantID"\`` 字段，``Meta.TenantIsolated = true``。
   绝大多数业务集合属于此类。
2. **全局集合**：全系统一份，模型没有该字段，``Meta.TenantIsolated` 保持零值 false`。
   当前候选：`cluster_addon_defs`、`plat_admin_role_bindings`、`component_defs`、`runtime_images`。
   `depservice_services`（服务目录）在旧方案中被列为全局表，需要和产品确认是否要支持租户自定义服务。
3. **计数器集合**：`store.Counter`（`counter.go`）文档结构固定为 `{_id: key, seq}`，不走 schema。
   见第 8.2 节。

### 4.2 关键决策：全表冗余 tenantID，而不是只在 workspaces 上标租户

两个选项：

- **A. 全表冗余**：45+ 张租户集合每张都带 `tenantID`。
- **B. 只有 `workspaces` 带 `tenantID`**，其余集合靠已有的 `workspaceID` 间接隔离。

**推荐 A。** 理由：

- B 在 store 层根本无法强制。`builds.GetByID(ctx, recordID)`、
  `appmodels.Get(ctx, ByApp(appID))` 这类不带 `workspaceID` 的查询在现有代码里大量存在，
  store 层拿不到 workspace 就没法判断归属，只能回到"每个调用点自己记得先查 workspace"——
  这正是三版旧方案都想消灭的那类静默漏洞。
- B 的每次访问都要多一跳 workspace 查询，或者引入缓存，复杂度并不比回填低。
- A 的增量成本主要在一次性数据回填，而不在代码：有了第 5 节的机制，
  模型加一个字段 + 一条迁移就完成接入，没有 per-store 代码。

A 的代价是数据冗余和一次性回填工作量，以及"同一个 workspace 下的数据 tenantID 必须自洽"
这个不变量需要靠回填脚本和索引来保证，而不是靠外键。考虑到 Mongo 本来就没有外键、
现有 `workspaceID` 冗余也是同样的模式，这个代价是可接受的。

### 4.3 字段定义

```go
type Application struct {
    ID string `json:"id" bson:"id,omitempty" validate:"required"`
    // TenantID 所属租户，由 store 层按 ctx 自动写入，业务代码不应手动赋值
    TenantID string `json:"-" bson:"tenantID"`

    WorkspaceID string `json:"workspaceID" bson:"workspaceID"`
    // ...

}
```

- `json:"-"`：租户不对外暴露，前端通过请求头确定租户，响应里不需要回显。
- 不加 `omitempty`：空租户必须写成空字符串被索引发现，而不是字段消失。

可以考虑提供一个内嵌结构体减少重复：

```go
// store.TenantScoped 可内嵌到租户模型中，提供由 store 层维护的 tenantID 字段
type TenantScoped struct {
    TenantID string `json:"-" bson:"tenantID"`
}
```

`path` 包已支持 inline 嵌入字段的展开（`schema.go:56` 的注释），内嵌与直接声明等价。

## 5. store 层设计

### 5.1 Meta 声明租户属性

`Meta` 增加一个 bool 字段：

```go
// TenantIsolated 集合是否按租户隔离读写。
// 零值 false 表示不隔离（全系统共用一份）；为 true 时模型必须有 tenantID 字段
TenantIsolated bool
```

**为什么是 bool 而不是枚举。** 曾考虑过三值枚举，用来区分"隔离且禁止任何跨租户访问"
与"隔离但允许平台侧跨租户访问"。结论是这两者在运行时行为一致：
平台管理迟早要统计、清理任意一张表，把某些表标成"永久禁止跨租户"只会在那一天回来改声明。
既然只剩两个值，bool 就够了，也省掉了给枚举类型起名的问题。

**零值 `false` 让迁入可以一张表一张表地做。** 没声明的模型行为与现在完全一致，
51 个 Meta 在第一期一行不用改，第三期每迁一张表才加一行。
代价是新加的租户模型如果忘了写 `TenantIsolated: true` 会静默变成不隔离，
兜底靠下面的字段交叉校验：带了 `tenantID` 字段却没声明会直接报错。

**为什么不干脆去掉声明、只看模型有没有 `tenantID` 字段。** 推断能分出"隔不隔离"，
但会误伤把租户 ID 当业务数据存的模型（例如审计记录里记"本次操作影响了哪个租户"，
而它本身需要跨租户可查）。这类模型目前不存在，但显式声明的成本只有一行，
没必要为了省这一行引入一个会误判的规则。

`Meta` 增加一个字段：

```go
type Meta[T any] struct {
    Collection string
    TenantIsolated bool
    NotFound   error
    Duplicate  error
    // ...
}
```

`New`（`store.go:61`）在现有的集合名校验之后，增加交叉校验：

- `TenantIsolated` 为 true 但 schema 没扫到 `tenantID` 字段 → 报错。
- `TenantIsolated` 为 false 但 schema 扫到了 `tenantID` 字段 → 报错。

为什么要同时有声明和字段，而不是只看字段：漏加字段的后果是数据静默跨租户可见，
这类错误不能靠"忘了就是全局表"的默认行为兜底。两处交叉校验让新模型必须显式做一次决策，
而 `Of` 本来就会在配置错误时 panic（`store.go:55`），模型包的现有单测会立刻暴露。

51 个 Meta 都要加一行，是一次性的机械改动。

### 5.2 schema 识别字段

`schema`（`schema.go:44`）增加 `tenantID []int`，`newSchema` 在现有的
`_id` / `createdAt` / `updatedAt` 分支旁增加 `tenantID` 分支，并校验字段类型为 `string`
（与 `createdAt` 必须是 `time.Time` 的校验同构）。

对应增加一个辅助方法，与 `fillID` / `setUpdated` 同一风格：

```go
// fillTenant 将解析出的租户写入对象，覆盖调用方传入的值
func (s *schema) fillTenant(v reflect.Value, tenantID string)
```

不需要反向的"读出对象租户"：所有判断都基于 ctx / scope 解析出的租户，
对象上的值一律被覆盖，没有被读取的场景。

`schemaCache` 按 `reflect.Type` 缓存，无需改动。

### 5.3 访问范围

租户范围是一个 `Opt`，与过滤、排序等选项同级：

```go
// AllTenants 跨租户访问，不按租户过滤，用于平台管理与运维命令。
// 只有读、局部更新与删除接受它；写入整个文档的操作不接受
func AllTenants() TenantOpt

// ForTenant 指定租户，忽略 ctx 中的租户，用于 ctx 里没有租户的场景，如运维命令按租户遍历
func ForTenant(tenantID string) TenantOpt
```

调用形态：

```go
// 普通业务：租户来自 ctx，写法与现在完全一致
app, err := apps.Get(ctx, bkmsapp.ByName(wsID, name))

// 平台管理
list, err := workspaces.ListPage(ctx, page, size, opts, store.AllTenants())

// 运维命令指定租户
err := apps.Create(ctx, &app, store.ForTenant("tenant-a"))
```

设计取舍：

- **不隔离的集合上传入租户选项报错**，而不是静默忽略。
  旧方案（#249/#250）选择忽略，但那是因为它们按表名单识别全局表、调用方不一定知情；
  这里 `Meta` 就声明了属性，调用方写出来就是理解错了。
- **范围只能指定一次**，重复传入返回 `ErrTenantScopeDuplicated`。
- **写入整个文档的操作不接受 `AllTenants`**，新文档的 `tenantID` 必须是确定值。
  这条不需要专门的检查：`Create` / `Replace` / `Upsert` 在 `optKind` 位集合里
  只声明接受 `kindTenantFixed`，现成的 `errUnsupportedOpt` 就会拒绝。
- **`Combine` 里的租户选项被拒绝**，返回 `ErrTenantScopeCombined`，见下节。

为了让插入类操作也能指定租户，`Create` / `CreateMany` / `CreateManyIfAbsent` / `GetByID`
增加了 `opts ...Opt` 参数。变参是源码兼容的，现存调用点一处不用改。

#### 5.3.1 选型过程：Opt vs GORM 风格 `Scopes` vs 派生 Store

三种形态都实现过或评估过：

| | `Scopes(AllTenants())`（三版旧方案） | 派生 Store `s.AllTenants()` | `Opt`（当前） |
|---|---|---|---|
| 引入的新概念 | `ScopeFunc` | Store 上的一套派生方法 | 无，复用 `Opt` |
| 错误何时返回 | 执行时 | 执行时，且要靠 `scope.err` 延迟记录 | 执行时，`parseOpts` 直接返回 |
| 分操作限制（如 Create 禁止跨租户） | 手写 | 手写专门的 `ErrWriteAllTenants` | `optKind` 位集合白送 |
| 跨租户能否被藏起来 | 能 | 不能 | 能，故显式拒绝 `Combine` |
| 跨租户在一个 service 里声明几次 | 每次调用 | 构造时一次 | 每次调用 |

**最终选 `Opt`**，主要是前三行：它不引入并行概念，错误能在 `parseOpts` 里就地返回，
而且每个操作接受哪种范围直接落在已有的 `optKind` 门禁上，不需要为"Create 不许跨租户"单写一条规则。
派生 Store 为了能链式调用必须返回 `*GenericStore[T]` 而非 `(*GenericStore[T], error)`，
由此被迫引入一套延迟报错机制，这部分复杂度在 `Opt` 下整个消失。

**代价在最后一行，要正视。** GORM 的 `Scopes` 价值在于把可复用的查询片段抽成函数再自由组合，
而租户范围恰恰是不该被复用的东西——它是安全边界，每次越过都应该在调用点被看到。
`Opt` 形态天然可以被 `store.Combine` 打包进领域包的具名 Opt，
`apps.List(ctx, bkmsapp.ForDashboard(wsID))` 从签名上完全看不出它跨了租户。
所以 `parseOpts` 显式拒绝顶层 Opt 是 `combinedOpt` 且携带租户范围的情况，
把"必须出现在调用点"这条约束保留下来。

另一处代价：平台管理的 service 以前可以在构造时绑定一次跨租户 Store，
现在每个方法都要记得传 `store.AllTenants()`。platmgt 的 ctx 里有中间件塞的租户，
漏传本来会静默只返回那一个租户的数据；现在由运行时标记把它变成报错，见 8.3。

### 5.4 注入点清单

**`parseOpts` 改为接收 ctx**（`query.go:87`），在字段名解析完成之后注入租户条件：

```go
func (s *GenericStore[T]) parseOpts(ctx context.Context, op string, opts []Opt, accepted optKind) (*query, error)
```

注入逻辑：

1. 集合是 `Global` → 不注入，直接返回。
2. 否则按 scope 解析租户：`AllTenants` 不注入；`ForTenant(id)` 用 id；
   未设置范围时，ctx 被 `RequireExplicitTenantScope` 标记过则返回 `ErrTenantScopeRequired`，
   否则 `tenant.GetTenantID(ctx)`，取不到返回 `ErrTenantIDRequired`。
3. 若 `q.conds` 里已经有 `tenantID`，**报错**；否则追加解析出的条件。

第 3 步的三种可选语义里选"报错"：

- *直接追加*不可行——`query.filter()`（`query.go:58`）遇到重复 key 会退化成 `$and`，
  调用方伪造的条件仍会参与匹配，等于没拦住。
- *静默丢弃*（旧方案的做法）能保证安全，但会让"以为自己在按租户过滤"的代码继续跑且结果不符预期。
- *报错*把问题暴露在第一次执行时，且错误信息可以直接指向 `AllTenants()` / `ForTenant()`。

跨租户视图（`AllTenants`）下不做这个检查：此时按 `tenantID` 过滤正是平台管理筛选租户的正常用法。

`writeQuery` 相应改签名。两者的调用方全在 `pkg/store` 包内（约 16 处），**公开 API 不变**。

注意顺序：注入必须在 `mapFilterKeys(q.conds, s.schema.resolveKey)` 之后，
注入的是已经解析好的 bson key，避免被二次解析。

**注入会废掉"写操作必须带条件"的保护，必须一并处理。** `writeQuery` 通过
`requireFilter`（`query.go`）拒绝不带任何条件的写操作，判据是 `len(q.conds) == 0`。
注入之后租户集合的 `q.conds` 恒非空，于是 `DeleteMany(ctx)` 会从"报错"变成
"删光当前租户的整张表"——隔离机制反而制造了一个更危险的操作。

解法是在 `query` 上记一个 `callerConds int`，在注入之前赋值为 `len(q.conds)`，
`requireFilter` 改判 `callerConds`。另有一处要配合：`prepareUpsert` 会把对象自带的 `_id`
追加进 `q.conds`，这等同于调用方给出的条件，追加时要同步 `callerConds++`，
否则 `Upsert` 一个带 `_id` 的对象会被误判成无条件写。

这类"安全网被新机制悄悄绕过"的点，是本次改造里比注入本身更需要用例锁住的部分。

**写入路径**：

| 操作 | 改动 |
|---|---|
| `Create` / `CreateMany` / `CreateManyIfAbsent` | 加 `opts ...Opt`（只接受 `kindTenantFixed`）；`prepareInsert`（`write.go:111`）在 `fillID` / `fillCreated` 之后调 `fillTenant`，覆盖调用方传入的值；回填到 obj 让调用方可见 |
| `Replace` | `write.go:76`，在 `beforeSave` 之后、`encodeDoc` 之前 `fillTenant`。filter 已由 `writeQuery` 注入，跨租户替换会 NotFound |
| `Update` / `UpdateMany` | `fieldSelection.paths`（`update.go:187`）的全字段分支跳过 `tenantID`；`resolveUpdatePath`（`update.go:234`）把 `tenantID` 加入拒绝列表，和 `_id` / `createdAt` 一起 |
| `Upsert` / `UpsertMany` | filter 注入后，`tenantID` 会成为等值条件，`insertOnlyFields`（`upsert.go:189`）据此把它排除出 `$setOnInsert`，插入时由 MongoDB 从 filter 取值——**这条路径刚好自动正确，但必须有用例锁住**。同时 `updateTarget` 走 `paths`，`tenantID` 已被排除出 `$set`。插入后应把租户回填到 obj（与 `inserted` 回填 `_id` / `createdAt` 同处） |
| `Modify` / `ModifyMany` | 无需单独改动：`modifyDoc`（`modify.go:133`）已把每个字段过一遍 `resolveUpdatePath`，上一行的拒绝列表自动生效 |
| `Delete*` / `GetAndDelete` | 只靠 filter 注入，无额外改动 |

**聚合路径**：

- `ListLatestBy`（`aggregate.go:39`）与 `Aggregate`（`aggregate.go:106`）都只在
  `len(filter) > 0` 时才插入首个 `$match`。注入后租户集合的 filter 恒非空，自然满足。
- **但 `Aggregate` 的管道由领域层自由定义**，`$lookup` / `$graphLookup` / `$unionWith`
  会读到另一个集合且不受首个 `$match` 约束，`$merge` / `$out` 会写出集合。
  租户集合上应拒绝这几个 stage，并在错误信息里说明需要跨集合聚合时自行处理租户条件。
  旧方案 #249/#250 采用白名单，本方案建议用黑名单：现有管道已经在用的 stage 不该被新规则打断。

**`ListPage`**（`page.go:54`）内部复用 `parseOpts` 与 `find`，无额外改动。

## 6. 租户来源与传播

数据层只认 ctx（或显式 `ForTenant`），所以每条进入业务逻辑的路径都要保证 ctx 里有租户。

| 入口 | 现状 | 要做的事 |
|---|---|---|
| HTTP 业务 API | `tenant.Required` 已挂在 `/bkms/v1/bkms-server`（`server.go:186`） | 无 |
| 其它路由组 / 健康检查 | 未挂中间件 | 确认这些路由不碰租户集合，或补挂 |
| 异步任务 | `payloadEnvelope` 只有 `_authUser`（`envelope.go`） | 增加 `_tenantID`：入队时从 ctx 取，取不到直接失败；消费时写回 ctx。升级前已在队列、没有该字段的存量任务回落 `default` |
| 定时任务 / worker 自发逻辑 | 无 ctx 租户 | 显式 `AllTenants()` 遍历，或按租户 `ForTenant` 循环 |
| `cmd/migration/*` 一次性脚本 | 无 ctx 租户 | 显式 `AllTenants()`，与旧方案一致（`refresh_workspace_bkmonitor_perms`、`backfill_default_alert_strategies`、`bind_bscp_project`、`list_bscp_projects`、`app_bscpcfg_mgr`） |
| 平台管理 API（`pkg/platmgt/**`） | 走普通中间件，`RequirePlatformRole` 额外打上显式范围标记 | 每次访问租户集合都必须显式 `AllTenants()` / `ForTenant()`，漏传报错 |
| 单测 | 无 ctx 租户 | `testutil` 提供带租户的 ctx 构造，见第 10 节 |

存量任务回落 `default` 是一次性兼容，应该带上线窗口说明，并在队列排空后移除。

## 7. 索引与数据迁移

### 7.1 索引

租户集合的索引要按用途分两类处理：

- **查询索引**：按租户过滤后再按业务条件查，`tenantID` 应作为前缀。
  但现有大量索引以 `workspaceID` 打头，而 workspace 本身归属唯一租户，
  `workspaceID` 的选择性已经足够高，`tenantID` 前缀带来的收益有限。
  建议**不批量重建**，只在实际出现慢查询的集合上调整。
- **唯一索引**：这里才是必须决策的地方。
  - `workspaces` 的唯一键 `id` 是**用户自选的 slug**
    （`serializer/workspace.go:394`，`binding:"required,workspace_id"`）。
    保持全局唯一意味着 A 租户可以占用 B 租户想要的空间 ID，并能通过"创建失败"探测到别的租户用了哪些 ID。
    改成 `tenantID + id` 唯一则需要同步评估：URL 路径 `/workspaces/:workspaceID/...` 不受影响
    （租户来自 header），但 IAM 资源 ID、BCS/BKCI/BKRepo 等外部系统里的命名空间与项目名
    是否会撞车需要逐个确认。
  - 其余集合的业务唯一键大多已经以 `workspaceID` 或 `appID` 打头
    （如 `release_strategies` 的 `workspaceID + name`），workspace 归属租户后这些键天然不会跨租户冲突，
    可以不动。
  - `id` / `_id` 这类由 `bson.NewObjectID()` 生成的主键保持全局唯一，不加租户前缀。

**建议第一期保持 `workspaces.id` 全局唯一**（与三版旧方案的结论一致），
把它作为产品决策单独推进，不阻塞数据层落地。

### 7.2 迁移

`db/migrations` 当前最新是 `000020`，租户迁移从 `000021` 起。按集合拆分，不要一个巨型迁移：

1. `000021_workspaces_tenant_id`：`workspaces` 回填 `tenantID = "default"`，
   条件 `{"tenantID": {"$exists": false}}` 保证幂等，并建 `tenantID_1` 索引。
2. 后续每批业务集合一个迁移。有 `workspaceID` 的集合可以直接回填 `default`
   （单租存量数据全部属于 `default`），不需要 join `workspaces`——
   只有将来存在多个真实租户时的二次迁移才需要 join。
3. `down` 先删索引，再只 `$unset` 值为 `default` 的字段，不回滚真实多租数据。

迁移格式是 MongoDB 原生命令 JSON 数组，规范见 `db/AGENTS.md` 与 `README.md` 的"数据库迁移"节。
每个迁移配套 `.md` 说明文件。

注意 webserver 启动不会自动跑迁移（`cmd/webserver.go` 只做 `database.InitClient` + `storereg.Init`），
需要运维执行 `bkms-server migrate up`。**回填迁移必须先于新代码上线**，
否则存量文档没有 `tenantID`，按租户过滤会一条都查不到。

## 8. 逃生舱与残留风险

### 8.1 `GenericStore.Collection()`

`store.go:91` 暴露底层 `*mongo.Collection`，绕过全部注入。
当前生产代码只有一处使用：`pkg/build/chart/semver/store.go:66`（配合 `store.Counter`），
其余都在 `_test.go` 里。改造时处理这一处即可，并在方法注释里写明
"返回的集合不带租户条件，租户集合上使用前请确认"。

### 8.2 `store.Counter`

`counter.go` 是独立的计数器集合，文档为 `{_id: key, seq}`，不经过 schema 与 Meta。
如果计数器的 key 是按 workspace / app 拼出来的（helm chart semver 的用法），
租户隔离由 key 本身承担，不需要改；如果存在全局 key，需要把租户拼进 key。
接入时逐个确认，并在 `NewCounter` 的注释里写明这一点。

### 8.3 跨租户调用点漏传 `AllTenants()`

这是 `Opt` 形态引入的风险（见 5.3.1）。平台管理的某个方法忘了传 `store.AllTenants()` 时，
ctx 里有中间件塞的租户，查询会静默只返回那一个租户的数据。失败模式是"少看到数据"而不是越权，
但排查成本高。

**运行时标记**：`platmgt.RequirePlatformRole` 通过权限检查后，用 `store.RequireExplicitTenantScope`
给 ctx 打标记。带标记的 ctx 上，租户集合的操作若没有传 `AllTenants()` / `ForTenant()`，
`resolveTenant` 直接返回 `ErrTenantScopeRequired`，不再使用 ctx 里的租户。

- 漏传在第一次执行时报错，平台管理 service 的用例在带标记的 ctx 下调用即可覆盖。
- 按 ctx 判断，经过多少层领域函数都能拦住；静态 linter 对经可变参数透传的 opts 和间接调用无能为力，故未采用。
- 标记只要求显式选择范围，不授予跨租户能力，跨租户仍必须在调用点写 `AllTenants()`，
  与 5.3.1"租户范围必须出现在调用点"的约束一致。
- 不隔离的集合不受影响。

其余缓解：

- 平台管理的接口用例要覆盖"存在多个租户的数据时能全部读到"，而不只是单租户下的正确性。
- `grep -rn 'store.AllTenants()'` 是跨租户调用点的完整清单。
- 标记只挂在平台管理路由上；`cmd/migration/*` 的 ctx 里本来就没有租户，漏传会报 `ErrTenantIDRequired`，不需要标记。

### 8.4 其它

- **跨集合聚合**：见 5.4，靠 stage 黑名单兜住。
- **`workspaceID` 与 `tenantID` 不自洽**：store 层只按 `tenantID` 过滤，
  如果某条数据的 `tenantID` 和它所属 workspace 的不一致，表现为数据"消失"而不是越权，
  属于可接受的失败模式。回填脚本要保证一致性。
- **IAM / BCS / BKRepo 等外部系统**：本方案只覆盖 bkms 自己的 MongoDB，
  外部系统的租户隔离是独立议题。

## 9. 落地分期

每一期都可独立上线、独立回滚。

**第一期：机制落地（无行为变化）**

- `pkg/store`：`Meta.TenantIsolated`、`schema.tenantID`、`ForTenant` / `AllTenants` 两个 Opt、
  `parseOpts` / `writeQuery` 改签名、各写入路径改动、聚合 stage 黑名单。
- 51 个 Meta 一行不改：`TenantIsolated` 零值 false，行为与现在完全一致。
- `pkg/store` 的单测补齐租户用例（见第 10 节）。

**第二期：链路打通**

- taskq envelope 增加 `_tenantID`。
- 确认 / 补齐各路由组的 `tenant.Required`。
- `pkg/platmgt/**` 与 `cmd/migration/*` 显式 `AllTenants()`。
- `testutil` 提供带租户的 ctx 与 Store 构造。

**第三期：按集合迁入**

从 `workspaces` 开始，之后按领域分批（core → workload → deploy → build → extension → bkintegrations → observability）。
每批的动作固定为三步：模型加字段、`Meta.TenantIsolated` 改为 true、写回填迁移，
再加一组租户隔离用例。

**第四期：收尾**

- 移除存量任务的 `default` 回落。
- 复查 `Collection()` 与 `Counter` 的使用。
- 按实际慢查询调整索引。

## 10. 测试

机制集中在 `pkg/store`，所以测试重心也在那里，而不是像 #251 那样要求每个迁入的 Store 重复一整组隔离用例。

**`pkg/store` 包内（必须覆盖全部注入点）**：

- 每个公开方法在 A 租户 ctx 下看不到 / 改不到 / 删不到 B 租户的数据。
  这组用例要逐方法枚举，因为"漏改一个方法"是本方案唯一的真实风险。
- `Create` / `Replace` / `Upsert` 写入时覆盖调用方伪造的 `tenantID`。
- `Update` / `Modify` 试图修改 `tenantID` 时报错。
- `Upsert` 插入路径确实写入了正确的租户（锁住 5.4 里"自动正确"的那条链路）。
- 过滤条件里传入 `tenantID` 时被丢弃而不是进 `$and`。
- ctx 无租户时每个方法都返回 `ErrTenantIDRequired`。
- `AllTenants` 可跨租户读改删，传给写入整个文档的操作时被 `optKind` 拒绝；
  `ForTenant` 优先于 ctx，且能指定写入的租户；范围重复指定报错；`Combine` 包住租户范围时报错。
- ctx 带显式范围标记时，未传范围报 `ErrTenantScopeRequired`，传了 `AllTenants` / `ForTenant` 正常工作。
- `Global` 集合不注入、且调用 `AllTenants` / `ForTenant` 报错。
- `Meta.TenantIsolated` 与模型字段不一致时 `New` 返回错误。
- 租户集合上使用 `$lookup` 等 stage 被拒绝。

**领域包**：迁入时只需在已有用例里确认关键查询仍然工作，
不必重复隔离逻辑——隔离已经由 `pkg/store` 保证。
按 `bkms-server/AGENTS.md` 的约定，用 `testutil.NewStore(GinkgoT(), XxxMeta)` 真实打库，不 mock store。

**`testutil`**：补一个带租户的 ctx 工厂，默认注入 `default`，
让现有 1000+ 条用例在第三期迁入后不需要逐个改 ctx。

## 11. 已落地的 demo

以 `workspaces` 为试点集合实现了第一到第三期的一个纵切，与三版旧方案的试点范围一致，
可以直接跑起来验证上面的设计。

**机制（第一期）**

| 文件 | 内容 |
|---|---|
| `pkg/store/tenant.go` | 五个错误、`TenantOpt`、`AllTenants` / `ForTenant`、`RequireExplicitTenantScope`、`resolveTenant` / `injectTenant` / `tenantForWrite` |
| `pkg/store/opts.go` | `optKind` 扩到 uint16，新增 `kindTenantFixed` / `kindTenantAll` 并并入各操作的接受集合 |
| `pkg/store/schema.go` | 识别 `tenantID` 字段并校验类型，`fillTenant` |
| `pkg/store/meta.go` / `store.go` | `Meta.TenantIsolated`，`New` 的交叉校验，`GetByID` 加 opts |
| `pkg/store/query.go` | `parseOpts` / `writeQuery` 收 ctx 并注入，`callerConds`，拒绝 `Combine` 里的租户范围 |
| `pkg/store/write.go` / `upsert.go` / `update.go` | 写入时填租户，更新时拒改 `tenantID` |
| `pkg/store/aggregate.go` | 跨集合 stage 黑名单 |

**链路（第二期）**

- `pkg/infras/taskq/envelope.go` / `client.go`：envelope 增加 `_tenantID`，
  入队时 ctx 无租户直接失败，消费时写回 ctx，无该字段的存量任务回落 `default`。
- `pkg/platmgt/workspace/{service,admin/service}.go` 与 `cmd/migration/` 下 5 个命令，
  共 8 个跨租户调用点传 `store.AllTenants()`；`workspace.Update` 相应加 `opts ...store.Opt`。
- `pkg/platmgt/middleware.go`：`RequirePlatformRole` 通过后给 ctx 打显式租户范围标记。
- `pkg/common/testutil/tenant.go`：`WithDefaultTenant()`。

**试点集合（第三期）**

- `pkg/core/workspace/workspace.go` 加 `TenantID` 字段，`store.go` 声明 `TenantIsolated: true`。
- `db/migrations/000021_workspaces_tenant_id.{up,down}.json` 回填 `default` 并建 `tenantID_1` 索引。

**验证**：`pkg/store/tenant_test.go` 覆盖第 10 节列出的全部条目，
`pkg/store` 与 `pkg/infras/taskq` 两个套件通过。

## 12. 待决问题

1. `workspaces.id` 是否改为租户内唯一？涉及产品语义与外部系统命名，建议单独评估，第一期保持全局唯一。
2. `depservice_services`、`component_defs`、`runtime_images` 这几张"平台内置 + 可能允许自定义"的集合
   是全局还是租户隔离？需要产品确认。
3. 平台管理后台的租户切换交互：是沿用 `X-Bk-Tenant-Id` 头，还是走独立的跨租户接口？
   影响 `AllTenants()` 的使用范围。
4. `system` / `superadmin` 两个保留租户（`tenant/types.go`）在数据层的语义：
   是普通租户值，还是等价于 `AllTenants`？建议作为普通租户值，跨租户一律走显式 API。
5. 回填节奏：是一次性全量回填，还是跟随第三期分批？分批期间系统处于"部分集合隔离"的中间态，
   需要确认不存在跨集合的一致性要求。
