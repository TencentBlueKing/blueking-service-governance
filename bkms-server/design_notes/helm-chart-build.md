# Helm Chart 构建功能设计文档

## 设计背景与目标

为 Helm 类型应用提供从 Git 源码打包并推送 Helm Chart 到 BkRepo Helm 仓库的能力。

核心目标：

- **Semver 版本管理**：构建时自动递增版本号（patch/minor/major），并发安全。
- **一键构建**：触发后自动完成 semver 生成 → 蓝盾流水线触发 → 构建记录落库 → 异步轮询跟踪状态。
- **凭证安全**：Helm 仓库凭证 AES 加密存储，蓝盾凭证幂等初始化。
- **异步轮询**：固定常量设置轮询间隔和超时，轮询状态实时持久化。

## 核心架构与工作流

### 同步触发 (ExecuteChartBuild)

依次完成 semver 递增 → 流水线/代码库/凭证初始化 → 触发蓝盾构建 → 构建记录落库 → 启动异步轮询。

### 异步轮询 (taskq.pollingHelmChartBuildStatus)

每 5s 轮询蓝盾构建状态，状态变更时持久化到 DB；终态时退出轮询并记录审计。

## 关键领域模型

| 模型                   | 集合                           | 主键/唯一索引           | 核心字段                          |
|----------------------|------------------------------|-------------------|-------------------------------|
| Build Record         | `helm_chart_build_records`   | `appID + buildID` | Num, ChartVersion, Status     |
| Semver Counter       | `helm_chart_semver_counters` | `appID`           | Major, Minor, Patch           |
| Helm Repo Credential | `helm_repo_credentials`      | `workspaceID`     | CredentialID, Password(AES加密) |

Semver 递增规则：

- BumpPatch → patch+1；
- BumpMinor → minor+1, patch 归零；
- BumpMajor → major+1, minor+patch 归零。初始值 0.0.0。

## 接口设计

| 接口                   | 方法   | 路径                            | 请求体                    | 响应                                                                                                                               |
|----------------------|------|-------------------------------|------------------------|----------------------------------------------------------------------------------------------------------------------------------|
| CreateHelmChartBuild | POST | `/apps/{appID}/charts/builds` | `{"bumpType":"patch"}` | `{"data":{"chartVersion":"0.0.1","buildID":"b-xxx"}}`                                                                            |
| GetHelmChartSemver   | GET  | `/apps/{appID}/charts/semver` | `?bumpType=patch`（可选）  | `{"data":{"latest":{"major":0,"minor":0,"patch":1,"version":"0.0.1"},"next":{"major":0,"minor":0,"patch":2,"version":"0.0.2"}}}` |

