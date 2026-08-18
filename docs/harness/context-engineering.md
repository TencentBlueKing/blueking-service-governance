# 上下文工程（Context Engineering）

> 目标：让 Agent "知道该知道的信息"——确保 Agent 在任务执行中能获取准确、及时、适量的上下文。

## 1. 知识来源定义

### 1.1 唯一知识来源（Single Source of Truth）

| 知识类型 | 存储位置 | 更新频率 |
|---------|---------|---------|
| 项目入口与硬约束 | 根 [`AGENTS.md`](../../AGENTS.md)（短头 + 项目记忆） | 入口约定变更时 |
| 工作单元入口 | 非根 `**/AGENTS.md`（[`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md)、[`bkms-cli/AGENTS.md`](../../bkms-cli/AGENTS.md) 等，局部约定优先） | 组件边界变更时 |
| 服务端设计笔记 | [`bkms-server/design_notes/`](../../bkms-server/design_notes/) | 设计决策变更时 |
| 编码/技术规范 | `docs/standards/`（按加载预算按节读） | 技术栈/规范变更时 |
| 前端规范 | `docs/standards/frontend-vue3.md` | 前端技术栈/规范变更时 |
| 接口规范 | `docs/standards/api-swagger.md` | 接口协议变更时 |
| 后端规范 | `docs/standards/backend-gin.md` | 技术栈/规范变更时 |

### 1.2 禁止的知识来源

以下渠道的信息不应作为 Agent 决策依据（容易过时或缺乏版本控制）：
- 即时通讯记录（飞书、微信等）
- 未纳入版本控制的外部 Wiki
- 口头约定或会议记录

## 2. 渐进式上下文披露

### 2.1 三层结构

```
第一层（入口）：根 AGENTS.md（短头 + 项目记忆 + 局部入口索引）
  └── 改某路径前：阅读向上最近的工作单元 AGENTS.md（局部优先于根）

第二层（导航）：docs/harness/README.md、docs/standards/README.md、docs/dev-map/README.md
  └── 对第一层只做指针/短摘要，不复制大段局部 AGENTS

第三层（详情）：代码、局部 AGENTS 细则、standards 分册、design_notes/、深度参考
  └── 按任务按节加载，禁止默认全文灌入
```

### 2.2 上下文预算管理

- Agent 的上下文窗口视为有限资源，需精心管理
- 优先加载与当前任务直接相关的文档
- `bkms-server/AGENTS.md` 明确提示：本仓源文件可能很长，读取前先检查文件大小，评估是否需要全文读取
- 大文件（>300 行）通过目录索引定位相关段落，避免全量加载；本仓已接入的 `docs/dev-map/`（graphify 知识图谱）可用于按概念/路径检索，替代全文扫描

## 3. 动态上下文

| 数据源 | 用途 | 存储位置 |
|-------|------|---------|
| 结构化日志 | 请求链路 / 异步任务链路排障 | [`bkms-server/pkg/common/logging`](../../bkms-server/pkg/common/logging)（详见 [`bkms-server/README.md#日志使用`](../../bkms-server/README.md)） |
| APM / bk-monitor 指标与追踪 | 服务可观测性 | [`bkms-server/pkg/observability/`](../../bkms-server/pkg/observability/)（`apm/`、`bkmonitor/`、`metrics/`、`instancelog/`） |

## 4. 上下文更新机制

### 4.1 触发条件

- 代码架构发生重大变更（新增模块、调整分层）
- API 接口新增或变更（需同步运行 `make apidocs`，见 [`bkms-server/Makefile`](../../bkms-server/Makefile)）
- 业务规则更新
- 依赖的外部系统发生变更

### 4.2 更新流程

1. 变更方在 PR 中同步更新相关文档（含工作单元 `AGENTS.md`、`design_notes/`）
2. Code Review 时检查文档是否同步更新（参考 `docs/standards/quality-code-review/`）
3. 文档巡检（harness-gardening）定期扫描检测遗漏

## 检查清单

- [ ] 所有知识类型都有明确的存储位置
- [ ] 根 AGENTS 短头精简；项目记忆按 `agents-merge.md` 理解后保留
- [ ] 已发现的工作单元 `**/AGENTS.md` 在根有索引，并写明局部优先（nearest）
- [ ] 动态数据源已配置接入方式
- [ ] 上下文更新机制已建立
