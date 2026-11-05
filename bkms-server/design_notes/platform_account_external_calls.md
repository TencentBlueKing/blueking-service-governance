# 平台级虚拟账号与外部系统调用

本文说明 BKMS 调用外部系统时使用的身份，以及引入平台级虚拟账号（下称"虚拟账号"）后每个调用点的处理方式。

- 代码状态以本分支为准；标注"第二阶段"的改动在后续的检查改造分支中完成。
- 外部系统的行为结论来自阅读 TencentBlueKing 组织下的开源仓库，未在真实环境逐项验证，见第九节。

---

## 一、背景

### 1.1 BKMS 现在怎么调用外部系统

BKMS 代用户调用蓝盾、BSCP、BCS 等外部系统时，绝大多数请求带的是**当前登录用户本人的凭证**（`bk_ticket` 或 `access_token`），外部系统按这个用户的权限鉴权。BKMS 的应用凭证只用于通过网关的应用认证。

用户之所以在外部系统有权限，是因为 BKMS 在权限中心替空间成员授了权：

1. 创建空间时，BKMS 以自身系统的身份调用权限中心管理接口，为空间创建分级管理员，授权范围包含空间绑定的蓝盾项目、BCS 项目、监控空间等资源；
2. 在分级管理员下为管理员、开发者、SRE、运营者各建一个用户组，按 `pkg/bkintegrations/bkiam/scope/template/<系统>/<角色>.json` 授予各外部系统的权限；
3. 把成员加入 BKMS 的内置角色，就是把他加入对应的用户组，从而获得这些外部权限；
4. 外部系统鉴权时向权限中心查询该用户的策略，来自 BKMS 用户组的策略同样生效。

也就是"BKMS 给人授权 → 以人的身份调用 → 外部系统按人鉴权"，BKMS 自身不持有外部系统的权限。

### 1.2 为什么需要虚拟账号

环境级权限会引入自定义角色。自定义角色按设计**不授予外部系统的权限**：成员不能绕过 BKMS 直接到外部系统上操作，需要的人自行去外部系统申请。

这样一来，自定义角色成员通过 BKMS 构建、读取配置时，如果仍以本人身份调用外部系统，会因为没有外部权限而失败。

方案是：**BKMS 先按自己的规则鉴权，通过后统一以一个平台级虚拟账号调用外部系统**。所有人都一样，不区分内置角色和自定义角色。

---

## 二、虚拟账号

### 2.1 账号与配置

- 管理员在用户管理后台创建一个虚拟账号，并为它签发 `access_token`。虚拟账号不能登录，只用这个 token 调用外部系统。
- 整个平台只用一个虚拟账号。
- 服务配置：

```yaml
platformAccount:
  enabled: false
  username: ""
  # 虚拟账号的 access_token，属于敏感信息
  accessToken: ""
```

- `enabled` 为 `true` 时，`username`、`accessToken` 必填（`validate:"required_if=Enabled true"`）。
- 未启用时，所有调用仍使用请求用户本人，与引入前完全一致。
- `accessToken` 不能打印到日志，也不能返回给前端。

### 2.2 代码接口

`pkg/infras/account/platformaccount`：

| 函数 | 返回 | 用途 |
| --- | --- | --- |
| `Get() (auth.User, bool)` | 虚拟账号本身；未启用时 `ok` 为 `false` | 需要知道"虚拟账号是谁"的场景：加入角色、成员列表中隐藏、迁移命令 |
| `GetCaller(ctx) (auth.User, error)` | 启用时为虚拟账号，未启用时为 `auth.GetUser(ctx)` | 调用外部系统，可能没有登录用户的场景（后台任务、可降级的查询） |
| `MustGetCaller(ctx) auth.User` | 启用时为虚拟账号，未启用时为 `auth.MustGetUser(ctx)` | 调用外部系统的请求处理路径 |

- 启用时返回的虚拟账号不依赖 ctx 中的用户，也不带租户 ID。
- `MustGetCaller` 未启用时直接调用 `auth.MustGetUser`，与原来的调用完全等价。
- **调用身份只在后端决定**，不接受前端传入。

### 2.3 授权

