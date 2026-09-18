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
 * 列表类场景（S9/S11/S14/S15/S18）的 vxe（@blueking/table）jsdom 垫片与表格 stub。
 * 背景与打法见 docs/vitest/guides/TEST_PLAYBOOK.md「vxe 垫片 + 表格 stub」：
 *  - vxe-table 依赖元素尺寸决定渲染，jsdom 中 offsetHeight/clientHeight 恒为 0 → 表格不渲染行（需尺寸垫片）；
 *  - 纯字段列（非插槽渲染）在 jsdom 下不产出 DOM，无法断言 → 需表格 stub（方案 A）；
 *  - 页面上的 CustomFilter / useElementHeight 仍触碰 vxe DOM 工具 → 垫片与 stub 并用。
 *
 * 使用方式：
 *   import { installVxeShims, tableMockFactory } from '../helpers/mock-table';
 *   beforeAll(() => { installVxeShims(); });
 *   vi.mock('@blueking/table', tableMockFactory);
 */

import { Comment, defineComponent, h } from 'vue';

/** 安装 vxe 依赖的 jsdom 垫片（在测试文件 beforeAll 中调用一次） */
export function installVxeShims() {
  // vxe-table 的 DOM 工具会引用 HTMLDocument 判断文档类型，jsdom 未暴露该全局
  (globalThis as Record<string, unknown>).HTMLDocument = Document;
  // jsdom 未实现元素滚动 API，vxe 虚拟滚动会调用
  Element.prototype.scrollTo = function scrollTo() {};
  const size = (value: number) => ({ configurable: true, get: () => value });
  Object.defineProperty(HTMLElement.prototype, 'offsetHeight', size(600));
  Object.defineProperty(HTMLElement.prototype, 'offsetWidth', size(1200));
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', size(600));
  Object.defineProperty(HTMLElement.prototype, 'clientWidth', size(1200));
  // useElementHeight 通过 getBoundingClientRect 测高
  Element.prototype.getBoundingClientRect = function getBoundingClientRect() {
    return {
      width: 1200,
      height: 600,
      top: 0,
      left: 0,
      bottom: 600,
      right: 1200,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect;
  };
  globalThis.ResizeObserver = class {
    cb: ResizeObserverCallback;
    constructor(cb: ResizeObserverCallback) {
      this.cb = cb;
    }
    disconnect() {}
    observe(target: Element) {
      this.cb([{ target } as ResizeObserverEntry], this as never);
    }
    unobserve() {}
  } as never;
}

/** TableColumn stub：透传列插槽（供 TableStub 按行渲染），无插槽时输出注释节点占位 */
export const TableColumnStub = defineComponent({
  name: 'TableColumnStub',
  props: {
    field: { type: String, default: '' },
    label: { type: String, default: '' },
    type: { type: String, default: '' },
  },
  setup(_props, { slots }) {
    return () => (slots.default ? h('span', slots.default({ row: {}, rowIndex: 0 })) : h(Comment, ''));
  },
});

/**
 * Table stub：按 data 渲染行并逐列求值插槽；data 为空时渲染 #empty 插槽（空态可断言）。
 * 行标记 data-testid="table-row"，空态区块标记 data-testid="table-empty"。
 */
export const TableStub = defineComponent({
  name: 'TableStub',
  props: { data: { type: Array, default: () => [] } },
  setup(props, { slots, expose }) {
    // 契约：页面可能经 ref 调用 getVxeTableInstance().scrollTo()（如 env.vue:545 恢复滚动位置）
    expose({ getVxeTableInstance: () => ({ scrollTo: () => Promise.resolve() }) });
    return () => {
      const columns = (slots.default?.() ?? []).filter(Boolean);
      const rows = props.data as Record<string, unknown>[];
      const body = rows.length
        ? rows.map((row, rowIndex) =>
            h(
              'div',
              { key: rowIndex, 'data-testid': 'table-row' },
              columns.map(
                (col: { children?: { default?: (s: unknown) => unknown }; props?: { field?: string } }, i: number) =>
                  h('span', { key: i }, [
                    col.children?.default
                      ? col.children.default({ row, rowIndex })
                      : String(row?.[col.props?.field ?? ''] ?? '--'),
                  ]),
              ),
            ),
          )
        : slots.empty
          ? h('div', { 'data-testid': 'table-empty' }, [slots.empty()])
          : null;
      return h('div', { 'data-testid': 'table-stub' }, [body]);
    };
  },
});

/** vi.mock('@blueking/table') 的标准 factory：替换为轻量表格 stub */
export const tableMockFactory = () => ({
  Table: TableStub,
  TableColumn: TableColumnStub,
});
