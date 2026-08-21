# 权限管理（Permission）模块设计说明

> **当前状态**：PR1 / PR2 / PR3 / PR4 / PR5 均已落地。本文档以「**当前仓库状态**」为视角描述权限相关代码的分层与职责，不是
> changelog。
> **历史背景**：authmanager 微服务已以「蚂蚁搬家」方式完整合并进 bkms-server，仓库内对原 authmanager PB 模块的引用已经归零。

---

## 1. 背景

### 1.1 总体目标

将原独立微服务 `apps/authmanager/`（trpc-go）以"内嵌包"形式合并进 bkms-server，从跨进程 RPC 调用切换为同进程函数调用，并切换为
v2 gin 风格调用模式（无 trpc handler、无 PB 类型外泄）。

### 1.2 三层架构（当前形态）

```
bkms-server/pkg/
├── infras/cloudapi/iam/        # L1: 蓝鲸 IAM 网关 HTTP 客户端（最底层）
├── bkintegrations/bkiam/         # L2: IAM 领域服务（角色 / 范围 / 编排）
│   ├── role/                    #   └─ 角色领域类型 + Mongo 存储
│   ├── scope/                   #   └─ 可授权范围生成器 + JSON 模板
│   ├── actions/                 #   └─ 资源操作枚举
│   ├── dto.go                   #   └─ 纯 Go DTO
│   ├── service.go               #   └─ IAMService 编排层
│   └── migrate/                 #   └─ IAM 模型迁移库（由 cmd 子命令调用）
└── infras/perm/                # L3: 业务侧权限管理器入口
```

业务侧调用链：

```
其他业务代码 (pkg/core/..., pkg/server/...)
    │
    ▼
pkg/infras/perm.NewManager() -> perm.Manager
    │
    ▼
pkg/infras/perm.LocalManager
    │
    ▼
pkg/bkintegrations/bkiam.IAMService
    │
    ├──▶ pkg/bkintegrations/bkiam/role.RoleStoreMongo (复用全局 mongo client)
    ├──▶ pkg/bkintegrations/bkiam/scope
    └──▶ pkg/bkintegrations/bkiam/actions
                │
                ▼
        pkg/infras/cloudapi/iam.IAMClient
                │
                ▼
        蓝鲸 IAM 网关 (HTTP)
```

本仓库内**不再存在** `pkg/infras/authmanager/` 包，业务代码不再依赖任何原 authmanager PB 模块，老 trpc authmanager
链路已经被完全删除。

---

## 2. 当前已落地的层：`pkg/infras/cloudapi/iam`

L1 层：蓝鲸 IAM 网关 HTTP 客户端。本层只负责协议交互，不包含任何业务领域语义。

### 2.1 目录结构

```
pkg/infras/cloudapi/iam/
├── client.go              # IAMClient 接口定义、BKIAMClient 实现、NewIAMClient 工厂
├── grade_manager.go       # GradeManager 6 个方法（Create / Update / Delete / GetByName / Add/DeleteMembers）
├── user_group.go          # UserGroup 6 个方法（Create / Delete / GrantPolicies / Add/Delete/ListMembers）
├── operation_mock.go      # bkapi 的 Operation/BkApiClient mock，供本包及上层包单测复用（非 _test.go，普通 .go）
├── iam_suite_test.go      # ginkgo 测试入口
├── client_test.go         # ginkgo 单测：BKIAMClient 相关 case，描述全英文
└── types/
    ├── base.go            # 通用类型：ResourceType / ResourcePath / Resource / Action / AuthorizationScope / SubjectScope
    ├── req.go             # 请求体：CreateGradeManagerReq / UpdateGradeManagerReq / UserGroupParam / UserMemberParam / AddUserGroupMembersReq
    └── resp.go            # 响应体：Resp / CreateUserGroupsResp / GradeManager(Data) / UserGroup / UserMember(Data)
```

### 2.2 包职责

