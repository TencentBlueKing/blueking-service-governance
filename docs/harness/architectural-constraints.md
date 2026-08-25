# 架构约束（Architectural Constraints）

> 目标：让 Agent "做正确的事"——通过刚性约束确保代码结构的一致性和可维护性。

## 1. 仓库级分层模型

本仓为多模块 monorepo，模块间以独立部署单元划分，非单体分层：

| 模块 | 目录 | 职责 | 允许的依赖 |
|------|------|------|-----------|
| 服务端 | `bkms-server/` | Gin REST API、领域服务、异步任务（worker）、数据访问 | `libs/bkms-adapter`（通过 [`go.mod` replace](../../bkms-server/go.mod)） |
| 前端 | `bkms-ui/` | Vue 3 产品化交互界面，消费 bkms-server API | 无仓内 Go 模块依赖 |
| CLI | `bkms-cli/` | Cobra 命令行工具，调用 bkms-server API | 独立 Go module，不依赖 bkms-server |
| Dockerfile 生成器 | `bkms-dockerfile-generator/` | 镜像构建流程工具，读取流水线环境变量渲染 Dockerfile | 独立 Go module |
| 公共适配层 | `libs/bkms-adapter/` | 封装与外部系统（如 Polaris）对接逻辑 | 被 `bkms-server` 引用，自身无仓内依赖 |

### 1.1 bkms-server 内部分层（示例，见 [`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md)）

- 各业务能力包（`pkg/account`、`pkg/core`、`pkg/deploy`、`pkg/workload` 等）内部遵循 router → handler → serializer 的请求处理路径
- 新增 Store 参考 `pkg/core/app/store.go`；数据库迁移见 [`bkms-server/db/AGENTS.md`](../../bkms-server/db/AGENTS.md)

### 1.2 依赖规则

- Go module 间依赖以各自 `go.mod` 的 `require` / `replace` 为唯一事实来源；`bkms-cli`、`bkms-dockerfile-generator` 均为独立 module，**不得**跨仓相互引用业务包
- `libs/bkms-adapter` 不得反向依赖 `bkms-server`
- `bkms-ui` 与 `bkms-server` 之间只通过 HTTP API（Swagger 契约）交互，不共享代码

## 2. Generated / codegen

- `bkms-server` 的 Swagger/OpenAPI 生成物位于 [`bkms-server/docs/apis/`](../../bkms-server/docs/apis)（`docs.go`、`swagger.json`、`swagger.yaml`），由 `make apidocs` 生成，**禁止手改**
- `bkms-ui/src/@types/v1/**` 由 `pnpm gen:api:v1` 重新生成，`bkms-ui/src/components.d.ts` 由构建时自动生成——均不含 license header（见根 [`AGENTS.md`](../../AGENTS.md)「Files that intentionally have no header」）

## 3. Parse, Don't Validate

在数据进入系统的边界处，将原始数据**解析**为强类型的领域模型，后续代码只操作解析后的类型：

| 边界 | 输入类型 | 解析目标 | 处理位置 |
|------|---------|---------|---------|
| bkms-server API 请求 | JSON | Gin `ShouldBind*` 绑定的请求结构体 | Handler 层（见 `docs/standards/backend-gin.md`） |
| bkms-cli YAML Spec | YAML | `AppCreateSpec`（`bkms-cli/pkg/handler/app/types.go`） | CLI 命令层，`go-playground/validator` 校验 |
| bkms-server 配置 | YAML | 配置结构体（`bkms-server/configs/config.yaml`；本地可用 `*.local.yaml` 覆盖，见 `.gitignore`） | 启动阶段 |

## 检查清单

- [ ] 各 Go module 依赖关系清晰，无跨仓业务包引用
- [ ] Swagger 生成物未被手改
- [ ] 数据边界处的 Parse 策略已明确
- [ ] 若根/工作单元 AGENTS 已有分层或关系图：本文档仅摘要 + 指针，未另编冲突分层
