# bkms-cli app-appspec Reference

`bkms-cli app appspec` 用于管理应用部署规格（AppSpec）的各个配置段（section），包括启动命令、资源限制、更新策略、生命周期钩子、健康探针、underlay IP 网络、开发模式、标签和注解。

当前包含以下子命令组：

- `view`：聚合查看所有 section 的配置。
- `start-command`：管理应用启动命令（view / edit）。
- `resources`：管理容器资源配置（view / edit / reset）。
- `update-strategy`：管理滚动更新策略（view / edit / reset）。
- `lifecycle`：管理容器生命周期钩子（view / edit / reset）。
- `probe`：管理健康探针配置（view / edit / reset）。
- `underlay-ip`：管理 underlay IP（VPC-CNI）网络模式开关（view / enable / disable / reset）。
- `dev-mode`：管理开发模式开关，**仅支持环境级配置**（view / enable / disable / reset），所有子命令必须指定 `--env`。
- `labels`：管理 Kubernetes 标签（view / edit / reset）。
- `annotations`：管理 Kubernetes 注解（view / edit / reset）。

**默认配置 vs 环境配置：**

- 不带 `--env`：操作的是应用级默认配置，所有环境共享。
- 带 `--env`：操作的是环境级覆盖配置。view 时展示合并后的生效配置；edit / enable / disable 时修改环境覆盖层；reset 时删除环境覆盖（恢复为使用默认配置）。
- `start-command` 是全局配置，不区分环境，没有 `--env` 参数和 reset 操作。
- `dev-mode` 只有环境级配置，服务端没有应用默认级接口，因此所有 `dev-mode` 子命令都必须指定 `--env`。聚合 `view` 也只在带 `--env` 时才展示 dev mode。

**开关型 section：**

`underlay-ip` 与 `dev-mode` 只有一个 `enabled` 布尔开关，用 `enable` / `disable` 子命令切换，**不提供 `edit -f`**。其余 section 仍通过 `edit -f <yaml>` 修改。

**输出格式：**

所有 view 命令默认以表格形式输出，支持 `-o json`、`-o yaml`、`-o 'jq=<expr>'` 三种格式化输出。

## 常用场景

查看应用的所有 AppSpec 配置。

```bash
# 查看默认配置（表格格式）
bkms-cli app appspec view --app my-app

# 查看 prod 环境生效配置（JSON 格式）
bkms-cli app appspec view --app my-app --env prod -o json
```

修改单个 section 的配置，先编写 YAML 文件再用 edit 写入。

```bash
# 修改默认资源配置
bkms-cli app appspec resources edit --app my-app -f resources.yaml

# 修改 prod 环境的更新策略
bkms-cli app appspec update-strategy edit --app my-app --env prod -f update-strategy.yaml
```

重置环境覆盖配置，使环境恢复使用默认配置。

```bash
# 重置 prod 环境的资源配置为默认
bkms-cli app appspec resources reset --app my-app --env prod
```

开关型 section 用 enable / disable 切换，无需 YAML 文件。

```bash
# 在默认配置上开启 underlay IP（所有环境共享）
bkms-cli app appspec underlay-ip enable --app my-app

# 只在 prod 环境开启 underlay IP
bkms-cli app appspec underlay-ip enable --app my-app --env prod

# 开启 stag 环境的开发模式（dev-mode 必须带 --env）
bkms-cli app appspec dev-mode enable --app my-app --env stag
```

使用 jq 表达式提取特定字段。

```bash
# 提取当前副本数
bkms-cli app appspec resources view --app my-app -o 'jq=.replicas'

# 提取启动命令
bkms-cli app appspec start-command view --app my-app -o 'jq=.command'

# 判断 underlay IP 是否已开启
bkms-cli app appspec underlay-ip view --app my-app -o 'jq=.enabled'
```

## view（聚合查看）

`view` 命令一次性查看应用所有 AppSpec section 的配置。

```bash
# 查看默认配置
bkms-cli app appspec view --app my-app

# 查看 prod 环境生效配置
bkms-cli app appspec view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec view --app my-app -o json

# YAML 格式输出
bkms-cli app appspec view --app my-app --env prod -o yaml

# 使用 jq 表达式提取资源配置
bkms-cli app appspec view --app my-app -o 'jq=.resources'
```