| 时机 | 做什么 | 位置 |
| --- | --- | --- |
| 创建空间 | 把虚拟账号加入空间的管理员角色 | `core/workspace/dependency.go` `AddPlatformAccount`，在 `InitWorkspaceUser` 最后调用 |
| 存量空间 | 同上，补加 | `cmd/migration/grant_platform_account.go` |

- 管理员角色同时是空间在权限中心的分级管理员，所以虚拟账号也会成为分级管理员成员。选择管理员角色，是因为管理员的外部权限模板覆盖开发者的全部权限，并为以后可能用到的权限留有余量。
- 当前实际用到的外部权限：蓝盾流水线的创建、编辑、执行、查看和代码库查看（构建）；BCS 查看（事件、集群名称）。
- BSCP 不使用虚拟账号：配置文件引用的服务以本人身份读取，配置管理扩展在部署时不调用 BSCP 接口，见第 4.2 节。

### 2.4 成员列表中隐藏

虚拟账号不是真实成员。`perm.LocalManager.ListRoleMembers` 在启用时统一去掉虚拟账号，所有依赖它的地方都不会出现虚拟账号，包括成员列表、同步到监控告警组的人员。

由此带来的行为：

- 通过空间成员接口移除虚拟账号时，因为它不在任何角色的成员列表中，请求直接成功但不做任何事；通过平台管理回收管理员时返回"不是管理员"。虚拟账号不会被误删。

### 2.5 上线步骤

1. 创建虚拟账号，签发 `access_token`。
2. 准备一份开启了 `platformAccount` 的配置，执行 `grant_platform_account --srvCfg=<配置> [--dry-run] [--workspace-id=a,b]`，为存量空间补发权限。命令处理所有状态的空间，可重复执行；失败的空间会在日志里给出重试命令。
3. 服务配置开启 `platformAccount` 并发布。之后新建的空间会自动加入。

第 3 步必须在第 2 步之后：虚拟账号还没加入管理员角色时就改用它调用，构建等功能会失败。

---

## 三、调用身份的类型

| 身份 | 涉及系统 | 外部系统是否检查调用者的权限 |
| --- | --- | --- |
| 本人凭证 | 蓝盾、BSCP、CMDB、HCM、集群资源接口（事件）、BCS 用户 token 接口 | 检查，按调用者本人的权限 |
| 应用凭证 + 请求头中的用户名 | BCS 非用户模式（`X-Bcs-Username`）、监控（`X-Bkapi-User-Name`）、制品库 | 用户名由应用传入，网关不校验其真实性；是否按它鉴权因接口而异：BCS 项目管理接口按它鉴权，BCS 终端只检查应用白名单，监控以它作为操作人 |
| 只用应用凭证 | kubeinsight、DBM、`txcmdb`、`tof`、用户管理、权限中心 | 不涉及用户 |
| 平台集群凭证、库中保存的凭证 | K8s 集群操作（部署、实例、日志、终端以外的集群访问）、镜像仓库、Helm 仓库、北极星 | 不涉及用户 |

虚拟账号主要改变第一类。第二类中会按用户名鉴权的接口（如部署总览查询 BCS 集群名称），同样改为传入虚拟账号的用户名；不按用户名鉴权的保持原样。

### 3.1 一个调用点该用谁的身份

按顺序判断：

1. **外部系统要求本人授权，或者结果要交给本人使用**：必须用本人。例如 Git OAuth，交给 CLI 使用的 BCS token。
2. **外部资源由前端传入**（业务 ID、项目 ID 等），BKMS 无法判断用户能否访问：必须用本人。否则任何人都能借虚拟账号读到其他空间的数据。
3. **创建空间**：用本人。虚拟账号此时还没加入空间，绑定的项目、业务也应当是创建人自己的。
4. **BKMS 已经鉴权，且外部资源由后端确定**（从空间、环境、部署记录中取）：用虚拟账号。

虚拟账号在所有空间都有外部权限，第 2 条是安全底线：**先在 BKMS 按应用、环境鉴权，再用虚拟账号调用；外部资源的标识必须来自后端数据**。

---

## 四、调用点清单

"状态"一列：

- **已改**：本分支已改为虚拟账号；
- **第二阶段**：在检查改造分支中改为虚拟账号；
- **保持**：按第 3.1 节必须用本人，或不受影响；
- **待处理**：见第七节。

