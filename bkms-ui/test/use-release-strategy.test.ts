import { nextTick, reactive, ref } from 'vue';
import type { Component, MaybeRefOrGetter } from 'vue';

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
 * composable 级测试：useReleaseStrategy 的实例数锁定 / 同步与分批失效回退
 *
 * 纯函数 resolveCanaryBlockReason 的优先级由 release-strategy-eligibility.test.ts 覆盖，
 * 这里只验证「分批更新不能同时扩缩容」这条保障在组合层是否真的生效：
 *   1. 选中分批后，表单实例数被强制同步为当前生效值（旧值不允许随分批一起提交）
 *   2. 生效实例数无效（undefined / 0）时不参与判定，也不覆盖用户填写的值
 *   3. 分批期间把实例数改成别的值 → 静默回落为一次更新全部（用户可自行改回，不必提示环境不支持）
 *   4. 硬性不可用原因（切到从源码构建）→ 回落并提示「当前环境不支持分批发布」
 *   5. 锁定态下的输入框说明文案随发布方式出现 / 消失
 *
 * stub/mock：use-deploy（部署接口）、ReleaseStrategiesService、stores、Message、vue-i18n。
 */
import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useReleaseStrategy } from '~/pages/application/detail/deploy/use-release-strategy';

import type { ReleaseStrategyModel } from '~/pages/application/detail/deploy/use-release-strategy';

const mocks = vi.hoisted(() => ({
  listDeployRecords: vi.fn(),
  listLatestDeployRecords: vi.fn(),
  listStrategies: vi.fn(),
  message: vi.fn(),
}));

/** 与线上文案一致：直译 key 并处理 {0} 形式的插值 */
function interpolate(text: string, params?: unknown) {
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (raw, key: string) => {
    const value = Array.isArray(params) ? (params as unknown[])[Number(key)] : (params as Record<string, unknown>)[key];
    return value === undefined ? raw : String(value);
  });
}

vi.mock('vue-i18n', async importOriginal => ({
  ...(await importOriginal<object>()),
  useI18n: () => ({ t: interpolate, te: () => true }),
}));

vi.mock('bkui-vue', () => ({ Message: mocks.message }));

vi.mock('~/api/modules/v1/release-strategies', () => ({
  ReleaseStrategiesService: { listReleaseStrategies: mocks.listStrategies },
}));

vi.mock('~/pages/application/detail/deploy/use-deploy', () => ({
  useDeployAPIs: () => ({
    listDeployRecords: mocks.listDeployRecords,
    listLatestDeployRecords: mocks.listLatestDeployRecords,
  }),
}));

vi.mock('~/stores/app-detail', () => ({
  useAppDetail: () => ({ appID: 'app-a', appType: 'trpc' }),
}));
vi.mock('~/stores/space', () => ({
  useSpaceStore: () => ({ currentSpace: 'ws-1' }),
}));

interface SetupOptions {
  /** 当前生效实例数（props.effectiveReplicas） */
  currentReplicas?: MaybeRefOrGetter<number | undefined>;
  /** 本次是否「从源码构建」 */
  isBuildFromSource?: MaybeRefOrGetter<boolean>;
  /** 表单里的实例数初始值 */
  replicas?: number;
}

/** 在真实组件实例内运行 composable（onMounted / watch 都需要活跃实例），返回其对外 API 与表单模型。 */
function setup(options: SetupOptions = {}) {
  const model = reactive<ReleaseStrategyModel>({
    releaseMode: 'all',
    releaseStrategyID: '',
    replicas: options.replicas ?? 10,
  });
  let api!: ReturnType<typeof useReleaseStrategy>;
  // 普通对象而非 defineComponent：避免 vue/one-component-per-file，且本文件只需要一个空渲染宿主。
  const harnessComponent = {
    setup() {
      api = useReleaseStrategy({
        currentReplicas: options.currentReplicas,
        envName: 'prod',
        isAppModelApp: true,
        isBuildFromSource: options.isBuildFromSource ?? false,
        isFederationEnv: false,
        model,
      });
      return () => null;
    },
  };
  const wrapper = mount(harnessComponent as Component);
  return { api, model, wrapper };
}

const REPLICAS_CHANGED_REASON = '分批更新时不能修改实例数，请将实例数先改回 20';
const REPLICAS_LOCKED_HINT = '分批更新时不能修改实例数。如需调整，请选择「一次更新全部实例」，或发布完成后再扩缩容。';
const ENV_UNSUPPORTED_MESSAGE = '当前环境不支持分批发布，已重置为一次更新全部实例';

