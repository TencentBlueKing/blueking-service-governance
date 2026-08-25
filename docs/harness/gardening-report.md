# 文档园艺报告

> 扫描时间：2026-08-25
> 模式：全量扫描
> 触发来源：手动
> last-commit: 63de70135d34b82c1e5013a9e656d4f9d430c502

## 摘要

| 维度 | 状态 | P0 已修复 | P1 待确认 | Skip |
|------|------|----------|----------|------|
| 路径有效性 | FIXED | 1 | 0 | 0 |
| Skill 清单 | PASS | 0 | 0 | 0 |
| 架构描述 | FIXED | 0 | 1 | 0 |
| 技术规范同步 | FIXED | 1 | 0 | 0 |
| 词汇表完整性 | PASS | 0 | 0 | 0 |
| 目录结构 | PASS | 0 | 0 | 0 |
| 工具依赖一致性 | PASS | 0 | 0 | 0 |
| Dev Map 与 IDE 集成一致性 | PASS | 0 | 0 | 0 |
| 工作流文档同步 | PASS | 0 | 0 | 0 |
| 业务规范一致性 | SKIP | 0 | 0 | 1 |
| Standards Rules 一致性 | PASS | 0 | 0 | 0 |
| AGENTS 项目记忆保留 | PASS | 0 | 0 | 0 |

## 已自动修复（本次）

- [P0] 路径有效性 / `docs/harness/execution-verification.md`：工作单元验证表（`HARNESS_WORK_UNIT_VERIFICATION` 标记区块）缺失 25 条已发现的 build/lint/test 命令（`bkms-cli`、`bkms-dockerfile-generator`、`bkms-server`、`bkms-ui` 四个工作单元），已用 `render-work-unit-verification.py --write` 重新渲染
- [P0] 技术规范同步 / `docs/standards/quality-code-review.md` 及其分册 `docs/standards/quality-code-review/*.md`：与预设库 `../assets/standards/quality-code-review*` 存在编号漂移（分册 03/04/05/06 应为 01/02/03/04），已用预设覆写入口文件并重命名/覆写四个分册

## 已修复（P1，用户确认）

- [P1] 架构描述一致性 / `docs/harness/architectural-constraints.md` 第 40-41 行：Parse-Don't-Validate 表格中 `bkms-cli YAML Spec` 一行路径补全为 `bkms-cli/pkg/handler/app/types.go`（原缺模块前缀）；`bkms-server 配置` 一行原引用不存在于仓库中的 `config.local.yaml`（属 `.gitignore` 中 `*.local.yaml` 通配的本地文件，从未提交），已按用户确认方案改写为「配置结构体（`bkms-server/configs/config.yaml`；本地可用 `*.local.yaml` 覆盖，见 `.gitignore`）」

## 待确认方案

无

## 跳过项

- [Skip] 业务规范一致性：`docs/business-standards/` 目录不存在，本项目未登记自定义业务规范，无需处理