路径均相对于 `bkms-server/pkg/`。"触发场景"按调用链追到的入口填写：接口写对应的页面操作，后台任务写任务名。标注"前端未调用"的接口，本仓库的前端和 CLI 目前都没有请求。

### 4.1 蓝盾

| 调用点 | 位置 | 触发场景 | 身份 | 状态 |
| --- | --- | --- | --- | --- |
| 创建或绑定蓝盾项目 | `bkintegrations/bkci/manager.go` `createProject`、`createBKCIProject` | 创建空间 | 本人 | 保持 |
| 创建空间时的初始流水线 | `core/workspace/dependency.go` `EnsureBkSystems` 调用 `PipelineManager.Initialize`，未调用 `WithCaller` | 创建空间 | 本人 | 保持：虚拟账号此时还没入组 |
| 内置流水线缺失时创建、模板版本落后时升级 | `bkintegrations/bkci/manager.go` `PipelineManager`（`WithCaller` / `callerOf`），由 `build/image/build.go`、`build/chart/build.go` 调用 | 镜像构建（手动构建、构建并部署）、创建 Chart 版本 | 虚拟账号 | 已改 |
| 登记自定义流水线 | `build/image/build.go` `ExecuteBKCIPipelineBuild`（`SourceTypePipeline` 分支，本地没有记录时调用） | 用自定义流水线构建 | 本人 | 保持，见第六节 |
| 触发镜像构建 | `build/image/build.go` `ExecuteBKCIPipelineBuild` | 手动构建；构建并部署（tRPC、TAF、bkmsapp） | 内置流水线用虚拟账号，自定义流水线用本人 | 已改 |
| 触发 Chart 构建 | `build/chart/build.go` `execute` | 创建 Chart 版本 | 虚拟账号 | 已改 |
| 关联代码库 | `bkintegrations/bkci/manager.go` `createRepository`（按空间加别名，本地没有记录时才调用），由 `bkci.EnsureWorkspaceRepositories`、`build/image/build.go`、`build/chart/build.go` 调用 | 创建应用（带构建配置）、修改构建配置、修改 Helm 配置；镜像构建、创建 Chart 版本时兜底 | 本人 | 保持，见第六节 |
| 构建状态轮询 | `server/taskqtask/buildpoll/poll.go`、`server/taskqtask/chartbuildpoll/poll.go` `runTick` | 后台任务：镜像构建、Chart 构建提交后轮询 | 虚拟账号；未启用时为触发者，见第五节 | 已改 |
| 镜像构建日志 | `build/build/handler/build_log.go` `prepareBuildLogDeps` | 查看构建日志（检查查看应用） | 虚拟账号 | 已改 |
| Chart 构建日志 | `build/chart/handler/build_log.go` `prepareBuildLogDeps` | 同上 | 虚拟账号 | 已改 |
| 构建触发专用流水线 | `bkintegrations/bkci/trigger_pipeline.go` `Ensure`、`Cleanup` | 目前没有调用方，为构建触发预留 | 虚拟账号 | 已改 |
| 代码库分支、标签 | `bkintegrations/handler/bkci.go` `ListBkCIRepositoryBranches`、`ListBkCIRepositoryTags` | 选择代码分支或标签：构建、快速部署、实例全量更新、创建 Chart 版本 | 本人 | 第二阶段：同时补上"查看空间"检查。原接口不检查任何权限，改用虚拟账号前必须先补 |
| OAuth 授权地址 | `bkintegrations/handler/bkci.go` `GetBkCIOAuthUrl` | 选择代码库时检查 Git OAuth 授权 | 本人 | 保持：OAuth 只能是本人 |
| OAuth 代码库列表 | `bkintegrations/handler/bkci.go` `ListBkCIOAuthGitProjects` | 前端未调用，前端直接请求蓝盾，见第 4.5 节 | 本人 | 保持：OAuth 只能是本人 |
| 流水线详情、变量 | `bkintegrations/handler/bkci.go` `GetBkCIPipeline`、`GetBkCIPipelineVariables` | 配置自定义流水线构建、执行自定义流水线 | 本人 | 保持：与自定义流水线由本人执行一致 |
| 流水线列表 | `bkintegrations/handler/bkci.go` `ListBkCIPipelines` | 前端未调用 | 本人 | 保持：同上 |
| 容器项目对应的蓝盾项目 | `bkintegrations/handler/bcs.go` `ListBCSAuthorizedProjects` | 创建空间、创建或编辑环境时选择容器项目 | 本人 | 保持 |
| 创建镜像仓库凭证 | `core/workspace/dependency.go` `CreateImageCredential` | 创建空间时填写外部镜像仓库；平台管理修改空间镜像仓库 | 本人 | 保持 |
| 删除镜像仓库凭证 | `platmgt/workspace/imageregistry/manager.go` `deleteCredential` | 平台管理修改空间镜像仓库失败时回滚 | 本人 | 保持：仅平台管理员 |
| 初始化制品库镜像仓库凭证 | `bkintegrations/bkrepo/manager.go` `initImageRegistryCredentials` | 创建空间 | 本人 | 保持 |
| Helm 仓库凭证兜底创建 | `workload/helmcore/credential/credential.go` `EnsureCredential`（本地没有凭证记录时才调用） | 创建空间；创建 Chart 版本；helm-git 应用的部署、部署预览，以及查看 Chart 列表、文件、版本、values | 本人 | 待处理 |

