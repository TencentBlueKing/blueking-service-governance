# bkms-cli app list Reference

`bkms-cli app list` 用于列出工作空间下当前用户有权限查看的所有应用。

默认以表格形式输出应用列表摘要信息，也可通过 `--output` 参数切换为 JSON、YAML 或使用 jq 表达式提取特定字段。

## 返回字段

每条应用记录包含以下字段：

| 字段 | 说明 |
|------|------|
| `id` | 应用 ID |
| `name` | 应用名称 |
| `displayName` | 应用显示名称 |
| `type` | 应用类型（bkmsApp / trpc / taf / helm / agones） |
| `framework` | 技术栈框架（trpc / taf / blank），Helm / Agones 为空 |
| `language` | 应用语言（go / cpp / nodejs / python） |
| `creator` | 创建者 |

## 常用场景

列出当前默认工作空间下的所有应用（表格输出）。

```bash
# 使用默认工作空间，表格格式输出
bkms-cli app list
```

指定工作空间并以 JSON 格式输出完整数据。

```bash
# 指定工作空间，JSON 格式输出
bkms-cli app list --workspace ws-demo -o json
```

以 YAML 格式输出。

```bash
# YAML 格式输出
bkms-cli app list -o yaml
```

使用 jq 表达式提取特定字段，例如仅获取所有应用的 ID 列表。

```bash
# 提取所有应用 ID
bkms-cli app list -o 'jq=[.[] | .id]'

# 提取所有 trpc 类型应用的名称
bkms-cli app list -o 'jq=[.[] | select(.type == "trpc") | .name]'

# 提取第一个应用的 ID
bkms-cli app list -o 'jq=.[0].id'
```

结合 workspace 设置简化日常使用。

```bash
# 先设置默认工作空间（一次性操作）
bkms-cli workspace set ws-demo

# 之后无需每次指定 --workspace
bkms-cli app list
bkms-cli app list -o json
```
