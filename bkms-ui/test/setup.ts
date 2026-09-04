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
