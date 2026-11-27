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

import { ref } from 'vue';
import type { MaybeRefOrGetter } from 'vue';

import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it } from 'vitest';
import { useTableSettings } from '~/composables/use-table-settings';
import { useUiStore } from '~/stores/ui';

const DEFAULT_COLUMNS = ['id', 'status', 'image', 'latestPublish', 'resources'];
const TABLE_ID = 'instance-table-v2';
const PREFERENCE_KEY = `table:${TABLE_ID}`;

function createSettings(options: { disabled?: MaybeRefOrGetter<string[]>; tableId?: string } = {}) {
  return useTableSettings(options.tableId ?? TABLE_ID, {
    defaultChecked: DEFAULT_COLUMNS,
    disabled: options.disabled ?? ['id'],
  });
}

beforeEach(() => {
  setActivePinia(createPinia());
});

describe('useTableSettings：列设置差异', () => {
  it('无偏好记录时按默认勾选展示，并强制并入不可隐藏列', () => {
    const { settings } = createSettings();
    expect(settings.value.checked).toEqual(DEFAULT_COLUMNS);
    expect(settings.value.disabled).toEqual(['id']);
  });

  it('记录为 hidden 的默认列不再展示', () => {
    useUiStore().setPreference(PREFERENCE_KEY, { hidden: ['image'], shown: [] });
    const { settings } = createSettings();
    expect(settings.value.checked).not.toContain('image');
    expect(settings.value.checked).toContain('status');
  });

  it('记录为 shown 的非默认列补进展示列表', () => {
    useUiStore().setPreference(PREFERENCE_KEY, { hidden: [], shown: ['age'] });
    const { settings } = createSettings();
    expect(settings.value.checked).toContain('age');
  });

  it('保存时取消默认列记为 hidden，勾选非默认列记为 shown', () => {
    const { handleSettingChange } = createSettings();
    handleSettingChange({ checked: ['id', 'image', 'latestPublish', 'resources', 'age'] });

    expect(useUiStore().preferences[PREFERENCE_KEY]).toEqual({ hidden: ['status'], shown: ['age'] });
  });

  it('不可隐藏列不落盘差异', () => {
    const { settings, handleSettingChange } = createSettings();
    expect(settings.value.checked).toContain('id');

    handleSettingChange({ checked: ['status', 'image', 'latestPublish', 'resources'] });

    expect(useUiStore().preferences[PREFERENCE_KEY]).toBeUndefined();
  });

  it('勾选状态与默认一致时清除该表记录', () => {
    useUiStore().setPreference(PREFERENCE_KEY, { hidden: [], shown: ['age'] });
    const { handleSettingChange } = createSettings();

    handleSettingChange({ checked: [...DEFAULT_COLUMNS] });

    expect(useUiStore().preferences[PREFERENCE_KEY]).toBeUndefined();
  });

  it('行高只记录 medium/mini，组件库默认 small 视为未个性化', () => {
    const { handleSettingChange } = createSettings();

    handleSettingChange({ checked: [...DEFAULT_COLUMNS], size: 'small' });
    expect(useUiStore().preferences[PREFERENCE_KEY]).toBeUndefined();

    handleSettingChange({ checked: [...DEFAULT_COLUMNS], size: 'medium' });
    expect(useUiStore().preferences[PREFERENCE_KEY]).toEqual({ hidden: [], shown: [], size: 'medium' });
  });

  it('同一 tableId 的另一个实例保存时保留已记录的列偏好', () => {
    useUiStore().setPreference(PREFERENCE_KEY, { hidden: ['image'], shown: [] });

    // 另一个实例读写同一份偏好：基线固定，保存时按同一基线重算也会得到相同的差异。
    const another = createSettings();
    expect(another.settings.value.checked).not.toContain('image');

    another.handleSettingChange({ checked: another.settings.value.checked });

    expect(useUiStore().preferences[PREFERENCE_KEY]).toEqual({ hidden: ['image'], shown: [] });
  });

  it('不同 tableId 的列设置互不影响', () => {
    useUiStore().setPreference('table:deploy-overview', { hidden: ['type'], shown: [] });
    const { settings } = createSettings();

    expect(settings.value.checked).toContain('status');
    expect(useUiStore().preferences['table:deploy-overview']).toEqual({ hidden: ['type'], shown: [] });
  });

  it('不可操作列不在面板里，保存时不会被算成隐藏', () => {
    // 非开发模式环境：热更新列不渲染，面板回传的 checked 不含它
    const { handleSettingChange } = createSettings({ disabled: ['id', 'latestPublish'] });

    handleSettingChange({ checked: ['id', 'image', 'resources'] });

    expect(useUiStore().preferences[PREFERENCE_KEY]).toEqual({ hidden: ['status'], shown: [] });
  });

  it('记录里已有的不可操作列差异被原样保留', () => {
    useUiStore().setPreference(PREFERENCE_KEY, { hidden: ['latestPublish'], shown: [] });

    const { handleSettingChange } = createSettings({ disabled: ['id', 'latestPublish'] });
    handleSettingChange({ checked: ['id', 'status', 'image', 'resources'] });

    expect(useUiStore().preferences[PREFERENCE_KEY]).toEqual({ hidden: ['latestPublish'], shown: [] });
  });

  it('不可操作列支持响应式，切换后同一份记录语义随之变化', () => {
    useUiStore().setPreference(PREFERENCE_KEY, { hidden: ['latestPublish'], shown: [] });
    const devMode = ref(false);
    const { settings, handleSettingChange } = createSettings({
      disabled: () => (devMode.value ? ['id'] : ['id', 'latestPublish']),
    });

    // 非开发模式：该列被强制并入 checked，保存保留既有 hidden
    expect(settings.value.checked).toContain('latestPublish');
    handleSettingChange({ checked: settings.value.checked });
    expect(useUiStore().preferences[PREFERENCE_KEY]).toEqual({ hidden: ['latestPublish'], shown: [] });

    // 开发模式：回到由 hidden 决定显隐
    devMode.value = true;
    expect(settings.value.checked).not.toContain('latestPublish');
    expect(settings.value.disabled).toEqual(['id']);
  });
});
