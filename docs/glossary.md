# 词汇表（Glossary）

> 本项目涉及的核心概念、术语和缩写定义。Agent 和人类成员均以此为术语的唯一解释来源。

## Harness Engineering 核心概念

| 术语 | 英文 | 定义 |
|------|------|------|
| 驾驭工程 | Harness Engineering | 为 AI Agent 构建可靠运行环境的方法论，通过工具、上下文、约束、验证机制系统性提升 Agent 表现 |
| 上下文工程 | Context Engineering | 确保 Agent 能获取准确、及时、适量上下文信息的实践 |
| 架构约束 | Architectural Constraints | 通过刚性规则保障代码结构一致性和可维护性的约束体系 |
| 熵管理 | Entropy Management | 控制系统熵增速度（文档漂移、技术债累积）的自动化机制 |
| 工具能力 | Tooling | Agent 可调用的标准化工具接口集合及其稳定性保障 |
| 执行与验证 | Execution & Verification | Agent 执行任务循环与任务完成前的强制验证机制 |
| 开发地图 | Dev Map | 基于 graphify 的知识图谱，用于 Agent 查询代码结构与概念关联 |

## 项目业务术语

| 术语 | 英文/缩写 | 定义 |
|------|----------|------|
| 蓝鲸服务治理 | BKMS（BlueKing Service Governance） | 面向游戏开发者、SRE 的一站式应用全生命周期管理平台 |
| 空间 | Workspace | bkms-server 中应用托管的顶层组织单元 |
| 环境 | Env | 应用部署所依赖的运行时环境定义（`pkg/core/env`） |
| 应用 | App | 服务治理平台托管的业务应用单元（`pkg/core/app`） |
| 制品托管 | Artifact Hosting | 基于代码提供标准化 CI 制品构建，事件驱动交付能力 |
| 开发联调 | Dev Mode | 为开发者提供一特性一环境的个人/团队调试能力（`pkg/extension/component/devmode`） |
| 部署 | Deploy | 基于应用制品完成服务部署、滚动更新和灰度更新（`pkg/deploy`） |
| 组件 | Component | 基于业务场景构建的组件实例，降低开发者配置和接入成本（`pkg/extension/component`） |
| 集群资源附件 | Cluster Addon | 环境关联的集群资源扩展能力（`pkg/core/env/clusteraddon`） |

## 工具与平台

| 术语 | 英文/缩写 | 定义 |
|------|----------|------|
| Gin | Gin | bkms-server 使用的 Go HTTP Web 框架 |
| Swagger/OpenAPI | Swagger/OpenAPI | bkms-server 通过 swaggo/gin-swagger 注解生成的 API 契约文档 |
| Ginkgo | Ginkgo | Go 生态 BDD 测试框架，bkms-server 与 bkms-cli 单元测试均采用 |
| Cobra | Cobra | bkms-cli 使用的 Go CLI 框架 |
| Pinia | Pinia | bkms-ui 使用的 Vue 3 状态管理库 |
| golang-migrate | golang-migrate | bkms-server 数据库迁移工具，迁移文件为 MongoDB JSON 格式 |
| golangci-lint | golangci-lint | Go 代码静态检查与格式化工具，各 Go module 均配置 `.golangci.yaml` |

## 工程实践术语

| 术语 | 英文 | 定义 |
|------|------|------|
| 分层架构 | Layered Architecture | Handler → Service → Store 的请求处理路径约定（见 `docs/standards/backend-gin.md`） |
| 生成物 | Generated Artifact | 由工具自动生成、禁止手改的文件（如 Swagger JSON、前端 API 类型） |
| 局部入口 | Work Unit AGENTS | 非根目录的 `AGENTS.md`，记录该子树的局部开发约定，优先于根 AGENTS |

---

*持续补充中——遇到新术语时请直接在对应分类下添加。*