L1 层只负责 **协议交互**，不包含任何业务领域语义：

- **HTTP 调用**：通过 `bk-apigateway-sdks/core/bkapi` 对接蓝鲸 IAM 网关，处理 path / query / body / 错误码解析
- **鉴权 SDK**：通过嵌入 `iam-go-sdk` 的 `*iamsdk.IAM`，提供 `IsAllowed` / `BatchResourceMultiActionsAllowed`
- **资源类型枚举**：在 `types/base.go` 中维护 BKMS / BKCI / BCS / BKMonitor / BKLog / BKRepo 等业务系统的资源类型常量

### 2.3 接口划分

`IAMClient` 由两个子接口组合：

| 接口                | 方法数 | 职责                                             |
|-------------------|-----|------------------------------------------------|
| `UserRoleManager` | 12  | 分级管理员（6）+ 用户组（6）的全部 CRUD / 成员管理                |
| `Authenticator`   | 2   | `IsAllowed`、`BatchResourceMultiActionsAllowed` |

实现：

| 类型            | 用途       | 编译期检查                                   |
|---------------|----------|-----------------------------------------|
| `BKIAMClient` | 真实的网关客户端 | `var _ IAMClient = (*BKIAMClient)(nil)` |

### 2.4 配置接入

本 PR 在 `pkg/common/config` 中新增以下字段，并在示例配置（`configs/config.yaml`）和测试配置（`tests/configs/config.yaml`
）中提供示例值：

| 字段                    | 说明                                                            |
|-----------------------|---------------------------------------------------------------|
| `BkApiStages.BkIAM`   | IAM 网关 stage（如 `prod` / `test`）                               |
| `BkIAMSystemIDs.Bkms` | bkms-server 自身在权限中心注册的系统 ID（其他系统 ID 字段同时也已加入，便于 PR2 / PR3 复用） |

构造客户端的核心鉴权信息复用既有 `BkApp.Code` / `BkApp.Secret`，网关地址复用既有 `BkPlatUrls.BkApiUrlTmpl`。

### 2.5 内部约定

- **未导出工厂**：`newIAMClient(define.BkApiClient)` 用于单测注入 mock，生产代码统一走 `NewIAMClient()`
- **常量化魔数**：`ConflictCode`、`NeverExpireTimestamp`、`defaultRequestTimeout`、分页参数等全部以 const 形式声明
- **错误处理**：统一使用 `pkg/errors`。下层错误使用 `errors.Wrap(err, errPrefix)`；网关返回业务错误码时使用
  `errors.Errorf("%s: %s", errPrefix, resp.Message)`

### 2.6 依赖边界

- 仅依赖 `pkg/common/config`、蓝鲸 SDK（`bk-apigateway-sdks` / `iam-go-sdk`）、通用工具（`pkg/errors`、`mapstructure`）
- **不引入** bkms-server 其他业务包，避免循环依赖

---

## 3. 当前已落地的层：`pkg/bkintegrations/bkiam/{role,scope,actions}`

L2 层中下半段三个领域子包，负责角色持久化、可授权范围生成、资源操作枚举等领域能力，供 L2 层 `IAMService` 与 migrate 库复用。

### 3.1 目录结构

