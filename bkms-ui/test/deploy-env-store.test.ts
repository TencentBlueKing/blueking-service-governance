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

import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useDeployEnvStore } from '../src/stores/deploy-env';

const STORAGE_KEY = 'bkms_deploy_env_app_selections';

describe('deploy-env store', () => {
  beforeEach(() => {
    localStorage.clear();
    setActivePinia(createPinia());
    vi.restoreAllMocks();
    let now = 1;
    vi.spyOn(Date, 'now').mockImplementation(() => now++);
  });

  it('keeps app environment selections isolated by scope key', () => {
    const store = useDeployEnvStore();

    store.updateAppEnvSelection('space-a:app-a', {
      mode: 'multi',
      selectedEnvs: ['dev', 'test'],
    });
    store.updateAppEnvSelection('space-a:app-b', {
      mode: 'single',
      selectedEnvs: ['prod'],
    });

    expect(store.getAppEnvSelection('space-a:app-a')).toMatchObject({
      mode: 'multi',
      selectedEnvs: ['dev', 'test'],
    });
    expect(store.getAppEnvSelection('space-a:app-b')).toMatchObject({
      mode: 'single',
      selectedEnvs: ['prod'],
    });
  });

  it('keeps only the 30 most recently updated app selections', () => {
    const store = useDeployEnvStore();

    for (let index = 0; index < 31; index++) {
      store.updateAppEnvSelection(`space-a:app-${index}`, {
        mode: 'multi',
        selectedEnvs: [`env-${index}`],
      });
    }

    expect(Object.keys(store.appEnvSelections)).toHaveLength(30);
    expect(store.getAppEnvSelection('space-a:app-0')).toBeUndefined();
    expect(store.getAppEnvSelection('space-a:app-30')).toMatchObject({
      selectedEnvs: ['env-30'],
    });
  });

  it('restores selections from localStorage after the store is recreated', () => {
    const store = useDeployEnvStore();

    store.updateAppEnvSelection('space-a:app-a', {
      mode: 'multi',
      selectedEnvs: ['dev', 'test'],
    });

    setActivePinia(createPinia());
    const restoredStore = useDeployEnvStore();

    expect(JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')).toMatchObject({
      'space-a:app-a': {
        mode: 'multi',
        selectedEnvs: ['dev', 'test'],
      },
    });
    expect(restoredStore.getAppEnvSelection('space-a:app-a')).toMatchObject({
      mode: 'multi',
      selectedEnvs: ['dev', 'test'],
    });
  });
});
