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

import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/vue';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import DeployPage from '~/pages/application/detail/deploy/deploy.vue';
import { useAppDetail } from '~/stores/app-detail';

import { i18nGlobalMocks, I18nTStub } from '../helpers/mock-i18n';

const mocks = vi.hoisted(() => ({
  env: {
    name: 'prod',
    displayName: '生产环境',
    kind: 'standard',
    type: 'production',
    status: 'Ready',
    cluster: { isFederation: false },
  },
  latest: vi.fn(),
  records: vi.fn(),
  create: vi.fn(),
  build: vi.fn(),
  updateInstances: vi.fn(),
  federation: vi.fn(),
  envVars: vi.fn(),
}));
vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());
vi.mock('vue-router', async () => {
  const { reactive } = await import('vue');
  const route = reactive({ params: { space: 'ws-1', name: 'app-a' }, query: { activeTab: 'overview' } });
  return {
    useRoute: () => route,
    useRouter: () => ({
      replace: ({ query }: { query: { activeTab: string } }) => {
        route.query = query;
      },
    }),
  };
});
vi.mock('~/stores/space', () => ({ useSpaceStore: () => ({ currentSpace: 'ws-1' }) }));
vi.mock('~/api/modules/v1', () => ({
  EnvService: { listFeatureEnvs: vi.fn().mockResolvedValue([]) },
  AppSpecService: { getEnvEffectiveAppSpecResources: vi.fn().mockResolvedValue({ replicas: 2 }) },
  InstanceService: { updateAppInstances: mocks.updateInstances },
}));
vi.mock('~/pages/application/detail/deploy/use-deploy', () => ({
  useDeployAPIs: () => ({
    listLatestDeployRecords: mocks.latest,
    listDeployRecords: mocks.records,
    createDeployDirectly: mocks.create,
    buildAndCreateDeploy: mocks.build,
  }),
}));
vi.mock('~/pages/application/detail/deploy/use-federation-resource-precheck', () => ({
  useFederationResourcePrecheck: () => ({ check: mocks.federation }),
}));
vi.mock('~/pages/application/detail/deploy/use-env-var-precheck', () => ({
  useEnvVarPrecheck: () => ({ check: mocks.envVars }),
}));
vi.mock('~/pages/application/components/image-select.vue', () => ({
  default: {
    props: { value: String },
    emits: ['update:value'],
    template: '<input aria-label="镜像 Tag" :value="value" @input="$emit(\'update:value\', $event.target.value)" />',
  },
}));
vi.mock('~/components/env-select-panel/env-select-panel.vue', async () => {
  const { defineComponent } = await import('vue');
  return {
    default: defineComponent({
      emits: ['update:envList'],
      mounted() {
        this.$emit('update:envList', [mocks.env]);
      },
      template: '<div />',
    }),
  };
});
vi.mock('~/components/code-revision/code-revision-field.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/overview/deploy-overview.vue', () => ({
  default: {
    emits: ['view-instances'],
    template: "<button @click=\"$emit('view-instances', 'prod')\">查看隐藏环境</button>",
  },
}));
vi.mock('~/pages/application/detail/deploy/quickly-deploy.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/instance-list/instance-list.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/multi-env-instance-table.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/instance-list/scale-instance.vue', () => ({
  default: { template: '<div />' },
}));
vi.mock('~/pages/application/detail/deploy/deploy-history.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/deploy-event.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/components/topo/index.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/create-feature-env.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/feature-env-sideslider.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/deploy/remove-deploy.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/components/view-build-log/index.vue', () => ({ default: { template: '<div />' } }));
vi.mock('~/pages/application/detail/components/view-build-log/build-log-panel.vue', () => ({
  default: { template: '<div />' },
}));

const FlexRowStub = { template: '<div><slot name="left" /><slot name="right" /></div>' };

