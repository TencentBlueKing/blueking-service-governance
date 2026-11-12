# 北极星接入点

## 1. 背景

社区版业务使用自建北极星（如腾讯云 TSE 北极星），一个空间内不同集群可能连接不同的北极星。北极星在平台内有 4 条链路：

| 链路 | 改造前的接入信息来源 |
| --- | --- |
| 服务注册（OpenAPI） | depservice `polaris` 服务 `default` 方案的 `baseUrl`，请求不带鉴权头 |
| 实例注册（集群 `bcs-polaris-operator`） | 集群插件安装参数 |
| 实例查询（SDK） | 进程级单例，启动时用 `config.Polaris.Address/JoinPoint` 初始化 |
| tRPC 框架配置 | 只注入 `plugins.registry.polaris.service`，不注入地址 |

方案：空间管理员登记“北极星接入点”，再把接入点绑定到集群；环境、北极星配置、operator 都由集群绑定推导出所用接入点。
4 条链路都改为从**解析得到的接入点**（`accesspoint.AccessPoint`）取接入信息，不再各自读全局配置。

## 2. 接入点类型与能力

| 类型 | ID | 来源 | 是否入库 |
| --- | --- | --- | --- |
| `platform` 平台托管 | 固定为 `platform` | SDK 地址取自 `config.Polaris`，OpenAPI 地址取自 depservice `polaris/default` 方案的 `baseUrl` | 否，不可编辑 |
| `workspace` 空间自建 | ObjectID hex | 空间管理员登记 | 是 |

自建接入点只登记 `clientHosts`（IP 或域名，1–10 个），端口固定：OpenAPI 为 `8090`，SDK 为 `8091`。
OpenAPI 只用第一个地址（`http://{clientHosts[0]}:8090`），SDK 使用全部地址（`host:8091`）。

两类接入点的差异全部表达为能力字段（`accesspoint.Capabilities`，`CapabilitiesOf(kind)`）。**前端与后端只看能力，不判断接入点类型或版本**：

| 能力 | `platform` | `workspace` | 消费方 |
| --- | --- | --- | --- |
| `editable` 可编辑、删除 | false | true | 02 |
| `weightFactor` 权重因子、动态权重 | true | false | 05 |
| `namespaceSource` 命名空间来源 | `fixed`（Test/Production/Development/Pre-release） | `live`（实时读取北极星） | 02、05 |
| `serviceTokenRequired` 引入已有服务时需服务 Token | true | false | 05 |
| `injectClientToken` 向 Pod 注入 `{instanceKey}_polarisToken` | true | false | 05 |

## 3. 两版差异的处理

内部版与社区版同仓，差异只由配置驱动，不在代码里散落版本判断，内部仓库不新增 overlay 文件。

| 配置 | 默认值 | 社区版 | 内部版 |
| --- | --- | --- | --- |
| `polaris.allowWorkspaceAccessPoints` | `false` | `true` | `false` |
| `network.egressIPs` | 空 | 平台出口 IP | 不需要 |
| `polaris.address` / `polaris.joinPoint` | 空 | 有平台北极星时填写 | `joinPoint` |

`allowWorkspaceAccessPoints=false` 时：

- 只有平台托管接入点，所有集群隐式绑定到它；
- 接入点写接口返回 `INVALID_REQUEST`（`ErrAccessPointDisabled`，02 实现）；
- 集群绑定视图、组件状态视图、组件安装、环境组件状态 4 个接口不注册（`accesspoint.Register` 的第三个参数）。

前端通过 `GET /frontend-features`（`pkg/misc/frontendfeature`）读取 `polarisWorkspaceAccessPoint`，
它的值就是 `allowWorkspaceAccessPoints`。为 false 时前端隐藏接入点管理、集群绑定、组件安装等入口。
开关只由后端配置决定，不另设前端部署变量，避免前后端两份配置不一致；前端隐藏入口只是展示层面的处理，
后端接口仍按上面的规则各自拦截。

内部版的私有 SDK 通过 go.work replace 替换 `libs/bkms-adapter`，见第 7 节。

## 4. 解析规则

`accesspoint.Resolver`（`NewResolver(db)`，也由 `accesspoint.FxModule` 提供）是各链路获取接入点的唯一入口：

