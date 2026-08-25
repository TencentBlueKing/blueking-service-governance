# 熵管理（Entropy Management）

> 目标：让系统"保持整洁"——通过自动化机制控制系统熵增速度，确保长期可维护。

## 1. 文档园艺机制

### 1.1 一致性检测范围

| 检测项 | 方式 | 责任人 |
|-------|------|-------|
| `docs/harness/**`、`docs/standards/**`、`docs/dev-map/README.md` 引用的路径是否仍存在 | harness-gardening 巡检 | 文档维护者 |
| `docs/standards/README.md`「当前项目选用的规范」是否与实际技术栈一致 | `detect-standards.sh` 重新探测 | 各模块 Owner |
| `docs/harness/tooling.md` 的 Skill/MCP/CLI 清单是否与安装布局一致 | harness-gardening 巡检 | Agent 维护者 |
| `bkms-server` Swagger 生成物是否与注解一致 | `make apidocs` 后 diff | bkms-server Owner |

### 1.2 园艺流程

1. 检测到不一致 → 记录差异并提示修复
2. 简单不一致（路径变更等）→ 直接提交修复
3. 复杂不一致（逻辑变更等）→ 通知责任人人工处理

## 2. 架构违规检测

| 检测类型 | 触发时机 | 工具 |
|---------|---------|------|
| Go 代码规范 | 提交前 | `golangci-lint`（`make lint`，配置见各模块 `.golangci.yaml`） |
| 前端代码规范 | 提交前 | ESLint / Stylelint / Biome（`bkms-ui/package.json` scripts） |
| License header 缺失 | 提交前（仅 `bkms-ui`）| ESLint `codecc/license` 规则（其余文件类型人工检查，见根 [`AGENTS.md`](../../AGENTS.md)） |

## 3. 技术债追踪

本仓当前未建立集中式技术债看板；已知技术债追踪渠道：

| 债务类型 | 记录位置 |
|---------|---------|
| 待完善的本地环境搭建指引 | [`docs/DEVELOP_GUIDE.md`](../DEVELOP_GUIDE.md) 中标注的 TODO（如"提供更简便的测试环境搭建指南"） |
| 数据库迁移风险 | [`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md)「Database model & migrations」章节说明的 seq 序号冲突风险 |

## 检查清单

- [ ] 文档园艺检测机制已配置（harness-gardening 巡检）
- [ ] 各模块 lint 已接入本地/CI 流程
- [ ] License header 检测已覆盖可机器检查的文件类型（`bkms-ui`）
