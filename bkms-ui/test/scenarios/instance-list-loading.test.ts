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
 * 场景级测试：实例列表加载状态。
 * 回归路径：慢首包中隐藏页面 → 恢复后展示骨架屏 → 大列表返回（加载态/成功态，V = 2）。
 * 保留真实 InstanceList、Skeleton 与实例数据源；隔离 API、Store、操作入口及表格展示。
 */
import { render, screen } from '@testing-library/vue';
import { flushPromises } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import InstanceList from '~/pages/application/detail/deploy/instance-list/instance-list.vue';

import { i18nGlobalMocks } from '../helpers/mock-i18n';

import type { AppInstanceOutputObj } from '~/@types/v1/instance';

const mocks = vi.hoisted(() => ({
  listAppInstances: vi.fn(),
  watchAppInstances: vi.fn(),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('~/api/modules/v1', () => ({ InstanceService: mocks }));
vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({ appID: 'app-1', app: 'demo', fetchAppDetail: async () => undefined }),
}));
vi.mock('~/stores/trpc-deploy', () => ({
  useTrpcDeployStore: () => ({
    curEnvItem: { name: 'prod', cluster: { isFederation: true } },
    getDeploySpec: async () => undefined,
  }),
}));
vi.mock('bkui-vue', () => ({ SearchSelect: { template: '<div />' } }));
vi.mock('~/components/skeleton/skeleton-layout', () => ({
  default: {
    shape: { template: '<span />' },
    table: { template: '<div role="status" aria-label="正在加载实例" />' },
  },
}));
vi.mock('~/components/table-exception.vue', () => ({ default: { template: '<p>暂无数据</p>' } }));
vi.mock('~/composables/use-search-filter', async () => {
  const { ref } = await import('vue');
  return { default: () => ({ filterOptions: ref({}), handleFilterChange: vi.fn() }) };
});
vi.mock('~/pages/application/detail/deploy/instance-list/instance-utils', () => ({
  isPolarisHealthy: () => false,
  matchesInstanceIPFilter: () => true,
}));
vi.mock('~/pages/application/detail/deploy/instance-list/composables/instance-watch-notifier', () => ({
  notifyInstanceWatchError: vi.fn(),
}));
vi.mock('~/pages/application/detail/deploy/instance-list/composables/use-instance-list-controller', () => ({
  useInstanceListController: () => ({
    canAdjustWeight: false,
    canGrayDeploy: false,
    handleRowAction: vi.fn(),
    instanceActions: {},
  }),
}));
vi.mock('~/pages/application/detail/deploy/instance-list/components/instance-actions-host.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/components/instance-batch-toolbar.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/components/instance-table.vue', () => ({
  default: {
    props: ['data'],
    template: `<table><tbody>
      <tr v-for="row in data" :key="row.id"><td>{{ row.id }}</td></tr>
      <tr v-if="!data.length"><td><slot name="empty" /></td></tr>
    </tbody></table>`,
  },
}));

interface ListResult {
  resourceVersion: string;
  results: AppInstanceOutputObj[];
}

function deferredList() {
  let resolve!: (value: ListResult) => void;
  const promise = new Promise<ListResult>(done => {
    resolve = done;
  });
  return { promise, resolve: (results: AppInstanceOutputObj[]) => resolve({ results, resourceVersion: 'rv-1' }) };
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe('实例列表：加载状态', () => {
  it('当用户返回浏览器页签时，应持续展示骨架屏直到大列表首包返回', async () => {
    let visibility: DocumentVisibilityState = 'visible';
    vi.spyOn(document, 'visibilityState', 'get').mockImplementation(() => visibility);
    const first = deferredList();
    const resumed = deferredList();
    mocks.listAppInstances.mockReturnValueOnce(first.promise).mockReturnValueOnce(resumed.promise);
    render(InstanceList, { props: { hasDeployRecord: true }, global: { mocks: i18nGlobalMocks } });
    expect(screen.getByRole('status', { name: '正在加载实例' })).toBeInTheDocument();

    visibility = 'hidden';
    document.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();
    expect(mocks.listAppInstances.mock.calls[0][1].signal.aborted).toBe(true);

    visibility = 'visible';
    document.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();

    expect(mocks.listAppInstances).toHaveBeenCalledTimes(2);
    expect(screen.getByRole('status', { name: '正在加载实例' })).toBeInTheDocument();
    expect(screen.queryByText('暂无数据')).not.toBeInTheDocument();

    resumed.resolve(Array.from({ length: 2000 }, (_, index) => ({ id: `pod-${index}` })));
    await flushPromises();

    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    expect(screen.getByRole('cell', { name: 'pod-1999' })).toBeInTheDocument();
    expect(screen.queryByText('暂无数据')).not.toBeInTheDocument();
  });
});
