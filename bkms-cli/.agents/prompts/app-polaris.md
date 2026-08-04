# bkms-cli app polaris Reference

`bkms-cli app polaris` 用于管理应用的北极星（Polaris）服务注册配置。包含以下子命令：

- `list`：列出指定应用的所有北极星配置。
- `create`：从 YAML spec 文件创建新的北极星配置。
- `update`：从 YAML spec 文件更新已有的北极星配置（部分更新）。
- `delete`：删除指定的北极星配置。

北极星配置是应用级别的依赖配置，用于为应用注册北极星服务。`app polaris` 按应用维度管理配置，不提供 `--env` 命令行参数。

配置的环境范围通过数据字段表达：
- `create` 时在 YAML 中使用 `scopeEnvNames` 指定生效环境；
- `list` 返回中通过 `scopeEnvNames` 展示生效环境；
- `update` 可通过 `scope.scopeEnvNames` 调整生效环境。

## list

`list` 命令负责列出指定应用的所有北极星配置。

### 返回字段

每条北极星配置记录包含以下字段：

| 字段 | 说明 |
|------|------|
| `appID` | 所属应用 ID |
| `name` | 配置名称（应用内唯一） |
| `instanceKey` | 组件实例标识，用于环境变量拼接 |
| `polarisName` | 北极星实例名称 |
| `polarisNamespace` | 北极星命名空间（Test / Production / Development / Pre-release） |
| `polarisToken` | 北极星 Token（返回时脱敏） |
| `servicePort` | 服务端口 |
| `direct` | 是否为直连模式 |
| `keepNotReadyPod` | 是否保留未就绪的 Pod |
| `enableHealthCheck` | 是否启用健康检查 |
| `weight` | 服务权重 |
| `serviceLabels` | 服务标签 |
| `scopeType` | 生效范围类型 |
| `scopeEnvNames` | 生效的环境列表 |
| `operator` | 负责人 |
| `createdAt` | 创建时间 |
| `updatedAt` | 更新时间 |

### 常用场景

```bash
# 列出应用的所有北极星配置（表格输出）
bkms-cli app polaris list --app my-app

# JSON 格式输出
bkms-cli app polaris list --app my-app -o json

# YAML 格式输出
bkms-cli app polaris list --app my-app -o yaml

# 使用 jq 表达式提取所有配置名称
bkms-cli app polaris list --app my-app -o 'jq=[.[] | .name]'

# 提取所有生效在 prod 环境的配置
bkms-cli app polaris list --app my-app -o 'jq=[.[] | select(.scopeEnvNames | index("prod"))]'

# 提取第一个配置的服务端口
bkms-cli app polaris list --app my-app -o 'jq=.[0].servicePort'
```

## create

`create` 命令负责从 YAML spec 文件创建新的北极星配置。YAML spec 文件结构与后端 API 请求体一致。

**注意：创建配置后需要触发一次部署才能在集群中生效。**

创建成功后输出配置名称：

```
✓ Polaris config created successfully
  Name: polaris-a1b2c
```

### 常用场景

```bash
# 从 YAML spec 文件创建北极星配置
bkms-cli app polaris create --app my-app -f polaris.yaml
```

### 完整 YAML 示例

#### 使用已有北极星服务（手动指定 Token）

```yaml
# 关联已有的北极星服务
scopeEnvNames:
  - prod
instanceKey: my_polaris
polarisName: my-service
polarisNamespace: Production
polarisToken: "your-polaris-token-here"
servicePort: 8080
direct: false
keepNotReadyPod: true
enableHealthCheck: false
weight: 10
serviceLabels:
  env: prod
  team: backend
```

#### 由平台创建新的北极星服务

```yaml
# 平台自动创建北极星服务并回填 Token
createNewService: true
scopeEnvNames:
  - test
  - staging
instanceKey: auto_polaris
polarisName: my-new-service
polarisNamespace: Test
servicePort: 9090
direct: true
keepNotReadyPod: true
enableHealthCheck: true
weight: 20
operator: zhangsan
```

#### 最小化配置（仅必填字段）

```yaml
scopeEnvNames:
  - prod
instanceKey: svc_polaris
polarisName: my-service
polarisNamespace: Production
polarisToken: "xxxx"
servicePort: 8080
```

