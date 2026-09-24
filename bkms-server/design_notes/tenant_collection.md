# Collection 租户隔离

业务集合的读写由 Collection 注入 `tenant_id`，避免每个 Store 手写过滤。默认租户来自 `ctx`，也可以用 `Scopes` 绑死某一个租户或跨租户访问。本轮只接入 `workspaces`，文档只描述当前这份实现已经做到的行为，以及还没补上的边界。

## 要点

- 默认租户只认 `ctx`，不认请求参数、不认调用方传入的 `tenant_id`（防伪造，注入时覆盖）。
- `Scopes(WithTenant(id))` 把这次访问绑到指定租户，不再读 `ctx`。`Scopes(AllTenants())` 跳过注入。
- HTTP 入口由 `tenant.Required` 写入 ctx：单租统一为 `default`，多租校验 header 与用户归属。
- 异步任务由 taskq envelope 携带 `_tenantId`，worker 恢复到 ctx。入队时 ctx 没有租户直接失败。升级前已在队列里、没有 `_tenantId` 的任务，恢复时回落 `default`。
- 缺租户直接失败（`ErrTenantIDRequired`），数据层不静默回落，避免漏中间件变成错数据。
- 字段名统一 `tenant_id`（`tenant.FieldTenantID`）。本轮不改 `workspaces.id` 全局唯一索引。

## 当前实现

当前不是在每个 Store 方法里手写 `tenant_id`，而是把“是否注入租户”收敛到 `dbtenant.Collection` 这一层。

### 1. 入口只负责把 tenant 放进 `ctx`

- HTTP 请求：`tenant.Required` 写入 `ctx`
- taskq worker：从 envelope 恢复到 `ctx`
- 命令行按 workspace id 或全表操作：显式走 `WorkspaceStore.Scopes(AllTenants())`
- 命令行指定某一个租户：`WorkspaceStore.Scopes(WithTenant(id))`，不把租户塞进 ctx

也就是说，业务侧只需要区分“这次是普通租户访问”还是“这次明确跨租户访问”，不用自己拼 `tenant_id` 过滤条件。

### 2. `Collection` 内部有三种访问视角

- `scopeTenant`：默认视角。集合带 `tenant_id`，按 `ctx` 注入租户条件
- `scopeGlobal`：集合本身没有 `tenant_id`，完全跳过注入
- `scopeCrossTenant`：集合带 `tenant_id`，但这次调用明确要看全部租户，也跳过注入

其中：

- `WrapTenant(coll)` 负责构造默认视角，并按集合名自动识别 `global` 表
- `Scopes(AllTenants())` 从同一底层集合派生出跨租户视图

这个区分很关键：`global` 表和 `cross-tenant` 都“不注入”，但前者是**表模型没有租户字段**，后者是**同一张租户表这次故意跨租户访问**。

### 3. 各类操作的租户规则

| 操作 | 处理方式 |
|---|---|
| Find / FindOne / Delete / Count | 只在 filter 上补 `tenant_id` |
| Insert / Replace | 在写入文档里补 `tenant_id`，调用方传入的同名字段会被覆盖 |
| Update | 只在 filter 上补 `tenant_id`，同时拒绝修改 `tenant_id` |
| Aggregate | 在管道首部补 `$match: {tenant_id}`；当前仅允许单集合安全 stage |

所以 Store 的方法签名不需要变化，仍然是普通的 `Find(ctx, filter)`、`Update(ctx, ...)`。差异体现在传入的 `ctx`，以及拿到的是默认 store、`Scopes(WithTenant(id))` 还是 `Scopes(AllTenants())` store。

### 4. 当前落地范围

- `WorkspaceStoreMongo` 已接入：构造时使用 `dbtenant.WrapTenant(coll)`
- `workspaces` 存量数据已通过迁移 `000018` 回填 `tenant_id=default`
- 当前实际生效的方法是 workspace 的 `List / ListWithPagination / Get / Create / Update / Delete / CountByState`

本轮还没有把所有业务集合都切到这套包装层。

## 为什么放 Collection，而不是 Store 小函数

核心原因是把“租户隔离”变成一条集中规则，而不是每个 Store 自己记得补一次 filter。

- 隔离规则只写一份，漏写 filter 不会静默跨租户读
- Store 接口保持普通的 `Find(ctx, filter)` / `Update(ctx, ...)`，不把租户做成额外查询参数
- 单租 / 多租的差异停在入口。默认数据层认 `ctx`；`WithTenant(id)` 把租户绑在这次拿到的 Collection 上

这里也没有把 Wrapper 做成完整的 `mongo.Collection` 替代物，而是只包装当前 Store 实际会用到的方法，控制改造面并降低后续维护成本。

## 跨租户怎么处理

分两种情况看：两类当前都已经有实现，但做法不一样。

**1. 表本身没有 `tenant_id`**（全系统一份）  
这个现在已经有实现：继续用 `globalTables`。集合名加进去，`Wrap` 后 `isGlobal` 为 true，读写都不注入。现有三个：`cluster_addon_defs`、`depservice_services`、`plat_admin_role_bindings`。`workspaces` 不要加进去。

**2. 表有 `tenant_id`，但这次调用要看全部租户**  
这个当前也已经实现，但实现方式不是改 ctx，而是走同一个 Store 的跨租户视图：

- `WorkspaceStore` 暴露 `Scopes(...)`。`WithTenant(id)` 绑死租户，`AllTenants()` 跨租户访问
- `WorkspaceStoreMongo.Scopes` 只替换内部 `Collection`，Store 方法本身不拼 `tenant_id`
- `Collection` 内部用 `scopeCrossTenant` 标记这个视图；只有默认 `scopeTenant` 才会注入 `tenant_id`
- platmgt workspace / platmgt workspace admin 已经显式使用 `h.registry.WorkspaceStore.Scopes(dbtenant.AllTenants())`

也就是说，当前 `workspaces` 已经支持“同一张表，普通业务按租户隔离，平台管理走跨租户视图”。

平台管理侧现在的写法类似：

```go
func (h *GinHandler) service() *platmgtworkspace.Service {
	return platmgtworkspace.NewService(
		h.registry.WorkspaceStore.Scopes(dbtenant.AllTenants()),
		h.registry.AppStore,
		h.registry.EnvStore,
	)
}
```

按 workspace id 或全表操作的命令行已经复用 `WorkspaceStore.Scopes(AllTenants())`：`refresh_workspace_bkmonitor_perms`、`backfill_default_alert_strategies`、`bind_bscp_project`、`list_bscp_projects`、`app_bscpcfg_mgr`。后续同类调用同样走这个视图，不给 `List/Get` 增加 `SkipTenant` 参数。
这样做的好处是：业务侧和平台侧都还是普通的 `store.List/Get/CountByState`，差异只体现在拿到的是默认 store 还是 `Scopes(AllTenants())` store。

## 当前边界

- 当前只有 `workspaces` 接入了 Wrapper，其它集合还没有统一切换。
- 全局表已经支持按集合名跳过租户注入。
- `workspaces` 已经支持 `Scopes(AllTenants())`；但这套模式目前还没有推广到其它租户表。