## start-command

`start-command` 管理应用的容器启动命令和参数，或一段完整的 shell 启动脚本。该命令为全局配置，不区分环境。

### start-command view

```bash
# 查看启动命令
bkms-cli app appspec start-command view --app my-app

# JSON 格式输出
bkms-cli app appspec start-command view --app my-app -o json
```

### start-command edit

从 YAML 文件更新启动命令。`--app` 和 `-f/--file` 均为必填。

- `type=bkmsApp` 的应用走 startup-config 接口，`command` / `args` / `shCommand` **整体替换**，YAML 里没写的字段会被清空；
- 旧的 trpc / taf 应用走各自的 spec 接口，当前的 trpcSpec/tafSpec 会自动保留，除非在 YAML 中显式覆盖。

```bash
# 从 YAML 文件更新启动命令（-f 必填）
bkms-cli app appspec start-command edit --app my-app -f start-command.yaml
```

YAML 文件格式：

```yaml
# 基本格式
command:
  - /usr/local/trpc/bin/container-start.sh
args:
  - -conf
  - /usr/local/trpc/bin/trpc-go.yaml

# shell 脚本格式：以 /bin/sh 执行，支持多行、变量、管道等；与 command/args 互斥，写入一种会替换另一种
# 开发模式下 tRPC 应用需在脚本中以绝对路径启动二进制
# shCommand: |
#   export GOTRACEBACK=crash
#   /usr/local/trpc/bin/server -conf /usr/local/trpc/conf/trpc_go.yaml

# 带 trpcSpec 覆盖（可选，通常无需手动指定）
# command:
#   - ./server
# args:
#   - --config
#   - /app/conf/trpc_go.yaml
# trpcSpec:
#   language: go
#   fileName: trpc_go.yaml
#   filePath: /app/conf
```

## resources

`resources` 管理容器的资源配额（CPU、内存、副本数）。

### resources view

```bash
# 查看默认资源配置
bkms-cli app appspec resources view --app my-app

# 查看 prod 环境生效资源配置
bkms-cli app appspec resources view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec resources view --app my-app -o json
```

### resources edit

从 YAML 文件更新资源配置。

```bash
# 修改默认资源配置
bkms-cli app appspec resources edit --app my-app -f resources.yaml

# 修改 prod 环境资源配置
bkms-cli app appspec resources edit --app my-app --env prod -f resources.yaml
```

YAML 文件格式：

```yaml
replicas: 3
cpuRequests: "500m"
cpuLimits: "2000m"
memoryRequests: "512Mi"
memoryLimits: "2Gi"
```

### resources reset

删除环境级资源覆盖配置，恢复为使用默认配置。必须指定 `--env`。

```bash
# 重置 prod 环境资源配置
bkms-cli app appspec resources reset --app my-app --env prod
```

## update-strategy

`update-strategy` 管理滚动更新策略（maxSurge、maxUnavailable）。

### update-strategy view

```bash
# 查看默认更新策略
bkms-cli app appspec update-strategy view --app my-app

# 查看 prod 环境生效更新策略
bkms-cli app appspec update-strategy view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec update-strategy view --app my-app -o json
```

### update-strategy edit

从 YAML 文件更新滚动更新策略。

```bash
# 修改默认更新策略
bkms-cli app appspec update-strategy edit --app my-app -f update-strategy.yaml

# 修改 prod 环境更新策略
bkms-cli app appspec update-strategy edit --app my-app --env prod -f update-strategy.yaml
```

YAML 文件格式：

```yaml
maxSurge: "25%"
maxUnavailable: "25%"
```

### update-strategy reset

删除环境级更新策略覆盖配置。必须指定 `--env`。

```bash
# 重置 prod 环境更新策略
bkms-cli app appspec update-strategy reset --app my-app --env prod
```

## lifecycle

`lifecycle` 管理容器生命周期钩子（postStart、preStop）和优雅终止时间。

### lifecycle view

