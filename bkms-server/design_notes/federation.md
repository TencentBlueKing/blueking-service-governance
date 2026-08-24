# 联邦环境适配

联邦 Host 对 GameDeployment 和 Kubernetes SSA 的支持不完整。BKMS 在识别为联邦的环境上做了一系列后台适配，对用户尽量透明。

## 1. 如何判断联邦

不以 BCS 实时查询为准, 判定来源是配置列表：

```yaml
bcs:
  federationClusterIDs:  
    - BCS-K8S-xxxxx
```

实现：`cluster.Config.IsFederation()` 读 `config.G.BCS.FederationClusterIDs`。环境侧入口是 `env.IsFederationCluster(clusterID)`。

部署热路径**不再查 BCS**，只读环境上已经写好的 `cluster.isFederation`。

## 2. 环境标记

| 位置 | 说明 |
|------|------|
| 落库 | `Environment.cluster.isFederation`（`bson:"isFederation"`） |
| 写入时机 | 创建环境、更新环境集群绑定 |
| 入参 | 前端**不传**该字段，后端对照配置自行写入 |
| 出参 | 环境详情可返回 `isFederation`，供排查，前端不必展示 |
| 特性环境 | 从源环境复制 `IsFederation` |
| 存量 | 字段缺失视为 `false` |

前端可能需要这个字段来判断一些功能开关。

## 3. Workload：原生 Deployment

Builder **永远先造 GameDeployment**，组件 / 北极星 / 环境变量都作用在 GD 上。最后若 `env.Cluster.IsFederation`，再转成 `apps/v1 Deployment`。

转换白名单（`gameDeploymentToDeployment`）：

- 拷贝：Name / Namespace / Labels / Annotations、Replicas、Selector、Template、RollingUpdate 的 MaxUnavailable / MaxSurge
- 丢掉：GD 独有字段（`podsToDelete`、原地更新、灰度分区等）
- 不剥离 `io.tencent.bcs.dev/*` 一类 BCS 注解

`BuildResult` 带 `WorkloadKind` + `MainWorkload`。部署记录写入 `Record.WorkloadKind`。

### 能力差异

| 能力 | GameDeployment | 联邦 Deployment |
|------|----------------|-----------------|
| 滚动更新 / 改副本 | 支持 | 支持（patch `spec.replicas`） |
| 原地更新 / 灰度指定实例 | 支持 | **明确报错** |
| 删除指定运行中 Pod | 走 `podsToDelete` | **明确报错**（终止态 Pod 仍可直接删） |

不在联邦上复刻 GD 特有能力。

## 4. K8s Client

### Create 补 namespace

调用方仍禁止 manifest 自带 `metadata.namespace`。`Create` 校验后 DeepCopy 并 `SetNamespace`，满足联邦网关 / 准入 webhook。

### Upsert 双路径

| 集群 | 路径 | 原因 |
|------|------|------|
| 非联邦 | Server-Side Apply（`ApplyPatchType` + `Force=true` + FieldManager） | 与既有行为一致 |
| 联邦 | 三路 JSON Merge Patch | BCS 网关不支持 `application/apply-patch+yaml` |

联邦 Merge 路径要点：

- 注解：`bkms.tencent.com/last-applied-configuration`（不复用 kubectl 那条，避免互相覆盖）
- 算法：`jsonmergepatch.CreateThreeWayJSONMergePatch`
- GET 不存在则 Create；Create 撞 AlreadyExists **直接返回**，不在 client 里转 Patch（由调用方重试）
- `prepareFederationDesired`：克隆、sanitize、写 last-applied；**不剥 status**（与 SSA 对齐）
- 只清「上次本路径写过、本次省略」的字段；从未写入的字段（Service `clusterIP`、status、其他控制器注入的 label）保留
- 资源上还没有 last-applied 时，第一次 upsert **不会**删除实况里的多余字段
- JSON Merge Patch 把数组当成整体替换；`opts.Force` 在此无效

拓扑 YAML 展示会剥掉 kubectl / bkms 两条 last-applied 注解。

## 5. 部署状态

`latest-status` **只读库**，不现场查集群。轮询任务走 `DeployStateGetter.Get`：

- kind = Deployment 且 `clusterCfg.IsFederation()` → `deploystatus.ParseForFederation`
- kind = Deployment 且非联邦 → `deploystatus.Parse`（严格）
- kind = GameDeployment → `gamedeploystatus.Parse`

`ParseForFederation` 的原因：联邦网关返回的 Deployment status 通常只有 `replicas` / `readyReplicas` / `updatedReplicas` / `availableReplicas`，没有 `observedGeneration` 和 `conditions`。宽松解析只看副本是否一致 → Available。

拓扑节点状态同样按集群判定：`getResourceStatus` 在 `clusterCfg.IsFederation()` 时对 Deployment 走 `ParseForFederation`，与部署轮询一致。

## 6. GPA

联邦侧目前需要通过单独的 CR (FedGeneralPodAutoscaler)来进行自动扩缩容，且不支持基于指标的扩缩容， 因此本期**不支持 GPA** 。创建 / 更新 / 开启返回 `INVALID_REQUEST`（`gpa is not supported in federation environment`）。关闭和删除仍可用，便于清理误下发的配置。



## 7. 关键代码

| 主题 | 路径 |
|------|------|
| 配置 | `pkg/common/config/types.go`（`BCSConfig.FederationClusterIDs`） |
| 判定 | `pkg/infras/kubernetes/cluster/config.go`、`pkg/core/env/federation.go` |
| 环境落库 | `pkg/core/env/model/env.go`、`pkg/core/env/handler/env.go` |
| Workload 转换 | `pkg/workload/appmodelcore/workload/to_deployment.go` |
| 部署 / 扩缩 / 实例操作 | `pkg/deploy/appmodel/deployer.go` |
| 部署状态 | `pkg/deploy/appmodel/state_getter.go`、`pkg/infras/kubernetes/status/workload/deployment/deployment.go` |
| 拓扑 Deployment 状态 | `pkg/workload/topology/node_status.go`、`builder.go` |
| K8s Client | `pkg/infras/kubernetes/client/client.go`、`merge_patch.go` |
| GPA（联邦拒绝） | `pkg/extension/addon/gpa/service.go`、`handler/gpa.go` |
| 拓扑 YAML | `pkg/workload/topology/manifest.go` |
