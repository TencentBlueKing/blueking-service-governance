# 多租户 DB 层隔离方案

在 `GenericStore`（`pkg/store`）内部实现租户隔离：把 `tenantID` 做成 store 自动维护的字段，
与 `_id` / `createdAt` / `updatedAt` 同级。所有模型共用这一套实现，领域包不需要写任何隔离代码。

## 1. 核心约定

- 租户字段统一为 `tenantID`（bson / camelCase），由 store 写入和过滤，业务代码不手动赋值。
- 集合的数据是否按租户划分由 `Meta.TenantScoped` 显式声明，并与模型字段交叉校验。
  `TenantScoped` 表示读写默认按租户过滤，不代表禁止跨租户访问，平台管理等场景显式传 `TenantAll()`。
- 租户默认取自 ctx。ctx 里没有租户时直接报错，不回落到 `default`，避免漏挂中间件时写出错误数据。
- 不使用 ctx 租户时，在 opts 中指定租户范围：`store.TenantAll()` 跨租户，`store.TenantOf(id)` 指定某个租户。
  租户范围必须写在调用点，不能藏进 `Combine`。
- 业务代码的调用写法不变，例如 `appStore.Get(ctx, bkmsapp.ByName(wsID, name))`：不传租户选项时按 ctx 中的租户过滤；
  需要时追加租户选项即可，如 `appStore.Get(ctx, bkmsapp.ByName(wsID, name), store.TenantOf("tenant-a"))`。

## 2. 数据模型

### 2.1 集合分类

| 类型 | 声明 | 说明 |
|---|---|---|
| 租户集合 | 模型有 `TenantID string \`bson:"tenantID"\``，`TenantScoped: true` | `workspaces` 与其余 50 个集合，清单见迁移 `000024_tenant_id.up.json` |
| 共享读取的租户集合 | 租户集合，读取时传 `TenantWithPlatform` | 平台租户中满足条件的文档对传了该选项的读取可见，目前只有 `component_defs`，见 3.5 与第 6 节 |
| 计数器 | `store.Counter`，文档为 `{_id: key, tenantID, seq}` | 经 `ModifyOrCreate` 按租户写入，见 3.6；镜像/Chart 构建号、特性环境序号 |
| 全局集合 | 无该字段，`TenantScoped` 为零值 false | 全系统一份，见下表 |

全局集合只存放平台数据，写入只来自平台路径（启动加载、命令行、平台管理接口），任何租户的请求都能读到全量数据。

| 集合 | 原因 |
|---|---|
| `runtime_images`、`cluster_addon_defs`、`depservice_services`、`bkci_pipeline_templates` | 平台目录，启动或命令行写入，租户只读 |
| `image_snapshots`、`image_tag_suppressions`、`repo_snapshot_statuses` | 远端镜像仓库的缓存，以 `repoKey = SHA256(仓库名 + 账号 + 密码)` 为 key，没有凭据算不出 key，读不到其他租户的快照 |
| `plat_admin_role_bindings` | 平台角色，按用户名全局唯一 |

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

- `Meta.TenantScoped bool`：零值 false 表示全局集合，因此可以逐个集合迁入，未声明的模型行为不变。
- `schema` 识别 `tenantID` 字段并要求类型为 `string`。
- `store.New` 交叉校验：声明隔离但没有字段、或有字段但没声明，都直接报错（`Of` 会 panic，模型单测即可发现）。

### 3.2 租户范围

```go
// 普通业务：租户来自 ctx
app, err := apps.Get(ctx, bkmsapp.ByName(wsID, name))

// 平台管理 / 运维：跨租户
list, err := workspaces.ListPage(ctx, page, size, opts, store.TenantAll())

// 指定租户（忽略 ctx）
err := apps.Create(ctx, &app, store.TenantOf("tenant-a"))

// 本租户加平台共享数据（只读），见 3.5
def, err := defs.Get(ctx, component.ByNameVersion(name, version), store.TenantWithPlatform(component.BuiltinShared()))
```

