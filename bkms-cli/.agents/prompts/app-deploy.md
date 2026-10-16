# bkms-cli app deploy Reference

`bkms-cli app deploy` 用于管理应用的部署，包括创建部署、查看部署记录、更新已有部署、删除部署以及部署前检查。支持 helm、trpc、taf 三种应用类型。

`--env` 参数支持多环境（逗号分隔），操作将对每个环境依次执行。`create`、`update`、`delete`、`list` 还支持 `--env-type` 参数按环境类型批量操作（如 `--env-type test,development`），将对指定类型下的所有环境依次执行。二者互斥，必须提供其一。

## precheck

部署前检查：验证指定环境下应用配置文件、组件属性及 Polaris 服务标签中引用的环境变量是否已定义，以及所需集群组件是否已安装。仅支持 trpc 和 taf 应用。

命令退出码：0 = 两项检查通过；1 = 检查未通过或执行失败。

```bash
# 检查 prod 环境部署前置条件
bkms-cli app deploy precheck --app myapp --env prod

# 以 JSON 格式输出（便于脚本处理）
bkms-cli app deploy precheck --app myapp --env prod -o json

# 检查通过示例输出：
# ✓ Pre-check passed for app myapp in env prod

# 检查失败示例输出（表格）：
# ✗ Pre-check FAILED: 2 undefined environment variable(s), 1 missing required cluster addon(s) found
#
#   KEY                            REFERENCED BY
#   APP_CONFIG                     appConfigFile:prod, component:app-config
#   UPSTREAM_HOST                  appConfigFile:prod
#
# Fix: use 'bkms-cli envvar create' to define missing variables, then re-run precheck.
#
#   MISSING CLUSTER ADDONS
#   Gamedeploy
#
# Fix: install the missing cluster addons from the environment page, then re-run precheck.
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 是 | 环境名称 |
| `-o, --output` | 否 | 输出格式：json / yaml / table / jq=\<expr\> |

## create

创建新部署（触发发布）。通过 YAML spec 文件指定部署参数，字段因应用类型不同而有所差异。

### YAML spec 说明

**helm 应用**
```yaml
# 镜像 tag（必填）
imageTag: v1.0.0
# Chart 版本（可选）
chartVersion: 1.2.3
# Values 文件 ID（可选）
valuesFile: values-prod
# 泳道名称（可选）
trafficLane: canary
```

**trpc / taf 应用**
```yaml
# 镜像 tag（必填）
imageTag: v1.0.0
# 副本数，>= 1（必填）
replicas: 3
```

### 常用场景

```bash
# trpc/taf 应用部署
bkms-cli app deploy create --app myapp --env prod -f trpc-deploy.yaml

# 同时部署到多个环境
bkms-cli app deploy create --app myapp --env prod,staging,test -f trpc-deploy.yaml

# 按环境类型批量部署（test、development 类型下的所有环境）
bkms-cli app deploy create --app myapp --env-type test,development -f trpc-deploy.yaml

# helm 应用部署
bkms-cli app deploy create --app myapp --env prod -f helm-deploy.yaml

# 指定工作空间
bkms-cli app deploy create --workspace ws-demo --app myapp --env prod -f deploy.yaml
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 二选一 | 环境名称（支持逗号分隔多环境） |
| `--env-type` | 二选一 | 环境类型（支持逗号分隔）：development / test / staging / production，展开为该类型下所有环境 |
| `-f, --deploy-spec-file` | 是 | 部署 spec 文件路径 |
| `--workspace` | 否 | 工作空间 ID |

## list

查看最近 10 条部署记录。支持按环境、关键字、泳道过滤。

### 返回字段（trpc/taf）

| 字段 | 说明 |
|------|------|
| `id` | 部署记录 ID |
| `imageTag` | 部署的镜像 Tag |
| `replicas` | 副本数 |
| `status` | 部署状态（Deploying / Success / Failed 等） |
| `operator` | 触发人 |
| `createdAt` | 创建时间 |

### 常用场景

