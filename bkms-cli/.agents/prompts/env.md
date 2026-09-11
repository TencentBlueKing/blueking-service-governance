# bkms-cli env Reference

`bkms-cli env` 用于管理 BKMS 环境。环境是应用部署的目标，关联到特定的集群和命名空间。环境类型分为：`development`、`test`、`staging`、`production`。**环境名称（name）创建后不可修改**（无 `--name` 更新接口）；类型（type）可通过 `env update --type` 修改。

## list

列出工作空间下的标准环境。

默认及 table 输出仅展示名称、展示名、类型、就绪状态和类别；
使用 `-o json` 或 `-o yaml` 查看 ID、集群等完整字段。

如果要查询某个应用可用的全部环境（含专属特性环境），使用 `bkms-cli app env list --app <应用>`
创建和管理特性环境见 [应用环境](app-env.md)。

### 返回字段

| 字段 | 说明 |
|------|------|
| `id` | 环境 ID |
| `name` | 环境名称（不可变） |
| `displayName` | 环境显示名称 |
| `type` | 环境类型（development / test / staging / production） |
| `description` | 环境描述 |
| `cluster.clusterID` | 集群 ID |
| `cluster.clusterType` | 集群类型 |
| `cluster.namespace` | Kubernetes 命名空间 |
| `cluster.projectCode` | 项目 Code |
| `updatedAt` | 最近更新时间 |

### 常用场景

```bash
# 列出默认工作空间的所有环境
bkms-cli env list

# 指定工作空间
bkms-cli env list --workspace ws-demo

# JSON 格式
bkms-cli env list -o json

# 提取所有环境名称
bkms-cli env list -o 'jq=[.[] | .name]'

# 查找 production 类型环境
bkms-cli env list -o 'jq=[.[] | select(.type == "production")]'
```

## create

从 YAML spec 文件创建新环境。**name 创建后不可修改（无更名接口），type 可通过 `env update --type` 修改，请仍谨慎填写。**

```bash
# 从 YAML 文件创建（需要 --workspace）
bkms-cli env create --workspace ws-demo -f env.yaml
```

### YAML spec 示例

```yaml
# 环境名称（字母/数字/中划线，1-20 字符，创建后不可改）
name: staging
# 显示名称（可后续修改）
displayName: Staging Env
# 类型：development | test | staging | production
type: staging
description: Staging environment for QA validation
cluster:
  # BCS 集群 ID（从 BCS 平台获取）
  clusterID: BCS-K8S-12345
  # BCS 集群类型（见下方说明）
  clusterType: single
  # Kubernetes 命名空间
  namespace: bkms-staging
```

### YAML 字段说明

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` | 是 | 环境名称，创建后不可改 |
| `displayName` | 是 | 显示名称 |
| `type` | 是 | 类型：development / test / staging / production（可通过 `env update --type` 修改）|
| `description` | 否 | 描述 |
| `cluster.clusterID` | 是 | BCS 集群 ID，从 BCS 平台集群列表获取 |
| `cluster.clusterType` | 是 | BCS 集群类型（见下方说明） |
| `cluster.namespace` | 是 | Kubernetes 命名空间 |

**`cluster.clusterType` 说明**

该值与 BCS 平台集群的实际类型对应，需与所用集群一致。已知取值：

| 值 | 说明 |
|----|------|
| `single` | 独立集群（最常见） |
| `virtual` | 虚拟集群 |

> 实际可用值以 BCS 平台返回为准。填写前建议在 BCS 控制台查看目标集群的类型字段，或通过 `env list -o json` 查看已有环境的 `cluster.clusterType` 值作为参考。

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--workspace` | 是 | 工作空间 ID |
| `-f, --file` | 是 | YAML spec 文件路径 |

## get

通过环境名称查看环境完整详情。

```bash
# 查看环境详情（表格）
bkms-cli env get --workspace ws-demo --env staging

# YAML 格式
bkms-cli env get --workspace ws-demo --env staging -o yaml

# JSON 格式
bkms-cli env get --workspace ws-demo --env staging -o json
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--workspace` | 是 | 工作空间 ID |
| `--env` | 是 | 环境名称 |
| `-o, --output` | 否 | 输出格式：json / yaml / table / jq=\<expr\> |

## update

更新环境的显示名称或类型。**name 不可修改。**`--display-name` 和 `--type` 至少传一个。

```bash
# 更新显示名称
bkms-cli env update --workspace ws-demo --env staging --display-name "New Display Name"

# 切换环境类型（如从 test 切换为 staging）
bkms-cli env update --workspace ws-demo --env staging --type production
```

**注意**：将类型改为 `production` 后，`port-forward` 和 `publish` 等开发类操作将被限制。

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--workspace` | 是 | 工作空间 ID |
| `--env` | 是 | 环境名称 |
| `--display-name` | 否 | 新显示名称 |
| `--type` | 否 | 新类型：development / test / staging / production |

## delete

永久删除环境。**不可逆操作**，删除前请确保该环境的所有部署已清理。

```bash
# 删除环境（会提示确认）
bkms-cli env delete --workspace ws-demo --env staging
```

建议删除前先检查该环境是否有活跃部署：

```bash
bkms-cli app deploy list --app myapp --env staging
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--workspace` | 是 | 工作空间 ID |
| `--env` | 是 | 环境名称 |
