# 工具能力（Tooling）

> 目标：让 Agent "能行动"——封装标准化工具接口，保障执行稳定性。

## 1. 工具清单

### 1.0 Skill 清单与触发（Harness 基线）

| Skill | 触发词 | 功能概要 |
|-------|--------|---------|
| agent-task-execution-evaluation | 评估 agent 执行、LLM-as-a-judge、评分 rubric | 依据工作 Agent 的标准化任务文档与实际输出，评估执行是否符合预期 |
| ai-engineer | AI 工程开发、Skill/Agent/Rule/Hook 开发 | 基于 Skill 规范驱动与 Eval 驱动方法论的 AI 工程开发能力套件 |
| backend-developer | 后端开发、API 设计、Clean Architecture | 基于 API-First、DDD、TDD、ADR 的后端需求分析到实现交付能力 |
| bcs-cluster-checklist | 集群 checklist、上线前巡检 | BCS 集群上线前 checklist 自动化检查，批量采集集群配置生成结果文件 |
| bk-security-redlines | 代码安全审查 | 基于 IEG 安全规范，覆盖输入校验、鉴权、数据加密三大高危领域 |
| business-analyst | 产品概念梳理、商业分析、用户画像 | 把模糊产品想法转化为产品简报、用户画像、触发器地图 |
| code-review | 代码评审 | 基于 Google Code Review 指南的代码评审技能 |
| frontend-developer | 前端开发、组件设计 | 基于 Component-Driven Development、Atomic Design 的前端交付能力套件 |
| go-micro-service | go-micro、grpc 服务开发、proto 文件编写 | go-micro 框架微服务开发（新建项目结构、proto 接口、代码生成、处理器实现） |
| graph-engineering | graph-engineering、Story/Bug/Issue 编排 | 为单个 Story/Bug/Issue 编排可恢复的多阶段 Graph Run |
| graphify | 代码库/架构问题、开发地图查询 | 将代码/文档转为持久化知识图谱，支持 query/path/explain 查询 |
| handoff | 会话交接、handoff | 将当前会话压缩为交接文档，供下一位 Agent 无缝续接 |
| harness-engineering | Harness Engineering、AI Agent 运行环境规范 | 编排 harness-generating（规范生成）与 harness-gardening（文档巡检） |
| issue-batch-analysis | 批量处理多个 Issue、并行前置分析 | 多工蜂 Issue 批量前置分析与并行交接，跨 Issue 依赖分析后统一物化 |
| issue-feasibility-analysis | Issue 可行性分析 | 拉取工蜂 Issue 详情，分析技术可行性、资源需求与项目定位匹配度 |
| micro-service-project-init | 初始化微服务项目、go-micro 脚手架 | 一键生成 go-micro v5 微服务项目结构（proto、代码生成、公共库、Docker） |
| product-manager | 写 PRD、路线图、产品需求 | 基于 Working Backwards、JTBD、RICE 的产品经理能力套件 |
| qa-engineer | 测试用例、质量报告 | 对抗性审查为核心，结合 Risk-Based Testing、BDD、测试金字塔 |
| requirement-dependency-planning | 需求依赖分析、开发规划 | 梳理需求间依赖关系，构建 DAG，按拓扑序划分开发组 |
| speckit-analyze | spec/plan/tasks 一致性分析 | 对 spec.md、plan.md、tasks.md 进行跨文档一致性与质量分析 |
| speckit-checklist | 生成自定义 checklist | 基于用户要求为当前特性生成自定义检查清单 |
| speckit-clarify | 需求澄清提问 | 识别特性 spec 中信息不足之处，提出最多 5 个针对性澄清问题 |
| speckit-constitution | 项目章程创建/更新 | 从交互或用户输入创建/更新项目章程，同步依赖模板 |
| speckit-git-commit | Spec Kit 命令后自动提交 | Spec Kit 命令完成后自动提交变更 |
| speckit-git-feature | 创建特性分支 | 以序号或时间戳命名创建特性分支 |
| speckit-git-initialize | 初始化 Git 仓库 | 初始化 Git 仓库并完成首次提交 |
| speckit-git-remote | 检测 Git 远程地址 | 检测 Git remote URL 以支持 GitHub 集成 |
| speckit-git-validate | 校验分支命名 | 校验当前分支是否遵循特性分支命名规范 |
| speckit-implement | 执行实现计划 | 处理并执行 tasks.md 中定义的全部任务 |
| speckit-plan | 执行规划工作流 | 使用 plan 模板生成设计产出物 |
| speckit-specify | 创建/更新特性 spec | 依据自然语言描述创建或更新特性规格说明 |
| speckit-tasks | 生成任务列表 | 基于可用设计产出物生成依赖排序的 tasks.md |
| speckit-taskstoissues | 任务转 Issue | 将现有任务转换为依赖排序的可执行 GitHub Issue |
| sre-engineer | SRE 上线准备与运营 | 基于 SLO/Error Budget、四黄金信号等方法论，接入 bkm-bkte 监控体系 |
| story-specify | graph-engineering 技术澄清 worker | 审查 req.md 技术澄清维度并生成 spec.md，遵守 worker-contract 五态信封 |
| tapd-bug-clarification | 缺陷澄清、根因分析 | 拉取 TAPD 缺陷详情，通过 5W1H+5 Whys+ODC 确认根因并回写单据 |
| tapd-bug-evaluation | 缺陷评估、缺陷工时 | 基于 PERT 三点估算给出工时，按五维评分输出规模并回写 TAPD |
| tapd-iteration-plan | 迭代规划、需求入迭代 | 基于 approved 需求池编排迭代，控制迭代总规模上限 |
| tapd-iteration-runner | 迭代执行、批量需求实现 | 批量开发一个迭代中的全部需求，含依赖分析与逐需求调度 |
| tapd-product-discovery | 产品前置、PRD、原型 | 产品前置调研、竞品分析、PRD 起草与角色拆单 |
| tapd-story-clarification | 需求澄清 | 对"规划中"需求进行多维度澄清并标准化输出 |
| tapd-story-evaluation | 需求评估、拆分、size/RICE 评分 | 需求拆分、规模评分（斐波那契）与业务价值（RICE）评分 |
| tapd-story-govern-pipeline | 需求整理流水线 | 调度需求澄清/评审/评估三个 skill，推进需求至 approved |
| tapd-story-pipeline | 单需求实现流水线 | 串联技术澄清、开发计划、任务拆分、TDD 实现、校验、代码提交 |
| tapd-story-review | 需求评审 | 发起评审、汇总评论意见，驱动重新澄清或拆分直至通过 |
| tech-lead | 技术需求分析、架构设计 | 技术需求分析、系统架构设计、任务拆解、架构合规评审 |
| ux-designer | UX 设计、用户旅程、设计系统 | 用户旅程、页面规格、设计系统三阶段知识闭环 |