一次操作的租户按以下顺序确定：

1. 全局集合：不注入；若传了租户选项，报 `ErrTenantScopeNotSupported`。
2. 传了 `TenantAll()`：不按租户过滤。
3. 传了 `TenantOf(id)`：用 `id`，为空报 `ErrTenantIDRequired`。
4. 没传，或传了 `TenantWithPlatform`：ctx 带显式范围标记时报 `ErrTenantScopeRequired`（见 5.2）；
   否则取 ctx 租户，取不到报 `ErrTenantIDRequired`。`TenantWithPlatform` 再按 3.5 放宽条件。

约束：

- 租户选项只能传一个，重复传报 `ErrTenantScopeDuplicated`。
- `TenantAll()` 只用于读、局部更新（`Update` / `Modify`）和删除；`Create` / `Replace` / `Upsert` 只接受 `TenantOf`，
  因为新文档必须落在确定的租户；`TenantWithPlatform` 只用于读。由 `optKind` 的类别检查实现，无需额外规则。
- 被 `Combine` 包住的租户选项报 `ErrTenantScopeCombined`，保证跨租户在调用点可见、可被 `grep store.TenantAll()` 找全。
- 租户范围只存在于每次调用新建的 `query` 中，`GenericStore` 实例不持有任何租户状态，多处共用同一实例互不影响。
  不要在 store 上增加会修改实例状态的方法。

### 3.3 注入点

所有读写都经过 `parseOpts`（写操作经 `writeQuery`），租户条件在这里统一注入：
字段名解析完成后，向 `q.conds` 追加 `tenantID = <租户>`。

- 调用方自己在条件里写 `tenantID` 会报错，而不是覆盖或丢弃：重复 key 会被 `filter()` 合成 `$and` 仍参与匹配，
  静默丢弃又会掩盖调用方的误解。`TenantAll()` 下允许按 `tenantID` 筛选，这是平台管理的正常用法。
- 写操作"必须带过滤条件"的保护改为看 `callerCondCount`（注入前调用方条件数），否则注入后条件恒非空，
  `DeleteMany(ctx)` 会变成删光当前租户的数据。`Upsert` 把对象自带的 `_id` 计入调用方条件。

写入路径：

| 操作 | 处理 |
|---|---|
| `Create` / `CreateMany` / `CreateManyIfAbsent` | 写入前用解析出的租户覆盖对象上的值，并回填到对象 |
| `Replace` | 同上；过滤条件已注入，跨租户替换返回 NotFound |
| `Update` / `Modify` | 拒绝修改 `tenantID`；全字段更新时跳过 `tenantID` |
| `Upsert` | 过滤条件中的 `tenantID` 等值条件在插入时由 MongoDB 写入，`$set` 不含 `tenantID` |
| `ModifyOrCreate` | 同 `Upsert`；只接受 `TenantOf` |
| `Delete*` / `GetAndDelete` | 仅靠过滤条件注入 |

Encode 钩子可能整体重建对象（如 appspec 的 `normalizeSpec` 经 `Clone` 重建），重建时漏复制租户会写出空 `tenantID`。
store 在 Encode 钩子执行后，按已填好租户的原对象把租户写回副本。

### 3.4 聚合

`Aggregate` / `ListLatestBy` 把租户条件作为管道首个 `$match`。

租户集合上拒绝 `$lookup`、`$graphLookup`、`$unionWith`、`$merge`、`$out`：它们读写其他集合，不受首个 `$match` 约束。
目前业务代码没有用到这几种阶段（仅 `db/migrations` 和 `cmd/migration` 直接操作原始集合时使用，不受影响），
需要关联时分别经各自的 Store 查询再在代码里拼装。

检查递归覆盖整个管道：这些阶段藏在 `$facet`、`$lookup.pipeline`、`$unionWith.pipeline` 的子管道里同样会被拒绝。

将来确需放开时的方案（按需再做）：

