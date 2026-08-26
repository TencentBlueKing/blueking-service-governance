# 文档园艺报告

> 扫描时间：2026-08-26
> 模式：全量扫描
> 触发来源：手动
> last-commit: 3a0b3ba945ec36c53e0f048678aeaf72c8375673

## 摘要

| 维度 | 状态 | P0 已修复 | P1 待确认 | Skip |
|------|------|----------|----------|------|
| 路径有效性 | PASS | 0 | 0 | 0 |
| Skill 清单 | PASS | 0 | 0 | 0 |
| 架构描述 | PASS | 0 | 0 | 0 |
| 技术规范同步 | FIXED | 1 | 0 | 0 |
| 词汇表完整性 | PASS | 0 | 0 | 0 |
| 目录结构 | PASS | 0 | 0 | 0 |
| 工具依赖一致性 | PASS | 0 | 0 | 0 |
| Dev Map 与 IDE 集成一致性 | PASS | 0 | 0 | 0 |
| 工作流文档同步 | SKIP | 0 | 0 | 1 |
| 业务规范一致性 | SKIP | 0 | 0 | 1 |
| Standards Rules 一致性 | FIXED | 1 | 0 | 0 |
| AGENTS 项目记忆保留 | PASS | 0 | 0 | 0 |

## 已自动修复（本次）

- [P0] 技术规范同步 / `docs/standards/README.md`：`detect-standards.sh` 显示 frontend/api/backend 均已 Level-1（`bkms-ui` Vue3、`bkms-server` swagger + gin），仍残留「未覆盖的技术栈」节，正文为「无——三端均已匹配」。按「无未匹配分类时整节省略」删除整节；未改成「未覆盖的技术栈（无）」。章节快速索引 `--check` 仍通过。
- [P0] Standards Rules / 根 `AGENTS.md` 门闩第 3 步：该节已不存在，将无条件「按 README「未覆盖的技术栈」处理」改为「若 README 含该节则按其处理，否则向用户确认」。License headers 等项目记忆未改。
- [P0] `docs/harness/README.md`：项目概述「镶嵌规范」改为「规范」（上一轮未提交的表述修正一并纳入）。

## 已修复（P1，用户确认）

无

## 待确认方案

无

## 跳过项

- [Skip] 工作流文档同步：维度 9 已退役；根 `AGENTS.md` 无 `workflow-agent` /「不允许跳过」残留
- [Skip] 业务规范一致性：`docs/business-standards/` 目录不存在，本项目未登记自定义业务规范

## 复核（无偏差）

- 路径有效性：`AGENTS.md`、`docs/harness/*.md`、`docs/standards/README.md`、`docs/glossary.md` 中抽查的仓内链接均存在；`tooling.md` 无被 ignore 的安装根路径引用
- Skill 清单：`tooling.md` §1.0 所列顶层 Skill 均存在于 `.agents/skills/`；`speckit-*` / `story-specify` 未登记，按规则 Skip
- 架构描述：工作单元验证表 `render-work-unit-verification.py --check` 通过；`bkms-server` / `bkms-ui` / `bkms-cli` / `bkms-dockerfile-generator` / `libs/bkms-adapter` 目录与依赖方向断言仍成立
- 技术规范同步：`frontend-vue3.md` / `api-swagger.md` / `backend-gin.md` / `security-bk-redlines.md` / `quality-code-review.md`（含分册）与安装根预设逐字节一致
- 词汇表完整性：Harness 核心术语与 BKMS 业务术语均有条目
- 目录结构：根 `AGENTS.md` 目录树与实际一致；无 `<!-- TODO` / gardening 操作约束泄漏
- 工具依赖一致性：基线节无环境状态列；项目自有 `bkms-dev-ginapi` 已 git 跟踪
- Dev Map：`docs/dev-map/README.md` + `.gitignore` 白名单就位；`.agents/rules/graphify.mdc` / `.md` 存在
- Standards Rules：`sync-standards-rules.sh --dry-run` 仅重写 ignore 下的 `.agents/rules`（安装产物，不入库）
- AGENTS 项目记忆：局部入口覆盖 `git ls-files` 的 4 个非根 `AGENTS.md`；License headers 保留
