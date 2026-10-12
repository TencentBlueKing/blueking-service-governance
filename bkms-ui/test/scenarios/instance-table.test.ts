/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License; you may not use this file except
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

import { defineComponent, h } from 'vue';
import type { VNodeChild } from 'vue';

/**
 * 场景级测试：实例列表热更新列。
 *
 * 路径清单（V = 5）：成功记录、失败记录、无记录占位符、未知状态、详情字段缺省值。
 * 热更新数据由父组件通过 data 传入；本文件只隔离表格与 UI 组件契约，不连接 API、SSE 或真实 Popover。
 */
import { render, screen, within } from '@testing-library/vue';
import { beforeAll, describe, expect, it, vi } from 'vitest';

import { I18nTStub } from '../helpers/mock-i18n';

import type { AppInstanceOutputObj } from '~/@types/v1/instance';

type TableColumnVNode = {
  children?: null | {
    default?: (slot: { row: Record<string, unknown>; rowIndex: number }) => VNodeChild;
  };
  props?: {
    'class-name'?: string;
    className?: string;
    field?: string;
  };
};

vi.mock('@blueking/table', async () => {
  const { defineComponent, h } = await import('vue');
  const TableColumn = defineComponent({
    name: 'TableColumnStub',
    props: {
      className: { type: String, default: '' },
      field: { type: String, default: '' },
    },
    setup() {
      return () => h('span');
    },
  });
  const Table = defineComponent({
    name: 'TableStub',
    props: { data: { type: Array, default: () => [] } },
    setup(props, { expose, slots }) {
      expose({ getVxeTableInstance: () => ({ scrollTo: () => Promise.resolve() }) });
      return () => {
        const columns = (slots.default?.() ?? []).filter(Boolean);
        const rows = props.data as Record<string, unknown>[];
        return h(
          'div',
          { 'data-testid': 'table-stub' },
          rows.map((row, rowIndex) =>
            h(
              'div',
              { key: rowIndex, 'data-testid': 'table-row' },
              columns.map((rawColumn, columnIndex: number) => {
                const column = rawColumn as unknown as TableColumnVNode;
                const className = column.props?.className ?? column.props?.['class-name'];
                const cell =
                  column.children?.default?.({ row, rowIndex }) ?? String(row[column.props?.field ?? ''] ?? '--');
                return h('span', { key: columnIndex, class: className }, [cell]);
              }),
            ),
          ),
        );
      };
    },
  });
  return { Table, TableColumn };
});

vi.mock('bkui-vue', async () => {
  const { defineComponent, h } = await import('vue');
  const passthrough = (name: string, tag = 'span') =>
    defineComponent({
      name,
      setup(_, { slots }) {
        return () => h(tag, slots.default?.());
      },
    });
  const dropdownMenu = passthrough('DropdownMenu', 'div');
  const dropdownItem = passthrough('DropdownItem', 'div');
  const dropdown = Object.assign(passthrough('Dropdown', 'div'), {
    DropdownMenu: dropdownMenu,
    DropdownItem: dropdownItem,
  });
  const popover = defineComponent({
    name: 'PopoverStub',
    setup(_, { slots }) {
      return () => h('div', { 'data-testid': 'popover' }, [slots.default?.(), slots.content?.()]);
    },
  });
  return {
    Button: passthrough('Button', 'button'),
    Checkbox: passthrough('Checkbox', 'input'),
    Dropdown: dropdown,
    Popover: popover,
    Tag: passthrough('Tag'),
  };
});

vi.mock('bkui-vue/lib/icon', async () => {
  const { defineComponent, h } = await import('vue');
  const Icon = (name: string) => defineComponent({ name, setup: () => () => h('i') });
  return { AngleDownLine: Icon('AngleDownLine'), Copy: Icon('Copy'), RightShape: Icon('RightShape') };
});

vi.mock('~/common/util', () => ({
  copyText: vi.fn(),
  formatTimeByTimezone: (value: string) => `formatted:${value}`,
}));
vi.mock('~/composables/use-gpa-config-polling', async () => {
  const { ref } = await import('vue');
  return { useGPAConfigPolling: () => ({ enabled: ref(false), status: ref(''), updatePolling: vi.fn() }) };
});
vi.mock('~/composables/use-language', () => ({ useLanguage: () => ({ isEnglish: false }) }));
vi.mock('~/composables/use-resource-spec-display', () => ({
  useResourceSpecDisplay: () => ({ getResourceText: () => '--', getResourceTips: () => '' }),
}));
vi.mock('~/composables/use-table-settings', async () => {
  const { ref } = await import('vue');
  return { useTableSettings: () => ({ settings: ref({ checked: [] }), handleSettingChange: vi.fn() }) };
});
vi.mock('~/pages/application/detail/deploy/instance-list/composables/use-instance-list-watch', async () => {
  const { ref } = await import('vue');
  return {
    useInstanceListWatch: () => ({
      clear: vi.fn(),
      instances: ref([]),
      lastError: ref(null),
      refresh: vi.fn(),
    }),
  };
});
vi.mock('~/stores/app-detail', () => ({ useAppDetail: () => ({ appID: 'app-1' }) }));