```
pkg/bkintegrations/bkiam/
├── role/
│   ├── types.go            # 领域类型：ResourceType / WorkspaceResourceType / PermissionScope / Role / WorkspaceGradeManager / RoleQueryParams
│   ├── builtin.go          # BuiltinRoleCode（admin/developer/sre/operator） + WorkspaceScopeBuiltinRoles
│   ├── utils.go            # GenGradeManagerName / GenWorkspaceRoleName 名称生成 helper
│   ├── store.go            # RoleStore 接口（7 个方法） + RoleStoreMongo 实现
│   ├── role_suite_test.go  # ginkgo 入口
│   ├── store_test.go       # ginkgo 同包单测（英文描述，覆盖 7 个方法 + ListRoles 各过滤组合）
├── scope/
│   ├── generator.go        # AuthScopesGenerator 接口 + GenerateAuthScopes 聚合 + generateFromTemplate
│   ├── {bcs,bkci,bklog,bkmonitor,bkms,bkrepo,bscp}.go  # 7 个业务系统的 RoleScopesGenerator 实现
│   ├── scope_suite_test.go # ginkgo 入口（注入确定性 IAMSystemIDs 用于断言）
│   ├── {bcs,bkci,bklog,bkmonitor,bkms,bkrepo}_test.go     # 6 个剪枝单测（admin + 1 个非 admin 角色）
│   └── template/
│       ├── embed.go        # //go:embed *.json + */*.json 嵌入模板 FS
│       ├── tpl.go          # GetRoleScopeTemplatePath helper，引用 role.BuiltinRoleCode
│       ├── anonymous.json  # 未知角色码的默认空范围
│       └── {bcs,bkci,bklog,bkmonitor,bkms,bkrepo,bscp}/{admin,developer,sre,operator}.json  # 7 * 4 = 28 个模板
└── actions/
    └── actions.go          # BKMS / BKCI / BCS / BKMonitor / BKLog / BKRepo 全部 action ID 常量
```

### 3.2 子包职责

| 子包                | 职责                                | 外部依赖                                                                                    |
|-------------------|-----------------------------------|-----------------------------------------------------------------------------------------|
| `role/`           | 角色 / 分级管理员领域模型 + Mongo 存储         | 标准库 + `mongo-driver/v2` + `pkg/errors`                                                  |
| `scope/`          | 可授权范围生成（1 接口 + 7 个业务系统 generator） | `cloudapi/iam/types` + `bkintegrations/bkiam/role` + `common/config` + `scope/template` |
| `scope/template/` | 模板资源与路径 helper                    | `bkintegrations/bkiam/role`（仅取 `BuiltinRoleCode`）+ 标准库                                  |
| `actions/`        | 资源操作 ID 常量枚举                      | 标准库（零依赖）                                                                                |

依赖边界为**单向**：`scope` 只能引用 `role` 与 `scope/template`；`role` 不引用 `scope`/`actions`/`cloudapi/iam`；`actions`
不引用任何其他项目包。

### 3.3 Mongo 集合与索引

为了与线上数据兼容，**集合名与原 authmanager 完全一致**：

| 集合                  | 索引                                 | 类型    | 用途                                                          |
|---------------------|------------------------------------|-------|-------------------------------------------------------------|
| `iam_grade_manager` | `{workspaceID: 1}`                 | 唯一    | 保证每个 workspace 仅一个分级管理员；Get/Delete WorkspaceGradeManager 加速 |
| `iam_role`          | `{id: 1}`                          | 唯一    | 角色 ID 业务唯一；GetRoleByID 加速                                   |
| `iam_role`          | `{workspaceID: 1, userGroupID: 1}` | 复合非唯一 | DeleteRolesByUserGroupIDs 与 ListRoles 按 workspace 过滤加速      |

索引在 `NewRoleStoreMongo` 里通过 `Indexes().CreateMany` 创建，已存在则由 mongo driver 自动跳过，对已上线数据库无破坏性。

### 3.4 内部约定

- `BuiltinRoleCode` 上提到 `role/builtin.go`（原仅在 `scope/template/tpl.go`），语义归属角色子包。
- `scope` 不重复定义 `AuthorizationScope`，统一复用 `pkg/infras/cloudapi/iam/types`。
- generator 清一色从 `config.G.BkIAMSystemIDs.{BCS,BkCI,BkMonitor,BkLog,BkRepo,BSCP,Bkms,BkCC}` 读 SystemID（与原
  authmanager 直读全局 config 行为对等）。PR3 重构是否改为参数注入到时再说。
- 错误处理统一 `pkg/errors.Wrap` / `Wrapf`，不使用 `errorx`、不使用 `fmt.Errorf` 包错。
- 接口实现均带编译期检查（`var _ Iface = (*Impl)(nil)`）。