```bash
# 列出 prod 环境最近部署记录
bkms-cli app deploy list --app myapp --env prod

# 列出多环境部署记录
bkms-cli app deploy list --app myapp --env prod,staging

# 按环境类型列出部署记录
bkms-cli app deploy list --app myapp --env-type test,development

# 按关键字过滤
bkms-cli app deploy list --app myapp --env prod --keyword v1.0

# JSON 格式输出
bkms-cli app deploy list --app myapp --env prod -o json

# 获取最新部署的镜像 tag
bkms-cli app deploy list --app myapp --env prod -o 'jq=.[0].imageTag'
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 二选一 | 环境名称（支持逗号分隔多环境） |
| `--env-type` | 二选一 | 环境类型（支持逗号分隔）：development / test / staging / production，展开为该类型下所有环境 |
| `--keyword` | 否 | 关键字过滤 |
| `--trafficLane` | 否 | 泳道名称过滤 |
| `-o, --output` | 否 | 输出格式 |
| `--workspace` | 否 | 工作空间 ID |

## update

更新已有部署。仅支持 trpc 和 taf 应用。支持四种更新模式：

| 模式 | YAML 字段 | 说明 |
|------|-----------|------|
| `Full` | `imageTag` | 全量更新（镜像 + 配置），重建 Workload 和 Pod |
| `Config` | 无额外字段 | 仅更新配置，重建 Workload 和 Pod |
| `Image` | `imageTag`, `strategy` | 仅更新镜像；strategy: `InplaceUpdate`（原地更新，推荐）或 `RollingUpdate` |
| `Grayscale` | `imageTag`, `instanceIDs` | 灰度更新指定实例，`instanceIDs` 用分号分隔 |

### 常用场景

```bash
# 全量更新（更新镜像和配置）
bkms-cli app deploy update --app myapp --env prod -f update-full.yaml

# 仅更新镜像（原地更新，推荐）
bkms-cli app deploy update --app myapp --env prod -f update-image.yaml

# 灰度更新指定实例
bkms-cli app deploy update --app myapp --env prod -f update-grayscale.yaml

# 多环境同步更新
bkms-cli app deploy update --app myapp --env prod,staging -f update-image.yaml

# 按环境类型批量更新（test、development 类型下的所有环境）
bkms-cli app deploy update --app myapp --env-type test,development -f update-image.yaml
```

### YAML 示例

```yaml
# 全量更新（镜像 + 配置同时生效，重建 Workload 和 Pod）
updateMode: Full
imageTag: v1.1.0
```

```yaml
# 仅更新配置（不换镜像，重建 Workload 和 Pod，常用于 appspec 变更后重新部署）
updateMode: Config
```

```yaml
# 镜像原地更新（推荐，Pod 原地重启，不重建 Workload，影响最小）
updateMode: Image
imageTag: v1.1.0
strategy: InplaceUpdate
```

```yaml
# 滚动更新镜像（重建 Pod，适合需要优雅迁移的场景）
updateMode: Image
imageTag: v1.1.0
strategy: RollingUpdate
```

```yaml
# 灰度更新（仅更新指定 Pod，验证新版本）
updateMode: Grayscale
imageTag: v1.1.0
instanceIDs: "pod1;pod2"
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 二选一 | 环境名称（支持逗号分隔多环境） |
| `--env-type` | 二选一 | 环境类型（支持逗号分隔）：development / test / staging / production，展开为该类型下所有环境 |
| `-f, --update-spec-file` | 是 | 更新 spec 文件路径 |
| `--workspace` | 否 | 工作空间 ID |

## delete

删除（卸载）应用在指定环境的部署。此操作会从集群中移除工作负载。`--env` 支持逗号分隔多环境，`--env-type` 支持按环境类型批量卸载。

- **helm 应用**：需通过 `--deploy-id` 指定要删除的部署记录 ID（从 `deploy list` 获取），且一次仅支持单个环境。
- **trpc / taf 应用**：删除整个环境部署，无需 `--deploy-id`。

### 常用场景

```bash
# 删除 trpc/taf 应用部署
bkms-cli app deploy delete --app myapp --env test

# 删除 helm 应用的指定部署
bkms-cli app deploy delete --app myapp --env test --deploy-id deploy1

# 删除多个环境部署
bkms-cli app deploy delete --app myapp --env test,staging

# 按环境类型批量卸载
bkms-cli app deploy delete --app myapp --env-type test,development
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 二选一 | 环境名称（支持逗号分隔多环境） |
| `--env-type` | 二选一 | 环境类型（支持逗号分隔）：development / test / staging / production，展开为该类型下所有环境 |
| `--deploy-id` | helm 必填 | 部署记录 ID（helm 应用必须指定，从 `deploy list` 获取） |