```bash
# 查看默认生命周期配置
bkms-cli app appspec lifecycle view --app my-app

# 查看 prod 环境生效配置
bkms-cli app appspec lifecycle view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec lifecycle view --app my-app -o json
```

### lifecycle edit

从 YAML 文件更新生命周期钩子配置。

```bash
# 修改默认生命周期配置
bkms-cli app appspec lifecycle edit --app my-app -f lifecycle.yaml

# 修改 prod 环境生命周期配置
bkms-cli app appspec lifecycle edit --app my-app --env prod -f lifecycle.yaml
```

YAML 文件格式：

```yaml
postStart:
  # EXEC | HTTP
  type: EXEC
  exec:
    command: ["/bin/sh", "-c", "echo hello"]
preStop:
  type: EXEC
  exec:
    sleepSeconds: 5
    shCommand: "echo shutting down"
terminationGracePeriodSeconds: 30

# Handler 类型说明：
#   EXEC - exec.command (字符串数组) 或 exec.shCommand (shell 命令字符串)
#          exec.sleepSeconds (可选, >= 0, 在执行命令前等待的秒数)
#   HTTP - http.url (请求路径), http.headers (可选, 请求头)
```

HTTP 类型示例：

```yaml
preStop:
  type: HTTP
  http:
    url: /shutdown
    headers:
      X-Shutdown-Token: "abc123"
terminationGracePeriodSeconds: 60
```

### lifecycle reset

删除环境级生命周期覆盖配置。必须指定 `--env`。

```bash
# 重置 prod 环境生命周期配置
bkms-cli app appspec lifecycle reset --app my-app --env prod
```

## probe

`probe` 管理容器健康探针（liveness、readiness、startup）。

### probe view

```bash
# 查看默认探针配置
bkms-cli app appspec probe view --app my-app

# 查看 prod 环境生效探针配置
bkms-cli app appspec probe view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec probe view --app my-app -o json
```

### probe edit

从 YAML 文件更新健康探针配置。

```bash
# 修改默认探针配置
bkms-cli app appspec probe edit --app my-app -f probe.yaml

# 修改 prod 环境探针配置
bkms-cli app appspec probe edit --app my-app --env prod -f probe.yaml
```

YAML 文件格式：

```yaml
liveness:
  handler:
    # EXEC | HTTP | TCP
    type: HTTP
    url: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  failureThreshold: 3
readiness:
  handler:
    type: EXEC
    command: ["/bin/sh", "-c", "cat /tmp/ready"]
  periodSeconds: 10
startup:
  handler:
    type: TCP
    port: 8080
  failureThreshold: 30
  periodSeconds: 10

# Handler 类型说明：
#   EXEC - command (字符串数组) 或 shCommand (shell 命令字符串)
#   HTTP - url (请求路径), port (端口), headers (可选)
#   TCP  - port (端口)
#
# 通用阈值字段：
#   initialDelaySeconds - 容器启动后首次探测前的等待秒数
#   periodSeconds       - 探测间隔秒数
#   failureThreshold    - 连续失败多少次后判定为不健康
#   successThreshold    - 连续成功多少次后判定为健康（仅 readiness）
#   timeoutSeconds      - 单次探测超时秒数
```

### probe reset

删除环境级探针覆盖配置。必须指定 `--env`。

```bash
# 重置 prod 环境探针配置
bkms-cli app appspec probe reset --app my-app --env prod
```

## labels

`labels` 管理 Kubernetes Pod 标签。

### labels view

```bash
# 查看默认标签配置
bkms-cli app appspec labels view --app my-app

# 查看 prod 环境生效标签配置
bkms-cli app appspec labels view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec labels view --app my-app -o json
```

### labels edit

从 YAML 文件更新标签配置。

```bash
# 修改默认标签配置
bkms-cli app appspec labels edit --app my-app -f labels.yaml

# 修改 prod 环境标签配置
bkms-cli app appspec labels edit --app my-app --env prod -f labels.yaml
```

YAML 文件格式：

```yaml
labels:
  app.kubernetes.io/team: platform
  app.kubernetes.io/version: v1.2.0
```

### labels reset

删除环境级标签覆盖配置。必须指定 `--env`。