---

## 4. 当前已落地的层：`pkg/bkintegrations/bkiam`（编排 + DTO + migrate）

在 L2 三个领域子包之上，`pkg/bkintegrations/bkiam` 提供 `IAMService` 编排层、纯 Go DTO 与 `migrate` 库子包。业务侧
`pkg/infras/perm` 入口包直接调用 `IAMService` 完成鉴权与角色生命周期管理。

### 4.1 目录结构

```
pkg/bkintegrations/bkiam/
├── dto.go                 # 纯 Go DTO（WorkspaceData / 各业务系统 Options / BSCPService）
│                          # Role / PermissionScope 通过 type alias 复用 role 子包，避免重复
├── service.go             # IAMService 编排：构造函数 NewIAMService(client, store)
├── iam_suite_test.go      # ginkgo 测试入口
├── service_test.go        # ginkgo 单测：13 个核心 case，复用同包 fakeIAMClient + fakeRoleStore
└── migrate/
    ├── driver.go          # IAMDriver：基于 iam-go-sdk/iammigrate 把模型推送到 IAM
    ├── migrate.go         # Migrate(MongoConfig, Config) 入口（仅落库，PR5 才接 cobra）
    └── migrations/
        ├── embed.go       # //go:embed *.json 嵌入迁移模板
        └── 0000_bkms.up.json / 0001_workspace.up.json /
            0002_app.up.json / 0003_env.up.json
```

### 4.2 IAMService 职责

`IAMService` 是 L2 层的**编排器**（orchestrator），把以下三件事拼起来：

1. **网关调用**：通过 PR1 的 `cloudapi/iam.IAMClient` 操作分级管理员 / 用户组 / 鉴权
2. **持久化**：通过 PR2 的 `role.RoleStore` 读写 `iam_grade_manager` / `iam_role` 集合
3. **范围生成**：通过 PR2 的 `scope.AuthScopesGenerator` 把 WorkspaceData 转成 IAM 授权范围

方法分组（共 18 个公开方法）：

| 分组       | 方法                                                                                                                                                        |
|----------|-----------------------------------------------------------------------------------------------------------------------------------------------------------|
| 工作空间生命周期 | `CreateWorkspaceAdmin` / `UpdateWorkspaceAdmin` / `CreateWorkspaceScopeBuiltinRoles` / `UpdateWorkspaceScopeBuiltinRoles` / `DeleteAllRolesByWorkspaceID` |
| 角色 / 成员  | `ListRoles` / `ListRoleMembers` / `AddRoleForUsers` / `DeleteRoleForUsers`                                                                                |
| 鉴权（单资源）  | `WorkspaceCreateIsAllowed` / `WorkspaceActionIsAllowed` / `AppCreateIsAllowed` / `AppActionIsAllowed` / `EnvCreateIsAllowed` / `EnvActionIsAllowed`       |
| 鉴权（批量）   | `WorkspacesMultiActionsAllowed` / `AppsMultiActionsAllowed` / `EnvsMultiActionsAllowed`                                                                   |

构造函数严格遵循「先依赖后数据」：

```go
func NewIAMService(client cloudapiiam.IAMClient, store role.RoleStore) *IAMService
```

### 4.3 纯 Go DTO

本包对外暴露的请求 / 响应类型全部是 `dto.go` 中定义的纯 Go 结构体，不依赖任何 PB 模块。字段定义以代码为准，此处只记录结构约定。

- `WorkspaceData`：创建 / 更新 workspace 分级管理员与内置角色所需的全部信息。除 `WorkspaceID` / `WorkspaceName` 外，
  `BKCI` / `BCS` / `BKMonitor` / `BKLog` / `BKRepo` / `BSCP` 六个业务系统字段均为可选指针，为 nil 时跳过对应的可授权范围生成器