### 4.2 BSCP

BKMS 中 BSCP 有两种用法，身份不同：

- **配置文件引用 BSCP 服务**：配置文件的内容来源选择 BSCP，引用的服务由用户自己带入。凡是要算出配置文件内容的地方，BKMS 都以本人身份读取，需要本人有该服务的查看权限。
- **配置管理扩展**：BKMS 在空间绑定的 BSCP 项目下为应用创建服务并绑定环境。部署时 BKMS 不调用 BSCP 接口，只把本地的绑定快照和空间的 BSCP 凭证注入到 Pod（`workload/bscpcfg/builder.go` `BuildFromStore`），由容器内的 init 容器和 sidecar 用这个凭证拉取配置，与部署人的权限无关。开启配置管理、绑定环境时以本人身份调用 BSCP。

| 调用点 | 位置 | 触发场景 | 身份 | 状态 |
| --- | --- | --- | --- | --- |
| 读取配置内容：列服务版本、取配置 | `core/app/appcfg/models.go` `BSCPConfig.FetchContent`，经 `appcfg` 编辑器的 `GetCompiledContent` 调用 | 查看配置文件详情；修改配置文件内容或覆盖内容、预览覆盖合并；部署预检、部署（tRPC、TAF、bkmsapp、Helm）、Helm 部署预览；查询 APM 服务名；查看、校验北极星配置（检查服务名）；后台任务：构建后自动部署（身份为构建触发者），部署成功后同步 APM 服务配置（身份为部署人） | 本人 | 保持：引用的服务由用户自己带入，平台不代为授权 |
| 读取配置内容：下载文件 | `infras/cloudapi/bscp/types.go` `File.Content` | 同上，文件型配置 | 与获取该文件的客户端相同 | 已改：`File` 保存获取它的客户端，下载沿用其身份，不再在内部从 ctx 取用户 |
| 业务、服务、配置的列表和详情 | `bkintegrations/handler/bscp.go` | 前端未调用（供选择 BSCP 配置使用） | 本人 | 保持：业务 ID 由前端传入 |
| 保存时校验配置 | `core/app/appcfg/handler/app_config_file_validation.go`、`core/app/appcfg/appcfgfiledef/validation.go` `validateBSCPConfig` | 创建、修改配置文件（含 Helm values 文件）且来源为 BSCP | 本人 | 保持：同上 |
| 绑定 BSCP 项目 | `core/workspace/bscp.go` `BindBscpProject` | 创建空间 | 本人 | 保持 |
| 配置管理扩展：创建后置钩子 | `extension/bscpcfg/service/manager.go` `InitMetadata` | 开启配置管理 | 本人 | 待处理 |
| 配置管理扩展：创建环境和服务、绑定钩子、刷新凭证范围 | `extension/bscpcfg/service/manager.go` `CreateEnvBinding` | 为环境绑定配置管理 | 本人 | 待处理 |
| 配置管理扩展：授权时查业务名 | `extension/bscpcfg/service/permission.go` `addBSCPPermissions` | 同上 | 本人 | 随上一项 |

### 4.3 BCS 与集群资源

