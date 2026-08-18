# 架构约束（Architectural Constraints）

> 目标：让 Agent "做正确的事"——通过刚性约束确保代码结构的一致性和可维护性。

## 1. 组件与目录职责

本仓是多模块 monorepo，各组件独立演进，无跨组件共用的统一分层模型；组件内部约定详见对应工作单元 `AGENTS.md`。

| 组件 | 目录 | 职责 |
|------|------|------|
| Gin REST API 服务端 | `bkms-server/` | 领域服务、异步任务、数据访问；详见 [`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md) |
| Cobra CLI | `bkms-cli/` | 命令行子命令、API 客户端调用；详见 [`bkms-cli/AGENTS.md`](../../bkms-cli/AGENTS.md) |
| Dockerfile 生成工具 | `bkms-dockerfile-generator/` | 基于流水线配置生成平台默认 Dockerfile |
| 公共适配层 | `libs/bkms-adapter/` | 封装与外部系统 / 基础设施的对接逻辑 |
| 前端 | `bkms-ui/` | Vue 3 产品化交互界面 |
| 部署清单 | `charts/` | Helm Chart |

### 1.1 bkms-server：领域包内分层

按 [`bkms-server/.agents/skills/bkms-dev-ginapi/SKILL.md`](../../bkms-server/.agents/skills/bkms-dev-ginapi/SKILL.md) 约定，新增 Gin REST API 遵循固定的领域包内分层：

```
pkg/<domain>/
  router.go          # 路由与小型 handler interface
  handler/<feature>.go   # 视图逻辑，依赖 *store.Registry
  serializer/<feature>.go # Input/Output 结构与转换
```

- 路由注册放在模块的 `router.go`；handler 文件只放视图逻辑
- 可复用 Gin 工具放在 `pkg/bkmssrv/ginutils`；鉴权放在 `pkg/common/auth`；错误渲染放在 `pkg/common/bkerrs`
- **禁止**为了"分层"额外包一层没有复用价值的私有函数

### 1.2 依赖规则（导入环）

- `pkg/<domain>/router.go` 只定义路由和小型 handler interface；`handler.New(registry)` 在上层完成构造后再调用 `domain.Register(...)`，避免 import cycle（见 bkms-dev-ginapi SKILL.md「核心规则」）
- `bkms-cli`：子命令入口在 `cmd/<name>/`，API 调用逻辑放 `pkg/client/`，业务逻辑放 `pkg/handler/`（见 [`bkms-cli/AGENTS.md`](../../bkms-cli/AGENTS.md)「Adding a new subcommand」）

## 2. Linter 与静态检查

| 组件 | 工具 | 配置 |
|------|------|------|
| `bkms-server` | golangci-lint v2 | [`bkms-server/.golangci.yaml`](../../bkms-server/.golangci.yaml)，通过 `make lint` 执行 |
| `bkms-cli` | golangci-lint v2 | [`bkms-cli/.golangci.yaml`](../../bkms-cli/.golangci.yaml)，通过 `make lint` 执行 |
| `bkms-ui` | ESLint + Stylelint | `bkms-ui/eslint.config.mjs`，通过 `pnpm lint` 执行；`codecc/license` 规则强制校验开源许可证头 |

## 3. Parse, Don't Validate

### 3.1 数据边界

| 边界 | 输入类型 | 解析目标 | 处理位置 |
|------|---------|---------|---------|
| Gin API 请求 | JSON/URI | Serializer Input 结构 | `pkg/<domain>/serializer/`（`ginutils.BindURI` / `ginutils.BindJSON`，见 bkms-dev-ginapi SKILL.md「Handler 约定」） |
| `bkms-cli app create` YAML | YAML | `AppCreateSpec` | `bkms-cli/pkg/handler/app/types.go`（见 [`bkms-cli/AGENTS.md`](../../bkms-cli/AGENTS.md)「app create subcommand design」） |

## 4. 架构决策记录（ADR）

本仓当前以 `bkms-server/design_notes/` 记录服务端设计笔记（如 `appdefaults.md`、`component.md`、`permission.md` 等），尚无独立的 `docs/adr/` 目录。若需引入正式 ADR 流程，建议：

- 存储位置：`docs/adr/`
- 命名格式：`NNNN-标题.md`
- 在做出跨组件架构决策前，先检索 `design_notes/` 与已有 ADR，确保不与历史决策冲突

## 检查清单

- [ ] 新增 Gin API 遵循 router/handler/serializer 分层，无 import cycle
- [ ] 修改前已阅读对应工作单元 `AGENTS.md` 与领域 `design_notes/`（若有）
- [ ] 对应组件 lint 命令（`make lint` / `pnpm lint`）已通过
- [ ] 数据边界处按 Serializer / Spec 结构完成 Parse
- [ ] 跨组件架构变更已记录到 `design_notes/` 或 ADR