- 各业务系统 Options 结构对称：BKCI / BCS / BKRepo 为 `{ProjectID, ProjectName}`，BKMonitor / BKLog 为
  `{SpaceID, SpaceName}`，BSCP 为 `{BizID, BizName, Services}`，其中 `Services` 是 `[]BSCPService{ID, Name}`

**构造点**：`WorkspaceData` 当前有 5 处构造，新增字段时以这几处为对照——`pkg/infras/perm/local.go` 的
`CreateWorkspaceAdmin` / `CreateWorkspaceScopeBuiltinRoles`（由标量入参组装，只填 BKCI / BCS / BKRepo）、
`pkg/server/taskqtask/workspace/handler.go`、`pkg/extension/bscpcfg/service/permission.go`，以及迁移命令
`cmd/migration/refresh_workspace_bkmonitor_perms.go`。

### 4.4 migrate 子包

migrate 子包负责把 IAM 系统模型（resource_type / instance_selection / action）注册到蓝鲸权限中心。

- **入口**：`Migrate(mongoCfg MongoConfig, iamCfg Config) error`，由 `cmd/migration/migrate_iam_system_model.go` 子命令调用（详见
  §6）
- **网关入口**：使用 `iam-go-sdk` 提供的 `IAMBackendClient`（IAM 后台模型管理端口），与 `cloudapi/iam.IAMClient`（鉴权 /
  用户组端口）是两个不同的 IAM 接口面，两者并存而非替代
- **配置传入**：通过 `Config{BkApiGatewayURL, BkmsSystemID, AppCode, AppSecret, BkmsHost}` 显式传参，不读全局 `config.G`
  ，保持库的自含与可复用性
- **JSON 模板**：4 个迁移文件 `0000_bkms` / `0001_workspace` / `0002_app` / `0003_env`

### 4.5 内部约定

- **接口落点**：`IAMService` 是结构体类型，不暴露接口；其依赖 `cloudapiiam.IAMClient`（PR1）与 `role.RoleStore`
  （PR2），两者均带编译期检查
- **Role / PermissionScope**：通过 `type alias` 复用 `role` 子包定义，避免在 dto.go 中重复声明同名类型；同时让 `service.go`
  直接接收 `*role.Role` 既可读又能与 store 层无缝互通
- **错误处理**：全部使用 `pkg/errors.Wrap` / `Wrapf`，禁用 `fmt.Errorf` 包错；测试代码亦同
- **常量化魔数**：IAM 资源类型字符串（`workspace` / `app` / `env`）、subject 类型（`user`）、`_bk_iam_path_` 属性键、
  `/workspace,%s/` 父路径模板均提取为 const
- **空接口**：统一使用 `any`（如 `_bk_iam_path_` 的属性值 map）
- **零循环依赖**：本包仅依赖 `pkg/infras/cloudapi/iam{,/types}`、`pkg/bkintegrations/bkiam/{role,scope,actions}`、
  `pkg/common/config` 与第三方库；**不**依赖 `pkg/server/*`、`pkg/core/*`、`pkg/infras/perm`

### 4.6 单测组织

- `iam_suite_test.go`：ginkgo suite 入口（`package iam`，同包以便访问私有结构）
- `service_test.go`：13 个核心 case，描述全英文，覆盖：
    - workspace admin 生命周期：创建、幂等返回、网关错误传播、空用户组返回防御、更新
    - 内置角色：missing-only 创建 + 重复调用幂等
    - 角色 / 成员：列角色（按 scope 过滤）、列成员、加成员、删成员（GM + UG 两层）
    - 鉴权：单资源 IsAllowed、批量 MultiActionsAllowed
    - workspace 拆除：DeleteAllRolesByWorkspaceID 的级联删除
- mock 策略：手写 `fakeIAMClient` + `fakeRoleStore` 两个同包 fake，记录调用参数 + 返回预设值；不依赖 mockey、不依赖远程
  MongoDB