const passthroughComponent = (name: string) => ({
  name,
  template: '<span><slot /></span>',
});
vi.mock('~/components/hover-copy.vue', () => ({ default: passthroughComponent('HoverCopy') }));
vi.mock('~/components/middle-ellipsis.vue', () => ({ default: passthroughComponent('MiddleEllipsis') }));
vi.mock('~/components/status-dot-icon.vue', () => ({
  default: { name: 'StatusDotIcon', template: '<i data-testid="publish-status-dot" />' },
}));
vi.mock('~/components/status-icon.vue', () => ({ default: passthroughComponent('StatusIcon') }));
vi.mock('~/components/table-exception.vue', () => ({ default: passthroughComponent('TableException') }));
vi.mock('~/pages/application/detail/components/auto-scale-tag.vue', () => ({
  default: passthroughComponent('AutoScaleTag'),
}));
vi.mock('~/pages/application/detail/deploy/instance-list/components/instance-group-filter.vue', () => ({
  default: passthroughComponent('InstanceGroupFilter'),
}));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

beforeAll(() => {
  (window as Window & { i18n?: { t: (value: string) => string } }).i18n = { t: (value: string) => value };
});

async function renderTable(data: AppInstanceOutputObj[]) {
  const { default: InstanceTable } =
    await import('~/pages/application/detail/deploy/instance-list/components/instance-table.vue');
  return render(InstanceTable, {
    props: {
      data,
      envName: 'prod',
      mode: 'single',
      showEnvHeader: false,
    },
    global: {
      mocks: { $t: (value: string) => value },
      components: {
        DetailItem: defineComponent({
          props: { label: { type: String, default: '' } },
          setup(props, { slots }) {
            return () => h('div', [h('span', props.label), slots.default?.()]);
          },
        }),
        'i18n-t': I18nTStub,
      },
      directives: { 'bk-tooltips': () => undefined },
    },
  });
}

describe('实例列表：热更新列', () => {
  it('成功记录展示已更新状态和完整详情字段', async () => {
    await renderTable([
      {
        id: 'pod-success',
        latestPublish: {
          binaryName: 'demo',
          md5: '0123456789abcdef',
          operator: 'admin',
          status: 'success',
          updatedAt: '2026-10-14T10:00:00Z',
        },
      },
    ]);

    expect(screen.getByText('已更新')).toBeInTheDocument();
    expect(screen.getByText('demo')).toBeInTheDocument();
    expect(screen.getByText('01234567')).toBeInTheDocument();
    expect(screen.getByText('admin')).toBeInTheDocument();
    expect(screen.getByText('formatted:2026-10-14T10:00:00Z')).toBeInTheDocument();
    expect(screen.getByText('成功')).toBeInTheDocument();
  });

  it('失败记录展示更新失败状态和失败原因', async () => {
    await renderTable([
      {
        id: 'pod-failed',
        latestPublish: { status: 'failed', message: 'binary mismatch' },
      },
    ]);

    expect(screen.getByText('更新失败')).toBeInTheDocument();
    expect(screen.getByText('失败')).toBeInTheDocument();
    expect(screen.getAllByText('binary mismatch')).toHaveLength(2);
  });

  it('没有热更新记录时展示占位符', async () => {
    await renderTable([{ id: 'pod-empty' }]);

    const publishCell = screen.getByTestId('table-row').querySelector<HTMLElement>('.instance-publish-col');
    expect(publishCell).not.toBeNull();
    expect(publishCell).toHaveTextContent('--');
    expect(screen.queryByText('已更新')).not.toBeInTheDocument();
    expect(screen.queryByText('更新失败')).not.toBeInTheDocument();
  });

  it('详情字段缺省时分别展示占位符', async () => {
    await renderTable([{ id: 'pod-incomplete', latestPublish: { status: 'success' } }]);

    const publishCell = screen.getByTestId('table-row').querySelector<HTMLElement>('.instance-publish-col');
    if (!publishCell) throw new Error('热更新单元格未渲染');
    const details = within(publishCell).getByTestId('popover');
    expect(within(details).getAllByText('--')).toHaveLength(4);
  });

  it('未知热更新状态原样展示且不渲染成功失败状态点', async () => {
    await renderTable([{ id: 'pod-publishing', latestPublish: { status: 'publishing' } }]);

    expect(screen.getAllByText('publishing')).toHaveLength(2);
    expect(screen.queryByTestId('publish-status-dot')).not.toBeInTheDocument();
  });
});
