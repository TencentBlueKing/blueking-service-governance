# 执行与验证（Execution & Verification）

> 目标：让 Agent "做对事"——通过执行循环和强制验证确保任务被正确完成。

## 1. Agent Loop 执行循环

### 1.1 循环结构

```
while 任务未完成:
    1. 观察（Observe）— 获取当前环境状态
    2. 推理（Think）— 分析状态，规划下一步
    3. 行动（Act）— 调用工具执行操作
    4. 验证（Verify）— 检查操作结果是否符合预期
    5. 更新（Update）— 更新任务状态
```

## 2. 强制验证机制

### 2.1 预完成检查清单

Agent 在宣称任务完成前，必须逐项确认。工作单元验证表由渲染脚本生成。命令来自工作单元 AGENTS、Makefile、package scripts 或 CI；配置只证明规则存在，不执行 format/lint/test/build。无明确入口时不得补写或声称 lint 已通过。

<!-- HARNESS_WORK_UNIT_VERIFICATION:START -->
| 工作单元 | 类别 | 工作目录 | 命令 | 命令证据 | 规则证据 |
|---|---|---|---|---|---|
| `bkms-cli` | build | `bkms-cli` | `make build` | [`Makefile`](../../bkms-cli/Makefile) | - |
| `bkms-cli` | build | `bkms-cli` | `make build-all` | [`Makefile`](../../bkms-cli/Makefile) | - |
| `bkms-cli` | build | `bkms-cli` | `make build-all compress` | [`bkms-cli.yml`](../../.github/workflows/bkms-cli.yml) | - |
| `bkms-cli` | build | `bkms-cli` | `make build-darwin` | [`Makefile`](../../bkms-cli/Makefile) | - |
| `bkms-cli` | build | `bkms-cli` | `make build-linux` | [`Makefile`](../../bkms-cli/Makefile) | - |
| `bkms-cli` | build | `bkms-cli` | `make build-windows` | [`Makefile`](../../bkms-cli/Makefile) | - |
| `bkms-cli` | build | `bkms-cli` | `make check-build-vars` | [`Makefile`](../../bkms-cli/Makefile) | - |
| `bkms-cli` | build | `bkms-cli` | `make e2e-build` | [`Makefile`](../../bkms-cli/Makefile) | - |
| `bkms-cli` | lint | `bkms-cli` | `make install-golangci-lint` | [`Makefile`](../../bkms-cli/Makefile) | [`.golangci.yaml`](../../bkms-cli/.golangci.yaml) |
| `bkms-cli` | lint | `bkms-cli` | `make lint` | [`AGENTS.md`](../../bkms-cli/AGENTS.md) | [`.golangci.yaml`](../../bkms-cli/.golangci.yaml) |
| `bkms-cli` | test | `bkms-cli` | `make e2e-go-test` | [`AGENTS.md`](../../bkms-cli/AGENTS.md) | - |
| `bkms-cli` | test | `bkms-cli` | `make test` | [`AGENTS.md`](../../bkms-cli/AGENTS.md) | - |
| `bkms-dockerfile-generator` | build | `bkms-dockerfile-generator` | `make build` | [`Makefile`](../../bkms-dockerfile-generator/Makefile) | - |
| `bkms-dockerfile-generator` | lint | `bkms-dockerfile-generator` | `make golangci-lint` | [`Makefile`](../../bkms-dockerfile-generator/Makefile) | - |
| `bkms-dockerfile-generator` | lint | `bkms-dockerfile-generator` | `make lint` | [`Makefile`](../../bkms-dockerfile-generator/Makefile) | - |
| `bkms-dockerfile-generator` | test | `bkms-dockerfile-generator` | `make test` | [`Makefile`](../../bkms-dockerfile-generator/Makefile) | - |
| `bkms-server` | build | `bkms-server` | `make build` | [`AGENTS.md`](../../bkms-server/AGENTS.md) | - |
| `bkms-server` | build | `bkms-server` | `make build-dev` | [`Makefile`](../../bkms-server/Makefile) | - |
| `bkms-server` | lint | `bkms-server` | `make golangci-lint` | [`Makefile`](../../bkms-server/Makefile) | [`.golangci.yaml`](../../bkms-server/.golangci.yaml) |
| `bkms-server` | lint | `bkms-server` | `make lint` | [`AGENTS.md`](../../bkms-server/AGENTS.md) | [`.golangci.yaml`](../../bkms-server/.golangci.yaml) |
| `bkms-server` | test | `bkms-server` | `make test` | [`AGENTS.md`](../../bkms-server/AGENTS.md) | - |
| `bkms-ui` | build | `bkms-ui` | `pnpm build` | [`package.json`](../../bkms-ui/package.json) | - |
| `bkms-ui` | lint | `bkms-ui` | `pnpm lint` | [`AGENTS.md`](../../AGENTS.md) | [`eslint.config.mjs`](../../bkms-ui/eslint.config.mjs)、[`stylelint.config.mjs`](../../bkms-ui/stylelint.config.mjs) |
| `bkms-ui` | lint | `bkms-ui` | `pnpm stylelint` | [`package.json`](../../bkms-ui/package.json) | [`eslint.config.mjs`](../../bkms-ui/eslint.config.mjs)、[`stylelint.config.mjs`](../../bkms-ui/stylelint.config.mjs) |
| `bkms-ui` | test | `bkms-ui` | `pnpm test:unit` | [`package.json`](../../bkms-ui/package.json) | - |
<!-- HARNESS_WORK_UNIT_VERIFICATION:END -->

| 检查项 | 验证方式 | 跳过条件 |
|-------|---------|---------|
| 对照任务 Spec | 逐条检查需求点 | 无 |
| 文档已同步更新 | 检查关联文档 | 无文档影响 |
| API 契约同步 | 改动 bkms-server API 后重跑 make apidocs 并 diff | 未涉及 API 改动 |

### 2.2 验证失败处理

- 验证失败 → 自动回到执行循环修复
- 连续多次修复失败 → 暂停并请求人工介入

## 3. 任务漂移检测

| 信号 | 含义 | 处理方式 |
|------|------|---------|
| Agent 开始处理 Spec 以外的任务 | 任务漂移 | 回退到最近的检查点 |
| 修改与任务无关的文件 | 范围蔓延 | 撤销变更并提醒 |
| 手改 bkms-server/docs/apis/ 生成物 | 违反 codegen 约定 | 撤销，改跑 make apidocs |

## 检查清单

- [ ] Agent Loop 执行循环已定义
- [ ] 预完成检查清单已制定，且工作单元验证表由发现器/渲染器生成（无虚构命令）
- [ ] 任务漂移检测规则已明确