- migrate 子包：按剪枝原则未补充单测（搬迁后逻辑等价于 authmanager 既有测试覆盖；PR5 接入 cmd 时再视情况补充）

---

## 5. 当前已落地的层：`pkg/infras/perm`（业务侧入口）

业务侧权限管理器入口。`pkg/infras/perm` 是所有 bkms-server 业务代码访问权限管理能力的唯一入口包。

### 5.1 目录结构

```
pkg/infras/perm/
├── perm.go              # Manager 接口（共 23 个方法）+ NewManager() 工厂（仅 UseStubPerm 一段分支 + 单例缓存）
├── local.go             # LocalManager：通过 iamServicer 接口转发到 bkiam.IAMService
├── stub.go              # StubAllowAnyManager：本地开发桩，全部放行 + 4 个固定 UUID 角色
├── action.go            # WorkspaceAction / AppAction / EnvAction 常量
├── roles.go             # RoleCodeAdmin / Sre / Developer / Operator + IsValidRoleCode / RoleCodes
├── types.go             # ResourceType 枚举：WorkspaceResourceType / AppResourceType / EnvResourceType
├── perm_suite_test.go   # ginkgo suite 入口（同包 package perm）
├── local_test.go        # LocalManager 7 个核心 case + 同包 fakeIAMServicer
└── stub_test.go         # StubAllowAnyManager 4 个核心 case
```

### 5.2 包职责

| 类型                    | 职责                                                                                                                                                                                                  |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `Manager` 接口          | 业务侧权限管理器入口（v2 风格），返回值与 DTO 型入参均取自 `pkg/bkintegrations/bkiam`（`*role.Role` / `bkiam.WorkspaceData`）；`Update*` 系列直接收 DTO，`Create*` 系列仍收标量入参（workspaceID / bkCIProjectID 等），由 `LocalManager` 内部组装成 DTO |
| `LocalManager`        | `Manager` 的进程内本地实现：内部从 ctx 通过 `auth.GetUser` 取 user，再调用 `bkiam.IAMService.*IsAllowed` / `*ActionIsAllowed` 等                                                                                        |
| `StubAllowAnyManager` | 本地开发与测试桩；全部 `Has*Perm` 返回 `nil`、`FilterViewable*` 返回入参全集、`ListRoles` 返回 4 个固定 UUID 角色                                                                                                               |
| `iamServicer`（包内私有接口） | 收敛 `LocalManager` 真正用到的 `bkiam.IAMService` 方法集合，便于阅读与单测替换；编译期校验 `var _ iamServicer = (*bkiam.IAMService)(nil)`                                                                                      |

### 5.3 NewManager() 实现

`perm.NewManager()` 使用懒加载单例，首次调用时完成底层依赖装配，后续调用复用同一个 `Manager` 实例，避免在请求路径重复创建
IAM client、Mongo 角色存储和 IAMService。

```
perm.NewManager()
  ├── managerOnce.Do(buildManager)
  └── return cached Manager

buildManager()
  ├── if config.G.Development.UseStubPerm == true:
  │       return &StubAllowAnyManager{}            # 本地开发，禁止生产环境
  └── else:
          cli   <- cloudapi/iam.NewIAMClient()     # 蓝鲸 IAM 网关 client
          store <- role.NewRoleStoreMongo(database.Client(), database.Name())  # Mongo 角色存储
          svc   <- bkiam.NewIAMService(cli, store) # IAMService 编排
          return &LocalManager{svc: svc}
```

失败时通过 `log.Fatalf` 终止启动，与项目其它基础设施（database / redis 等）初始化策略保持一致。

### 5.4 唯一开关 `Development.UseStubPerm`

`pkg/common/config/types.go` 的 `Development` 子结构中只保留一个权限链路开关：

| 开关                                 | 默认值     | 含义                                                                                 |
|------------------------------------|---------|------------------------------------------------------------------------------------|
| `config.G.Development.UseStubPerm` | `false` | `true` 时 `perm.NewManager()` 返回 `StubAllowAnyManager`（全部放行），仅本地开发使用，**禁止在生产环境中打开** |

