# bkms-cli app instance Reference

`bkms-cli app instance` 用于管理应用运行中的实例（Pod）。支持查看实例列表、删除实例、调整北极星权重/隔离状态，以及执行管理命令。端口转发功能见 [app-instance-port-forward.md](app-instance-port-forward.md)。

## list

列出应用在指定环境下的所有运行实例。

### 返回字段

| 字段 | 说明 |
|------|------|
| `id` | 实例 ID（Pod 名称） |
| `ip` | Pod IP |
| `image` | 当前运行的镜像 |
| `restartCount` | 重启次数 |
| `status` | 状态（Running / Pending / Failed 等） |
| `isHealthy` | 健康状态（探针检查结果） |
| `age` | 存活时长 |
| `polarisInfos` | 北极星注册信息列表（权重、健康状态等） |

### 常用场景

```bash
# 列出实例（表格输出）
bkms-cli app instance list --app myapp --env test

# JSON 格式
bkms-cli app instance list --app myapp --env test -o json

# 提取所有 Running 实例的 ID
bkms-cli app instance list --app myapp --env test -o 'jq=[.[] | select(.status == "Running") | .id]'

# 查看特定实例的北极星信息
bkms-cli app instance list --app myapp --env test -o 'jq=[.[] | select(.id=="pod1") | .polarisInfos]'

# 按 Pod IP 精确查找实例（可重复传入，多值取 OR）
bkms-cli app instance list --app myapp --env test --pod-ip 10.0.0.1 --pod-ip 10.0.0.2

# 查找运行在指定节点上的实例
bkms-cli app instance list --app myapp --env test --node-ip 192.168.0.1
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 是 | 环境名称 |
| `--pod-ip` | 否 | 按 Pod IP 精确过滤，可重复或逗号分隔 |
| `--node-ip` | 否 | 按 Node IP 精确过滤，可重复或逗号分隔；与 `--pod-ip` 同时传时取 OR |
| `-o, --output` | 否 | 输出格式：json / yaml / table / jq=\<expr\> |

## polaris

调整指定实例的北极星流量权重或隔离状态。`--weight` 和 `--isolate` 至少传一个。

### 风险与注意事项

**权重调整（`--weight`）：**
- 权重 `0` 表示该实例不再接收新流量（已建立的连接不会立即中断）。权重降为 0 后，该实例从北极星的路由中摘除，但 **Pod 仍在运行**，适合维护前排空流量。
- 摘流后**必须手动恢复**权重（设为 100 或其他正常值），否则实例永远不再参与流量分配。
- **仅影响该实例在北极星上的注册权重**，不影响 Kubernetes 层面的流量（如 Service/Ingress 路由）。
- 权重范围约定：0 = 完全摘流，100 = 正常权重。实际业务可按比例调整，用于灰度流量分配。

**隔离状态（`--isolate`）：**
- 隔离后该实例从北极星中摘除注册（等效于权重 0 但语义更强）。
- 取消隔离（`--isolate=false`）后实例恢复注册，**权重恢复为 Pod annotation 中记录的值**。
- 隔离操作适合故障快速摘除场景，比单纯降权响应更及时。

**两者的区别：**
- `--weight 0`：保留注册，权重为 0，仍可通过直连访问该实例。
- `--isolate=true`：摘除注册，无法通过北极星路由访问该实例。

### 典型用法（运维摘流 + 恢复）

```bash
# 1. 摘流（维护前将权重降为 0）
bkms-cli app instance polaris --app myapp --env prod --instance-ids pod1 --weight 0

# 2. 确认流量排空（观察北极星权重已生效）
bkms-cli app instance list --app myapp --env prod -o 'jq=[.[] | select(.id=="pod1") | .polarisInfos]'

# 3. 执行维护操作（重启、热更新等）...

# 4. 恢复权重（切记，否则实例永远不再参与流量）
bkms-cli app instance polaris --app myapp --env prod --instance-ids pod1 --weight 100

# 快速隔离故障实例
bkms-cli app instance polaris --app myapp --env prod --instance-ids pod1 --isolate

