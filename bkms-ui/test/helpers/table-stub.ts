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

import { defineComponent, h, ref } from 'vue';

/** Table/Column 插槽 vnode 的最小结构（stub 只消费 default/header 插槽与 field/label props） */
interface SlotColumn {
  props?: { field?: string; label?: string };
  children?: {
    default?: (s: unknown) => unknown;
    header?: (s: unknown) => unknown;
  };
}

/**
 * Table stub：按 data 渲染行并逐列求值插槽；data 为空时渲染 #empty 插槽（空态可断言）。
 * 行标记 data-testid="table-row"，空态区块标记 data-testid="table-empty"。
 * 页面传入 :settings（useTableSettings 产物）时，额外按 settings.checked 渲染列头，
 * 标记 data-testid="table-header-{field}"（默认勾选基线由此可断言）；
 * 未传 settings 的既有用法不渲染列头，行为不变。
 */
export const TableStub = defineComponent({
  name: 'TableStub',
  props: {
    data: { type: Array, default: () => [] },
    settings: { type: Object, default: null },
  },
  setup(props, { slots, expose }) {
    /** 当前高亮行（模拟 vxe row-config.isCurrent + setCurrentRow） */
    const currentRow = ref<null | Record<string, unknown>>(null);
    // 契约：页面可能经 ref 调用 getVxeTableInstance() 上的方法
    //  - scrollTo()：如 env.vue:545 恢复滚动位置
    //  - setCurrentRow(row)：保存后高亮定位到刚保存的行，对应行渲染 data-current="true"
    expose({
      getVxeTableInstance: () => ({
        scrollTo: () => Promise.resolve(),
        setCurrentRow: (row: Record<string, unknown>) => {
          currentRow.value = row;
          return Promise.resolve();
        },
      }),
    });
    return () => {
      const columns = (slots.default?.() ?? []).filter(Boolean) as SlotColumn[];
      const rows = props.data as Record<string, unknown>[];
      const checked: null | string[] = (props.settings as null | { checked?: string[] })?.checked ?? null;
      const headers = checked
        ? columns
            .map((col: SlotColumn, i: number) => {
              const { field = '', label = '' } = col.props ?? {};
              // 面板按 field 收录；无 field 列（如操作列）不受设置控制，恒显示
              if (field && !checked.includes(field)) return null;
              return h(
                'span',
                { key: i, 'data-testid': `table-header-${field || i}` },
                // #header 插槽（如 CustomFilter 筛选头）优先，否则退回 label
                col.children?.header ? (col.children.header({}) as never) : label,
              );
            })
            .filter(Boolean)
        : null;
      const body = rows.length
        ? rows.map((row, rowIndex) =>
            h(
              'div',
              {
                key: rowIndex,
                'data-testid': 'table-row',
                // 高亮行标记：供「保存后选中该行」类断言使用
                'data-current': row === currentRow.value ? 'true' : undefined,
              },
              columns.map((col: SlotColumn, i: number) =>
                h('span', { key: i }, [
                  col.children?.default
                    ? (col.children.default({ row, rowIndex }) as never)
                    : String(row?.[col.props?.field ?? ''] ?? '--'),
                ]),
              ),
            ),
          )
        : slots.empty
          ? h('div', { 'data-testid': 'table-empty' }, [slots.empty()])
          : null;
      return h('div', { 'data-testid': 'table-stub' }, [headers, body]);
    };
  },
});