配置示例（`configs/config.yaml`）：

```yaml
development:
  useStubPerm: false
```

### 5.5 LocalManager 关键转发关系

```
LocalManager.HasCreateAppPerm(ctx, wsID)
   └── auth.GetUser(ctx) -> user.ID
        └── IAMService.AppCreateIsAllowed(user.ID, wsID) -> (bool, error)
             └── 转换：bool=false -> errors.New("no permission to create app")

LocalManager.FilterViewableWorkspaces(ctx, ids)
   └── auth.GetUser(ctx) -> user.ID
        └── IAMService.WorkspacesMultiActionsAllowed(user.ID, ids, [WorkspaceAction.View])
             └── map[wsID]map[action]bool -> 聚合为 *set.StringSet

LocalManager.CreateWorkspaceAdmin(ctx, wsID, displayName, users, bkCIID, bcsID, bkRepoID)
   └── 拆分参数包装为 bkiam.WorkspaceData
        └── IAMService.CreateWorkspaceAdmin(ctx, data, users) -> *role.Role（结果丢弃，仅返回 error）

LocalManager.GetRole(ctx, wsID, roleCode)
   └── ListRoles(ctx, wsID) -> 遍历匹配 RoleCode -> 找不到时返回 "role not found"
```

**字段映射 quirk 警告**：`CreateWorkspaceAdmin` / `CreateWorkspaceScopeBuiltinRoles` 的 `BCS.ProjectName` 取
`bkCIProjectID`（**不是** `bcsProjectID`）。这是从老 `SvcBasedAuthManager` 完整迁移过来的历史习惯，保留以兼容线上 IAM 数据。

### 5.6 错误信息兼容性

`LocalManager` 在权限被拒绝时返回的 error 文本与历史调用方约定保持一致：

| 场景             | 错误文本                                                 |
|----------------|------------------------------------------------------|
| 创建应用拒绝         | `no permission to create app`                        |
| 创建工作空间拒绝       | `no permission to create workspace`                  |
| 创建环境拒绝         | `no permission to create env`                        |
| 应用 action 拒绝   | `no permission to %s application %s in workspace %s` |
| 工作空间 action 拒绝 | `no permission to %s workspace %s`                   |
| 环境 action 拒绝   | `no permission to %s env %s in workspace %s`         |

### 5.7 单测组织

- `perm_suite_test.go`：ginkgo suite 入口（同包 `package perm`）
- `local_test.go`：覆盖 `HasCreateWorkspacePerm`（含 ctx 取 user）/ `HasViewAppPerm` / `HasDeployEnvPerm` /
  `FilterViewableWorkspaces` / `CreateWorkspaceAdmin`（含字段映射 quirk 验证）/ `ListRoles` / `GetRole` /
  `ListRoleMembers`，使用同包 `fakeIAMServicer` 替换底层依赖
- `stub_test.go`：4 个 case 覆盖 `StubAllowAnyManager` 全部放行、`FilterViewable*` 入参回传、4 个固定 UUID 角色返回、写方法
  no-op

### 5.8 内部约定

- **纯 Go DTO**：所有结构体字段使用 `pkg/bkintegrations/bkiam` 的纯 Go DTO（`*role.Role` / `bkiam.WorkspaceData` /
  `role.PermissionScope`）
- **零反向依赖**：`pkg/infras/perm/` 不 import `pkg/server/*`、`pkg/core/*`
- **错误处理**：全部使用 `pkg/errors.Wrap` / `Wrapf`，禁用 `fmt.Errorf` 包错
- **常量化**：错误信息模板、stub 角色 UUID、stub user-group ID 全部提取为 `const`
- **空接口**：统一使用 `any`
- **接口编译期检查**：`var _ Manager = (*LocalManager)(nil)`、`var _ Manager = (*StubAllowAnyManager)(nil)`、
  `var _ iamServicer = (*bkiam.IAMService)(nil)`

