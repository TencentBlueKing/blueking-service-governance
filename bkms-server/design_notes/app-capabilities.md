# 应用类型能力现状梳理

本文梳理各应用类型（helm / agones / trpc / taf，及规划中的 standard）当前具备的能力，独立于具体类型的
设计文档，供设计新类型时对照。

## 能力清单（全量）

按「归属」分四类：

| 归属 | 能力 | 现状（适用范围） |
|------|------|----------------|
| 通用基座 | 应用生命周期：创建/查询/列表/删除、显示名、ID 后缀、组件管理 | 各类型通用 |
| 通用基座 | 构建：构建配置、Tag 策略、开始构建、构建记录/日志、推荐 Tag | 各类型通用 |
| 通用基座 | 配置文件：CRUD、内容编辑、版本管理、overlay/base、按环境、local/BSCP、审计 | 各类型通用 |
| 通用基座 | 部署：AppModel 部署（记录/状态/快照/下架）、预检、总览、扩缩容、灰度、批量删除 | 仅 AppModel 类应用 |
| 通用基座 | 环境与泳道：标准环境 CRUD、特性环境、泳道列表、部署状态聚合 | 各类型通用 |
| 通用基座 | 工作负载渲染：GameDeployment、规格、配额、探针、生命周期、卷、更新策略、副本、拉取密钥、labels/annotations、Route ENI、组件、联邦转换 | 仅 AppModel 类应用 |
| 通用基座 | 实例：列表/watch、WebConsole、端口转发、实例日志、环境事件 | 仅 AppModel 类应用 |
| 通用基座 | 环境变量：应用级、scoped、内置、运行时、导入导出、依赖服务 | 应用级仅 AppModel 类，scoped 各类型通用 |
| 通用基座 | AppSpec：resources/updateStrategy/probe/lifecycle/labels/annotations/tkeRouteEni + 三段式 | 仅 AppModel 类应用 |
| 通用基座 | 组件：定义 CRUD、预览、内置组件 | 仅 AppModel 类应用 |
| 通用基座 | 可观测：拓扑、监控指标、告警策略、APM 实例 | 各类型通用 |
| 通用基座 | 镜像：列表、快照、晋级、tag 占用/删除、部署记录、builder/runner、自定义镜像 | 各类型通用 |
| 开放能力 | polaris：注册/权重/隔离/统计 | 当前耦合 trpc yaml（patch/校验） |
| 开放能力 | devmode：开发模式热更新 | 当前 trpc/taf 两套路径/脚本 |
| 开放能力 | BSCP：配置元信息、环境绑定与下发 | 当前仅 trpc 注入 workloadKind |
| 开放能力 | GPA 自动扩缩容 | 当前仅经部署总览暴露 |
| 开放能力 | HostPort：联邦集群随机端口映射 | 仅联邦环境生效 |
| 开放能力 | 端口池：PortPool 管理 | 环境维度 |
| 开放能力 | 平台通用构建 platform | 当前仅 trpc-go |
| 开放能力 | 一键构建部署 / 构建触发 | 当前 trpc/taf 各一份，触发为 stub |
| 框架特有 | 框架配置文件及渲染（ConfigMap + init 容器） | 仅 trpc/taf |
| 框架特有 | admin 命令（trpc HTTP / taf 私有协议） | 仅 trpc/taf |
| 框架特有 | APM 服务名提取（trpc yaml / taf xml） | 仅 trpc/taf |
| 框架特有 | tRPC 服务名解析 / Telemetry 解析 | 仅 trpc/taf |
| Helm 特有 | chart 来源/values/部署/chart 构建/制品/semver/PostRenderer/Service 同步/部署锁 | 仅 helm/agones |

## 各类型特有能力清单

| 能力 | helm | agones | trpc | taf | standard（规划） |
|------|:---:|:---:|:---:|:---:|:---:|
| Helm chart 来源（Helm/BCS/Git） | ✅ | ✅ | - | - | - |
| values 文件（默认 default） | ✅ | ✅ | - | - | - |
| Helm 部署（install/rollback） | ✅ | ✅ | - | - | - |
| GameDeployment 部署 | - | - | ✅ | ✅ | ✅ |
| 框架配置文件（framework） | - | - | ✅ | ✅ | - |
| plain 配置文件 | - | - | -（迁移中） | -（迁移中） | ✅ |
| admin 命令 | - | - | ✅ | ✅ | - |
| APM 服务名提取 | - | - | ✅ | ✅ | - |
| polaris（开放类，待下沉） | - | - | ✅（YAML patch） | - | 待支持 |
| devmode（开放类，待抽象） | - | - | ✅ | ✅ | 待支持 |
| 语言子字段 | - | - | go/cpp | - | go/python/node |
| AppSpec / envVars / 组件 | - | - | ✅ | ✅ | ✅ |
| 一键构建部署（构建+自动部署） | - | - | ✅ | ✅ | 待加 |
| Helm chart 构建 / 制品 / semver | ✅ | ✅ | - | - | - |
| 实例操作（扩缩容/灰度/批量删除/端口转发） | - | - | ✅ | ✅ | ✅ |
