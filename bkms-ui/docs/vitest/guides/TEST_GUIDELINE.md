# Vitest 交互测试指南

> 本文件是 Vitest 场景测试的**最高基准（章程）**。与本指南冲突时，以本指南为准。
>
> 文件体系（均位于 `docs/vitest/`）：
>
> ```
> guides/TEST_GUIDELINE.md          ← 本文件：目的 + 方法 + 规范 + 流程（章程）
> guides/TEST_SCENARIOS_ROUTES.md   ← 路由全量枚举评分 + 场景台账（审批与验收摘要）
> guides/TEST_PLAYBOOK.md          ← 跨场景可复用打法（按主题，不按 S 编号）
> guides/TEST_PILOT_LOG.md          ← 已知问题速查（仅影响后续写法的环境事实）
> test/scenarios/*.test.ts          ← 用例落地层（文件头注释 = 场景说明）
> ```
>
> **新场景默认零新增 md**：只更新台账场景卡 + 测试文件头；禁止再建 `TEST_PILOT_SXX` / `TEST_REVIEW_SXX`。仅当沉淀出**新的跨场景打法**时，追加一行到 `TEST_PLAYBOOK.md`。

## 1. 设计目的

1. **固化复杂交互（回归保护网）**：场景级用例将「用户可感知的正确行为」固化为可执行断言。
2. **用例即文档**：`it` 标题使用业务语言；跑 `pnpm test:unit` 的输出即功能说明书。
3. **降低回归成本 / 提升重构信心**：断言用户可见行为，不绑内部实现。

### 非目标

- 不追求 100% 覆盖率；不测直线逻辑（V=1）、样式细节、整页 E2E、纯展示组件。

## 2. 方法依据

### 2.1 依据链

```
① src/modules/router.ts 路由表 → 不遗漏
② 频率 × 影响 × 复杂度（各 1~3）→ ≥7 入选 / 落选须留理由
③ 基本路径 / 等价类 / 判定表 / 状态迁移 → 场景内 V 定量
```

### 2.2 挑选与颗粒度

- 只做评分表入选对象；禁止未经评分直接新增场景。
- **黄金法则**：含 ≥1 个判定节点才建场景；破坏性操作例外。
- **V = 用户可感知判定分支数 + 1**（基本路径法）；写入路径清单交审批，用例数 ≈ V。
- 一个 `it` = 一个用户可感知目标；`describe` =「模块：交互主题」。

## 3. 规范约束（违反即打回）

1. 存量场景必须来自评分表入选对象。**ROUTES 已冻结（S1~S19 不再增补）**：新模块/新页面按 `bkms-scenario-test-authoring` skill 走轻量流程——路径清单写测试文件头，不进册（日后是否补录由维护者决定）。
2. 写用例前先产出路径清单（含校准 V）交审批。
3. `it` 标题：**「当用户____时，应____」**；`describe`：**「模块：交互主题」**。
4. 查询以 `getByRole` / `getByText` / `findBy*` 为主；允许 placeholder / displayValue / `getByTestId`（仅 stub 契约）；禁止依赖 CSS 类名与内部 DOM 结构。**唯一例外**：目标信号在 jsdom 不可达时（如 bkui tooltip 文案、`is-error` 态、无 role 的组件库控件），允许以组件库内部类名为最后手段，用例须注释原因。
5. 交互用 `@testing-library/user-event`；断言用户可见行为（组件契约单测可用 VTU emit，见 PLAYBOOK）。
6. API 一律 `vi.mock`，禁止真实请求；复用 `test/setup.ts`。
7. **共享 helpers（禁止重复造轮子）**：`vi.mock` 内须动态 `import()`，避免 hoisting TDZ。

   | 文件                           | 用途                      |
   | ------------------------------ | ------------------------- |
   | `test/helpers/mock-i18n.ts`    | vue-i18n                  |
   | `test/helpers/mock-router.ts`  | vue-router                |
   | `test/helpers/mock-table.ts`   | vxe 垫片 + 表格 stub      |
   | `test/helpers/mock-service.ts` | Proxy anyService          |
   | `test/stubs/monaco-editor.ts`  | monaco alias              |
   | `test/stubs/bkui-vue-lite.ts`  | 轻量 bkui（按需 vi.mock） |

   目录约定：**helpers/ 放“生成 mock 的工厂函数”（如 `tableMockFactory()`），stubs/ 放“整包替身模块本体”（被 alias 或被 vi.mock 直接 import）**。

8. 全局 `afterEach(cleanup)` 已在 `test/setup.ts`；勿重复手写。VTU 场景须自行 `unmount`（见 PLAYBOOK）。
9. P0/P1 须正向 + 反向 + 边界/异常。
10. 推断规则标 `推断/需确认`。
11. 场景用例放 `test/scenarios/`，文件名 = 业务模块名（无 S 前缀）；编号映射见台账。
12. 单文件 **tests 段** < 10s（不含冷启动编译）。
13. **场景文件头注释**（强制，见 §4.4）：编号 + 台账链接、V 与路径、stub 边界、backlog；禁止写过程轮次与评审分数。

## 4. 场景实施与验收

> 打法见 `TEST_PLAYBOOK.md`。

### 4.1 实施流程

1. 摸底源码与 `defineExpose` 契约 → 再写 stub。
2. 路径清单（V + it 草稿）→ 审批。
3. 写用例（规范 §3 + PLAYBOOK）。
4. 验证（§4.2）。
5. 收尾（§4.3）：**只改台账场景卡 + 测试文件头**；有新打法则改 PLAYBOOK 一行。

### 4.2 验收标准

- [ ] 路径清单已审批，与用例一一对应
- [ ] 符合 §3
- [ ] `vitest run` 全绿且连跑 3 次无 flaky（watch 结果不作全绿依据）
- [ ] tests 段耗时符合条款 12
- [ ] 变异 ≥3 处注入后对应用例变红，并立即恢复业务代码
- [ ] 文档性验收：不熟模块的同事能从 `it` 标题复述交互（可人工）

**不要求**撰写独立六维评审 md 或多轮复审报告；验收字段写入台账场景卡即可。

### 4.3 场景收尾

- 通过 → 更新 `TEST_SCENARIOS_ROUTES.md` 场景卡（用例路径、V、连跑、变异、backlog、打法指针）；仅影响后续写法的问题记入 `TEST_PILOT_LOG.md`；新打法追加 `TEST_PLAYBOOK.md`；规范变更回写本文件。
- 不通过 → 修订指南或缩小范围，**不强行铺开**。

### 4.4 测试文件头约定

```typescript
/**
 * 场景级测试：<模块>
 * （存量场景：路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md Sxx；
 *   ROUTES 未收录的新模块：本头即场景说明，V 与路径清单直接写于下方）
 *
 * 覆盖：… → V = n
 * stub/mock：…
 * backlog：…（可选）
 */
```

## 5. 文件职责边界

| 文件                       | 职责                     | 纪律                          |
| -------------------------- | ------------------------ | ----------------------------- |
| `TEST_GUIDELINE.md`        | 章程                     | 只放稳定规则                  |
| `TEST_SCENARIOS_ROUTES.md` | 枚举 + 台账 + 验收摘要   | 不放过程轮次                  |
| `TEST_PLAYBOOK.md`         | 跨场景打法               | 按主题；每条 ≤3 句 + 示例路径 |
| `TEST_PILOT_LOG.md`        | 已知问题速查             | 只记会影响后续写法的事实      |
| `test/scenarios/*.test.ts` | 用例 + 文件头说明        | 头注释不含评审分              |