---

## 6. IAM 模型迁移命令

bkms-server 二进制提供两个 cobra 子命令用于 IAM 相关迁移：

### 6.1 `migrate_iam_system_model`

把 bkms 系统模型（resource_type / instance_selection / action）一次性注册到蓝鲸权限中心。

```
bkms-server migrate_iam_system_model \
  --srvCfg=/path/to/biz_cfg.yaml \
  --bkms-host=https://bkms-server.example.com
```

- `--srvCfg`（必填）：bkms-server 配置文件路径
- `--bkms-host`（必填）：IAM 回调 bkms-server 的外部可达 URL，会渲染到系统模型的 `provider_config.host`
- 实现：基于 `pkg/bkintegrations/bkiam/migrate.Migrate(mongoCfg, iamCfg)`，迁移历史记录在 `iam_schema_migrations` 集合（与
  bkms 业务集合无关）

### 6.2 `migrate_iam_data`

按业务唯一键 upsert，把源端的 `iam_grade_manager` / `iam_role` 集合数据迁移到 bkms-server Mongo 库。

```
bkms-server migrate_iam_data \
  --srvCfg=/path/to/biz_cfg.yaml \
  [--source-uri=mongodb://user:pass@host:port] \
  [--source-db=authmanager_legacy] \
  [--dry-run]
```

- `--srvCfg`（必填）：bkms-server 配置文件路径
- `--source-uri`（可选）：源 Mongo 连接 URI；缺省时复用 bkms-server 全局 mongo client
- `--source-db`（可选）：源 Mongo 数据库名；当 `--source-uri` 非空时必填
- `--dry-run`（默认 `false`）：仅打印计划迁移的文档数量，不写入目标库
- 业务唯一键：
    - `iam_grade_manager` 用 `workspaceID`
    - `iam_role` 用 `id`

---

## 7. 调用链当前状态

业务侧默认调用链：

```
其他业务代码
   │  perm.Manager 接口
   ▼
pkg/infras/perm.NewManager()
   │  Development.UseStubPerm=false（默认）
   ▼
pkg/infras/perm.LocalManager
   ▼
pkg/bkintegrations/bkiam.IAMService
   │
   ├──▶ pkg/bkintegrations/bkiam/role.RoleStoreMongo (复用全局 mongo client)
   ├──▶ pkg/bkintegrations/bkiam/scope
   └──▶ pkg/bkintegrations/bkiam/actions
            │
            ▼
   pkg/infras/cloudapi/iam.IAMClient
            │
            ▼
   蓝鲸 IAM 网关 (HTTP)
```

开启 `Development.UseStubPerm=true` 后，`perm.NewManager()` 直接返回 `StubAllowAnyManager`，所有鉴权请求放行；仅用于本地开发。

---

## 8. PR 进度总览

| #   | 分支                 | 主题                                                                   | 状态    |
|-----|--------------------|----------------------------------------------------------------------|-------|
| PR1 | `merge-auth-mgr-1` | 底层 IAM 蓝鲸网关 Client → `pkg/infras/cloudapi/iam`                       | ✅ 已完成 |
| PR2 | `merge-auth-mgr-2` | IAM 角色 / 范围 / 操作动作 → `pkg/bkintegrations/bkiam/{role,scope,actions}` | ✅ 已完成 |
| PR3 | `merge-auth-mgr-3` | IAMService 编排 + 纯 Go DTO → `pkg/bkintegrations/bkiam`                | ✅ 已完成 |
| PR4 | `merge-auth-mgr-4` | 业务侧入口 `pkg/infras/perm` + 灰度开关 + PB ↔ DTO 桥接                         | ✅ 已完成 |
| PR5 | `merge-auth-mgr-5` | 业务侧改名 + 删除老 `pkg/infras/authmanager/` + 接入 migrate 子命令               | ✅ 已完成 |