| 调用点 | 位置 | 触发场景 | 身份 | 状态 |
| --- | --- | --- | --- | --- |
| 实例事件 | `workload/instance/handler/event.go` `ListEvents` | 查看实例事件（检查查看应用） | 虚拟账号；集群、命名空间取自部署记录 | 已改 |
| 拓扑节点事件 | `workload/topology/service.go` `ListNodeEvents` | 查看拓扑节点事件（检查应用环境） | 虚拟账号；项目编码取自环境记录 | 已改 |
| 部署总览中的集群名称 | `deploy/overview/cluster_names.go` `queryClusterNames` | 查看应用部署总览 | 应用凭证 + 虚拟账号用户名 | 已改；取不到时只是不显示集群名称 |
| Web 终端 | `workload/instance/handler/webconsole.go` `CreateAppInstanceWebConsole` | 打开实例终端 | 应用凭证 + 本人用户名 | 保持：终端服务只检查应用白名单，用户名仅用于记录 |
| 授权项目列表 | `bkintegrations/handler/bcs.go` `ListBCSAuthorizedProjects` | 创建空间、创建或编辑环境时选择容器项目 | 应用凭证 + 本人用户名 | 保持：BKMS 侧没有资源检查，必须按本人查 |
| 项目详情、集群、命名空间列表 | `bkintegrations/handler/bcs.go` `GetBCSProject`、`ListClustersByProject`、`ListNamespacesByCluster` | 前端未调用 | 应用凭证 + 本人用户名 | 保持：同上 |
| 查项目关联的业务 | `core/workspace/dependency.go` `fetchCMDBInfo` | 创建空间时绑定已有蓝盾项目且未指定业务 | 本人 | 保持 |
| 获取用户 BCS token | `infras/account/bcstoken/token.go` | 开发模式发布预检；`/bcs/token` 接口（前端未调用） | 本人 token | 保持：token 交给 CLI 使用 |
| 开发模式重启 | `server/taskqtask/devmoderestart/task.go`（用本人 BCS token 直连集群） | 后台任务：开发模式重启进程（投递时逐个环境检查部署权限） | 本人 token | 保持，见第六节 |

### 4.4 其他系统

| 系统 | 调用点与触发场景 | 身份 | 状态 |
| --- | --- | --- | --- |
| HCM | `bkintegrations/handler/bkhcm.go`：地域、子网、VPC、可用区列表，申请负载均衡；前端未调用 | 本人 | 保持：纯透传，BKMS 侧没有资源检查 |
| CMDB | `bkintegrations/handler/bkcc.go` 授权业务列表（创建空间时选择业务）；`bkintegrations/cmdb` 创建空间时查业务；配置管理扩展授权时查业务名（见第 4.2 节） | 本人 | 保持 |
| 制品库 | `bkintegrations/bkrepo/manager.go` `Initialize`：创建制品库项目和仓库（创建空间） | 应用凭证 + 本人用户名 | 保持 |
| 监控 | APM 应用列表、创建与绑定（`observability/bkmonitor/apm_service.go`）；仪表盘；实例指标、实例排行；告警组；告警策略同步；告警事件；后台任务：部署成功后同步 APM 服务配置（`server/taskqtask/appmodeldeploypoll`，用部署记录创建人），空间初始化（`server/taskqtask/workspace`，用空间创建人） | 应用凭证 + 用户名 | 保持：不按该用户鉴权 |
| 只用应用凭证 | kubeinsight（环境的集群健康诊断）；DBM（依赖服务申请 Redis，`server/taskqtask/depsvcredis`）；`txcmdb`（创建空间时查业务）；`tof`（创建蓝盾项目时查组织架构）；用户管理（校验用户所属租户）；权限中心 | 不涉及用户 | 不受影响 |
| 依赖服务保存的凭证 | 北极星（`extension/depservice/provider/polaris`，用接入时保存的 token） | 不涉及用户 | 不受影响 |

### 4.5 前端直接调用的外部接口

以下请求由浏览器直接发往外部系统，带的是用户自己的登录态，不经过 BKMS 后端，不受虚拟账号影响（路径相对于 `bkms-ui/src/`）：

- BCS 权限查询：`directives/authority.ts`；
- 蓝盾 Git 代码库列表：`pages/application/components/repo-selector.vue`。

---

## 五、后台任务中的身份

