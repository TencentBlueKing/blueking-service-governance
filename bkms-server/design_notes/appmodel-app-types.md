# AppModel 应用类型体系

## 概述

本文描述 AppModel 大类的应用类型体系：trpc / taf 是框架特化类型，standard 是新增的「语言无关的通用
应用类型」，支持 go / python / nodejs 等语言。standard 复用 tRPC 应用的 AppModel + GameDeployment
部署链路，但**不绑定任何特定框架**。

与现有类型定位对照：

| 类型 | 大类 | 定义方式 |
|------|------|---------|
| helm / agones | Helm 大类 | Helm Chart + values |
| trpc / taf | AppModel 大类（框架特化） | 框架 + 框架配置文件 + appmodel |
| **standard** | AppModel 大类（通用基座） | 纯 appmodel + plain 配置文件 |

命名定为 `standard`，挂进 AppModel 大类，语言做子字段（`standardSpec.language`）。

## 与 AppModel 大类的关系

平台应用分两大族：**AppModel 族**（trpc/taf/standard）与 **Helm 族**（helm/agones）。

trpc / taf 本质上可视为 **standard 的「框架特化」版本**——它们在通用的 AppModel 基座（workload /
envVars / AppSpec / 组件 / GameDeployment 部署）之上，额外挂载了框架专属能力（框架配置文件、admin
命令、APM 提取、语言相关解析等）。standard 是去掉框架特化后的通用基座。

## 配置文件：使用 plain 配置（遵循 PR #142 设计）

standard 应用**有配置文件**，使用 PR #142 引入的 **plain 配置文件**（`configKind=plain`）。

plain 与 framework 配置文件的核心差异（决定 standard 的配置能力）：

1. 同一应用下可创建**多个** plain 文件（framework 仍是 1:1）
2. plain 文件可**选择挂载环境**（文件与环境 1:n），framework 仍挂载全部环境
3. plain 文件**存储完整内容**，非 patch 模式
4. plain 文件开启「按环境配置」后：环境未独立修改时不产生真实文件，内容随默认文件同步；一旦独立修改
   才产生一份独立文件，与默认文件解耦；可一键恢复默认（再次与默认同步）
5. plain 文件可**移除挂载**，移除后对应配置文件及版本信息全部删除

**三表模型与字段定义以 PR #142 为准，本文不重复给出。** standard 的配置能力直接复用该 PR 的
`appcfg` Meta 三表模型（`app_config_file_metas` / `app_config_files` / `app_config_file_versions`）
与 `plainfiles` 渲染，包含 `mountPath` 挂载、按环境配置、环境变量模板渲染。

standard 创建时：写入 `configKind=plain` 的配置文件（用户指定 name + mountPath + 内容）。后续
trpc/taf 的 framework 配置也迁移到这套模型，与 plain 共用能力。

## 构建（三种构建方式）

standard 需支持现有三种构建方式，`buildConfig.sourceType` 取值与 trpc/helm 一致：

| 构建方式 | sourceType | 说明 | 当前限制 | standard |
|---------|-----------|------|---------|---------|
| 源码仓库构建 | `codeRepository` | 从 Git 仓库构建镜像 | 平台通用构建仅 trpc-go | ✅ |
| 镜像仓库 | `imageRegistry` | 直接使用已构建镜像 | 现仅 helm | ✅（需放开） |
| 流水线构建 | `pipeline` | 蓝盾流水线构建 | 现仅 trpc | ✅（需放开） |

其中 `codeRepository` 内部又分两种镜像构建方式（`imageBuildMode`）：

- `repositoryDockerfile`：仓库内 Dockerfile 构建（默认），与类型无关。
- `platform`：平台通用构建（builder/runner 基础镜像 + 命令），当前后端硬校验仅 trpc-go，
  standard 需放开到 go/python/node。

构建配置统一存 `build_configs`（`sourceType` + 对应的 `imageBuildConfig` / `repoBuildConfig` /
`pipelineBuildConfig`），standard 与 trpc 同构、无需新增表；需改动的点是放开 imageRegistry /
pipeline / platform 三处现存的类型限制。