- 前提：`store.New` 登记"集合名 → 是否隔离"，查不到登记的集合按拒绝处理。
- `$lookup` / `$unionWith`：在子管道首部插入 `{$match: {tenantID: <租户>}}` 并递归处理子管道。
  `$lookup` 同时使用 `localField` 与 `pipeline` 需 MongoDB 5.0+。
- `$graphLookup`：把租户条件并入 `restrictSearchWithMatch`。
- `$facet`：递归处理每个子管道。
- `$merge` / `$out`：继续拒绝，回填放在迁移里做。

### 3.5 共享读取

平台维护、所有租户只读共用的数据存放在平台租户中，读取时在调用点传 `store.TenantWithPlatform(cond)`，
`cond` 指明平台租户中哪些文档共享，例如 `component_defs` 的 `{isBuiltin: true}`（由 `component.BuiltinShared()` 给出）：

| 操作 | 传 `TenantWithPlatform` 时的过滤 |
|---|---|
| 读（Get / List / ListPage / Exists / Count / ListLatestBy / Aggregate） | `tenantID = ctx 租户` 或 `tenantID = 平台租户 且满足 cond` |
| 写（Update / Modify / Replace / Delete / Create / Upsert / ModifyOrCreate） | 不接受该选项，报 option not supported；租户改不到平台数据 |

- 共享是一次读取的范围，而不是集合的属性：不传就只读本租户，默认读写对称，与其它集合一致。
  "读得到平台数据、写不到"只出现在显式传了该选项的调用点上，这些调用点读到结果后要先判断是否为平台数据再决定能否写
  （如修改、删除组件定义前检查 `IsBuiltin`）。
- 平台租户由 `tenant.PlatformTenantID()` 给出：多租户模式为 `system`，单租户模式为 `default`；配置未加载时返回错误。
  多租户环境为全新部署，不存在从单租户原地切换，平台租户在一个环境内是固定的。
- 它是 `TenantOpt`，与 `TenantAll()` / `TenantOf()` 互斥，同样不能放进 `Combine`。
  与 `TenantAll()` 一样写在调用点上，`grep store.TenantWithPlatform` 能找全所有读到平台数据的地方；
  领域包只封装共享条件（返回 `bson.D` 的函数），不封装选项本身。
- ctx 租户就是平台租户时只按本租户过滤；`cond` 为空时报错，避免把平台租户的全部数据共享出去。
- `cond` 要能区分平台数据与平台租户用户自己的数据：多租户下 `system` 也是登录租户。

### 3.6 绕过 store 的操作

`Collection()` 返回底层集合，直接操作不会注入租户条件，调用方须自己带上 `tenantID`。业务代码不再直接操作集合。

计数器需要原子递增并取回新值，走 `ModifyOrCreate`（`FindOneAndUpdate` + upsert）：
`store.Counter.Next`（`build_counters`、`helm_chart_build_counters`、`feature_environment_counters`）
与 `semver.Next`（`helm_chart_semver_counters`，归零用 `$mul: 0`，因为 `Modify` 不接受普通字段的 `$set`）。
`_id` 仍只是 key（应用 ID），因此 key 必须跨租户唯一，被其它租户占用时返回 `ErrDuplicate`。

直接操作原始集合的只剩迁移与运维命令，按设计跨租户：

- `db/migrations`
- `cmd/migration` 的 `scan_app_runtime_model`（只读）、`cleanup_orphan_app_config_file_versions`（按 appID 清理）
- `extension/component/migrate`（`migrate_component_patch`）：遍历全部组件定义并原地更新
- `workload/appmodelcore/appspec/migrate`（`migrate_tke_route_eni_component`）：跨租户读出，写入经 store 并用
  `ScopeToTenant` 落到文档所属租户

## 4. 租户来源

数据层只认 ctx（或 `TenantOf`），每个入口都要保证 ctx 里有租户：