| 方法 | 规则 | 调用方 |
| --- | --- | --- |
| `ResolveByID(ctx, ws, id)` | `platform` 返回平台托管接入点（不可用时 `ErrAccessPointNotFound`，见 4.1）；其余 ID 查库且必须属于该空间，否则 `ErrAccessPointNotFound` | 02、04 |
| `ResolveByCluster(ctx, ws, clusterID)` | `allowWorkspaceAccessPoints=false` 返回平台托管；否则查集群绑定，未绑定返回 `ErrClusterNotBound` | 03、04、05 |
| `ResolveForPolarisConfig(ctx, cfg)` | `cfg.AccessPointID` 为空（内部版存量数据）或为 `platform` 按平台托管处理；其余按配置所属应用的空间调用 `ResolveByID` | 05、06 |

- 解析结果里的 Token 是明文，只在服务端内部使用，不能出现在接口响应、日志与审计记录中。
- 数据库查询失败返回包装后的错误，**不静默回退**到平台托管接入点，避免把服务注册到错误的北极星。
- `ResolveForPolarisConfig` 的参数是接口 `PolarisConfigRef`（由 `polaris.PolarisConfig` 实现），
  因为 `addon/polaris` 依赖本包，本包不能反向引用 `polaris.PolarisConfig`。
  `PolarisConfig` 上没有空间 ID，因此需要多查一次应用。

### 4.1 平台托管接入点是否可用

`PlatformAvailable()` 决定平台托管接入点是否可用：
`allowWorkspaceAccessPoints=false` 时总是可用（内部版可以不配地址，由私有 SDK 使用内置接入点），
否则仅在配置了 `polaris.address` 或 `polaris.joinPoint` 时可用。

不可用时，平台托管接入点不出现在接入点列表中、不能被集群绑定，解析方法也返回 `ErrAccessPointNotFound`，
不会返回一个地址为空的接入点、让调用方到真正访问北极星时才失败。

存量 `PolarisConfig`（`accessPointID` 为空）只来自内部版，内部版的平台托管接入点总是可用，因此不受影响；
社区版没有存量数据，不需要为存量配置放宽这条规则。

### 4.2 集群可以绑定到平台托管接入点

`PolarisClusterBinding.AccessPointID` 可以是 `platform`。社区版同时配置了平台北极星时，
`allowWorkspaceAccessPoints=true` 下未绑定的集群会被 `ResolveByCluster` 拒绝，用户需要显式选择平台托管接入点。

### 4.3 删除接入点要检查北极星配置

`PolarisConfig.accessPointID` 记录的是服务实际所在的北极星，不随环境或绑定变化。
生效环境清空后集群可以被解绑，此时接入点已没有绑定，但配置仍固定在它上面。
如果允许删除，配置之后的 `ResolveForPolarisConfig` 会返回 `ErrAccessPointNotFound`，删除服务、查询实例都无法进行，
北极星上的服务成为孤儿。

因此删除接入点（02）要同时满足两个条件，任一不满足都返回 `ErrAccessPointInUse`：

1. 没有绑定集群；
2. 没有北极星配置的 `accessPointID` 等于它。错误详情中列出这些配置的应用与名称。

第 2 条的查询是 `polaris` store 中的 `ListConfigUsagesByAccessPoint(ctx, db, accessPointID)`。
本包只声明同签名的函数类型 `accesspoint.ConfigUsageLister`，由删除接口的 handler 把 `polaris.ListConfigUsagesByAccessPoint`
传给删除逻辑：`addon/polaris` 依赖本包，本包不能直接调用它；`accesspoint/handler` 引用 `addon/polaris` 不构成循环。
相反方向不需要限制：配置固定在接入点上，生效环境为空时换绑原来的集群不会影响配置后续的操作。

## 5. 数据模型

三个集合都是租户集合（`TenantScoped: true`），索引见迁移 `000026_polaris_access_points_idx`。组件安装任务见 5.1。
该迁移还给 `polaris_configs` 新增了 `accessPointID` 索引，供删除接入点前查询（见 4.3）。