beforeEach(() => {
  vi.clearAllMocks();
  setActivePinia(createPinia());
  const store = useAppDetail();
  store.updateAppID('app-a');
  store.updateAppType('trpc');
  store.appDetail = { id: 'app-a', visibleEnvNames: ['test'] };
  mocks.latest.mockResolvedValue({ stage: 'deploy', status: 'deployed', hasDeployRecord: true });
  mocks.records.mockResolvedValue({ results: [{ imageTag: 'image:old' }] });
  mocks.updateInstances.mockResolvedValue({});
  mocks.create.mockResolvedValue({});
  mocks.federation.mockResolvedValue([]);
  mocks.envVars.mockResolvedValue({});
});

async function viewDeployedEnv(federation = false) {
  const { useRoute } = await import('vue-router');
  useRoute().query.activeTab = 'overview';
  mocks.env.cluster.isFederation = federation;
  render(DeployPage, {
    global: {
      mocks: i18nGlobalMocks,
      components: { 'i18n-t': I18nTStub },
      stubs: {
        FlexRow: FlexRowStub,
        TabHeader: true,
        ColorIcon: true,
        KeyValueBadge: true,
        Popover: { template: '<div><slot /></div>' },
      },
      directives: { 'bk-tooltips': {} },
    },
  });
}

describe('隐藏环境的维护入口', () => {
  it.each(['trpc', 'taf'] as const)('当 %s 应用查看隐藏环境时，仍可仅更新镜像且不调用创建部署预检', async type => {
    useAppDetail().updateAppType(type);
    await viewDeployedEnv();
    await userEvent.click(screen.getByRole('button', { name: '查看隐藏环境' }));
    const openButton = await screen.findByRole('button', { name: '部署' });
    expect(openButton).toBeEnabled();
    await userEvent.click(openButton);
    expect(await screen.findByRole('radio', { name: '镜像+配置' })).toBeDisabled();
    expect(screen.getByRole('radio', { name: '仅配置' })).toBeDisabled();
    expect(screen.getByRole('radio', { name: '仅镜像（原地重启）' })).toBeChecked();
    await userEvent.type(screen.getByRole('textbox', { name: '镜像 Tag' }), 'new-tag');
    const buttons = screen.getAllByRole('button', { name: '部署' });
    await userEvent.click(buttons[buttons.length - 1]);
    await waitFor(() =>
      expect(mocks.updateInstances).toHaveBeenCalledWith({
        appID: 'app-a',
        envName: 'prod',
        imageTag: 'new-tag',
        updateStrategy: 'InplaceUpdate',
        instanceIDs: [],
      }),
    );
    expect(mocks.federation).not.toHaveBeenCalled();
    expect(mocks.envVars).not.toHaveBeenCalled();
    expect(mocks.create).not.toHaveBeenCalled();
    expect(mocks.build).not.toHaveBeenCalled();
  });

  it('当隐藏环境属于联邦集群时，仍禁止不受支持的原地镜像更新', async () => {
    await viewDeployedEnv(true);
    await userEvent.click(screen.getByRole('button', { name: '查看隐藏环境' }));
    expect(await screen.findByRole('button', { name: '部署' })).toBeDisabled();
    expect(mocks.updateInstances).not.toHaveBeenCalled();
    expect(mocks.envVars).not.toHaveBeenCalled();
  });

  it('当环境可见时，仍默认更新镜像和配置，并执行创建部署预检', async () => {
    useAppDetail().appDetail = { id: 'app-a', visibleEnvNames: [] };
    await viewDeployedEnv();
    await userEvent.click(screen.getByRole('button', { name: '查看隐藏环境' }));
    await userEvent.click(await screen.findByRole('button', { name: '部署' }));
    expect(await screen.findByRole('radio', { name: '镜像+配置' })).toBeChecked();
    expect(screen.getByRole('radio', { name: '仅配置' })).toBeEnabled();
    expect(mocks.envVars).toHaveBeenCalledWith('prod');
    await userEvent.type(screen.getByRole('textbox', { name: '镜像 Tag' }), 'new-tag');
    const buttons = screen.getAllByRole('button', { name: '部署' });
    await userEvent.click(buttons[buttons.length - 1]);
    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith({ appID: 'app-a', envName: 'prod', imageTag: 'new-tag', replicas: 2 }),
    );
    expect(mocks.updateInstances).not.toHaveBeenCalled();
  });
});