| 入口 | 处理 |
|---|---|
| HTTP 业务 API | `/bkms/v1/bkms-server` 路由组挂 `tenant.Required`，写入 ctx |
| 平台管理 API | 同上，另由 `RequirePlatformRole` 打显式范围标记，见 5.2 |
| 异步任务 | envelope 增加 `_tenantID`：投递时 ctx 无租户直接失败，消费时写回 ctx，详见 4.1；只访问全局集合的任务声明 `Global()`，见 4.2 |
| `cmd/migration/*` 与命令行 | ctx 无租户，访问租户集合时显式传 `TenantAll()`；后续调用业务函数或投递任务前用 `store.ScopeToTenant` 设置资源所属租户，见 5.1 |
| 定时 / worker 自发逻辑 | 显式 `TenantAll()` 遍历，或按租户 `TenantOf` 循环 |
| 单测 | `testutil.NewDefaultTenantContext()` 构造带默认租户的 ctx，见 4.3 |

命令行写入租户数据时按资源所属租户写入。例如端口转发白名单：先跨租户查出环境（同时校验存在、非生产），
按环境所属租户分组，用 `TenantOf` 写入；`List` / `Remove` 用 `TenantAll()`；`CheckPermission` 按请求租户过滤。

### 4.1 存量异步任务

- 带 envelope 但没有 `_tenantID` 的任务（升级前入队）消费时回落到 `default`，并打 warn 日志。前提是回填迁移已先执行。
- 没有 envelope 的更早期任务不会恢复租户，访问租户集合会失败。
- 滚动升级期间新旧版本混跑无影响：旧 worker 忽略 `_tenantID`；新 worker 对旧任务走回落。
- 轮询类任务在 handler 内重新投递时使用恢复后的 ctx，会自动带上 `_tenantID`。
- 回落逻辑可在 pending / scheduled / retry / archived 四类任务中都不再有缺 `_tenantID` 的任务后移除。

### 4.2 不属于任何租户的任务

有些任务只读写全局集合，本身没有租户概念，例如镜像 tag 详情同步（`DetailSyncTask`）：
它只读写 `image_snapshots` / `repo_snapshot_statuses`，按 `repoKey` 去重，同一个仓库与凭据全系统只同步一份（见 2.1）。
触发方可能是命令行刷新平台运行时镜像，也可能是某个租户的应用或工作空间，任务本身不属于触发方的租户。

这类任务在定义处声明 `Global()`，与集合用 `Meta.TenantScoped` 声明对称：

```go
DetailSyncTask = taskq.NewTaskType[ImageDetailSyncArgs](name, handle, opts...).Global()
```

- 投递：不要求 ctx 带租户，ctx 里有租户也不写入 envelope；其余任务仍在投递时校验租户，漏设立即报错。
- 消费：不往 ctx 写租户，也不走 4.1 的 `default` 回落。handler 误访问租户集合会报 `ErrTenantIDRequired`，
  而不是悄悄读到某个租户的数据。升级前入队、带着 `_tenantID` 的同类任务同样忽略这个字段。
- 不用 `tenant.DefaultTenantID` 代替：`default` 是真实租户，填它等于声称任务属于 `default`，还会掩盖上面这类误访问。
- 不用 `TenantAll()` 表达：它是作用于租户集合的 store 选项，传给全局集合会报 `ErrTenantScopeNotSupported`。
- 不做成投递时的选项：任务是否属于租户是任务本身的属性，不应由各个调用点分别决定。

### 4.3 单测

租户集合的用例在 `BeforeEach` 里用 `testutil.NewDefaultTenantContext()` 构造 ctx（带默认租户，用例对 ctx 的统一要求都加在这里），
后续所有读写（包括 `AfterEach` 的清理）都要用这个 ctx；直接插入的原始文档需带 `tenantID`。
`dbfactory` 只会给自己内部的 `Create` 补租户，不会改写调用方传入的 ctx。
验证租户过滤本身的用例应直接用 `tenant.WithTenantID` 构造两个不同租户的 ctx。

