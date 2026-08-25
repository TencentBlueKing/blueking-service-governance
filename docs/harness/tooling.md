# 工具能力（Tooling）

> 目标：让 Agent "能行动"——封装标准化工具接口，保障执行稳定性。

## 1. 工具清单

### 1.0 Skill 清单与触发（Harness 基线）

| Skill | 触发词 | 功能概要 |
|-------|--------|---------|
| agent-task-execution-evaluation | 评估 agent 执行、LLM-as-a-judge、评分 rubric、pass/fail 判定、输出分流 | 评估 worker agent 任务执行结果并输出结构化报告 |
| ai-engineer | Skill 开发、Agent 开发、Eval 驱动、Rule/Hook、质量验收 | AI 工程开发全流程能力套件 |
| backend-developer | API-First、Clean Architecture、DDD、TDD、ADR | 后端需求分析到 API/架构设计与实现套件 |
| bcs-cluster-checklist | BCS 集群、checklist、上线巡检、集群配置检查 | BCS 集群上线前自动化 checklist 巡检 |
| bk-security-redlines | 输入校验、鉴权、数据加密、安全红线 | 蓝鲸代码安全三大高危红线检查 |
| business-analyst | 产品简报、用户画像、persona、触发器地图、商业分析 | 模糊产品想法转化为产品简报与触发器地图 |
| code-review | 代码评审、PR/MR、Google 指南、质量 | 基于 Google 指南的代码评审专家技能 |
| frontend-developer | Component-Driven、Atomic Design、Core Web Vitals、TDD | 前端需求分析到组件设计与代码实现套件 |
| go-micro-service | go-micro、grpc、proto 定义、微服务接口 | go-micro 微服务 proto 定义与代码生成指南 |
| graph-engineering | Graph Run、多阶段编排、人工确认卡点、返工预算 | 多阶段可恢复的 Story/Bug/Issue 编排核心 |
| graphify | 知识图谱、社区检测、dev-map、代码架构查询 | 将代码/文档转为可持续查询的知识图谱 |
| handoff | 会话交接、handoff、会话压缩、交接文档 | 压缩当前会话为可续接的交接文档 |
| harness-engineering | AGENTS.md、上下文工程、文档园艺、dev map、harness 规范 | AI Agent 运行环境规范生成与巡检编排 |
| issue-batch-analysis | 批量 Issue、并行分析、依赖分析、worktree | 批量 Issue 前置可行性分析与并行分支物化 |
| issue-feasibility-analysis | 可行性分析、工蜂 Issue、技术可行性、规范需求文档 | 分析工蜂 Issue 技术可行性并生成需求文档 |
| micro-service-project-init | go-micro 脚手架、grpc-gateway、微服务初始化、etcd 服务发现 | 一键生成 go-micro 微服务项目脚手架 |
| product-manager | PRD、路线图、JTBD、RICE、Discovery | 需求发现到 PRD 与路线图规划套件 |
| qa-engineer | 测试用例、对抗性审查、BDD、测试金字塔 | 对抗性审查产出测试用例与质量报告 |
| requirement-dependency-planning | 需求依赖、DAG、拓扑序、开发分组 | 构建需求依赖 DAG 并划分开发顺序分组 |
| sre-engineer | SLO、Error Budget、四黄金信号、Post-mortem | 上线前准备与上线后 SRE 运营套件 |
| tapd-bug-clarification | 缺陷澄清、根因分析、5Whys、ODC 分类 | TAPD 缺陷根因分析澄清并回写单据 |
| tapd-bug-evaluation | 缺陷评估、PERT 估算、斐波那契 size | 缺陷工时与规模评估并回写 TAPD |
| tapd-iteration-plan | 迭代规划、需求依赖、DAG、迭代容量 | 按依赖与规模将需求编排入迭代 |
| tapd-iteration-runner | 迭代调度、批量需求实现、TDD 迭代 | 批量开发整个迭代中的全部需求 |
| tapd-product-discovery | 产品前置、PRD、竞品分析、角色拆单 | PRD 完成前的产品调研与角色拆单流程 |
| tapd-story-clarification | 需求澄清、多维度澄清、需求文档规范化 | 澄清 TAPD 需求并生成标准化文档 |
| tapd-story-evaluation | 需求拆分、规模评分、RICE 评分、斐波那契 | 需求拆分与规模/业务价值评分 |
| tapd-story-govern-pipeline | 需求整理流水线、澄清评审评估、approved 状态 | 编排需求澄清评审评估三阶段流水线 |
| tapd-story-pipeline | 需求实现、TDD 开发、六阶段、代码提交 | 单个 TAPD 需求从澄清到提交全流程 |
| tapd-story-review | 需求评审、@评审人、评论闭环 | 驱动 TAPD 需求评审直至通过闭环 |
| tech-lead | 架构设计、任务拆解、架构合规评审、需求分析 | 技术需求分析、架构设计与合规评审套件 |
| ux-designer | 用户旅程、页面规格、design system、wireframe | 用户旅程、页面规格与设计系统构建套件 |

