# Vitest 场景测试 Playbook

> 跨场景可复用打法（按主题）。
> 新打法：追加本节一行即可，**不要**新建 pilot/review。

## 1. Harness / hoisted / 消极断言（S12）

- **问题**：`render()` 拿不到 `defineExpose`；`vi.mock` 工厂读不到外部变量；`waitFor(未被调用)` 首检即过。
- **做法**：包装 Harness 驱动 open/close；可变 mock 放 `vi.hoisted`；消极断言前先等正向消费信号（可观察 `vi.fn`）。
- **示例**：`test/scenarios/component-management.test.ts`

## 2. 写 stub 前先读真实契约（S12）

- **问题**：`isValid` 同步/异步写错 → Promise 恒 truthy，假绿。
- **做法**：先读子组件 `defineExpose`，stub 签名与真实一致（含 sync/async）。
- **示例**：`test/scenarios/component-management.test.ts`

## 3. vxe 垫片 + 表格 stub（S14 / S9）

- **问题**：jsdom 下 vxe 不渲染行；纯字段列即使垫片也不可断言。
- **做法**：`installVxeShims()` + 需要时 `tableMockFactory` / `TableStub`（`data-testid`）；列表场景必做连跑。
- **示例**：`test/helpers/mock-table.ts`；`test/scenarios/application-list.test.ts`、`env-management.test.ts`

## 4. 重依赖页：Proxy service + Pinia + vxe（S18）

- **问题**：页面依赖面广，逐个 mock 成本高；无 active Pinia；历史表需垫片。
- **做法**：`createAnyServiceMock`；`createPinia()`；`installVxeShims()`。出现 `Failed to parse URL` 说明有真实请求，补 mock，勿放过。
- **示例**：`test/scenarios/build-management.test.ts`

## 5. 同文件 stub 与真实组件并存（S15）

- **问题**：文件级 `vi.mock` 全文件生效，再渲染同路径真实组件仍是 stub。
- **做法**：需要真实实现时用 `vi.importActual` 绕过。
- **示例**：`test/scenarios/helm-deploy.test.ts`

## 6. bkui 弹层延迟显隐 / 校验文案（S15）

- **问题**：Dialog/Sideslider 经 `setTimeout` 置位；FormItem 错误常走 tooltip。
- **做法**：弹层用 `findBy*`；校验用 body 级 `is-error` + 负断言兜底。
- **关闭断言**：未传 `render-directive="if"` 的 Dialog 关闭后 DOM 仅 v-show 隐藏不销毁，「消失」断言会超时；用 `not.toBeVisible()` 断言关闭。
- **示例**：`test/scenarios/helm-deploy.test.ts`、`test/scenarios/env-management.test.ts`

## 7. 全局注册组件打桩（S1）

- **问题**：`<RouterView>` 等由 `app.use` 注入，不走模块 `vi.mock`。
- **做法**：`render` 的 `global.components` 注册桩；带 `v-slot` 的桩须回传插槽参数。
- **示例**：`test/scenarios/create-application.test.ts`

## 8. 纯路由逻辑（S13）

- **问题**：`history.back` spy 为 0（实为 `history.go(-1)`）；`replace` 无 Promise。
- **做法**：spy `history.go`；用 `waitFor` 等 `currentRoute`；不可达分支勿硬写。
- **示例**：`test/scenarios/router-guards.test.ts`

## 9. 弹窗表单直接测本体（S16 / S6）

- **问题**：整页入口噪音大。
- **做法**：直接渲染弹窗；`editData` 有无切换新建/编辑；Harness 承载 `v-model`；校验用「文案出现 + 成功回调未触发」。
- **示例**：`test/scenarios/public-env-var-form.test.ts`、`delete-confirm.test.ts`

## 10. 防抖 + VTU 契约（S19）

- **问题**：`advanceTimersByTimeAsync(debounceMs)` 与源码边界等值耦合；VTU 不走全局 cleanup。
- **做法**：推进 `debounceMs * 2` 并注释；`afterEach` 显式 `unmount`；stub 用 `data-testid`。
- **示例**：`test/scenarios/repo-ref-select.test.ts`、`test/stubs/bkui-vue-lite.ts`

## 11. 多态配置子模块（S3）

- **问题**：判定逻辑在 composable 内，mock 掉就测空了。
- **做法**：不 mock 核心 composable；用代表性子模块 + harness 驱动 expose；默认/环境双写入必测。
- **示例**：`test/scenarios/app-config-resources.test.ts`

## 12. 验收口径

- 以 `vitest run`（或 `pnpm test:ci`）全量为准；`pnpm test:unit` watch 缓存不可作全绿依据。
- 共享 helpers 用法见 `TEST_GUIDELINE.md` §3 条款 7。