跨租户行为的用例：`pkg/store/tenant_test.go`（`ScopeToTenant`、Encode 回写、共享读取）、
`platmgt/workspace`（跨租户统计）、`platmgt/workspace/admin`（其它租户的空间授权与回收）、
`platmgt/workspace/imageregistry`（按空间所属租户读写）、`platmgt/portforward`（按环境租户写入）、
`store`（`ModifyOrCreate`、`Counter` 按租户计数）、`build/chart/semver`（按租户写计数器）、`extension/component`（内置定义共享、加载时跨租户检查同名）。

## 5. 跨租户访问

### 5.1 使用方

- 平台管理（`pkg/platmgt/**`）：已有调用点均显式传 `TenantAll()`。
  平台工作空间列表的应用数、环境数经 `app.CountByWorkspaceIDs` / `env.CountByWorkspaceIDs` 的 `scope` 透传 `TenantAll()`。
  领域函数需要让调用方选择租户范围时，参数写成 `scope ...store.TenantOpt` 并用 `store.ScopeOpts` 转换，
  而不是开放 `opts ...store.Opt`：调用方改不了函数的过滤、投影等语义，租户范围藏进 `Combine` 也会在编译期失败。
- 运维命令（`cmd/migration/*`）：`refresh_workspace_bkmonitor_perms`、`backfill_default_alert_strategies`、
  `bind_bscp_project`、`list_bscp_projects`、`app_bscpcfg_mgr`、`repair_trpc_taf_framework_defs`；
  以及 `cmd/portforward_whitelist.go`（经 `platmgt/portforward` 服务）。

跨租户查出资源后，要调用不接受租户选项、只从 ctx 取租户的函数（如 perm 管理器），
用 `store.ScopeToTenant(ctx, doc.TenantID)` 把 ctx 租户切到资源所属租户，同时解除 5.2 的显式范围标记。
`tenantID` 只能取自已查出的文档，不能取自请求参数。
使用点：`platmgt/workspace/admin`（GetRoleStatus / GrantAdmin / RevokeAdmin / 过期回收）、
`platmgt/workspace/imageregistry`、`cmd/migration` 的 `app_bscpcfg_mgr`、`backfill_default_alert_strategies`、
`repair_trpc_taf_framework_defs`、`refresh_workspace_bkmonitor_perms`，以及 appspec 迁移器。

谁有权跨租户由路由上的 `RequirePlatformRole` 决定；哪次查询跨租户由调用点的 `TenantAll()` 决定，store 不做权限判断。

### 5.2 防止漏传 `TenantAll()`

平台管理请求的 ctx 里是操作者自己的租户，漏传 `TenantAll()` 会静默只返回该租户的数据。

`RequirePlatformRole` 通过后用 `store.RequireExplicitTenantScope` 给 ctx 打标记。
带标记时，租户集合的操作若未传 `TenantAll()` / `TenantOf()`，返回 `ErrTenantScopeRequired`。

- 按 ctx 判断，经过多层领域函数调用也能拦住。
- 标记不授予跨租户能力，跨租户仍须在调用点写 `TenantAll()`。
- 全局集合不受影响；`cmd/migration` 的 ctx 本来没有租户，漏传会报 `ErrTenantIDRequired`，无需标记。
- 平台管理 service 的用例在带标记的 ctx 下调用，漏传会直接失败。

## 6. 组件定义

`component_defs` 原为全局集合，混存平台内置定义与租户自建定义：租户自建定义对所有租户可见，`name + version` 全局唯一，
内置定义上的实例计数是全部租户的累加值。改为租户集合，内置定义通过 3.5 的共享读取对所有租户可读，不拆表。

### 6.1 内置定义

- `LoadBuiltinFromFolder` 写入平台租户（`store.TenantOf(平台租户)`），并强制设置 `IsBuiltin = true`，不依赖 YAML 中的字段。
- 共享条件 `component.BuiltinShared()` 为 `{isBuiltin: true}`，`system` 租户用户自建的定义不会因此对所有租户可见。
  按名称解析定义（渲染、创建实例、预览）、列表、修改 / 删除前的查询都传 `store.TenantWithPlatform(component.BuiltinShared())`；修改、删除前带上它是为了明确拒绝内置定义，而不是当作不存在。