## JSON 模型

### 创建请求 `POST /workspaces/:workspaceID/apps`

```jsonc
{
  "id": "demo-standard-a1b2c3",
  "name": "demo-standard",
  "type": "standard",                      // 新增枚举值
  "buildConfig": { "...同 trpc（codeRepository 源码构建）..." },
  "appModelSpec": {
    "command": ["./server"], "args": ["--config", "conf/app.properties"],
    "envVars": [{ "key": "POD_IP", "value": "", "description": "pod ip", "isSensitive": false }],
    "standardSpec": { "language": "go" }   // go / python / node
  }
}
```

### 详情响应 `GET /apps/:appID`

```jsonc
{ "data": { "id": "demo-standard-a1b2c3", "workspaceID": "bkms-workspace",
    "name": "demo-standard", "type": "standard", "displayName": "demo-standard", "creator": "admin",
    "buildConfig": { "...同 input..." },
    "appModelSpec": {
      "command": ["./server"], "args": ["--config", "conf/app.properties"],
      "envVars": [{ "key": "POD_IP", "value": "", "description": "pod ip", "isSensitive": false }],
      "components": [], "standardSpec": { "language": "go" } } } }
```

### DB 落库（7 张表）

- `applications`：`{ ..., type:"standard", standardSpec:{language} }`
- `app_models`：`{ workload:{ type:"standard", name, command, args, envVars[], standardConfig:{language} } }`
- `app_specs`：与 trpc/taf 完全一致（默认 + 每环境一份）
- `build_configs`：与 trpc 一致
- `app_config_file_metas`：`{ _id, appID, name:"app.properties", configKind:"plain", mountPath:"/data/app/conf", isUnifiedConfig:true }`
- `app_config_files`：内容记录（`metaID` 关联 Meta，默认 + 各环境实例）
- `app_config_file_versions`：不可变版本快照

## 能力清单（全量）

| 归属 | 能力 | standard 处理 |
|------|------|--------------|
| 通用基座 | 应用生命周期：创建/查询/列表/删除、显示名、ID 后缀、组件管理 | 自动继承 |
| 通用基座 | 构建：构建配置、Tag 策略、开始构建、构建记录/日志、推荐 Tag | 自动继承 |
| 通用基座 | 配置文件：CRUD、内容编辑、版本管理、overlay/base、按环境、local/BSCP、审计 | 自动继承 |
| 通用基座 | 部署：AppModel 部署（记录/状态/快照/下架）、预检、总览、扩缩容、灰度、批量删除 | 自动继承 |
| 通用基座 | 环境与泳道：标准环境 CRUD、**特性环境**、**泳道列表**、部署状态聚合 | 自动继承 |
| 通用基座 | 工作负载渲染：GameDeployment、规格、配额、探针、生命周期、卷、更新策略、副本、拉取密钥、labels/annotations、Route ENI、组件、联邦转换 | 自动继承 |
| 通用基座 | 实例：列表/watch、WebConsole、端口转发（port-forward）、实例日志、环境事件 | 自动继承 |
| 通用基座 | 环境变量：应用级、scoped、内置、运行时、导入导出、依赖服务 | 自动继承 |
| 通用基座 | AppSpec：resources/updateStrategy/probe/lifecycle/labels/annotations/tkeRouteEni + 三段式 | 自动继承 |
| 通用基座 | 组件：定义 CRUD、预览、内置组件 | 自动继承 |
| 通用基座 | 可观测：拓扑、监控指标、告警策略、APM 实例 | 自动继承 |
| 通用基座 | 镜像：列表、快照、晋级、tag 占用/删除、部署记录、builder/runner、自定义镜像 | 自动继承 |
| 开放能力 | polaris：注册/权重/隔离/统计 | 后续下沉（当前耦合 trpc yaml） |
| 开放能力 | devmode：开发模式热更新 | 后续抽象（当前 trpc/taf 两套路径/脚本） |
| 开放能力 | BSCP：配置元信息、环境绑定与下发 | 后续支持（当前仅 trpc 注入 workloadKind） |
| 开放能力 | GPA 自动扩缩容 | 后续支持（当前仅经部署总览暴露） |
| 开放能力 | HostPort：联邦集群随机端口映射 | 后续支持 |
| 开放能力 | 端口池：PortPool 管理 | 后续支持 |
| 开放能力 | 平台通用构建 platform | 需放开到 go/python/node |
| 开放能力 | 一键构建部署 / 构建触发 | 待加（当前 trpc/taf 各一份） |
| 框架特有 | 框架配置文件及渲染（ConfigMap + init 容器） | 一期不做 |
| 框架特有 | admin 命令（trpc HTTP / taf 私有协议） | 一期不做 |
| 框架特有 | APM 服务名提取（trpc yaml / taf xml） | 一期不做 |
| 框架特有 | tRPC 服务名解析 / Telemetry 解析 | 一期不做 |
| Helm 特有 | chart 来源/values/部署/chart 构建/制品/semver/PostRenderer/Service 同步/部署锁 | 不适用 |

