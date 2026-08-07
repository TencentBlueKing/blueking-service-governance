# Workspace 应用默认 AppSpec

## 功能说明

Workspace 可以按环境类型配置 AppSpec 初始值。创建 tRPC 或 TAF 应用时，系统读取这些配置，为应用生成默认 AppSpec 和环境 AppSpec。

这项功能只负责“应用刚创建时写入什么”。配置写入应用后就属于应用自身，Workspace 中的规则不会再影响它。

本文中的几个名词：

| 名词 | 含义 |
|------|------|
| AppSpec | 应用的部署配置，里面包含资源规格、更新策略、探针等配置块 |
| 默认 AppSpec | 应用级配置，记录的环境名称为 `default` |
| 环境 AppSpec | 某个具体环境的配置，记录的环境名称为实际环境名称 |
| 规则 | Workspace 下根据环境类型生效的 AppSpec 初始化模板，只在创建应用时使用 |

## 适用范围

- 适用于 AppModel 类型的 tRPC、TAF 应用。
- 不适用于 Helm、Agones 应用。
- 只处理标准环境，不处理特性环境。
- 只在创建应用时执行，不迁移或补写存量应用。
- 修改或删除规则，不会更新已经创建的应用。
- Workspace 后续新增环境，也不会给已有应用补写环境 AppSpec。

## 创建应用时会写入什么

系统按下面的顺序准备初始配置：

1. 读取当前 Workspace 的全部环境类型规则和标准环境。
2. 开始计算时先生成默认 AppSpec，其中固定包含平台资源规格和更新策略。
3. 再按标准环境的类型匹配规则，为有规则命中的环境生成环境 AppSpec。
4. 将默认 AppSpec 应用到 AppModel，并保存默认 AppSpec 和环境 AppSpec。

读取规则或标准环境失败时，应用创建直接失败，不继续写入应用数据。

### 平台默认 AppSpec

每个新应用都会生成一份默认 AppSpec，固定包含以下内容：

| 配置 | 字段 | 默认值 |
|------|------|--------|
| 资源规格 | 实例数 | `1` |
| 资源规格 | CPU Requests | `1` 核 |
| 资源规格 | CPU Limits | `2` 核 |
| 资源规格 | Memory Requests | `2Gi` |
| 资源规格 | Memory Limits | `4Gi` |
| 更新策略 | maxUnavailable | `25%` |
| 更新策略 | maxSurge | `25%` |

这些值由代码直接生成，不是 Workspace 规则：

- 不写入 `workspace_app_spec_rules` 集合。
- 不出现在规则查询接口中。
- 不能通过规则接口编辑或删除。

### 环境 AppSpec

环境规则只按环境类型匹配。支持的环境类型为：

- `development`
- `test`
- `staging`
- `production`

同一类型下的所有标准环境使用同一组规则。例如，Workspace 有两个 `production` 环境，那么这两个环境都会命中 `production` 规则。

生成环境 AppSpec 时有以下约定：

- 只写入命中的 section。
- 没有任何规则命中的环境，不创建环境 AppSpec。
- 环境 AppSpec 不复制平台默认资源规格和更新策略。
- 这里不计算环境的最终完整配置，也不做默认配置与环境配置的字段合并。

默认 AppSpec 和环境 AppSpec 是两类独立的初始化记录。本模块不负责计算部署时的最终有效配置。

## 规则

一条规则表示：某个 Workspace 中，某种环境类型的某个 AppSpec 配置块应该用什么初始值。代码中将配置块称为 section。

规则由以下内容唯一确定：

```text
Workspace + AppSpec section + 环境类型
```

例如：

```text
demo-workspace + resources + production
```

表示在 `demo-workspace` 中创建应用时，所有 `production` 标准环境都使用这条资源规格初始化配置。

规则只支持环境类型，不支持“所有环境”规则和“特定环境”规则。平台默认 AppSpec 也不属于规则。

### 支持的 section

| section | API 路径 | 内容 |
|---------|----------|------|
| `resources` | `resources` | 实例数、CPU 和内存规格 |
| `devMode` | `dev-mode` | 开发模式 |

第一期只开放 `resources` 和 `devMode` 的规则接口。

### 规则的完整性