后台任务没有当前请求。`taskq` 投递任务时，把投递时 ctx 中的用户连同凭证一起写入任务 payload（`infras/taskq/envelope.go` 的 `_authUser`），执行时恢复到 ctx。所以后台任务中的"本人"指**投递任务的人**，凭证是投递那一刻的凭证。

构建状态轮询原来用这份凭证查询蓝盾。`bk_ticket` 有效期以小时计，构建时间长、排队或重试时可能已经过期，导致查不到构建状态。改用 `GetCaller` 后：

- 启用时用虚拟账号，不受触发者凭证过期影响；
- 未启用时仍为触发者，行为与原来一致，凭证失效时以 `ErrStopRetry` 结束。

---

## 六、已知限制

| 场景 | 原因 | 影响 |
| --- | --- | --- |
| 某空间首次关联某代码库 | 蓝盾按调用者的 Git OAuth 关联代码库，虚拟账号不能做 OAuth | 通常发生在创建应用、修改构建配置或 Helm 配置时，操作的人需要本人有 Git OAuth 和蓝盾代码库创建权限。关联后 BKMS 记录在本地（按空间加别名），之后所有人都能构建 |
| 已关联代码库的后续构建 | 蓝盾把关联人记为代码库的 OAuth 授权人，拉代码、列分支和标签都用关联人的 OAuth | 关联人离职、OAuth 失效或失去仓库权限后，该空间使用这个代码库的构建会失败；BKMS 本地已有记录，不会自动重新关联。这是现有行为，与虚拟账号无关 |
| 用自定义流水线构建 | 流水线内容由用户自定义；用虚拟账号执行，等于让有编辑应用权限的人能以管理员身份执行项目内的任意流水线 | 需要本人在蓝盾上有该流水线的执行权限 |
| 选择 BSCP 配置、保存时校验 | 业务 ID 由前端传入 | 需要本人有 BSCP 权限才能把配置文件改为 BSCP 来源 |
| 部署、预览时读取 BSCP 配置 | 配置文件引用的服务由用户自己带入，权限由用户在 BSCP 上自行管理，平台不代为授权，因此用本人读取 | 部署、预览的人需要本人有该服务的查看权限，自动部署取决于触发构建的人；没有权限时需要本人到 BSCP 申请。与引入虚拟账号前一致 |
| 在 BSCP 上修改、发布配置管理扩展的配置 | 绑定环境时只给内置角色授予这些服务的 BSCP 权限，自定义角色不授予 | 自定义角色的成员需要本人到 BSCP 申请。如果配置文件引用的正是这些服务，部署时同样需要本人申请查看权限 |
| 开发模式 | 预检把本人的 BCS token 交给 CLI，重启也用这个 token 直连集群 | 需要本人有 BCS 权限 |
| 外部系统审计 | 外部系统只能看到虚拟账号 | 区分不出是哪个空间、哪个人操作，需要依靠 BKMS 的操作记录追溯 |
| 虚拟账号 token | 单点依赖 | token 过期或被吊销后，构建、事件等功能全部失败；需要明确有效期和续期方式 |

---

## 七、待处理

### 7.1 配置管理扩展授权时覆盖分级管理员范围（自定义角色上线前处理）

这是 main 上已有的问题，与虚拟账号无关，记录在这里是因为整理调用点时发现。

`extension/bscpcfg/service/permission.go` `addBSCPPermissions` 在配置管理绑定服务时，用 `UpdateWorkspaceAdmin` 更新管理员，传入的数据只有 BSCP 这一项。

权限中心更新分级管理员时，授权范围是整体覆盖写入的（bk-iam-saas `saas/backend/api/management/v1/views/grade_manager.py` 的 `update`）。只传 BSCP 一项，范围就只剩 BKMS 和这个服务，之前绑定的其他服务以及蓝盾、BCS、监控等都被去掉。已经授出的权限不受影响，但之后给用户组授权会按这个范围检查（`saas/backend/biz/group.py` `check_before_grant`），超出就被拒绝。

目前绑定之后没有再授其他权限的场景，所以不会出错。自定义角色上线后，保存需要授监控权限的角色会失败。上线前需要改为按空间现有数据构造完整授权数据再传入。

### 7.2 Helm 仓库凭证兜底创建（建议改用虚拟账号）