## 各类型特有能力清单

| 能力 | helm | agones | trpc | taf | standard |
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

## AppModel 大类内差别梳理与维护机制

### 问题 1：trpc/taf/standard 的能力差异（详细对比）

三者在「通用基座」上完全一致，差异只落在「框架特化」与「开放能力」两个维度。逐项对比（末列标注该
差异归属哪一层）：

| 能力 | trpc | taf | standard | 归属 |
|------|------|-----|----------|------|
| workload 渲染（GameDeployment） | ✅ | ✅ | ✅ | 通用基座 |
| AppSpec / envVars / 组件 | ✅ | ✅ | ✅ | 通用基座 |
| 部署（记录/状态/快照） | ✅ | ✅ | ✅ | 通用基座 |
| 构建（三种方式） | ✅ | ✅ | ✅ | 通用基座 |
| 配置文件类型 | 框架配置（yaml） | 框架配置（xml） | plain 配置 | 框架特化（configKind） |
| 配置文件渲染 | 框架渲染 + polaris 注入 | 框架渲染 | plain 模板渲染 | 框架特化 |
| 创建逻辑 | 各自一份 | 各自一份 | 各自一份 | 框架特化 |
| spec 更新入口 | 有 | 有 | 有 | 框架特化 |
| 语言子字段 | go/cpp | 无 | go/python/node | 框架特化 |
| admin 命令 | HTTP + yaml 解析 | 私有协议 + xml | 无 | 框架特化 |
| APM 服务名提取 | yaml 解析 | xml 解析 | 无 | 框架特化 |
| polaris | yaml 注入（待下沉） | - | 待支持 | 开放能力 |
| devmode | 专用路径/脚本（待抽象） | 专用路径/脚本 | 待支持 | 开放能力 |

维护方式（三层抽象）：

1. **通用基座**（渲染 / AppSpec / envVars / 组件 / 部署 / 构建）→ 三类共享；新类型只需被识别为
   AppModel 族即可挂入，约 20 处功能门控自动生效。
2. **框架特化层**（配置文件类型与渲染、创建逻辑、spec、admin、APM、语言）→ 收敛为一个「框架特化组件」
   注册机制 + `configKind` 区分；每种框架一个特化组件，框架差异封闭在组件内，不散落类型分支。
3. **开放能力层**（polaris / devmode）→ 下沉为通用能力，按需启用，与框架解耦。

收益：新增一种框架只需「注册一个特化组件 + 定义 spec + 声明配置类型」，而非在每处类型分支补逻辑——
trpc/taf 当前恰恰是被写死成硬编码分支的「特化应用」，standard 是把它们抽掉框架特化后的通用基座形态。

### 问题 2：实现 standard 之后，如何扩展到更多框架

前提是 standard（通用基座 + plain 配置 + 语言子字段）已落地。此后若要扩展更多框架（开源 Go/Python
框架、gRPC-Go 等），有三条可行路径，需要一起权衡：

**方案 A：框架 = 通用应用的「特化组件」**

