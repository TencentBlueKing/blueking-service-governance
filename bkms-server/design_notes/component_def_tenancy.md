# 组件定义接入租户隔离

`component_defs` 原为全局集合，混存平台内置定义与租户自建定义。本文是不拆表的接入方案，已实现。

## 1. 问题

- 租户自建定义对所有租户可见：`scopeType=global` 由请求直接传入；`scopeWorkspaceIDs`、`managedByWorkspaceIDs` 可填其他租户的空间。
- `name + version` 全局唯一，租户之间互相占用名称。
- 内置定义上的实例计数是全部租户的累加值；删除保护依赖该计数。
- 删除租户时无法清理其自建定义。

## 2. 方案

### 2.1 平台租户

内置定义写入平台租户：多租户模式为 `system`，单租户模式为 `default`。
多租户环境为全新部署，不存在从单租户原地切换，因此不会出现两份内置定义。

- 平台租户只由 `tenant.PlatformTenantID()` 根据 `config.G.Tenant.EnableMultiTenantMode` 给出，各处不自行判断模式；配置未加载时返回错误。
- `LoadBuiltinFromFolder` 写入时传 `store.TenantOf(平台租户)`。
- `LoadBuiltinFromFolder` 强制设置 `IsBuiltin = true`，不依赖 YAML 中的字段：共享读取以它为准，漏写会导致内置定义对其他租户不可见。

### 2.2 store：共享读取

`component_defs` 设为 `TenantScoped`，并声明 `Meta.SharedFromPlatform: bson.D{{"isBuiltin", true}}`：

| 操作 | 默认过滤 |
|---|---|
| 读（Get / List / ListPage / Exists / Count / ListLatestBy / Aggregate） | `tenantID = ctx 租户` 或 `tenantID = 平台租户 且 isBuiltin = true` |
| 写（Update / Modify / Replace / Delete） | `tenantID = ctx 租户`，租户改不到平台定义 |
| 插入（Create / Upsert） | ctx 租户；平台定义由加载命令 `TenantOf` 写入 |

- 共享条件带 `isBuiltin`：多租户下 `system` 也是登录租户，其用户自建的定义不能因此对所有租户可见。
- 读操作经 `readQuery` 构造条件，写操作仍走 `parseOpts` / `writeQuery`，共享只可能出现在读路径上。
- `TenantAll()` / `TenantOf()` 语义不变，不叠加共享条件。ctx 租户就是平台租户时只按本租户过滤。

### 2.3 名称

- 唯一索引由 `name + version` 改为 `tenantID + name + version`，租户之间名称互不影响。
- 内置定义保留原名，不加前缀。
- 租户创建定义时，名称不得与内置定义重名（`BuiltinDefExists`）。
- 按名称解析（`evaluate`、创建实例、预览、修改、删除）统一用 `GetDef`：同名时本租户优先。
  只有平台新增的内置定义与某租户已有定义重名时才会出现同名；本租户优先保证该租户的存量实例渲染结果不变。
  实现上按 `isBuiltin` 升序取第一条，自建定义的 `isBuiltin` 总是 false。
- 组件列表通过 `ExcludeShadowedBuiltins` 过滤掉被本租户同名定义覆盖的内置定义。
- 平台租户自己的自建定义与新增内置定义同名时（单租户模式下即 `default` 的自建定义），两者落在同一租户，
  加载命令会因唯一索引失败，需要先改名其中一个。

### 2.4 可见范围

- 租户自建定义只在本租户内可见；`scopeType=global` 的含义变为本租户内全局。
- 创建时校验 `scopeWorkspaceIDs`、`managedByWorkspaceIDs` 均属于本租户；修改时只校验本次传入的值，
  存量值中已删除的空间不阻塞其他字段的修改。

### 2.5 实例计数

新增集合 `component_def_instance_counts`（`TenantScoped`），唯一键 `tenantID + defName + version`，
字段 `appCompInstanceCount`、`workspaceCompInstanceCount`。

- `workspace/comp_store.go`、`appmodel/store.go` 的 `adjustInstanceCounts` 改为调用 `component.UpdateInstanceCount` 写该表：
  先 `Modify` `$inc`；记录不存在时 `Create`，并发下撞唯一键则再 `Modify` 一次；记录不存在时的减少直接忽略。
  项目没有使用 Mongo 事务，撞唯一键不会中止外层操作，因此不需要给 store 增加 upsert `$inc`。
- `ComponentDef` 删除两个计数字段；列表接口按本页定义批量查询计数后拼装，返回结构不变。
- 删除保护只看本租户计数；删除定义时一并删除其计数记录。
- 计数与实例写入不在同一事务，会有偏差（现状如此）。

## 3. 迁移

`000026_component_defs_tenant_id`，细节见对应的 `.md`：

1. 存量环境都是单租户，定义全部回填 `tenantID = default`，与单租户模式下的平台租户一致，内置定义无需搬迁。
2. 唯一索引 `name_1_version_1` 替换为 `tenantID_1_name_1_version_1`。
3. 定义上的非零计数通过 `$merge` 写入计数表，再移除定义上的计数字段。
4. 新旧版本混跑期间旧 Pod 写在定义上的计数增量不搬迁：窗口很短，期间增删组件的概率低，偏差可接受。

## 4. 改动范围

| 位置 | 改动 |
|---|---|
| `pkg/infras/tenant` | `PlatformTenantID` |
| `pkg/store` | `Meta.SharedFromPlatform`、读路径 `readQuery` |
| `pkg/extension/component` | Meta、`GetDef`、`BuiltinDefExists`、`ExcludeShadowedBuiltins`、计数表、加载命令写平台租户 |
| `pkg/extension/component/handler` | 创建重名与空间校验、列表拼装计数、删除保护改用计数表 |
| `pkg/core/workspace`、`pkg/workload/appmodelcore/appmodel` | 计数写入改到计数表 |
| `db/migrations` | `000026`：回填 `tenantID`、替换唯一索引、计数表 |
