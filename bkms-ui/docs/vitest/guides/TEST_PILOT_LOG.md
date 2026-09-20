# 已知问题速查（Pilot Log）

> 只记录**会影响后续场景写法**的环境事实。单场景修 bug 过程不记；验收摘要写台账场景卡，不在此双写。
>
> 新条目：问题一句话 / 根因一句话 / 结论一句话 / 回写去向。打法细节见 `TEST_PLAYBOOK.md`。

## 速查表

| 问题 | 根因 | 结论 | 回写 |
|---|---|---|---|
| bkui FormItem blur 校验 unhandled rejection | 组件库缺陷 | `dangerouslyIgnoreUnhandledErrors` 已关并由 setup 分类器兜底；升级 bkui 后复核 | vite / setup |
| jest-dom v7 与 TL/vue8 peer 冲突 | 生态错位 | 锁定 jest-dom v6 | package |
| stub `isValid` 写成 Promise 导致假绿 | 未读真实 defineExpose | 写 stub 前先读契约 | PLAYBOOK §2 |
| 消极断言 `waitFor(未调用)` 首检即过 | 无正向消费信号 | 先等 guard `vi.fn` 再负断言 | PLAYBOOK §1 |
| vxe 在 jsdom 不渲染行；纯字段列垫片仍不可断言 | 依赖真实布局 | `installVxeShims` + 必要时 TableStub；列表必连跑 | PLAYBOOK §3 / mock-table |
| 文件级 `vi.mock` 全文件生效 | Vitest 机制 | 同路径真实组件用 `importActual` | PLAYBOOK §5 |
| bkui Dialog/Sideslider 延迟显隐；错误走 tooltip | modal setTimeout；v-bk-tooltips | `findBy*` + `is-error` | PLAYBOOK §6 |
| 全局注册组件（RouterView）mock 无效 | 非模块解析路径 | `global.components` 打桩并回传 slot 参数 | PLAYBOOK §7 |
| `history.back` spy 为 0 | 实为 `history.go(-1)` | spy `history.go`；路由完成用 `waitFor` | PLAYBOOK §8 |
| 模块级单例 ref 测试间泄漏 | composable 导出共享状态 | `beforeEach` 手动重置 | PLAYBOOK / helm-deploy |
| Proxy anyService 静默吞调用 | 兜底过宽 | 用 `createAnyServiceMock`（带 warning）；`Failed to parse URL` 须补 mock | mock-service |
| watch 模式与 `vitest run` 结果不一致 | 增量缓存 + 并行超时 | **验收以 `vitest run` 全量为准** | GUIDELINE §4.2 |
| pinia store setup 内 `useI18n()` 抛错 | vue-i18n ≥11.1.12 约束 | 改用全局 i18n 实例（space.ts 已修） | stores/space |
| 页面经 ref 调用 `getVxeTableInstance().scrollTo()` 逃逸成 unhandled error | TableStub 未暴露该契约 | TableStub 已 `expose({ getVxeTableInstance })`；列表页 stub 后缺 ref 方法先查页面 watch/回调 | mock-table |
| getEnv mock 返回 `undefined` 走「获取详情失败」分支 | 组件直接解引用 `detail.appDeployStatuses` | mock 接口返回值须按真实 shape 给对象（空对象即可） | delete-env-action.vue:81 |
| bkui Dialog 关闭后 `getByText` 仍查得到、waitFor 消失断言超时 | 未传 `render-directive="if"` 时默认 'show' 模式，关闭仅 v-show 隐藏不销毁 DOM（modal/index.js:473-476）；getByText 不筛隐藏元素 | 断言关闭用 `not.toBeVisible()`；需「消失」语义则组件显式传 `render-directive="if"` | PLAYBOOK §6 / delete-comfirm.vue:25 |
| 全局 `restoreMocks: true` 会红 2 条（height-chain） | mock 工厂里一次性 `mockResolvedValue`、beforeEach 不重设的 vi.fn 被 mockRestore 清空实现，await 后 `undefined.then` | 禁开全局 restoreMocks；spyOn 泄漏目前仅 router-guards 一处且被每文件 jsdom 隔离兜住，新写 spyOn 请文件内自行 restore | height-chain.test.ts:43-52 |

## 环境事实（前置）

- `vite.config.mts`：bkui-vue / monaco test.alias，仅影响 vitest。
- `test/setup.ts`：jsdom 垫片 + 全局 cleanup + unhandled 分类。
- TL/vue、user-event、jest-dom 已装；helpers 见 GUIDELINE §3 条款 7。