每次接口请求只能包含 URL 所表示的一个 section，并且需要提交该 section 的完整配置。例如，资源规格规则必须同时包含实例数、CPU Requests/Limits 和 Memory Requests/Limits，不能只提交其中一个字段。

更新规则时会整体替换原有 section，不做字段级合并。

## 示例

假设 Workspace 中有三个标准环境：

| 环境名称 | 环境类型 |
|----------|----------|
| `dev` | `development` |
| `test` | `test` |
| `prod` | `production` |

Workspace 配置了两类规则：

- `production` 的资源规格规则。
- `test` 的开发模式规则。

创建一个新的 tRPC 或 TAF 应用后，会得到：

| AppSpec | 初始内容 |
|---------|----------|
| 默认 AppSpec | 平台资源规格、平台更新策略 |
| `dev` 环境 AppSpec | 不创建，因为没有规则命中 |
| `test` 环境 AppSpec | 开发模式 |
| `prod` 环境 AppSpec | 资源规格 |

`prod` 环境的资源规格来自 Workspace 规则；默认 AppSpec 中的平台资源规格仍然保持不变。两者是不同的 AppSpec 记录。

## API

第一期的 `resources` 和 `dev-mode` 各自保留独立的增删改查接口：

```text
GET    /workspaces/:workspaceID/app-spec/:section
POST   /workspaces/:workspaceID/app-spec/:section
PUT    /workspaces/:workspaceID/app-spec/:section/:ruleID
DELETE /workspaces/:workspaceID/app-spec/:section/:ruleID
```

`:section` 使用上表中的 API 路径。

### 查询

查询接口只返回数据库中保存的环境类型规则。没有规则时返回空列表：

```json
{
  "data": []
}
```

列表顺序没有业务含义，调用方不应依赖返回顺序。

### 新增和编辑

`POST` 和 `PUT` 使用相同的请求结构。下面是资源规格规则的示例：

```json
{
  "envType": "production",
  "spec": {
    "replicas": 3,
    "cpuRequests": "2",
    "cpuLimits": "4",
    "memoryRequests": "4Gi",
    "memoryLimits": "8Gi"
  }
}
```

`PUT` 可以同时修改环境类型和 section 配置。section 由 URL 决定，不能通过请求体修改。

同一 Workspace、同一 section、同一环境类型只能有一条规则。新增或更新后发生重复时返回 `400`。

### 删除

删除接口根据 URL 中的 Workspace、section 和规则 ID 查找规则。规则不存在，或者规则不属于该 Workspace 和 section 时返回 `404`。

### 权限和审计

- `GET` 使用 Workspace 查看权限。
- `POST`、`PUT`、`DELETE` 使用 Workspace 编辑权限。
- 新增、编辑、删除都会记录 Workspace 操作审计。

## 校验规则

新增和编辑时会检查：

- `envType` 必须是支持的标准环境类型。
- `spec` 必须存在。
- 请求中的 `spec` 只能包含当前接口对应的 section。
- 需要完整提交的字段不能缺失。
- CPU、内存和开发模式等字段需要通过 AppSpec 原有校验。

请求内容不合法时返回 `400`，规则不存在时返回 `404`。

## 数据存储

环境类型规则保存在 MongoDB 集合 `workspace_app_spec_rules` 中。

主要字段如下：

```go
type Rule struct {
    ID          bson.ObjectID
    WorkspaceID string
    ConfigType  appspec.AppSpecSectionID
    EnvType     string
    Spec        *appspec.AppSpec
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

MongoDB 使用唯一索引 `(workspaceID, configType, envType)` 防止同一规则位置被重复占用。

删除 Workspace 时，会同时删除该 Workspace 的全部规则。删除单个环境时不需要清理规则，因为规则引用的是环境类型，不引用具体环境。

## 代码位置

| 文件 | 作用 |
|------|------|
| `defaults.go` | 平台默认资源规格和更新策略 |
| `model.go` | Rule 数据结构和 section 写入逻辑 |
| `validate.go` | 规则内容校验 |
| `resolve.go` | 创建应用时解析默认 AppSpec |
| `store.go` | MongoDB 读写和唯一索引 |
| `router.go` | API 路由 |
| `handler/` | 各 section 的规则校验、数据库操作和审计 |
| `serializer/` | 各 section 的请求和响应结构 |
| `hooks/` | 删除 Workspace 时清理规则 |
