# 其余集合接入租户隔离

机制见 `tenant_isolation.md`，本文只记录把 `workspaces` 之外的集合全部接入时遇到的问题和处理方式。

## 1. 范围

- 按租户隔离：`workspaces` + 47 个集合（清单见 `db/migrations/000025_tenant_scoped_collections_tenant_id.up.json`），
  模型加 `TenantID`，`Meta` 设 `TenantScoped: true`。
- 保持全局：

| 集合 | 原因 |
|---|---|
| `runtime_images`、`cluster_addon_defs`、`depservice_services`、`bkci_pipeline_templates` | 平台目录，启动或命令行写入，租户只读 |
| `image_snapshots`、`image_tag_suppressions`、`repo_snapshot_statuses` | 按仓库凭据哈希 `repoKey` 存储，没有凭据算不出 key |
| `component_defs` | 混有租户自建定义，后续拆表 |
| `plat_admin_role_bindings` | 平台角色，按用户名全局唯一 |
| `store.NewCounter` 计数器（镜像/Chart 构建号、特性环境序号） | key 为应用 ID，本身不会跨租户冲突 |

## 2. store 层覆盖不到的地方

### 2.1 跨租户定位资源后调用只认 ctx 的函数

平台管理（临时管理员授权、过期回收）和运维命令先用 `TenantAll()` 查出资源，再调用 perm 管理器等下游函数。
下游函数不接受租户选项，而平台管理 ctx 是操作者自己的租户（且带显式范围标记），运维命令 ctx 没有租户，直接调用会读写错租户或报错。

处理：新增 `store.ScopeToTenant(ctx, tenantID)`，把 ctx 租户设为资源所属租户并解除显式范围标记。
`tenantID` 必须取自已查出的文档，不能取自请求参数。

使用点：`platmgt/workspace/admin`（GetRoleStatus / GrantAdmin / RevokeAdmin / 过期回收）、
`cmd/migration` 的 `app_bscpcfg_mgr`、`backfill_default_alert_strategies`、`repair_trpc_taf_framework_defs`、
`refresh_workspace_bkmonitor_perms`，以及 appspec 迁移器。

### 2.2 绕过 store 直接操作集合

`semver.Next` 用 `FindOneAndUpdate` + upsert 直接写 `helm_chart_semver_counters`，store 不会注入租户。
处理：从 ctx 取租户加进过滤条件，upsert 插入时 MongoDB 会把等值条件写进新文档；ctx 无租户返回 `ErrTenantIDRequired`。

其余直接操作原始集合的只有 `db/migrations` 和部分 `cmd/migration`（清理、扫描），按设计不受租户约束。

### 2.3 Encode 钩子重建对象丢租户

appspec 的 `normalizeSpec` 通过 `Clone` 整体重建对象，`Clone` 没复制 `TenantID`，写入的 `tenantID` 为空串，
同一应用下的 spec 撞唯一键。

处理：`Clone` 补复制；store 在 Encode 钩子执行后，按已填好租户的原对象把租户写回副本，避免其它钩子再踩同样的坑。

### 2.4 写入方没有租户

端口转发白名单由命令行写入，ctx 没有租户。处理：先跨租户查出环境（同时校验存在、非生产），
按环境所属租户分组，用 `TenantOf` 写入；`List` / `Remove` 用 `TenantAll()`；`CheckPermission` 按请求租户过滤。

### 2.5 平台管理统计

平台工作空间列表的应用数、环境数原来按操作者租户统计。
处理：`app.CountByWorkspaceIDs` / `env.CountByWorkspaceIDs` 增加 `opts` 透传，平台管理传 `TenantAll()`；
`app.GetAppsByIDs` 同样增加 `opts`，供迁移器跨租户查询。

## 3. 索引

- 只新增 `tenantID_1`：用于只按租户的查询、平台管理按租户筛选、租户清理与回填核对。
  区分度低，日常查询仍走原有索引，写入多维护一个索引，代价可忽略。
- 不加 `{tenantID, ...}` 联合查询索引：查询索引多以 `workspaceID` / `appID` 开头，二者全局唯一，
  同一 workspace / app 的文档 `tenantID` 必然相同，加前缀不会少扫文档，只会增大索引。
- 唯一索引不改为租户内唯一：
  - 以 `workspaceID` / `appID` 开头的已经等价于租户内唯一；
  - 对应外部或物理资源的（`environments` 的集群 + 命名空间、`bkci_projects.code`、`bkmonitor_apm_inst_config.apmID`）
    本应全局唯一，改成租户内唯一反而允许两个租户占用同一资源；
  - 业务主键（`workspaces.id`、`applications.id` 等）保持全局唯一，见 `tenant_isolation.md` 6.1。
- `plat_admin_port_forward_whitelist` 以环境 ID 作 `_id`，按环境查询走 `_id` 索引，无需额外索引。

## 4. 测试

- 访问租户集合的用例统一用 `testutil.WithTenantOrDefault(context.Background())`；直接插入的原始文档需带 `tenantID`。
- 跨租户行为的用例：`pkg/store/tenant_test.go`（`ScopeToTenant`、Encode 回写）、
  `platmgt/workspace`（跨租户统计）、`platmgt/workspace/admin`（其它租户的空间授权与回收）、
  `platmgt/portforward`（按环境租户写入）、`build/chart/semver`（按租户写计数器）。

## 5. 遗留

- `semver` 计数器按租户隔离，而 `store.NewCounter` 计数器保持全局，两者 key 都是应用 ID，后续可统一。
- 全局集合中由租户产生的数据（`component_defs` 自建定义、快照）无法按租户清理。

## 6. 发布

先执行迁移 000025，再发布代码，全部实例升级后再回填一次，详见 `db/migrations/000025_tenant_scoped_collections_tenant_id.md`。