| 集合 | 模型 | 索引 |
| --- | --- | --- |
| `polaris_access_points` | `PolarisAccessPoint` | 唯一 `workspaceID + name` |
| `polaris_cluster_bindings` | `PolarisClusterBinding` | 唯一 `workspaceID + clusterID`（不做跨空间唯一）；`workspaceID + accessPointID` |
| `cluster_addon_install_tasks` | `installtask.InstallTask` | `clusterID + namespace + addonName + createdAt(-1)`；唯一 `clusterID + namespace + addonName`（仅 `status=running`，不带租户） |

- 自建接入点的 Token 选填，通过 store 的 Encode/Decode 钩子用 `crypto.AESEncrypt` 加密存储。
  Token 只写入 `PolarisConfig` CR，不放进 operator 安装参数，不注入 Pod；更换 Token 下次部署生效。
- `lastConnectivity` 记录最近一次连通测试：`{ok, stage, message, consoleAuth, clientAuth, checkedAt}`，
  `stage` 取值 `network` / `clientAuth` / `token`。
- 组件安装任务状态只有 `running` / `succeeded` / `failed`。“同一集群同时只允许 1 个运行中的任务”
  由部分唯一索引保证，并发创建时第二个返回 `installtask.ErrRunning`，不依赖先查后写。

### 5.1 组件安装任务放在 clusteraddon

`bcs-polaris-operator` 是内置的集群 Addon（`clusteraddon/assets/addons/bcs-polaris-operator.yaml`），
环境的“集群组件”页也能安装它。为了让两条安装路径看到同一份任务、互相排斥，安装任务是通用的集群 Addon 任务，
不是北极星专属模型：

- 模型与存储在 `pkg/core/env/clusteraddon/installtask`，不在 `clusteraddon` 根包，
  因为 `clusteraddon` 经 `deploy/helm` 依赖 `addon/polaris`，放在根包会让 `accesspoint` 引用它时形成循环依赖。
- 互斥键是 `clusterID + namespace + addonName`，与 `clusteraddon.NewInstallLock` 一致，不带租户与空间：
  集群是物理资源，不同租户、不同空间对同一 Addon 的安装都必须互斥。
- 04 的异步任务执行 helm 安装时**必须持有 `clusteraddon.NewInstallLock`**。唯一索引只拦得住两个异步任务，
  环境页的同步安装不写任务记录，只有锁能让两条路径互斥。
- 任务不记录接入点。`outdated` 由 release 的 values 与接入点当前的期望值比对得出，不依赖安装时的接入点。
- 北极星专属的部分留在本包：安装参数合并、`outdated` / `driftFields` 判定、组件状态视图与批量安装接口；
  组件状态视图按 `addonName=bcs-polaris-operator` 查询最近任务。
- `PolarisConfig` 新增 `accessPointID`：创建时由生效环境的集群绑定推导并固定，之后不变；为空表示平台托管。

## 6. REST API 契约

路由注册在 `accesspoint/router.go`，handler 在 `accesspoint/handler`，请求与响应结构在 `accesspoint/serializer`。
除网络信息外，handler 目前只做参数绑定与权限校验，然后返回 `NOT_IMPLEMENTED`，业务逻辑由对应子需求补齐。

| 接口 | 路由 | 权限 | 实现方 |
| --- | --- | --- | --- |
| 接入点列表 | `GET /workspaces/{ws}/polaris-access-points` | 空间查看 | 02 |
| 创建接入点 | `POST /workspaces/{ws}/polaris-access-points` | 空间编辑 | 02（`clusterIDs` 由 03） |
| 更新接入点 | `PATCH /workspaces/{ws}/polaris-access-points/{accessPointID}` | 空间编辑 | 02（`clusterIDs` 由 03） |
| 删除接入点 | `DELETE /workspaces/{ws}/polaris-access-points/{accessPointID}` | 空间编辑 | 02 |
| 连通测试 | `POST /workspaces/{ws}/polaris-access-points/connectivity-test` | 空间编辑 | 02 |
| 网络信息 | `GET /workspaces/{ws}/polaris-access-points/network-info` | 空间查看 | 已实现 |
| 命名空间 | `GET /workspaces/{ws}/polaris-access-points/{accessPointID}/namespaces` | 空间查看 | 02 |
| 集群绑定视图 | `GET /workspaces/{ws}/polaris-clusters` | 空间查看 | 03 |
| 组件状态视图 | `GET /workspaces/{ws}/polaris-operators` | 空间查看 | 04 |
| 安装/更新组件 | `POST /workspaces/{ws}/polaris-access-points/{accessPointID}/operator-installs` | 空间编辑 | 04 |
| 环境组件状态 | `GET /envs/{envID}/polaris-operator` | 环境查看 | 04 |