### YAML 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `scopeEnvNames` | []string | 是 | 生效的环境名称列表，配置仅在这些环境中生效 |
| `instanceKey` | string | 是 | 组件实例标识，用于环境变量前缀（如 `my_polaris` 会生成 `my_polaris_polarisToken` 等环境变量）。字母/数字/下划线，必须字母开头 |
| `polarisName` | string | 是 | 北极星服务名称 |
| `polarisNamespace` | string | 是 | 北极星命名空间：Test / Production / Development / Pre-release |
| `polarisToken` | string | 条件必填 | 北极星 Token（`createNewService` 为 false 时必填；为 true 时由平台自动创建并回填） |
| `servicePort` | int | 是 | 应用监听的服务端口（1-65535），将注册到北极星 |
| `createNewService` | bool | 否 | 为 true 时平台自动创建新北极星服务并回填 Token；为 false（默认）时需提供已有的 polarisToken |
| `direct` | bool | 否 | 直连模式（默认 false）。启用后将 Pod IP 直接注册到北极星，每个 Pod 的 IP:port 作为独立的北极星实例 |
| `keepNotReadyPod` | bool | 否 | 保留未就绪 Pod（默认 true）。为 true 时未就绪 Pod 以 0 权重保留在北极星实例列表中；为 false 时未就绪 Pod 会立即从北极星注销 |
| `enableHealthCheck` | bool | 否 | 启用北极星健康检查（默认 false）。启用后北极星会主动探测实例健康状态 |
| `weight` | int | 否 | 服务权重（默认 10）。作用于该服务的所有注册实例，权重越高分配的流量越多 |
| `serviceLabels` | map | 否 | 服务标签（key-value 对）。作用于所有注册的北极星实例，可用于北极星路由规则和流量管理 |
| `operator` | string | 否 | 操作人/负责人（仅 `createNewService` 为 true 时有效） |

## delete

`delete` 命令负责删除指定应用的某个北极星配置。

**注意：删除配置后需要触发一次部署才能生效，届时已注册的实例将从北极星注销。**

删除成功后输出：

```
✓ Polaris config deleted successfully
  Name: polaris-a1b2c
```

### 常用场景

```bash
# 删除指定的北极星配置
bkms-cli app polaris delete --app my-app --name polaris-cfg-abc123
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--name` | 是 | 要删除的北极星配置名称（可通过 `list` 命令获取） |

## update

`update` 命令负责从 YAML spec 文件更新已有的北极星配置。仅 YAML 文件中存在的字段会被更新，未出现的字段保持不变（部分更新语义）。

**生效时机：**
- `servicePort` 和 `polarisToken`：修改后需要重新部署才能生效
- 其他字段（`direct`、`keepNotReadyPod`、`enableHealthCheck`、`weight`、`serviceLabels`、`instanceKey`、`scope`）：修改后立即生效，无需重新部署

更新成功后输出：

```
✓ Polaris config updated successfully
  Name: polaris-a1b2c
```

### 常用场景

```bash
# 更新北极星配置
bkms-cli app polaris update --app my-app --name polaris-cfg-abc -f update.yaml
```

### YAML 示例

#### 更新服务端口和权重

```yaml
servicePort: 9090
weight: 20
```

#### 更新生效环境范围

```yaml
scope:
  scopeEnvNames:
    - prod
    - staging
```

#### 更新服务标签（全量替换）

```yaml
serviceLabels:
  version: v2
  region: shenzhen
```

#### 更新多个字段

```yaml
servicePort: 9090
direct: true
keepNotReadyPod: false
enableHealthCheck: true
weight: 30
polarisToken: "new-token-value"
```

### 可更新字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `servicePort` | int | 服务端口（1-65535）。修改后需重新部署才能在集群中生效 |
| `direct` | bool | 直连模式。启用后将 Pod IP 直接注册到北极星 |
| `keepNotReadyPod` | bool | 保留未就绪 Pod。为 false 时未就绪 Pod 会立即从北极星注销 |
| `enableHealthCheck` | bool | 启用北极星健康检查 |
| `weight` | int | 服务权重。作用于所有注册实例 |
| `serviceLabels` | map | 服务标签。传入时全量替换（不是合并） |
| `instanceKey` | string | 组件实例标识。修改后需重新部署才能生效 |
| `scope` | object | 生效范围。包含 `scopeEnvNames`（环境列表），`scopeType` 会自动设置为 `"environment"` |
| `polarisToken` | string | 北极星 Token。修改后需重新部署才能生效 |

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--name` | 是 | 要更新的北极星配置名称（可通过 `list` 命令获取） |
| `-f, --file` | 是 | YAML spec 文件路径 |