- 凭证在创建空间时已由制品库初始化流程创建，兜底只在老空间缺少凭证时触发。
- 这是平台凭证，与具体用户无关。用本人身份时，没有蓝盾权限的成员第一次创建 Chart 版本，或部署、查看 helm-git 应用的 Chart 时会失败。
- 需要确认蓝盾管理员模板包含凭证管理权限。

### 7.3 BSCP 配置管理扩展（待定）

接口按应用、环境的编辑权限放行，服务端以本人身份在 BSCP 上创建服务、环境、凭证范围和钩子。现有 BSCP 授权模板里没有这些操作，内置角色也依赖本人在 BSCP 上的权限。没有 BSCP 权限的成员无法使用。

- **方案 A**：作为已知限制，需要本人有 BSCP 权限。
- **方案 B**：改用虚拟账号。业务由后端从空间取，没有越权风险，但要给虚拟账号授予业务级的创建服务、环境、凭证、钩子权限，模板里都没有，改动较大。

### 7.4 蓝盾构建通知（待定）

内置流水线模板没有配置通知订阅，使用蓝盾的默认设置，默认的失败通知按触发人发送。改由虚拟账号触发后，真实用户收不到蓝盾的失败通知。构建状态仍可以在 BKMS 的构建记录中查看。

### 7.5 停用后的成员展示（待定）

成员列表中隐藏虚拟账号依赖 `enabled`。如果启用后又停用，虚拟账号仍在各空间的管理员组中，会重新出现在成员列表里。可以改成只要配置了用户名就隐藏，或者停用时提供移除命令。

### 7.6 构建触发回调（以后注意）

`build/trigger/handler/callback.go` 目前是空实现，构建触发专用流水线（`TriggerPipelineManager`）也还没有调用方。实现后回调请求的 ctx 中没有用户：

- 自定义流水线路径直接调用 `auth.MustGetUser`，会 panic；
- 未启用虚拟账号时，`MustGetCaller` 同样会 panic。

届时需要决定使用虚拟账号还是触发策略的创建人。

---

## 八、新增外部调用时的约定

1. 先按第 3.1 节判断身份，不要默认使用 `auth.MustGetUser`。
2. 使用虚拟账号的调用，入口必须已经完成 BKMS 鉴权，外部资源标识必须来自后端数据。
3. 不要在数据对象的方法里隐式从 ctx 取用户建客户端（`File.Content` 原来的写法），身份应由调用方决定、随客户端传递。
4. 后台任务使用 `GetCaller`，并考虑投递者凭证过期的情况。
5. 新的调用点需要补充本文第四节。

---

## 九、依据与待验证

**源码依据**（GitHub TencentBlueKing 组织下的开源仓库）

| 结论 | 位置 |
| --- | --- |
| 网关的 `access_token` 校验同时识别应用和用户 | blueking-apigateway |
| 蓝盾代码库按关联人的 OAuth 拉取代码、列分支和标签 | bk-ci |
| BSCP 读取配置和下载文件检查业务访问和服务查看权限 | bk-bscp：`cmd/config-server/service/config_item.go`（`ListReleasedConfigItems`）、`cmd/api-server/service/repo.go`（`DownloadFile`） |
| BCS 网关只认证不鉴权；终端只检查应用白名单；项目管理接口按请求用户鉴权 | bk-bcs：`bcs-services/bcs-bk-apisix-gateway/plugins/bcs-auth/authentication.lua`、`bcs-services/bcs-webconsole/route/permissions.go`、`bcs-services/bcs-project-manager/internal/auth/preset.go` |
| 监控在没有用户名时以管理员执行，有用户名时以其为操作人 | bk-monitor：`bkmonitor/kernel_api/middlewares/authentication.py`（`process_view`） |
| 分级管理员更新时整体覆盖授权范围；用户组授权按分级管理员范围检查 | bk-iam-saas：`saas/backend/api/management/v1/views/grade_manager.py`、`saas/backend/biz/group.py` |

**上线前需要实测**

- 虚拟账号的 `access_token` 能否通过各系统网关，有效期和续期方式；
- 用一个没有任何外部权限的账号，走一遍构建（内置流水线）、构建日志、实例事件、拓扑节点事件、部署总览中的集群名称；
- `grant_platform_account` 在存量空间上的执行结果。