describe('useReleaseStrategy：实例数锁定与同步', () => {
  const wrappers: ReturnType<typeof mount>[] = [];

  /** 挂载并等待两项资格检查（主工作负载 / 首次部署）返回 */
  async function mountStrategy(options: SetupOptions = {}) {
    const result = setup(options);
    wrappers.push(result.wrapper);
    await flushPromises();
    return result;
  }

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listStrategies.mockResolvedValue([]);
    // 主工作负载为 GameDeployment + 已有部署记录：除实例数外的资格条件全部满足
    mocks.listDeployRecords.mockResolvedValue({ count: 1, results: [{ workloadKind: 'GameDeployment' }] });
    mocks.listLatestDeployRecords.mockResolvedValue({ hasDeployRecord: true, status: 'deployed' });
  });

  afterEach(() => {
    for (const wrapper of wrappers.splice(0)) wrapper.unmount();
  });

  it('当选分批发布时，应把表单实例数同步为当前生效值并锁定输入', async () => {
    const { api, model } = await mountStrategy({ currentReplicas: ref(20), replicas: 5 });

    // 改过实例数时分批入口先被拦下，提示直接带出要改回的目标值
    expect(api.canaryDisabledReason.value).toBe(REPLICAS_CHANGED_REASON);
    expect(api.replicasLocked.value).toBe(false);

    // UI 上该选项此时是禁用的；这里直接调用以验证组合层的兜底：一旦进入分批，实例数必须回到生效值，
    // 否则「旧实例数 + 分批策略」会一起提交给后端。
    api.handleChangeReleaseMode('canary');
    await nextTick();

    expect(model.replicas).toBe(20);
    expect(api.replicasLocked.value).toBe(true);
    // 同步回生效值后分批恢复可用
    expect(api.canaryDisabledReason.value).toBe('');
  });

  it('当生效实例数不可用（undefined / 0）时，不应参与判定也不覆盖表单值', async () => {
    const currentReplicas = ref<number | undefined>(undefined);
    const { api, model } = await mountStrategy({ currentReplicas, replicas: 7 });

    // 环境未部署 / 未取到期望副本数：无法判断是否改过实例数，交由后端兜底
    expect(api.canaryDisabledReason.value).toBe('');
    api.handleChangeReleaseMode('canary');
    await flushPromises();
    expect(model.replicas).toBe(7);

    // 接口可能回填 0，同样按「未知」处理
    currentReplicas.value = 0;
    await flushPromises();
    expect(api.canaryDisabledReason.value).toBe('');
    expect(model.replicas).toBe(7);
  });

  it('当分批期间实例数被改成与生效值不同时，应静默回落为一次更新全部实例', async () => {
    const { api, model } = await mountStrategy({ currentReplicas: ref(20), replicas: 20 });
    api.handleChangeReleaseMode('canary');
    await flushPromises();
    model.releaseStrategyID = 'strategy-1';

    // 输入框虽已禁用，但表单重置 / 程序化写入仍可能改到实例数
    model.replicas = 25;
    await flushPromises();

    expect(model.releaseMode).toBe('all');
    expect(model.releaseStrategyID).toBe('');
    // 属于用户可自行修正的临时状态，不应提示「当前环境不支持分批发布」
    expect(mocks.message).not.toHaveBeenCalled();
  });

  it('当已选分批后切到从源码构建时，应回落为一次更新全部并提示环境不支持', async () => {
    const isBuildFromSource = ref(false);
    const { api, model } = await mountStrategy({ currentReplicas: ref(20), isBuildFromSource, replicas: 20 });
    api.handleChangeReleaseMode('canary');
    await flushPromises();
    model.releaseStrategyID = 'strategy-1';

    // 构建部署不支持分批发布
    isBuildFromSource.value = true;
    await flushPromises();

    expect(model.releaseMode).toBe('all');
    expect(model.releaseStrategyID).toBe('');
    expect(mocks.message).toHaveBeenCalledWith({ theme: 'warning', message: ENV_UNSUPPORTED_MESSAGE });
  });

  it('当发布方式切换时，实例数输入框的锁定说明应同步出现与消失', async () => {
    const { api } = await mountStrategy({ currentReplicas: ref(20), replicas: 20 });
    expect(api.replicasLockedHint.value).toBe('');

    api.handleChangeReleaseMode('canary');
    await flushPromises();
    expect(api.replicasLocked.value).toBe(true);
    expect(api.replicasLockedHint.value).toBe(REPLICAS_LOCKED_HINT);

    api.handleChangeReleaseMode('all');
    await flushPromises();
    expect(api.replicasLocked.value).toBe(false);
    expect(api.replicasLockedHint.value).toBe('');
  });
});