> 完整 Skill 目录（含 speckit-* 系列与 story-specify 等子 skill）见 `.agents/skills/*/SKILL.md`。上表仅列出与本项目开发场景直接相关的顶层 Skill。

### 1.1 MCP 工具（Harness 基线）

| MCP 名称 | 所需接口 | 必需 | 检测方式 |
|---------|---------|------|---------|
| tapd | `stories_get`、`stories_update` 等 | 条件 | 会话内 probe（TAPD 相关 Skill 使用时） |
| gongfeng | Issue / MR 查询等 | 条件 | 会话内 probe（工蜂相关 Skill 使用时） |

### 1.2 CLI 工具（Harness 基线）

| 工具 | 必需 | 检测条件 | 检测命令 |
|------|------|---------|---------|
| `git` | 是 | 始终 | `command -v git` |
| `bash` | 是 | 始终 | `command -v bash` |
| `jq` | 是 | 始终 | `command -v jq` |
| `go` | 是 | `go.mod` 存在 | `go version` |
| `node` | 否 | `package.json` 存在 | `node --version` |

### 1.3 项目自有工具

| 名称 | 用途 |
|------|------|
| bkms-dev-ginapi | 辅助 bkms-server Gin API 开发的项目自建 Skill |

## 2. 工具接口规范

### 2.1 统一调用协议

所有自定义工具应遵循以下接口约定：

- **输入**：结构化参数（JSON 格式），包含必填和可选字段
- **输出**：结构化结果（JSON 格式），包含 `success`、`data`、`error` 字段
- **错误处理**：返回明确的错误码和可读的错误信息

## 3. 稳定性保障

### 3.1 容错策略

| 策略 | 适用场景 |
|------|---------|
| 超时 | 所有外部调用（MCP / API） |
| 重试 | 网络请求类临时性故障 |
| 幂等 | 写操作（如 TAPD 单据回写） |

### 3.2 敏感操作防护

| 操作类型 | 防护措施 |
|---------|---------|
| 删除文件/目录 | 二次确认 |
| 数据库迁移（`bkms-server/db/migrations`） | seq 序号唯一性检查（见 [`bkms-server/db/AGENTS.md`](../../bkms-server/db/AGENTS.md)），避免重号导致迁移被永久跳过 |
| 访问生产环境 | 严格禁止 / 需特殊授权 |

## 检查清单

- [ ] Harness 基线表与权威清单一致；项目自有节为 `名称 | 用途`，且名称可在安装布局命中
- [ ] 运行 `harness-doctor` 确认本机缺口（结果不入库）
- [ ] 文档不含运行时状态列或状态单元格（本地是否安装 / 是否可用等）
- [ ] 工具接口遵循统一协议
- [ ] 敏感操作有防护措施
