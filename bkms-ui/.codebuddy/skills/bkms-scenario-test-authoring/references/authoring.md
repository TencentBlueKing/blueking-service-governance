# 写用例模板与常见陷阱（步骤③）

## 文件与命名

- 位置：`test/scenarios/<业务模块名>.test.ts`，一个场景族一个文件。
- `it` 标题：**「当用户____时，应____」**；`describe`：**「模块：交互主题」**。
- 文件头注释（强制）——新模块没有台账编号，文件头就是该模块的场景说明载体：

  ```typescript
  /**
   * 场景级测试：<模块>（ROUTES 未收录，本头即场景说明）
   *
   * V = n（判定节点：…、…、…）
   * 路径：① … ② … ③ …
   * stub/mock：…
   * backlog：…（不确定项，待业务确认）
   */
  ```
- 不含评审分数与过程记录；backlog 里的不确定项确认后回写断言并更新该行。

## 查询与断言

- 主力 `getByRole` / `getByText` / `findBy*`（异步信号一律 `findBy*` / `waitFor`）；允许 placeholder / displayValue / `getByTestId`（仅 stub 契约标记）。
- 禁止 CSS 类名与内部 DOM 结构查询。**唯一例外**：目标信号 jsdom 不可达（bkui tooltip 文案、`is-error` 态、无 role 的组件库控件）时允许组件库内部类名作最后手段，用例须注释原因。
- 交互用 `@testing-library/user-event`；断言用户可见行为。
- P0/P1 场景保证正向 + 反向 + 边界/异常。

## mock 边界四件套

| 层 | 处理 | 要点 |
|---|---|---|
| API | `vi.mock('~/api/modules/v1', ...)` | 各 service 方法给 `vi.fn().mockResolvedValue(...)`；mock 返回值必须按真实 shape 给**对象**（`undefined` 会误入失败分支） |
| @blueking/table | `installVxeShims()` + tableMockFactory | TableStub 已含 `getVxeTableInstance` 契约；纯 field 列 jsdom 不产出 DOM |
| bkui-vue | partial mock | 只替换命令式 API（`Message`），Dialog/Button/Input 等保持真实渲染 |
| 子组件 | 契约 stub（仅非验证对象） | 先读真实 `defineExpose` 契约再写 stub；同步/异步要对齐 |

- `vi.mock` 内动态 `import()` 工厂（`await import('../helpers/mock-xxx')`），避免 hoisting TDZ。
- 页面子组件 setup 依赖 pinia 时，测试内 `createPinia()` + `setActivePinia` 自装——禁止依赖其他文件残留的全局 activePinia（CI worker 调度下随机失败）。
- 依赖用户存储的 store：照 `mock-i18n.ts` 的三件套模式（hoisted vi.fn + setter + 工厂）。

## 高频写法

- 弹窗关闭断言：未传 `render-directive="if"` 的 bkui Dialog 关闭后 DOM 仅 v-show 隐藏，用 `not.toBeVisible()`，不要 waitFor「消失」。
- 破坏性操作（删除类）：校验禁用态、失败保留弹窗可重试、防重复提交（pending 期连点断言只调一次）。
- 组件契约单测（命令式入口 `defineExpose`）：用 Harness 复刻宿主页 2 行接线（button `show(data)` + 事件监听），别整页渲染。

## 常见陷阱（实证）

- `toHaveBeenCalledWith` 要求**全参匹配**（两参调用只断言第一参会假绿）。
- bkui 空值校验展示 rules 末条文案而非自定义 message——文案断言按末条确定性文案写并注释依据。
- 提交失败停留在当前步骤：断言「停留信号」（如 loading 复位、弹窗保留），别断言会消失的东西。
- 编辑模式回显字段 disabled：辅助函数按 mode 跳过填写。
- stub 返回值非响应式：面板/列表类用例须在挂载前赋值（stub 注释留档该限制）。
- 接口失败用例：等失败链路完成信号（loading 复位 / 按钮 enabled）再做稳定态断言，防「弹窗本就开着」式假绿。