### 1.1 MCP 工具（Harness 基线）

| MCP 名称 | 所需接口 | 必需 | 检测方式 |
|---------|---------|------|---------|
| tapd | `stories_get`、`stories_update`、`stories_create`、`tapd_id_get`、`iterations_get` 等 | 条件（TAPD 相关 Skill 使用时） | 会话内调用 `stories_get`（最小参数）probe |
| gongfeng | Issue/MR/提交查询接口 | 条件（工蜂相关 Skill 使用时） | 会话内调用 `get_current_user` 等只读接口 probe |
| bkm-bkte | metrics/logs/dashboards/tracing/alarms 等 | 条件（`sre-engineer` 使用时） | 会话内对各 `bkm-bkte-*` MCP 发起只读探测 |

### 1.2 CLI 工具（Harness 基线）

| 工具 | 必需 | 检测条件 | 检测命令 |
|------|------|---------|---------|
| `git` | 是 | 始终 | `command -v git` |
| `bash` | 是 | 始终 | `bash --version` |
| `jq` | 是 | 始终（Linux/macOS） | `jq --version` |
| `node` | 否 | `bkms-ui/package.json` 存在 | `node --version` |
| `graphify` | 否 | `docs/dev-map/` 需要生成或更新时 | `graphify --version` |

### 1.3 项目自有工具

| 名称 | 用途 |
|------|------|
| bkms-dev-ginapi | bkms-server — 在 bkms-server 中开发新的 Gin REST API 时使用；聚焦 router、handler、serializer、鉴权、错误处理、Swagger 与测试约定 |

## 2. 工具接口规范

### 2.1 统一调用协议

所有自定义工具应遵循以下接口约定：

- **输入**：结构化参数（JSON 格式），包含必填和可选字段
- **输出**：结构化结果（JSON 格式），包含 `success`、`data`、`error` 字段
- **错误处理**：返回明确的错误码和可读的错误信息

## 3. 稳定性保障

### 3.1 容错策略

| 策略 | 配置 | 适用场景 |
|------|------|---------|
| 超时 | 单次外部调用不超过合理等待时间 | MCP / API 调用 |
| 重试 | 临时性故障最多重试 3 次，指数退避 | 网络请求、API 调用 |
| 幂等 | 相同参数多次调用结果一致 | 写操作（如 TAPD/工蜂回写） |

### 3.2 敏感操作防护

| 操作类型 | 防护措施 |
|---------|---------|
| 删除文件/目录 | 二次确认 |
| 数据库迁移（`bkms-server/db/migrations`） | 按 [`bkms-server/db/AGENTS.md`](../../bkms-server/db/AGENTS.md) 约定，seq 序号不得与目标分支已有 migration 重复 |
| 访问生产环境 | 严格禁止 / 需特殊授权 |

## 检查清单

- [ ] Harness 基线表与权威清单一致；项目自有节为 `名称 | 用途`，且名称可在安装布局命中
- [ ] 运行 `harness-doctor` 确认本机缺口（结果不入库）
- [ ] 文档无运行时就绪状态列或就绪状态单元格（该类信息只进 `harness-doctor` 输出）
- [ ] 工具接口遵循统一协议
- [ ] 外部调用配置了超时和重试策略
- [ ] 敏感操作有防护措施
