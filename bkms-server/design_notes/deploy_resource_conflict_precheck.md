# 部署资源冲突检测

## 背景

tRPC 应用部署时，Deployer 对所有 K8s 资源使用 Upsert（SSA `Force=true`）。如果目标 namespace 中已存在外部创建的同名资源（kubectl、其他部署系统等），会被**静默覆盖**；若涉及不可变字段（selector、clusterIP）则**直接报错**。两种情况用户都缺少事前感知。

## 当前问题

1. 部署前无法得知目标 namespace 中是否已有同名资源
2. 静默覆盖可能破坏外部系统正在使用的资源
3. 不可变字段冲突导致部署失败，报错信息不直观

> 注：bkms 系统内部不存在跨应用冲突——环境已有 (clusterID, namespace) 全局唯一约束，资源名称由 app name 确定性派生。冲突仅来自外部资源。

## 方案

新增独立的部署资源冲突预检 API，在部署前检测并阻断，仅在首次部署时会做冲突检测

### 检测流程

```
1. Builder.Build(app, env) → BuildResult → 提取 ResourceKeys [{Kind, Name}, ...]

2. 查 deploy record（本 app + 本 env 最新活跃记录）
   ├── 有记录 → 待检测 = currentKeys - previousKeys（只查新增资源）
   └── 无记录 → 待检测 = 全部 currentKeys

3. 对待检测的每个 ResourceKey，K8s GET 目标 namespace
   ├── 404 → 无冲突
   └── 200 → 冲突

4. 返回冲突资源列表（Kind + Name）
```

### API

```
GET /apps/{appID}/envs/{envName}/trpc-deploys/resource-conflict-precheck
GET /apps/{appID}/envs/{envName}/taf-deploys/resource-conflict-precheck
```

响应：

```json
{
  "conflictResources": [
    { "kind": "ConfigMap", "name": "myapp" }
  ]
}
```

### 涉及的资源类型

| Kind | 命名规则 | 条件 |
|------|---------|------|
| GameDeployment / Deployment | `{appName}` | 每次必有 |
| ConfigMap（tRPC 配置） | `{appName}` | 配置了 tRPC config |
| ConfigMap（DevMode） | `{appName}-devmode-scripts` | 非生产 + DevMode |
| PolarisConfig + Service | `{appName}-{configName}-polaris[-service]` | 配置了北极星 |
| 组件资源 | `{appName}-{componentName}` | 挂载了组件 |

### 前端

- 硬阻断：有冲突时不可继续部署
- 在现有预检弹窗中新增"资源冲突"区块
- 新增 `useResourceConflictPrecheck` composable，接入 `useDeployPrecheck` 编排

### 复用的现有能力

- `Builder.Build()` — 与 env-var-precheck 相同的 dry-run 构建
- `buildResourceKeys()` — 从 BuildResult 提取资源列表
- `deploy record.ResourceKeys` — 排除自身已部署的资源
- `discovery.GetGroupVersionResource` + `client.Get` — K8s 资源存在性检查（参考 `state_getter.go`）

### 范围

- 本期：tRPC/TAF 应用
- 不含：Helm、CLI、自动解决冲突
