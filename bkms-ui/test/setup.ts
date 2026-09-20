/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */
/**
 * vitest 全局 setup（vite.config test.setupFiles）：jsdom 环境垫片。
 * 必须在此处执行而非各测试文件体内——静态 import 先于文件体运行，
 * 组件库在模块加载阶段就可能引用这些浏览器 API。
 */

// ── 未处理错误分类（预期 → warning，非预期 → 让用例失败）───────────────────────
// 机制：注册自己的处理器后，vitest 认为错误已被处理、不再报告（见 vitest 文档「未处理的错误」），
// 分类权因此转到我们手上。判定不看错误消息（业务自定义文案无法穷举），只看两类信号：
//   ① 显式哨兵：MockedApiError（测试主动抛的接口失败，生产由 interceptor 接管）
//   ② 调用链归属：栈里出现 test/ 或 src/ 的帧 → 是我们的锅（如传错 props 把组件库干崩）→ 非预期；
//      纯组件库栈或无栈（如 bkui-vue 校验 promise reject、modal 卸载后定时器回调）→ 预期。
// 策略取舍：仅按栈帧形态放行已知组件库缺陷（fail-open），带我们栈帧的错误一律不放过；
// 非预期项由 afterAll 抛错让 CI 变红。若要收紧为全 fail-closed，需先盘点组件库缺陷签名建白名单。
import { cleanup } from '@testing-library/vue';
import { afterAll, afterEach, expect } from 'vitest';

import { MockedApiError } from './helpers/mocked-api-error';
// jest-dom 语义化 DOM 断言匹配器（toBeInTheDocument / toBeDisabled 等）全局注册
import '@testing-library/jest-dom/vitest';

const unexpectedErrors: { err: unknown; test: string }[] = [];
/** 栈帧路径中出现 test/ 或 src/ 下的源码文件 */
const OUR_FRAME = /[\\/](test|src)[\\/][\w.\\/-]+\.(ts|tsx|vue|js|mts)/;

function describeError(err: unknown): string {
  return err instanceof Error ? `${err.name}: ${err.message}` : String(err);
}

// vitest 只给自己的 UI 元素上色，console 内容默认不着色，故自行加 ANSI；
// 注意不能用 process.stdout.isTTY 判断——vitest 在 worker 子进程跑用例，stdout 是管道恒为 false；
// 改为默认启用，仅 CI 或 NO_COLOR 时禁用（CI 日志里不希望出现转义码）。
const USE_COLOR = !process.env.NO_COLOR && !process.env.CI;
const paint = (color: string, text: string) => (USE_COLOR ? `\u001b[${color}m${text}\u001b[0m` : text);

function classifyUnhandled(err: unknown): void {
  const testName = expect.getState().currentTestName ?? '(文件级/用例外)';
  if (isExpectedUnhandled(err)) {
    console.warn(`${paint('33', '[expected unhandled]')} ${describeError(err)}`);
    return;
  }
  unexpectedErrors.push({ test: testName, err });
  console.error(`${paint('31', '[UNEXPECTED unhandled]')} [${testName}] ${err instanceof Error ? err.stack : err}`);
}

function isExpectedUnhandled(err: unknown): boolean {
  // ① 显式哨兵：测试主动模拟的接口失败
  if (err instanceof MockedApiError) return true;
  // ② 调用链经过我们的代码 → 非预期（典型：用例传错 props 导致组件库崩溃）。
  //    先剔除 node_modules 帧（pnpm 目录树可能含 src/ 段，如 .pnpm/<pkg>/node_modules/<pkg>/src/），
  //    避免三方包源码路径被 OUR_FRAME 误判为我们的代码
  const stack = (err as null | { stack?: string })?.stack ?? '';
  const appFrames = stack
    .split('\n')
    .filter(line => !line.includes('node_modules'))
    .join('\n');
  if (appFrames && OUR_FRAME.test(appFrames)) return false;
  // ③ 纯组件库内部栈 / 无栈（组件库已知缺陷形态）→ 预期
  return true;
}

process.on('unhandledRejection', classifyUnhandled);
process.on('uncaughtException', classifyUnhandled);
window.addEventListener('unhandledrejection', event => {
  classifyUnhandled(event.reason);
  event.preventDefault();
});
window.addEventListener('error', event => {
  classifyUnhandled(event.error ?? event.message);
  event.preventDefault();
});

afterAll(() => {
  if (unexpectedErrors.length) {
    const detail = unexpectedErrors.map(({ test, err }) => `  - [${test}] ${describeError(err)}`).join('\n');
    throw new Error(
      `出现 ${unexpectedErrors.length} 个非预期未处理错误（见上方 [UNEXPECTED unhandled] 输出）。` +
        `涉及用例：\n${detail}\n若为已知情况，请用 MockedApiError 或在 setup.ts 补充判定规则。`,
    );
  }
});

// jsdom 未实现 ResizeObserver，bkui-vue 的 ResizeLayout setup 时依赖
if (!globalThis.ResizeObserver) {
  globalThis.ResizeObserver = class {
    disconnect() {}
    observe() {}
    unobserve() {}
  } as unknown as typeof ResizeObserver;
}

// PointerEvent 在 jsdom 27+ 才实现（仓库为兼容 Node 18 基线使用 jsdom 26），组件库加载链会引用。
// 注意：空壳实现仅满足 instanceof / 类型引用，未实现 pointerId 等专有属性，不可用于事件派发
if (!globalThis.PointerEvent) {
  globalThis.PointerEvent = class PointerEvent extends MouseEvent {} as unknown as typeof PointerEvent;
}

// ── 全局 cleanup：每个用例结束后自动清理 Testing Library 创建的 DOM ──────────────
// 各测试文件不再需要手写 afterEach(cleanup)
afterEach(() => {
  cleanup();
});
