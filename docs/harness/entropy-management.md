# 熵管理（Entropy Management）

> 目标：让系统"保持整洁"——通过机制控制系统熵增速度，确保长期可维护。

## 1. 文档一致性

| 检测项 | 方式 |
|-------|------|
| Harness 规范（`docs/harness/`、`docs/standards/`、根/工作单元 `AGENTS.md`）与项目实际是否一致 | `harness-engineering` skill 的 harness-gardening 巡检（触发词：文档园艺、文档巡检） |
| 开发地图（`docs/dev-map/graph.json`）与代码结构是否一致 | 触发词"更新开发地图"，本地增量更新（AST，无 API 费用） |
| 数据库迁移文件 seq 序号是否与目标分支冲突 | 人工核对，见 [`bkms-server/db/AGENTS.md`](../../bkms-server/db/AGENTS.md)（seq 重号会导致后合入的迁移被永久跳过且不报错） |

维护流程：发现文档与代码不一致时，先确认是文档滞后还是代码变更未同步说明，再决定是直接修正文档还是提示责任人复核；`docs/business-standards/`（若启用）由业务 Owner 维护，harness-gardening 不覆写其正文。

## 2. 技术债追踪

本仓当前未接入统一的技术债看板；已知的、可通过工具核实的债务信号：

| 信号 | 识别方式 |
|------|---------|
| Lint 违规 | 各组件 `make lint` / `pnpm lint` 报错 |
| 代码评审遗留问题 | 按 [`docs/standards/quality-code-review/`](../standards/quality-code-review/) 检查维度与问题分级记录 |

## 3. 代码评审规范

Code Review / 质量评分任务应按需加载 [`docs/standards/quality-code-review.md`](../standards/quality-code-review.md) 及对应分册（`docs/standards/quality-code-review/`），涵盖核心原则、问题分级、检查维度、评分标准与评审报告格式；日常改代码不预加载该规范。

## 检查清单

- [ ] 文档一致性检测机制已知晓（harness-gardening / dev-map 增量更新）
- [ ] 数据库迁移 seq 冲突风险已在提交前核对
- [ ] 组件 lint 已纳入提交前自检
- [ ] Code Review 任务已按需加载 `quality-code-review` 规范分册
