# 执行与验证（Execution & Verification）

> 目标：让 Agent "做对事"——通过强制验证确保任务被正确完成。

## 1. 预完成检查清单

**验证命令与工作目录以根 / 最近工作单元 `AGENTS.md` 及仓库脚本为准**；按组件根执行，禁止虚构测试栈。

| 组件 | 构建 | 测试 | Lint | 依据 |
|------|------|------|------|------|
| `bkms-server` | `make build` | `make test`（需先设置 `BKMS_SERVER_CONFIG_PATH`，用 Ginkgo，可用 `./bin/ginkgo {TEST_FILE}` 跑单模块） | `make lint` | [`bkms-server/Makefile`](../../bkms-server/Makefile)，[`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md) |
| `bkms-cli` | `make build` | `make test`（用 Ginkgo v2 + Gomega），E2E 用 `make e2e-go-test` | `make lint` | [`bkms-cli/Makefile`](../../bkms-cli/Makefile)，[`bkms-cli/AGENTS.md`](../../bkms-cli/AGENTS.md) |
| `bkms-dockerfile-generator` | `make build` | `make test`（用 Ginkgo） | `make lint` | [`bkms-dockerfile-generator/Makefile`](../../bkms-dockerfile-generator/Makefile) |
| `bkms-ui` | `pnpm build` | `pnpm` 脚本 `test:unit`（Vitest），`pnpm typecheck` | `pnpm lint` | [`bkms-ui/package.json`](../../bkms-ui/package.json) |

其余检查：

| 检查项 | 验证方式 | 跳过条件 |
|-------|---------|---------|
| Swagger 文档同步 | `make apidocs`（`bkms-server`，见 [`bkms-server/AGENTS.md`](../../bkms-server/AGENTS.md)「Add new REST APIs」） | 未新增/修改 REST 接口 |
| Bruno API 测试 | 在 `bkms-server/tests/apis` 下 `bru run --env-file ./environments/local.bru -r {DIR或FILE}`（需先重新编译并重启服务） | 未涉及 API 行为变更 |
| License Header | 新增源文件按 [`AGENTS.md`](../../AGENTS.md)「License headers」核对；`bkms-ui` 由 `pnpm lint` 中的 `codecc/license` 规则强制校验 | 无新增文件 |
| 对照任务 Spec | 逐条检查需求点 | 无 |
| 文档已同步更新 | 检查工作单元 `AGENTS.md`、`design_notes/`、`docs/` 是否需要同步 | 无文档影响 |

`libs/bkms-adapter` 当前未提供独立 Makefile；如需验证，改动前先确认该目录下是否存在可用的 `go test ./...` 覆盖，无法确认时不得声称已通过测试。

## 2. 验证失败处理

- 验证失败 → 回到执行修复该问题，而非直接提交
- 连续修复仍失败 → 暂停并向用户说明失败原因，请求人工介入
- 涉及数据库迁移（`bkms-server/db/migrations`）的变更，额外核对 seq 序号未与目标分支冲突（见 [`bkms-server/db/AGENTS.md`](../../bkms-server/db/AGENTS.md)）

## 3. 任务漂移检测

| 信号 | 处理方式 |
|------|---------|
| 修改与任务无关的组件/文件 | 撤销范围外变更并提醒用户 |
| 自主决定改变技术方案（如更换框架、跨组件重构） | 暂停并请求用户确认 |
| 循环执行相同操作仍未解决问题 | 终止并向用户报告当前状态 |

## 检查清单

- [ ] 预完成检查清单命令来自仓内 Makefile / package.json / AGENTS，无虚构测试栈
- [ ] 涉及 REST 接口变更已运行 `make apidocs`
- [ ] 涉及数据库迁移已核对 seq 序号
- [ ] 新增源文件已核对 License Header
- [ ] 任务漂移检测规则已明确