```bash
# 重置 prod 环境标签配置
bkms-cli app appspec labels reset --app my-app --env prod
```

## annotations

`annotations` 管理 Kubernetes Pod 注解。

### annotations view

```bash
# 查看默认注解配置
bkms-cli app appspec annotations view --app my-app

# 查看 prod 环境生效注解配置
bkms-cli app appspec annotations view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec annotations view --app my-app -o json
```

### annotations edit

从 YAML 文件更新注解配置。

```bash
# 修改默认注解配置
bkms-cli app appspec annotations edit --app my-app -f annotations.yaml

# 修改 prod 环境注解配置
bkms-cli app appspec annotations edit --app my-app --env prod -f annotations.yaml
```

YAML 文件格式：

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "8080"
```

### annotations reset

删除环境级注解覆盖配置。必须指定 `--env`。

```bash
# 重置 prod 环境注解配置
bkms-cli app appspec annotations reset --app my-app --env prod
```

## underlay-ip

管理 underlay IP（VPC-CNI）网络模式开关。开启后 Pod 直接从 VPC 获取 IP，而不使用 overlay 网络。

该 section 只有一个 `enabled` 布尔开关，支持默认级与环境级配置。

### underlay-ip view

查看 underlay IP 配置。不带 `--env` 查看默认配置，带 `--env` 查看该环境生效配置。

```bash
# 查看默认 underlay IP 配置
bkms-cli app appspec underlay-ip view --app my-app

# 查看 prod 环境生效配置
bkms-cli app appspec underlay-ip view --app my-app --env prod

# JSON 格式输出
bkms-cli app appspec underlay-ip view --app my-app -o json
```

未配置时 `enabled` 为 `null`，表格中展示为 `-`。

### underlay-ip enable / disable

开启或关闭 underlay IP。不带 `--env` 修改默认配置，带 `--env` 修改该环境的覆盖配置。

```bash
# 在默认配置上开启（所有未单独覆盖的环境都会生效）
bkms-cli app appspec underlay-ip enable --app my-app

# 在默认配置上关闭
bkms-cli app appspec underlay-ip disable --app my-app

# 只在 prod 环境开启
bkms-cli app appspec underlay-ip enable --app my-app --env prod

# 只在 prod 环境关闭（默认配置为开启时，用它单独屏蔽某个环境）
bkms-cli app appspec underlay-ip disable --app my-app --env prod
```

### underlay-ip reset

删除环境级覆盖配置，使该环境恢复使用默认配置。必须指定 `--env`。

```bash
# 重置 prod 环境，恢复继承默认配置
bkms-cli app appspec underlay-ip reset --app my-app --env prod
```

## dev-mode

管理开发模式开关。**开发模式只支持环境级配置**，服务端没有应用默认级接口，因此所有子命令都必须指定 `--env`；缺少 `--env` 时命令会直接报错退出。

`bkms-cli app publish` 要求目标环境已开启开发模式，否则会拒绝执行。

### dev-mode view

查看某环境生效的开发模式配置。

```bash
# 查看 stag 环境的开发模式配置
bkms-cli app appspec dev-mode view --app my-app --env stag

# JSON 格式输出
bkms-cli app appspec dev-mode view --app my-app --env stag -o json
```

输出包含三个字段：

- `enabled`：是否已开启开发模式。
- `workPath`：开发模式工作根目录。
- `mountPath`：脚本挂载路径。

其中 `workPath` 与 `mountPath` 由服务端按应用框架（trpc / taf）推导，**只读**，CLI 不提供修改入口。

### dev-mode enable / disable

开启或关闭某环境的开发模式。`--env` 必填。

```bash
# 开启 stag 环境的开发模式
bkms-cli app appspec dev-mode enable --app my-app --env stag

# 关闭 stag 环境的开发模式
bkms-cli app appspec dev-mode disable --app my-app --env stag
```

### dev-mode reset

清除某环境的开发模式配置。必须指定 `--env`。

由于开发模式没有应用默认级配置，reset 是直接清除该环境的设置，而不是回退到某个默认值。

```bash
# 清除 stag 环境的开发模式配置
bkms-cli app appspec dev-mode reset --app my-app --env stag
```
