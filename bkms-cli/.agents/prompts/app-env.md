# 应用环境

特性环境归属于单个应用，从标准环境复制基础信息及该应用的相关配置，
并由服务端生成环境名称和独立命名空间。源环境后续变化不会自动同步。
应用命令的 `--app` 支持应用 ID 或名称，`--workspace` 可使用已配置的默认值。

## 常用流程

先查询应用可用环境，选择标准环境创建特性环境，再使用创建结果中的
`name` 作为部署、配置和实例命令的 `--env` 参数。

```bash
bkms-cli app env list --app my-app -o json
bkms-cli app env create --app my-app --source-env staging --display-name "登录改造" -o json

# 将 <返回的 name> 替换为创建结果中的 name，不是展示名称或 ID
bkms-cli app deploy create --app my-app --env <返回的 name> -f deploy.yaml
bkms-cli app deploy list --app my-app --env <返回的 name>
```

## app env list

返回工作空间标准环境以及当前应用拥有的特性环境。
`kind` 为 `standard` 或 `feature`；特性环境还包含 `ownerAppID` 和 `sourceEnvID`。
`status` 表示环境就绪状态（Ready / NotReady），不是应用部署状态。
默认表格展示名称、展示名、类别、类型和就绪状态；JSON/YAML 保留来源 ID、应用归属和集群等完整信息。
`--kind standard|feature` 按环境类别筛选；部署信息使用 `app deploy list` 查询。
`env list` 仍用于查询工作空间标准环境。

```bash
bkms-cli app env list --workspace ws-demo --app my-app
bkms-cli app env list --app my-app --kind feature
bkms-cli app env list --app my-app --kind standard
bkms-cli app env list --app my-app -o 'jq=[.[] | select(.kind == "feature") | .name]'
```

## app env create

此命令仅创建特性环境。工作空间标准环境使用顶层 `env create` 创建。
`--source-env` 接受源标准环境的名称或 ID，`--display-name` 为必填展示名称。
特性环境不能用作源环境，服务端负责完整业务校验和命名空间分配。
创建结果包含新环境的 `id`、`name`、`kind`、来源 ID 和集群信息。

```bash
bkms-cli app env create --app my-app --source-env staging --display-name "登录改造" -o json
```

创建和列表命令支持 `-o json|yaml|table|jq=<表达式>`。

## app env delete

按名称或 ID 删除当前应用的特性环境。先从应用可用环境中定位目标并校验类别及归属，
因此不能通过此命令删除标准环境或其他应用的环境。
默认展示应用、环境和集群信息并要求确认；脚本可使用 `--yes` 跳过确认。

```bash
bkms-cli app env delete --app my-app --env feat-1
bkms-cli app env delete --app my-app --env feat-1 --yes
```

删除前必须先卸载该环境中的应用部署，服务端会拒绝删除仍有部署应用的环境。
此命令复用现有环境删除接口；当前服务端不会回收 Kubernetes namespace，
因此删除环境不等于完整销毁所有集群资源。
