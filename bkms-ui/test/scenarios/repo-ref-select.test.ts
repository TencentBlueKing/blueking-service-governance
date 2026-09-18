import { nextTick, ref } from 'vue';

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
 * 场景级测试：RepoRefSelect Input 路径（路径清单见 docs/vitest/guides/TEST_SCENARIOS_ROUTES.md S19）
 *
 * 覆盖四组用户可感知行为：trim 立即同步 / trim 后无变化不确认 / 实际变化防抖确认 / 空串不确认 → V = 4
 * 说明：下拉（Select）路径依赖代码仓库接口，标注待补；本文件聚焦流水线手动输入（无 repositoryId）。
 */
import { type VueWrapper, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

/** 桩掉下拉数据 hook，本轮只测 Input 路径的 trim / 防抖确认；字段与真实 return 对齐 */
vi.mock('~/components/repo-ref-select/use-repo-ref-select', () => ({
  useRepoRefSelect: () => ({
    groups: ref([]),
    optionsLoading: ref(false),
    handleSearch: vi.fn(),
    refresh: vi.fn(),
    reset: vi.fn(),
    onDropdownOpen: vi.fn(),
    onDropdownClose: vi.fn(),
    ensureOptionsLoaded: vi.fn(),
  }),
}));

vi.mock('vue-i18n', async () => (await import('../helpers/mock-i18n')).i18nMockFactory());

/** Input 路径用轻量 stub，避免依赖真实 bkui-vue Input 内部实现 */
vi.mock('bkui-vue', async () => import('../stubs/bkui-vue-lite'));

import RepoRefSelect from '~/components/repo-ref-select/repo-ref-select.vue';

/** 当前用例挂载的 wrapper，供 afterEach unmount（触发 onBeforeUnmount 取消防抖） */
let activeWrapper: undefined | VueWrapper;

/**
 * 挂载流水线模式（无 repositoryId → Input）并同步 v-model。
 */
function mountPipelineInput(initialBranch = 'master') {
  const state = { modelValue: initialBranch };
  const wrapper = mount(RepoRefSelect, {
    props: {
      workspaceId: 'ws-test',
      repositoryId: '',
      modelValue: state.modelValue,
      'onUpdate:modelValue': (value: string) => {
        state.modelValue = value;
        void wrapper.setProps({ modelValue: value });
      },
    },
  });
  activeWrapper = wrapper;
  return { wrapper, state };
}

/** 向 Input 桩写入原始字符串，触发 handleInputChange */
async function typeBranch(wrapper: VueWrapper, raw: string) {
  const input = wrapper.find('[data-testid="repo-ref-input-stub"]');
  await input.setValue(raw);
  await nextTick();
}

describe('RepoRefSelect：Input trim 与防抖确认', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    // 卸载以执行 onBeforeUnmount 中的 debouncedInputConfirm.cancel()
    activeWrapper?.unmount();
    activeWrapper = undefined;
    vi.useRealTimers();
  });

  it('当用户输入带首尾空格的分支名时，应立即同步去空格后的绑定值', async () => {
    const { wrapper } = mountPipelineInput('master');

    await typeBranch(wrapper, '  main  ');

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['main']);
    expect(wrapper.props('modelValue')).toBe('main');
  });

  it('当用户输入仅含首尾空格且 trim 后与当前值相同时，防抖后不应触发分支确认', async () => {
    const { wrapper } = mountPipelineInput('master');

    await typeBranch(wrapper, '  master  ');
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['master']);

    // 源码防抖 500ms，推进 2 倍确保触发（避免与 debounce 边界等值耦合）
    await vi.advanceTimersByTimeAsync(1000);
    await nextTick();

    expect(wrapper.emitted('branchCommit')).toBeUndefined();
  });

  it('当用户输入实际变化的分支名时，防抖结束后应触发分支确认', async () => {
    const { wrapper } = mountPipelineInput('master');

    await typeBranch(wrapper, '  feature/a  ');
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['feature/a']);
    expect(wrapper.emitted('branchCommit')).toBeUndefined();

    // 源码防抖 500ms，推进 2 倍确保触发（避免与 debounce 边界等值耦合）
    await vi.advanceTimersByTimeAsync(1000);
    await nextTick();

    expect(wrapper.emitted('branchCommit')?.at(-1)).toEqual(['feature/a']);
  });

  it('当用户输入仅空格（trim 后为空）时，防抖结束后仍不应触发分支确认', async () => {
    const { wrapper } = mountPipelineInput('master');

    await typeBranch(wrapper, '   ');
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['']);

    // 源码防抖 500ms，推进 2 倍确保触发（避免与 debounce 边界等值耦合）
    await vi.advanceTimersByTimeAsync(1000);
    await nextTick();

    expect(wrapper.emitted('branchCommit')).toBeUndefined();
  });
});