后 4 个接口只在 `allowWorkspaceAccessPoints=true` 时注册。

另有全局接口 `GET /frontend-features`（登录即可访问，已实现）向前端下发 `polarisWorkspaceAccessPoint`，见第 3 节。

更新接口的字段都是指针：缺省表示不修改；`token` 显式传空字符串表示清除；`clusterIDs` 显式传空数组表示解除全部绑定。

serializer 只用 binding 校验必填。地址格式、数量上限、名称重复由领域逻辑校验。

### 6.1 错误码

领域错误定义在 `accesspoint/errors.go`，handler 统一用 `accesspoint.APIError(err, msg)` 包装。
错误码（决定 HTTP 状态码）用仓库通用的 `bkerrs.ErrCode`；只有前端需要按类型做特殊处理、而不只是展示错误信息的错误，
才附带详情码（`bkerrs.ErrDetailCodePolaris*`），与仓库已有的 `ENV_CLUSTER_NAMESPACE_OCCUPIED` 等一致：

| 领域错误 | 错误码 | 详情码 | 触发场景 |
| --- | --- | --- | --- |
| `ErrAccessPointNotFound` | `NOT_FOUND` | 无 | 接入点不存在或不属于该空间 |
| `ErrAccessPointDisabled` | `INVALID_REQUEST` | 无 | 不允许自建接入点时创建、更新、删除 |
| `ErrAccessPointNameExists` | `ALREADY_EXISTS` | 无 | 名称重复 |
| `ErrAccessPointInvalid` | `INVALID_ARGUMENT` | 无 | 地址格式错误、数量越界、集群不在候选范围内 |
| `ErrConnectivityFailed` | `INVALID_REQUEST` | 无 | 保存前重跑连通测试未通过（网络不通、`clientOpen=true`、Token 无效），失败阶段写在错误信息里 |
| `ErrClusterNotBound` | `INVALID_REQUEST` | 无 | 北极星配置选择了未绑定集群上的环境、对未绑定集群安装组件 |
| `ErrAccessPointInUse` | `INVALID_REQUEST` | `POLARIS_ACCESS_POINT_IN_USE` | 删除仍绑定集群、或仍有北极星配置固定使用的接入点 |
| `ErrClusterInUse` | `INVALID_REQUEST` | `POLARIS_CLUSTER_IN_USE` | 解绑或换绑的集群上有环境被北极星配置使用 |

- 结构化的连通失败阶段由连通测试接口的 `stage` 返回，前端在保存前调用它展示；保存失败时展示错误信息即可。
- 前端可以通过 `/frontend-features` 隐藏入口、通过环境的 `polarisAccessPointID` 过滤可选环境，
  因此 `DISABLED`、`CLUSTER_NOT_BOUND` 不需要详情码。
- 两个 `IN_USE` 详情码需要在 `extras` 中列出占用的应用与配置供前端展示，由 02、03 实现删除与换绑时
  参照 `bkerrs.WrapEnvClusterNamespaceOccupied` 单独包装。

## 7. SDK 适配层

`libs/bkms-adapter/polaris` 新增按接入点多实例化的工厂：

```go
type Consumer interface {
    GetInstances(ctx context.Context, namespace, serviceName string) ([]*Instance, error)
    Close() error
}

func NewConsumer(ctx context.Context, joinPoint string, addresses []string) (Consumer, error)
```

- 开源实现不支持 `joinPoint`，忽略它并只使用 `addresses`，`addresses` 为空时返回错误。
- 现有的单例 `Provider` / `Init` 保持不变，由 06 改造调用方后再移除。
- bkms-server 目前还没有调用 `NewConsumer`，因此内部私有实现（`bkms-adapter-te`）暂未实现它也不影响内部版构建；
  **06 合入前，`bkms-adapter-te` 需要先实现同名的 `Consumer` 与 `NewConsumer`。**
  接口只依赖 `context`、字符串与 `Instance`，私有 SDK 可以满足。

