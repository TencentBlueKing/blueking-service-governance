# 上下文工程（Context Engineering）

> 目标：让 Agent "知道该知道的信息"——确保 Agent 在任务执行中能获取准确、及时、适量的上下文。

## 1. 知识来源定义

### 1.1 唯一知识来源（Single Source of Truth）

| 知识类型 | 存储位置 | 维护责任人 | 更新频率 |
|---------|---------|-----------|---------|
| 项目入口与硬约束 | 根 [`AGENTS.md`](../../AGENTS.md)（短头 + License headers 项目记忆） | 各模块 Owner | 入口约定变更时 |
| 工作单元入口 | [`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md)、[`bkms-cli/AGENTS.md`](../../bkms-cli/AGENTS.md) 等非根 `AGENTS.md`（局部约定优先） | 模块 Owner | 组件边界变更时 |
| 本地开发环境搭建 | [`docs/DEVELOP_GUIDE.md`](../DEVELOP_GUIDE.md) | 各模块 Owner | 依赖服务/端口变更时 |
| 贡献与提交规范 | [`docs/CONTRIBUTING.md`](../CONTRIBUTING.md) | 项目维护者 | 流程变更时 |
| 后端 API 契约（bkms-server） | `bkms-server/docs/apis/swagger.json`（`make apidocs` 生成） | bkms-server Owner | 接口变更时 |
| 数据库迁移记录 | `bkms-server/db/migrations/`（配套 markdown 见同目录 [`AGENTS.md`](../../bkms-server/db/AGENTS.md)） | bkms-server Owner | 新增迁移时 |
| 编码/技术规范 | `docs/standards/`（按加载预算按节读） | 各模块 Owner | 技术栈/规范变更时 |
| 前端开发规范 | `docs/standards/frontend-vue3.md` | bkms-ui Owner | 技术栈/规范变更时 |
| 接口协议规范 | `docs/standards/api-swagger.md` | bkms-server Owner | 接口协议变更时 |
| 后端开发规范 | `docs/standards/backend-gin.md` | bkms-server Owner | 技术栈/规范变更时 |

### 1.2 禁止的知识来源

以下渠道的信息不应作为 Agent 决策依据（容易过时或缺乏版本控制）：
- 即时通讯记录（飞书、微信等）
- 未纳入版本控制的外部 Wiki
- 口头约定或会议记录

## 2. 渐进式上下文披露

### 2.1 三层结构

```
第一层（入口）：根 AGENTS.md（短头 + License headers 项目记忆 + 局部入口索引）
  └── 改某路径前：阅读向上最近的工作单元 AGENTS.md（局部优先于根）

第二层（导航）：docs/harness/README.md、docs/standards/README.md
  └── 对第一层只做指针/短摘要，不复制大段局部 AGENTS

第三层（详情）：代码、局部 AGENTS 细则、standards 分册、深度参考
  └── 按任务按节加载，禁止默认全文灌入
```

### 2.2 上下文预算管理

- Agent 的上下文窗口视为有限资源，需精心管理
- 优先加载与当前任务直接相关的文档
- 大文件（>300行）通过目录索引定位相关段落，避免全量加载
- bkms-server 源文件体量较大，改动前先按 [`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md) 提示评估文件大小

## 3. 上下文更新机制

### 3.1 触发条件

- 代码架构发生重大变更（新增模块、调整分层）
- API 接口新增或变更（需同步 `make apidocs` 生成物）
- 业务规则更新
- 依赖的外部系统发生变更

### 3.2 更新流程

1. 变更方在 PR 中同步更新相关文档
2. Code Review 时检查文档是否同步更新
3. 文档园艺 Agent 定期扫描检测遗漏

## 检查清单

- [ ] 所有知识类型都有明确的存储位置和维护责任人
- [ ] 根 AGENTS 短头精简；License headers 项目记忆按理解后保留
- [ ] 已发现的工作单元 `**/AGENTS.md` 在根有索引，并写明局部优先（nearest）
- [ ] 上下文更新机制已建立并有人负责