思路：把「框架」视为对 standard 通用基座的「特化」。框架专属行为（配置格式与渲染、admin 命令、APM
提取、devmode 路径、语言）封装为一个可插拔的「框架特化组件」，挂到通用基座之上；新增框架 = 新增一个
特化组件，通用基座（部署 / AppSpec / envVars / 组件 / 构建）与开放能力（polaris / devmode）不动。

**能力差异的处理**：框架特化能力（配置文件类型与渲染、创建逻辑、spec、admin、APM、语言）全部封闭在
特化组件内，组件对外统一接口、对内各自实现差异；没有某能力的框架（如 standard 无 admin/APM）组件内
留空即可。开放能力（polaris/devmode）不归属任何框架，下沉为通用能力按需启用，框架只是可选消费者。

- 优点：通用基座长期稳定，框架差异被隔离、不污染公共逻辑；新增是「加法」不是「改法」，风险低；与
  「trpc / taf 本质也是两个特化组件」的定位自洽。
- 缺点：需先把现有 trpc/taf 里散落的框架分支收敛成统一的特化组件（一次重构成本）；「特化组件」的
  能力边界（配置 / 渲染 / admin / APM / devmode / 语言）要提前定清，否则日后仍会回退到补分支。

**方案 B：框架的「声明式定义」（配置驱动，零代码扩展）**

思路：把框架能力做成「数据」而非「代码」——用一份「框架定义」声明配置格式与挂载、admin 命令模板、
APM 提取规则、devmode 路径、支持的语言，平台按定义通用解析。新增框架只写定义、不写代码。

**能力差异的处理**：所有能力差异数据化进「框架定义」——配置（格式/挂载/渲染方式）、admin（协议/端口
提取模板）、APM（service 名提取规则）、devmode（路径/脚本模板）、语言（枚举）各是一段声明，平台按定义
通用执行。难点是表达力：高度框架特异的行为需在定义里留「逃逸」出口（如允许嵌入脚本），否则覆盖不住。

- 优点：扩展门槛最低，非核心框架无需写代码；大量「相似框架」可用一套模板覆盖，维护成本低。
- 缺点：声明式 schema 的「表达力」是难点——高度框架特异的行为（如私有 admin 协议）很难用声明覆盖，
  最终会逃逸回代码；早期投入大、有过设计风险。

**方案 C：维持现状「平铺类型 + 硬编码分支」**

思路：继续为每种框架加一个类型，在各处补分支。

**能力差异的处理**：不抽象，每项能力差异在公共逻辑里各加一个类型分支——配置渲染、admin、APM、devmode、
语言每一处都要补一份。差异无法封闭，随框架数增长；语言若也平铺成类型，进一步放大。

- 优点：零重构，短期最快。
- 缺点：已被现有 trpc/taf 的「镜像复制、需同步维护」反证，分支随框架数近平方增长；语言若也做成类型
  更是爆炸，不可持续。

**推荐**：近期走 A（把 trpc/taf 收敛为「特化组件」、standard 作为通用基座落地），中长期向 B 演进
（把特化组件里可声明化的部分——配置格式、admin 模板、APM 规则——沉淀为「声明式框架定义」，仅对真正
框架特异的行为保留代码特化）。C 不推荐。

这样，当下做 standard 时就为「特化组件」预留好能力边界（哪些走通用基座、哪些走特化、哪些走开放能力），
未来加 gRPC-Go 或某 Go 框架时只需新增一个特化组件（或一份框架定义），不触碰公共逻辑。

## 一期实现范围与依赖

- **不依赖 PR #142 的部分**（可先做）：新增类型并归入 AppModel 大类、语言子字段
  （`standardSpec.language`）、工作负载的 standard 配置、standard 的创建逻辑、`/standard-deploys`
  部署路由、envVars/AppSpec/组件复用、spec 更新接口、放开三种构建方式（imageRegistry / pipeline / platform）。
- **依赖 PR #142 的部分**（待合入后再接）：plain 配置文件创建（三表模型 + `configKind=plain`）。
- **一期不做**：框架特有能力（admin 命令、APM 提取）；polaris/devmode 的通用化下沉（后续单独排期）。