# 故障修复后取消隔离
bkms-cli app instance polaris --app myapp --env prod --instance-ids pod1 --isolate=false

# 灰度分配：将两个实例流量降至 50%
bkms-cli app instance polaris --app myapp --env prod --instance-ids pod1,pod2 --weight 50
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 是 | 环境名称 |
| `--instance-ids` | 是 | 实例 ID，逗号分隔 |
| `--weight` | 否 | 目标权重（0=摘流，100=正常），不传则不修改 |
| `--isolate` | 否 | 隔离状态：`--isolate` 或 `--isolate=true` 设为隔离，`--isolate=false` 取消隔离，不传则不修改 |

> 与 `app polaris weight` 的区别：本命令针对**单个 Pod 实例**精细控制；`app polaris weight` 对**环境内全部实例**统一设置全局默认权重。

## delete

永久删除运行中的 Pod 实例。**此操作会同时减少副本数**，删除后实例数量将永久减少，除非手动触发部署恢复。

**警告**：此操作不可逆，请确认无误后再执行。

```bash
# 删除指定实例（会提示确认）
bkms-cli app instance delete --app myapp --env prod --instance-ids pod1,pod2
```

操作完成后建议验证当前实例数：

```bash
bkms-cli app instance list --app myapp --env prod
```

如需恢复副本数，重新触发部署：

```bash
bkms-cli app deploy create --app myapp --env prod -f deploy.yaml
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 是 | 环境名称 |
| `--instance-ids` | 是 | 实例 ID，逗号分隔 |

## list-admin-cmds

查询应用实例上可用的管理命令列表。仅支持 tRPC 框架的应用（trpc，或 framework=trpc 的 bkmsApp）。

```bash
# 查询 pod1 上可用的管理命令
bkms-cli app instance list-admin-cmds --app myapp --env test --instance-ids pod1

# 批量查询多个实例
bkms-cli app instance list-admin-cmds --app myapp --env test --instance-ids pod1,pod2
```

### 参数说明

| 参数 | 必填 | 说明 |
|------|------|------|
| `--app` | 是 | 应用 ID |
| `--env` | 是 | 环境名称 |
| `--instance-ids` | 是 | 实例 ID，逗号分隔 |
| `-o, --output` | 否 | 输出格式 |
| `--workspace` | 否 | 工作空间 ID |

## exec-admin-cmd

在应用实例上执行管理命令。根据应用框架（framework）自动路由，blank 框架不支持管理命令：
- **trpc**：需要 `--method`、`--url`；可选 `--params`、`--body`
- **taf**：需要 `--command`（如 `taf.viewversion`、`taf.setloglevel DEBUG`）

```bash
# trpc 应用：列出管理命令
bkms-cli app instance exec-admin-cmd --app myapp --env test --instance-ids pod1 --method GET --url /cmds

# trpc 应用：调用带参数的管理命令
bkms-cli app instance exec-admin-cmd --app myapp --env test --instance-ids pod1 \
  --method POST --url /config --params '{"key":"val"}' --body '{"value":"new"}'

# taf 应用：查看版本
bkms-cli app instance exec-admin-cmd --app myapp --env test --instance-ids pod1 --command "taf.viewversion"

# taf 应用：动态设置日志级别
bkms-cli app instance exec-admin-cmd --app myapp --env test --instance-ids pod1 --command "taf.setloglevel DEBUG"
```

### 参数说明

| 参数 | 适用类型 | 必填 | 说明 |
|------|----------|------|------|
| `--app` | 通用 | 是 | 应用 ID |
| `--env` | 通用 | 是 | 环境名称 |
| `--instance-ids` | 通用 | 是 | 实例 ID，逗号分隔 |
| `--method` | trpc | 条件必填 | HTTP 方法（GET/POST/PUT） |
| `--url` | trpc | 条件必填 | 管理命令 URL 路径 |
| `--params` | trpc | 否 | Query 参数（JSON 字符串） |
| `--body` | trpc | 否 | 请求体 |
| `--command` | taf | 条件必填 | taf 管理命令 |
| `-o, --output` | 通用 | 否 | 输出格式 |
| `--workspace` | 通用 | 否 | 工作空间 ID |