- 加载按 `平台租户 + name + version + isBuiltin = true` upsert，只覆盖已有的内置定义。
- 修改、删除接口拒绝内置定义。

### 6.2 名称与可见范围

- 唯一索引由 `name + version` 改为 `tenantID + name + version`，租户之间名称互不影响；内置定义保留原名，不加前缀。
- 自建定义不能与内置定义同名，保证每个租户按名称最多查到一条定义，按名称解析仍直接 `Get`：
  - 租户创建定义时由 `BuiltinDefExists` 拒绝；
  - 加载内置定义时跨租户检查已有的同名自建定义（含平台租户自己的），有冲突则整体失败并列出冲突的租户与定义。
    组件实例按名称引用定义，自建定义没有改名入口，平台新增内置定义时应避开已被占用的名称。
- 租户自建定义只在本租户内可见，`scopeType=global` 的含义变为本租户内全局。
  `scopeWorkspaceIDs`、`managedByWorkspaceIDs` 不做租户校验：定义本身按租户过滤，填入其他租户的空间也匹配不到。

### 6.3 实例计数

计数拆到 `component_def_instance_counts`（租户集合），唯一键 `tenantID + defName + version`，
字段 `appCompInstanceCount`、`workspaceCompInstanceCount`。

- `workspace/comp_store.go`、`appmodel/store.go` 的 `adjustInstanceCounts` 调用 `component.UpdateInstanceCount`：
  先 `Modify` `$inc`；记录不存在时 `Create`，并发下撞唯一键则再 `Modify` 一次；记录不存在时的减少直接忽略。
  项目没有使用 Mongo 事务，撞唯一键不会中止外层操作，因此不需要给 store 增加 upsert `$inc`。
- 列表接口按本页定义批量查询计数后拼装，返回结构不变。
- 删除保护只看本租户计数；删除定义时一并删除其计数记录。
- 计数与实例写入不在同一事务，会有偏差（现状如此）。

## 7. 索引与迁移

### 7.1 索引

- 只新增 `tenantID_1`：用于只 按租户的查询、平台管理按租户筛选、租户清理与回填核对。
  区分度低，日常查询仍走原有索引，写入多维护一个索引，代价可忽略。
- 不加 `{tenantID, ...}` 联合查询索引：查询索引多以 `workspaceID` / `appID` 开头，二者全局唯一，
  同一 workspace / app 的文档 `tenantID` 必然相同，加前缀不会少扫文档，只会增大索引。
- 唯一索引不改为租户内唯一（`component_defs` 除外，见 6.2）：
  - 以 `workspaceID` / `appID` 开头的已经等价于租户内唯一；
  - 对应外部或物理资源的（`environments` 的集群 + 命名空间、`bkci_projects.code`、`bkmonitor_apm_inst_config.apmID`）
    本应全局唯一，改成租户内唯一反而允许两个租户占用同一资源；
  - 业务主键 `workspaces.id` 是用户自选的 slug，改为租户内唯一涉及 IAM 资源 ID 和外部系统命名，作为产品决策单独评估。
- `plat_admin_port_forward_whitelist` 以环境 ID 作 `_id`，按环境查询走 `_id` 索引，无需额外索引。

### 7.2 迁移

迁移 `000024_tenant_id`，细节与发布顺序见对应的 `.md`：

- 回填 `tenantID = "default"`，条件 `{"tenantID": {"$exists": false}}` 保证幂等，并建 `tenantID_1` 索引；
  `down` 先删索引，再只 `$unset` 值为 `default` 的字段。
- `component_defs` 另外替换唯一索引，并把定义上的计数搬到计数表。
- webserver 启动不自动跑迁移。**回填迁移必须先于新代码上线**，否则存量文档按租户过滤会全部查不到。
- **全部实例升级完成后再执行一次回填**：迁移执行完到旧 Pod 全部退出之间，旧版本写入的文档不带 `tenantID`，
  对新代码不可见。回填是幂等的，重跑只会补上这部分文档。
